package fiscal_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

// --- fakes ---

type fakeOutboxRepo struct {
	mu sync.Mutex

	claimQueue []*ports.ClaimedOutboxEvent
	claims     []string

	beginCalls    []ports.BeginLegalCallCommand
	recoverCalls  []ports.RecoverStaleAttemptCommand
	completeCalls []ports.CompleteOutboxCallCommand

	beginErr    error
	completeErr error
	recoverErr  error
	loseLeaseOn string // complete EventID string → ErrFiscalLeaseLost

	attempts map[uuid.UUID]*ports.FiscalProviderAttempt
}

func newFakeOutbox() *fakeOutboxRepo {
	return &fakeOutboxRepo{attempts: map[uuid.UUID]*ports.FiscalProviderAttempt{}}
}

func (f *fakeOutboxRepo) ClaimNext(_ context.Context, owner string, _ time.Duration) (*ports.ClaimedOutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claims = append(f.claims, owner)
	if len(f.claimQueue) == 0 {
		return nil, ports.ErrFiscalOutboxEmpty
	}
	ev := f.claimQueue[0]
	f.claimQueue = f.claimQueue[1:]
	ev.Event.LeaseOwner = owner
	ev.Event.Status = ports.FiscalOutboxLeased
	return ev, nil
}

func (f *fakeOutboxRepo) BeginLegalCall(_ context.Context, cmd ports.BeginLegalCallCommand) (*ports.FiscalProviderAttempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.beginCalls = append(f.beginCalls, cmd)
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	att := &ports.FiscalProviderAttempt{
		ID: uuid.New(), FiscalDocumentID: cmd.DocumentID, OutboxEventID: cmd.EventID,
		ProviderKey: cmd.ProviderKey, Operation: cmd.Operation, OperationKey: cmd.OperationKey,
		AttemptNo: 1, Status: ports.FiscalAttemptStarted, RequestSHA256: cmd.RequestSHA256,
		StartedAt: time.Now().UTC(),
	}
	f.attempts[att.ID] = att
	return att, nil
}

func (f *fakeOutboxRepo) RecoverStaleAttempt(_ context.Context, cmd ports.RecoverStaleAttemptCommand) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recoverCalls = append(f.recoverCalls, cmd)
	return f.recoverErr
}

func (f *fakeOutboxRepo) CompleteCall(_ context.Context, cmd ports.CompleteOutboxCallCommand) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.completeCalls = append(f.completeCalls, cmd)
	if f.loseLeaseOn != "" && cmd.EventID.String() == f.loseLeaseOn {
		return ports.ErrFiscalLeaseLost
	}
	return f.completeErr
}

type recordingProvider struct {
	mu         sync.Mutex
	issues     int32
	reconciles int32
	voids      int32
	fetches    int32

	issueErr     error
	reconcileRes ports.FiscalReconcileResult
	reconcileErr error
	voidErr      error
	issueResult  ports.FiscalIssueResult
	lastIssueKey string
	lastReconKey string
}

func (p *recordingProvider) VerifyConnection(context.Context, ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	return ports.FiscalVerifyConnectionResult{}, nil
}
func (p *recordingProvider) Issue(_ context.Context, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	atomic.AddInt32(&p.issues, 1)
	p.mu.Lock()
	p.lastIssueKey = req.OperationKey
	p.mu.Unlock()
	if p.issueErr != nil {
		return ports.FiscalIssueResult{}, p.issueErr
	}
	if p.issueResult.ProviderReference == "" {
		return ports.FiscalIssueResult{
			FiscalOperationResponse: ports.FiscalOperationResponse{
				ProviderKey: "mock", OperationKey: req.OperationKey, ProviderReference: "MOCK-REF",
			},
			DocumentID: req.DocumentID, ProviderNumber: "FT 1",
		}, nil
	}
	return p.issueResult, nil
}
func (p *recordingProvider) Reconcile(_ context.Context, req ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	atomic.AddInt32(&p.reconciles, 1)
	p.mu.Lock()
	p.lastReconKey = req.OperationKey
	p.mu.Unlock()
	if p.reconcileErr != nil {
		return ports.FiscalReconcileResult{}, p.reconcileErr
	}
	return p.reconcileRes, nil
}
func (p *recordingProvider) Void(context.Context, ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	atomic.AddInt32(&p.voids, 1)
	if p.voidErr != nil {
		return ports.FiscalVoidResult{}, p.voidErr
	}
	now := time.Now().UTC()
	return ports.FiscalVoidResult{VoidedAt: &now}, nil
}
func (p *recordingProvider) FetchArtifact(context.Context, ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	atomic.AddInt32(&p.fetches, 1)
	return ports.FiscalFetchArtifactResult{}, nil
}

