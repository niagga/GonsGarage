package fiscal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

const (
	// DefaultWorkerLease is the bounded lease duration for outbox claims.
	DefaultWorkerLease = 30 * time.Second
	// DefaultWorkerPollInterval is the idle poll cadence between claim attempts.
	DefaultWorkerPollInterval = time.Second
	// DefaultWorkerOwner is used when no owner identity is configured.
	DefaultWorkerOwner = "fiscal-worker"
)

// WorkerConfig bounds leases and optional test hooks.
type WorkerConfig struct {
	Owner        string
	Lease        time.Duration
	PollInterval time.Duration
	Environment  string
	// AfterBeginHook runs after BeginLegalCall commits and before the provider call (tests).
	AfterBeginHook func()
}

// Normalize applies bounded defaults for lease settings.
func (c WorkerConfig) Normalize() WorkerConfig {
	if c.Lease <= 0 {
		c.Lease = DefaultWorkerLease
	}
	if c.PollInterval <= 0 {
		c.PollInterval = DefaultWorkerPollInterval
	}
	if c.Owner == "" {
		c.Owner = DefaultWorkerOwner
	}
	if c.Environment == "" {
		c.Environment = "default"
	}
	return c
}

// Worker processes fiscal outbox events outside HTTP requests.
//
// Transaction boundaries (design claim/pre-call/gateway/completion):
//  1. ClaimNext — short claim tx (SKIP LOCKED + lease)
//  2. BeginLegalCall — pre-call tx (state + started attempt)
//  3. Provider gateway call — outside any DB transaction
//  4. CompleteCall / RecoverStaleAttempt — completion tx under lease fencing
type Worker struct {
	outbox    ports.FiscalOutboxRepository
	provider  ports.FiscalProvider
	artifacts *ArtifactService
	cfg       WorkerConfig
}

// NewWorker builds a lease-fenced fiscal worker.
func NewWorker(outbox ports.FiscalOutboxRepository, provider ports.FiscalProvider, cfg WorkerConfig) *Worker {
	return &Worker{outbox: outbox, provider: provider, cfg: cfg.Normalize()}
}

// WithArtifactService wires immutable PDF archive/recovery (WU6).
func (w *Worker) WithArtifactService(svc *ArtifactService) *Worker {
	if w != nil {
		w.artifacts = svc
	}
	return w
}

// Run polls until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.ProcessOnce(ctx); err != nil && !errors.Is(err, ports.ErrFiscalOutboxEmpty) {
			if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ProcessOnce claims at most one event and processes it.
func (w *Worker) ProcessOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	claimed, err := w.outbox.ClaimNext(ctx, w.cfg.Owner, w.cfg.Lease)
	if err != nil {
		return err
	}
	return w.ProcessClaim(ctx, claimed)
}

// ProcessClaim handles an already-claimed event (tests and reclaim paths).
func (w *Worker) ProcessClaim(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	if claimed == nil {
		return ports.ErrFiscalOutboxEmpty
	}
	if claimed.HasStartedAttempt {
		return w.recoverStale(ctx, claimed)
	}
	switch claimed.Event.EventType {
	case ports.FiscalOutboxIssue:
		return w.processIssue(ctx, claimed)
	case ports.FiscalOutboxReconcileIssue:
		return w.processReconcileIssue(ctx, claimed)
	case ports.FiscalOutboxVoid:
		return w.processVoid(ctx, claimed)
	case ports.FiscalOutboxReconcileVoid:
		return w.processReconcileVoid(ctx, claimed)
	case ports.FiscalOutboxRecoverArtifact:
		return w.processArtifactRecovery(ctx, claimed)
	default:
		return fmt.Errorf("unsupported outbox event type %q", claimed.Event.EventType)
	}
}

func (w *Worker) recoverStale(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	unknown := domain.FiscalDocumentStateOutcomeUnknown
	if claimed.Event.EventType == ports.FiscalOutboxVoid || claimed.Document.State == domain.FiscalDocumentStateVoidPending {
		unknown = domain.FiscalDocumentStateVoidOutcomeUnknown
	}
	return w.outbox.RecoverStaleAttempt(ctx, ports.RecoverStaleAttemptCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: claimed.StartedAttemptID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		UnknownState: unknown, SafeError: "started attempt unfinished after lease expiry",
	})
}

func (w *Worker) processIssue(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	att, err := w.begin(ctx, claimed, ports.FiscalOutboxIssue, domain.FiscalDocumentStateDispatching)
	if err != nil {
		return err
	}
	if w.cfg.AfterBeginHook != nil {
		w.cfg.AfterBeginHook()
	}
	res, err := w.provider.Issue(ctx, ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
			OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
		},
		DocumentID: claimed.Document.ID, Payload: claimed.CanonicalBytes,
	})
	ref, num := "", ""
	var issuedAt *time.Time
	if err == nil {
		ref, num, issuedAt = res.ProviderReference, res.ProviderNumber, res.IssuedAt
	}
	return w.completeFromProvider(ctx, claimed, att.ID, err, ref, num, issuedAt, nil, false)
}

