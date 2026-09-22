package fiscal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// FinalizationService owns privileged freeze/retry/reconcile/void commands.
type FinalizationService struct {
	repo     ports.FiscalRepository
	invoices ports.InvoiceRepository
	users    ports.UserRepository
}

// NewFinalizationService builds the manager/admin finalization service.
func NewFinalizationService(repo ports.FiscalRepository, invoices ports.InvoiceRepository, users ports.UserRepository) *FinalizationService {
	return &FinalizationService{repo: repo, invoices: invoices, users: users}
}

// FinalizeRequest freezes a draft and enqueues issue sequence 1.
type FinalizeRequest struct {
	DocumentID      uuid.UUID
	ExpectedVersion int64
	ProviderKey     string
	ConnectionID    uuid.UUID
	PolicyVersionID uuid.UUID
	IssuerProfileID uuid.UUID
}

// ActionRequest is shared by retry/reconcile/void.
type ActionRequest struct {
	DocumentID      uuid.UUID
	ExpectedVersion int64
}

// Finalize freezes one intent atomically; repeats of an already-pending document are idempotent.
func (s *FinalizationService) Finalize(ctx context.Context, actorID uuid.UUID, req FinalizeRequest) (*ports.FinalizeDraftResult, error) {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return nil, err
	}
	agg, err := s.repo.GetAggregate(ctx, req.DocumentID)
	if err != nil {
		return nil, err
	}
	if agg.Document.State == domain.FiscalDocumentStatePending {
		return &ports.FinalizeDraftResult{Document: agg.Document.Frozen(), Sequence: 1}, nil
	}
	if agg.Document.State != domain.FiscalDocumentStateDraft {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	if req.ProviderKey == "" || req.ConnectionID == uuid.Nil {
		return nil, ports.ErrFiscalNotReady
	}
	intentKey := uuid.New()
	issueKey := "fiscal:issue:" + intentKey.String()
	frozenAt := time.Now().UTC()
	return s.repo.FinalizeDraft(ctx, ports.FinalizeDraftCommand{
		DocumentID:      req.DocumentID,
		ExpectedVersion: req.ExpectedVersion,
		ActorID:         actorID,
		ProviderKey:     req.ProviderKey,
		ConnectionID:    req.ConnectionID,
		IntentKey:       intentKey,
		IssueOpKey:      issueKey,
		FrozenAt:        frozenAt,
		PolicyVersionID: req.PolicyVersionID,
		IssuerProfileID: req.IssuerProfileID,
		Issuer:          agg.Snapshot.Issuer,
		Customer:        agg.Snapshot.Customer,
		BillingAddress:  agg.Snapshot.BillingAddress,
		Currency:        agg.Snapshot.Currency,
		GrossTotal:      agg.Snapshot.GrossTotal,
		DiscountTotal:   agg.Snapshot.DiscountTotal,
		NetTotal:        agg.Snapshot.NetTotal,
		TaxTotal:        agg.Snapshot.TaxTotal,
		RoundingAdj:     agg.Snapshot.RoundingAdj,
		PayableTotal:    agg.Snapshot.PayableTotal,
		CanonicalBytes:  freezeCanonical(agg),
		CanonicalSHA256: freezeSHA(agg),
		Lines:           agg.Snapshot.Lines,
	})
}

// Retry enqueues the same issue operation key from recoverable states.
func (s *FinalizationService) Retry(ctx context.Context, actorID uuid.UUID, req ActionRequest) (*domain.FiscalDocument, error) {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return nil, err
	}
	agg, err := s.repo.GetAggregate(ctx, req.DocumentID)
	if err != nil {
		return nil, err
	}
	if !agg.Document.CanAct(domain.FiscalActorRoleManager, false, domain.FiscalDocumentActionRetry) {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	return s.repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID:      req.DocumentID,
		ExpectedVersion: req.ExpectedVersion,
		ActorID:         actorID,
		EventType:       "issue",
		OperationKey:    agg.Document.IssueOperationKey,
		NextState:       domain.FiscalDocumentStatePending,
	})
}

// Reconcile enqueues reconciliation for unknown outcomes only.
func (s *FinalizationService) Reconcile(ctx context.Context, actorID uuid.UUID, req ActionRequest) (*domain.FiscalDocument, error) {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return nil, err
	}
	agg, err := s.repo.GetAggregate(ctx, req.DocumentID)
	if err != nil {
		return nil, err
	}
	if !agg.Document.CanAct(domain.FiscalActorRoleManager, false, domain.FiscalDocumentActionReconcile) {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	eventType := "reconcile_issue"
	if agg.Document.State == domain.FiscalDocumentStateVoidOutcomeUnknown {
		eventType = "reconcile_void"
	}
	return s.repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID:      req.DocumentID,
		ExpectedVersion: req.ExpectedVersion,
		ActorID:         actorID,
		EventType:       eventType,
		OperationKey:    agg.Document.IssueOperationKey,
		NextState:       agg.Document.State,
	})
}

// Void enqueues a void for an issued document, fixing one void operation key.
func (s *FinalizationService) Void(ctx context.Context, actorID uuid.UUID, req ActionRequest) (*domain.FiscalDocument, error) {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return nil, err
	}
	agg, err := s.repo.GetAggregate(ctx, req.DocumentID)
	if err != nil {
		return nil, err
	}
	if !agg.Document.CanAct(domain.FiscalActorRoleManager, false, domain.FiscalDocumentActionVoid) {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	voidKey := agg.Document.VoidOperationKey
	if voidKey == "" {
		if agg.Document.IntentKey == nil {
			return nil, ports.ErrFiscalNotReady
		}
		voidKey = "fiscal:void:" + agg.Document.IntentKey.String()
	}
	return s.repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID:       req.DocumentID,
		ExpectedVersion:  req.ExpectedVersion,
		ActorID:          actorID,
		EventType:        "void",
		OperationKey:     voidKey,
		NextState:        domain.FiscalDocumentStateVoidPending,
		VoidOperationKey: voidKey,
	})
}

func (s *FinalizationService) requireManager(ctx context.Context, actorID uuid.UUID) (domain.FiscalActorRole, error) {
	u, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", domain.ErrUserNotFound
	}
	switch u.Role {
	case domain.RoleManager:
		return domain.FiscalActorRoleManager, nil
	case domain.RoleAdmin:
		return domain.FiscalActorRoleAdmin, nil
	default:
		return "", domain.ErrUnauthorizedAccess
	}
}

func freezeCanonical(agg *ports.FiscalDocumentAggregate) []byte {
	if len(agg.Snapshot.CanonicalBytes) > 0 {
		return append([]byte(nil), agg.Snapshot.CanonicalBytes...)
	}
	return []byte(fmt.Sprintf(`{"documentId":%q,"currency":%q,"payable":%q}`, agg.Document.ID, agg.Snapshot.Currency, agg.Snapshot.PayableTotal))
}

func freezeSHA(agg *ports.FiscalDocumentAggregate) string {
	if agg.Snapshot.CanonicalSHA256 != "" {
		return agg.Snapshot.CanonicalSHA256
	}
	sum := sha256.Sum256(freezeCanonical(agg))
	return hex.EncodeToString(sum[:])
}