func pendingClaim(eventType ports.FiscalOutboxEventType, state domain.FiscalDocumentState) *ports.ClaimedOutboxEvent {
	docID := uuid.New()
	connID := uuid.New()
	intent := uuid.New()
	token := uuid.New()
	exp := time.Now().UTC().Add(30 * time.Second)
	opKey := "fiscal:issue:" + intent.String()
	return &ports.ClaimedOutboxEvent{
		Event: ports.FiscalOutboxEvent{
			ID: uuid.New(), FiscalDocumentID: docID, EventType: eventType,
			OperationKey: opKey, SequenceNo: 1, Status: ports.FiscalOutboxLeased,
			LeaseToken: token, LeaseExpiresAt: &exp, ClaimCount: 1,
		},
		Document: domain.FiscalDocument{
			ID: docID, State: state, ProviderKey: "mock", ConnectionID: &connID,
			IntentKey: &intent, IssueOperationKey: opKey, Version: 2,
		},
		CanonicalBytes: []byte(`{"ok":true}`),
	}
}

func TestWorker_StartedAttemptBeforeProviderCall(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	var sawBeginBeforeIssue atomic.Bool
	origIssue := provider.issues
	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{
		Owner: "worker-a", Lease: 30 * time.Second,
		AfterBeginHook: func() {
			if atomic.LoadInt32(&provider.issues) == origIssue {
				sawBeginBeforeIssue.Store(true)
			}
		},
	})

	err := worker.ProcessOnce(context.Background())
	require.NoError(t, err)
	require.True(t, sawBeginBeforeIssue.Load(), "BeginLegalCall must commit before Issue")
	require.Len(t, outbox.beginCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateDispatching, outbox.beginCalls[0].NextDocumentState)
	require.Len(t, outbox.completeCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateIssued, outbox.completeCalls[0].NextDocumentState)
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.issues))
}

func TestWorker_LeaseTokenFencingPreventsCompletion(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}
	outbox.loseLeaseOn = claim.Event.ID.String()

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "worker-a", Lease: time.Minute})
	err := worker.ProcessOnce(context.Background())
	require.ErrorIs(t, err, ports.ErrFiscalLeaseLost)
	require.Equal(t, int32(1), atomic.LoadInt32(&provider.issues), "provider already called")
	require.Len(t, outbox.completeCalls, 1)
}

func TestWorker_StaleStartedAttemptBecomesUnknownWithoutResubmit(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStateDispatching)
	claim.HasStartedAttempt = true
	claim.StartedAttemptID = uuid.New()
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "recover-b", Lease: time.Minute})
	err := worker.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&provider.issues), "must not call Issue on stale started")
	require.Len(t, outbox.recoverCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, outbox.recoverCalls[0].UnknownState)
	assert.Empty(t, outbox.beginCalls)
	assert.Empty(t, outbox.completeCalls)
}

func TestWorker_DefiniteTransientGoesRetryableNeverAutoResubmit(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{
		issueErr: ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "TEMP", "busy", nil),
	}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Len(t, outbox.completeCalls, 1)
	c := outbox.completeCalls[0]
	assert.Equal(t, domain.FiscalDocumentStateRetryableFailure, c.NextDocumentState)
	assert.Equal(t, ports.FiscalErrorClassTransient, c.Classification)
	assert.True(t, c.Definitive)
	assert.Equal(t, ports.FiscalAttemptFailed, c.AttemptStatus)
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.issues))
}

func TestWorker_AmbiguousTimeoutNeverAutoResubmitsIssue(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{
		issueErr: ports.NewFiscalProviderError(ports.FiscalErrorClassAmbiguous, "TIMEOUT", "lost", nil),
	}
	// ambiguous must not set DefinitiveNonAcceptance
	provider.issueErr.(*ports.FiscalProviderError).DefinitiveNonAcceptance = false
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Len(t, outbox.completeCalls, 1)
	c := outbox.completeCalls[0]
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, c.NextDocumentState)
	assert.Equal(t, ports.FiscalAttemptUnknown, c.AttemptStatus)
	assert.False(t, c.Definitive)
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.issues))
}

func TestWorker_ReconcileUsesSameFrozenProviderAndKey(t *testing.T) {
	outbox := newFakeOutbox()
	now := time.Now().UTC()
	provider := &recordingProvider{
		reconcileRes: ports.FiscalReconcileResult{
			FiscalOperationResponse: ports.FiscalOperationResponse{
				ProviderKey: "mock", ProviderReference: "MOCK-REF",
			},
			Matched: true, ResolvedAt: &now,
		},
	}
	claim := pendingClaim(ports.FiscalOutboxReconcileIssue, domain.FiscalDocumentStateOutcomeUnknown)
	claim.Document.ProviderReference = "MOCK-REF"
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.reconciles))
	assert.Equal(t, int32(0), atomic.LoadInt32(&provider.issues))
	assert.Equal(t, claim.Event.OperationKey, provider.lastReconKey)
	require.Len(t, outbox.completeCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateIssued, outbox.completeCalls[0].NextDocumentState)
	assert.Equal(t, "MOCK-REF", outbox.completeCalls[0].ProviderReference)
}