func (w *Worker) processReconcileIssue(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	att, err := w.begin(ctx, claimed, ports.FiscalOutboxReconcileIssue, claimed.Document.State)
	if err != nil {
		return err
	}
	if w.cfg.AfterBeginHook != nil {
		w.cfg.AfterBeginHook()
	}
	res, err := w.provider.Reconcile(ctx, ports.FiscalReconcileRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
			OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
		},
		DocumentID: claimed.Document.ID, ProviderReference: claimed.Document.ProviderReference,
	})
	if err != nil {
		return w.completeFromProvider(ctx, claimed, att.ID, err, "", "", nil, nil, true)
	}
	next := domain.FiscalDocumentStateOutcomeUnknown
	if res.Matched {
		next = domain.FiscalDocumentStateIssued
	}
	return w.outbox.CompleteCall(ctx, ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: ports.FiscalAttemptSucceeded, Definitive: res.Matched,
		ProviderReference: res.ProviderReference, NextDocumentState: next, IssuedAt: res.ResolvedAt,
	})
}

func (w *Worker) processVoid(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	att, err := w.begin(ctx, claimed, ports.FiscalOutboxVoid, domain.FiscalDocumentStateVoidPending)
	if err != nil {
		return err
	}
	if w.cfg.AfterBeginHook != nil {
		w.cfg.AfterBeginHook()
	}
	res, err := w.provider.Void(ctx, ports.FiscalVoidRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
			OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
		},
		DocumentID: claimed.Document.ID, ProviderReference: claimed.Document.ProviderReference,
	})
	ref := ""
	var voidedAt *time.Time
	if err == nil {
		ref, voidedAt = res.ProviderReference, res.VoidedAt
	}
	return w.completeFromProvider(ctx, claimed, att.ID, err, ref, "", nil, voidedAt, false)
}

func (w *Worker) processReconcileVoid(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	att, err := w.begin(ctx, claimed, ports.FiscalOutboxReconcileVoid, claimed.Document.State)
	if err != nil {
		return err
	}
	res, err := w.provider.Reconcile(ctx, ports.FiscalReconcileRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
			OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
		},
		DocumentID: claimed.Document.ID, ProviderReference: claimed.Document.ProviderReference,
	})
	if err != nil {
		return w.completeFromProvider(ctx, claimed, att.ID, err, "", "", nil, nil, true)
	}
	next := domain.FiscalDocumentStateVoidOutcomeUnknown
	if res.Matched {
		next = domain.FiscalDocumentStateVoided
	}
	return w.outbox.CompleteCall(ctx, ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: ports.FiscalAttemptSucceeded, Definitive: res.Matched,
		ProviderReference: res.ProviderReference, NextDocumentState: next, VoidedAt: res.ResolvedAt,
	})
}

func (w *Worker) processArtifactRecovery(ctx context.Context, claimed *ports.ClaimedOutboxEvent) error {
	att, err := w.begin(ctx, claimed, ports.FiscalOutboxRecoverArtifact, claimed.Document.State)
	if err != nil {
		return err
	}
	artID := uuid.Nil
	if claimed.Event.ArtifactID != nil {
		artID = *claimed.Event.ArtifactID
	}
	if w.artifacts != nil {
		classification := domain.FiscalArtifactClassificationLegal
		if claimed.Document.ProviderKey == "mock" {
			classification = domain.FiscalArtifactClassificationMock
		}
		_, err = w.artifacts.RecoverFromProvider(ctx, RecoverArtifactCommand{
			ArtifactID: artID, DocumentID: claimed.Document.ID, SourceInvoiceID: claimed.Document.SourceInvoiceID,
			Environment: w.cfg.Environment, Classification: classification,
			ProviderReference: claimed.Document.ProviderReference, Provider: w.provider,
			ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
			OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
		})
	} else {
		_, err = w.provider.FetchArtifact(ctx, ports.FiscalFetchArtifactRequest{
			FiscalOperationRequest: ports.FiscalOperationRequest{
				ConnectionID: connectionID(claimed), ProviderKey: claimed.Document.ProviderKey,
				OperationKey: claimed.Event.OperationKey, CorrelationKey: claimed.Event.ID.String(),
			},
			ArtifactID: artID,
		})
	}
	status := ports.FiscalAttemptSucceeded
	if err != nil {
		status = ports.FiscalAttemptFailed
	}
	class, definitive, code, msg := classifyProviderError(err)
	return w.outbox.CompleteCall(ctx, ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: status, Classification: class, Definitive: definitive,
		SafeProviderCode: code, SafeProviderMsg: RedactSecrets(msg),
		Diagnostics:       map[string]any{"message": RedactSecrets(msg)},
		NextDocumentState: claimed.Document.State, LastSafeError: RedactSecrets(msg),
	})
}