func TestWorker_GracefulShutdownStopsNewClaims(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim, pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{
		Owner: "w", Lease: time.Minute, PollInterval: time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.True(t, err == nil || errors.Is(err, context.Canceled))
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not shut down")
	}
	assert.GreaterOrEqual(t, len(outbox.claims), 1)
}

func TestWorker_CrashBeforeProviderCallLeavesStartedForRecovery(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}
	outbox.beginErr = errors.New("simulated crash before begin commit")

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	err := worker.ProcessOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&provider.issues), "must not call provider if begin fails")
	assert.Empty(t, outbox.completeCalls)
}

func TestWorker_CrashDuringCallTreatedAmbiguousOnRecovery(t *testing.T) {
	// Crash during call ≡ lease expires with started attempt still open.
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStateDispatching)
	claim.HasStartedAttempt = true
	claim.StartedAttemptID = uuid.New()
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "recover", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	assert.Equal(t, int32(0), atomic.LoadInt32(&provider.issues))
	require.Len(t, outbox.recoverCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, outbox.recoverCalls[0].UnknownState)
}

func TestWorker_CrashAfterProviderResponseLeaseLossDoesNotCommit(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}
	outbox.loseLeaseOn = claim.Event.ID.String()

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	err := worker.ProcessOnce(context.Background())
	require.ErrorIs(t, err, ports.ErrFiscalLeaseLost)
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.issues))
	require.Len(t, outbox.completeCalls, 1)
}

func TestWorker_DefinitePermanentRejectedNotUnknown(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{
		issueErr: ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "REJECT", "NIF invalid", nil),
	}
	claim := pendingClaim(ports.FiscalOutboxIssue, domain.FiscalDocumentStatePending)
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Len(t, outbox.completeCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateRejected, outbox.completeCalls[0].NextDocumentState)
	assert.Equal(t, ports.FiscalAttemptFailed, outbox.completeCalls[0].AttemptStatus)
	assert.True(t, outbox.completeCalls[0].Definitive)
}

func TestWorker_ReconcileInconclusiveRetainsUnknownWithoutIssue(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{
		reconcileRes: ports.FiscalReconcileResult{Matched: false},
	}
	claim := pendingClaim(ports.FiscalOutboxReconcileIssue, domain.FiscalDocumentStateOutcomeUnknown)
	claim.Document.ProviderReference = "MOCK-REF"
	frozenKey := claim.Event.OperationKey
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{claim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	assert.Equal(t, int32(0), atomic.LoadInt32(&provider.issues))
	assert.Equal(t, frozenKey, provider.lastReconKey)
	require.Len(t, outbox.completeCalls, 1)
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, outbox.completeCalls[0].NextDocumentState)
}

func TestWorker_VoidDispatchAndArtifactRecoveryPaths(t *testing.T) {
	outbox := newFakeOutbox()
	provider := &recordingProvider{}
	voidClaim := pendingClaim(ports.FiscalOutboxVoid, domain.FiscalDocumentStateVoidPending)
	voidClaim.Document.ProviderReference = "MOCK-REF"
	voidClaim.Document.VoidOperationKey = voidClaim.Event.OperationKey
	artClaim := pendingClaim(ports.FiscalOutboxRecoverArtifact, domain.FiscalDocumentStateIssued)
	artID := uuid.New()
	artClaim.Event.ArtifactID = &artID
	artClaim.Document.ProviderReference = "MOCK-REF"
	outbox.claimQueue = []*ports.ClaimedOutboxEvent{voidClaim, artClaim}

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "w", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.NoError(t, worker.ProcessOnce(context.Background()))
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.voids))
	assert.Equal(t, int32(1), atomic.LoadInt32(&provider.fetches))
	require.GreaterOrEqual(t, len(outbox.completeCalls), 2)
}

func TestRedactSecrets_StripsBearerAndTokens(t *testing.T) {
	in := "failed bearer abc.def refresh=super-secret Authorization: TokenXYZ"
	out := fiscalsvc.RedactSecrets(in)
	assert.NotContains(t, out, "abc.def")
	assert.NotContains(t, out, "super-secret")
	assert.NotContains(t, out, "TokenXYZ")
	assert.Contains(t, out, "[REDACTED]")
}