func (w *Worker) begin(ctx context.Context, claimed *ports.ClaimedOutboxEvent, op ports.FiscalOutboxEventType, next domain.FiscalDocumentState) (*ports.FiscalProviderAttempt, error) {
	sum := sha256.Sum256(claimed.CanonicalBytes)
	return w.outbox.BeginLegalCall(ctx, ports.BeginLegalCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		ProviderKey: claimed.Document.ProviderKey, Operation: op, OperationKey: claimed.Event.OperationKey,
		RequestSHA256: hex.EncodeToString(sum[:]), NextDocumentState: next,
	})
}

func (w *Worker) completeFromProvider(
	ctx context.Context,
	claimed *ports.ClaimedOutboxEvent,
	attemptID uuid.UUID,
	providerErr error,
	providerRef, providerNumber string,
	issuedAt, voidedAt *time.Time,
	reconcile bool,
) error {
	if providerErr == nil {
		next := domain.FiscalDocumentStateIssued
		if claimed.Event.EventType == ports.FiscalOutboxVoid {
			next = domain.FiscalDocumentStateVoided
		}
		return w.outbox.CompleteCall(ctx, ports.CompleteOutboxCallCommand{
			EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: attemptID,
			LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
			AttemptStatus: ports.FiscalAttemptSucceeded, Definitive: true,
			ProviderReference: providerRef, ProviderNumber: providerNumber,
			NextDocumentState: next, IssuedAt: issuedAt, VoidedAt: voidedAt,
		})
	}

	class, definitive, code, msg := classifyProviderError(providerErr)
	msg = RedactSecrets(msg)
	next, attemptStatus := mapFailureState(claimed.Event.EventType, class, definitive, reconcile)
	return w.outbox.CompleteCall(ctx, ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: attemptID,
		LeaseOwner: claimed.Event.LeaseOwner, LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: attemptStatus, Classification: class, Definitive: definitive,
		SafeProviderCode: code, SafeProviderMsg: msg,
		Diagnostics:       map[string]any{"message": msg},
		NextDocumentState: next, LastSafeError: msg,
	})
}

func mapFailureState(eventType ports.FiscalOutboxEventType, class ports.FiscalErrorClass, definitive, reconcile bool) (domain.FiscalDocumentState, ports.FiscalAttemptStatus) {
	if !definitive || class == ports.FiscalErrorClassAmbiguous {
		if eventType == ports.FiscalOutboxVoid || eventType == ports.FiscalOutboxReconcileVoid {
			return domain.FiscalDocumentStateVoidOutcomeUnknown, ports.FiscalAttemptUnknown
		}
		return domain.FiscalDocumentStateOutcomeUnknown, ports.FiscalAttemptUnknown
	}
	switch class {
	case ports.FiscalErrorClassValidation, ports.FiscalErrorClassPermanent:
		if eventType == ports.FiscalOutboxVoid {
			return domain.FiscalDocumentStateIssued, ports.FiscalAttemptFailed
		}
		return domain.FiscalDocumentStateRejected, ports.FiscalAttemptFailed
	case ports.FiscalErrorClassAuthorization, ports.FiscalErrorClassUnauthorized:
		return domain.FiscalDocumentStateConnectionActionNeeded, ports.FiscalAttemptFailed
	case ports.FiscalErrorClassTransient, ports.FiscalErrorClassRateLimit:
		if reconcile {
			return domain.FiscalDocumentStateOutcomeUnknown, ports.FiscalAttemptFailed
		}
		if eventType == ports.FiscalOutboxVoid {
			return domain.FiscalDocumentStateIssued, ports.FiscalAttemptFailed
		}
		return domain.FiscalDocumentStateRetryableFailure, ports.FiscalAttemptFailed
	default:
		return domain.FiscalDocumentStateOutcomeUnknown, ports.FiscalAttemptUnknown
	}
}

func classifyProviderError(err error) (ports.FiscalErrorClass, bool, string, string) {
	if err == nil {
		return "", true, "", ""
	}
	var pe *ports.FiscalProviderError
	if errors.As(err, &pe) && pe != nil {
		return pe.Class, pe.DefinitiveNonAcceptance, pe.Code, pe.Message
	}
	return ports.FiscalErrorClassAmbiguous, false, "provider_error", err.Error()
}

var secretPattern = regexp.MustCompile(`(?i)(bearer\s+\S+|token[=:]\s*\S+|refresh[=:]\s*\S+|authorization[=:]\s*\S+)`)

// RedactSecrets strips credential-bearing fragments from diagnostic text.
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}
	return secretPattern.ReplaceAllString(s, "[REDACTED]")
}

func connectionID(claimed *ports.ClaimedOutboxEvent) uuid.UUID {
	if claimed.Document.ConnectionID == nil {
		return uuid.Nil
	}
	return *claimed.Document.ConnectionID
}
