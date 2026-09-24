package fiscal_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, nil
}
func (r *fakeUserRepo) GetByRole(context.Context, string, int, int) ([]*domain.User, error) {
	return nil, nil
}
func (r *fakeUserRepo) List(context.Context, int, int) ([]*domain.User, error) { return nil, nil }
func (r *fakeUserRepo) Update(context.Context, *domain.User) error             { return nil }
func (r *fakeUserRepo) Delete(context.Context, uuid.UUID) error                { return nil }
func (r *fakeUserRepo) UpdatePassword(context.Context, uuid.UUID, string) error {
	return nil
}
func (r *fakeUserRepo) GetActiveUsers(context.Context, int, int) ([]*domain.User, error) {
	return nil, nil
}
func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

type fakeInvoiceRepo struct {
	byID map[uuid.UUID]*domain.Invoice
}

func (r *fakeInvoiceRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	inv, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	cp := *inv
	return &cp, nil
}

func (r *fakeInvoiceRepo) GetByRepairID(_ context.Context, repairID uuid.UUID) (*domain.Invoice, error) {
	for _, inv := range r.byID {
		if inv != nil && inv.RepairID != nil && *inv.RepairID == repairID {
			cp := *inv
			return &cp, nil
		}
	}
	return nil, domain.ErrInvoiceNotFound
}

func (r *fakeInvoiceRepo) Create(context.Context, *domain.Invoice) error { return nil }
func (r *fakeInvoiceRepo) Update(context.Context, *domain.Invoice) error { return nil }
func (r *fakeInvoiceRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (r *fakeInvoiceRepo) ListByCustomerID(context.Context, uuid.UUID, int, int) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}
func (r *fakeInvoiceRepo) ListForStaff(context.Context, int, int) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}

type fakeFiscalRepo struct {
	mu          sync.Mutex
	aggs        map[uuid.UUID]*ports.FiscalDocumentAggregate
	byInvoice   map[uuid.UUID]uuid.UUID
	failOutbox  bool
	finalizeN   int
	outboxCount map[uuid.UUID]int
}

func newFakeFiscalRepo() *fakeFiscalRepo {
	return &fakeFiscalRepo{
		aggs:        make(map[uuid.UUID]*ports.FiscalDocumentAggregate),
		byInvoice:   make(map[uuid.UUID]uuid.UUID),
		outboxCount: make(map[uuid.UUID]int),
	}
}

func (r *fakeFiscalRepo) HasProtectedFiscalHistory(_ context.Context, invoiceID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byInvoice[invoiceID]
	if !ok {
		return false, nil
	}
	return r.aggs[id].Document.State.IsFrozen(), nil
}

func (r *fakeFiscalRepo) CreateDraft(_ context.Context, input ports.CreateDraftInput) (*ports.FiscalDocumentAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byInvoice[input.SourceInvoiceID]; ok {
		if r.aggs[existing].Document.SupersededAt == nil {
			return nil, ports.ErrFiscalIntentConflict
		}
	}
	id := input.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	now := time.Now().UTC()
	agg := &ports.FiscalDocumentAggregate{
		Document: domain.FiscalDocument{
			ID:              id,
			SourceInvoiceID: input.SourceInvoiceID,
			IntentSlot:      input.IntentSlot,
			Kind:            input.Kind,
			State:           domain.FiscalDocumentStateDraft,
			Version:         1,
			CreatedBy:       input.CreatedBy,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		Snapshot: input.Snapshot,
	}
	if agg.Snapshot.ID == uuid.Nil {
		agg.Snapshot.ID = uuid.New()
	}
	r.aggs[id] = agg
	r.byInvoice[input.SourceInvoiceID] = id
	cp := *agg
	return &cp, nil
}

func (r *fakeFiscalRepo) UpdateDraft(_ context.Context, documentID uuid.UUID, expectedVersion int64, snapshot ports.FiscalSnapshotRecord) (*ports.FiscalDocumentAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg, ok := r.aggs[documentID]
	if !ok {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	if agg.Document.Version != expectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	if agg.Document.State != domain.FiscalDocumentStateDraft {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	agg.Snapshot = snapshot
	agg.Document.Version++
	agg.Document.UpdatedAt = time.Now().UTC()
	cp := *agg
	return &cp, nil
}

func (r *fakeFiscalRepo) GetAggregate(_ context.Context, documentID uuid.UUID) (*ports.FiscalDocumentAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg, ok := r.aggs[documentID]
	if !ok {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	cp := *agg
	return &cp, nil
}

func (r *fakeFiscalRepo) GetCurrentIntent(_ context.Context, invoiceID uuid.UUID, _ string) (*ports.FiscalDocumentAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byInvoice[invoiceID]
	if !ok {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	cp := *r.aggs[id]
	return &cp, nil
}

func (r *fakeFiscalRepo) DeleteDraft(_ context.Context, documentID uuid.UUID, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg, ok := r.aggs[documentID]
	if !ok {
		return ports.ErrFiscalDocumentNotFound
	}
	if agg.Document.Version != expectedVersion {
		return ports.ErrFiscalVersionConflict
	}
	if agg.Document.State != domain.FiscalDocumentStateDraft {
		return ports.ErrFiscalActionNotAllowed
	}
	delete(r.byInvoice, agg.Document.SourceInvoiceID)
	delete(r.aggs, documentID)
	return nil
}

func (r *fakeFiscalRepo) FinalizeDraft(_ context.Context, cmd ports.FinalizeDraftCommand) (*ports.FinalizeDraftResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg, ok := r.aggs[cmd.DocumentID]
	if !ok {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	if agg.Document.Version != cmd.ExpectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	if agg.Document.State == domain.FiscalDocumentStatePending {
		r.finalizeN++
		return &ports.FinalizeDraftResult{Document: agg.Document.Frozen(), OutboxID: uuid.New(), Sequence: 1}, nil
	}
	if r.failOutbox {
		return nil, errors.New("outbox insert failed")
	}
	conn := cmd.ConnectionID
	intent := cmd.IntentKey
	frozen, err := agg.Document.Finalize(domain.FiscalActorRoleManager, domain.FiscalFreezeInput{
		ProviderKey:       cmd.ProviderKey,
		ConnectionID:      cmd.ConnectionID,
		IntentKey:         cmd.IntentKey,
		IssueOperationKey: cmd.IssueOpKey,
		FinalizedBy:       cmd.ActorID,
		FrozenAt:          cmd.FrozenAt,
	})
	if err != nil {
		return nil, err
	}
	agg.Document.ProviderKey = cmd.ProviderKey
	agg.Document.ConnectionID = &conn
	agg.Document.IntentKey = &intent
	agg.Document.IssueOperationKey = cmd.IssueOpKey
	agg.Document.FrozenAt = &cmd.FrozenAt
	agg.Document.FinalizedBy = &cmd.ActorID
	agg.Document.State = domain.FiscalDocumentStatePending
	agg.Document.Version++
	agg.Snapshot.CanonicalBytes = append([]byte(nil), cmd.CanonicalBytes...)
	agg.Snapshot.CanonicalSHA256 = cmd.CanonicalSHA256
	agg.Snapshot.FrozenAt = &cmd.FrozenAt
	r.outboxCount[cmd.DocumentID]++
	r.finalizeN++
	return &ports.FinalizeDraftResult{Document: frozen, OutboxID: uuid.New(), Sequence: 1}, nil
}

func (r *fakeFiscalRepo) EnqueueAction(_ context.Context, cmd ports.EnqueueActionCommand) (*domain.FiscalDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg, ok := r.aggs[cmd.DocumentID]
	if !ok {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	if agg.Document.Version != cmd.ExpectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	if cmd.NextState != agg.Document.State {
		if err := agg.Document.TransitionTo(cmd.NextState, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	if cmd.VoidOperationKey != "" {
		agg.Document.VoidOperationKey = cmd.VoidOperationKey
	}
	agg.Document.Version++
	r.outboxCount[cmd.DocumentID]++
	cp := agg.Document
	return &cp, nil
}

func (r *fakeFiscalRepo) GetNextPending(_ context.Context) (*domain.FiscalDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var next *domain.FiscalDocument
	for _, agg := range r.aggs {
		switch agg.Document.State {
		case domain.FiscalDocumentStatePending,
			domain.FiscalDocumentStateRetryableFailure,
			domain.FiscalDocumentStateVoidPending:
		default:
			continue
		}
		if next == nil || agg.Document.UpdatedAt.Before(next.UpdatedAt) {
			doc := agg.Document
			next = &doc
		}
	}
	if next == nil {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	return next, nil
}

var _ ports.FiscalRepository = (*fakeFiscalRepo)(nil)

func seedActors(t *testing.T) (employee, manager, client uuid.UUID, users *fakeUserRepo, invoices *fakeInvoiceRepo, invoiceID uuid.UUID) {
	t.Helper()
	employee, manager, client = uuid.New(), uuid.New(), uuid.New()
	invoiceID = uuid.New()
	users = &fakeUserRepo{users: map[uuid.UUID]*domain.User{
		employee: {ID: employee, Role: domain.RoleEmployee},
		manager:  {ID: manager, Role: domain.RoleManager},
		client:   {ID: client, Role: domain.RoleClient},
	}}
	invoices = &fakeInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{
		invoiceID: {
			ID:                invoiceID,
			CustomerID:        client,
			Amount:            12.3,
			Status:            "open",
			FiscalEligibility: domain.FiscalEligibilityEligible,
		},
	}}
	return employee, manager, client, users, invoices, invoiceID
}

func emptySnapshot() ports.FiscalSnapshotRecord {
	return ports.FiscalSnapshotRecord{
		Issuer:         json.RawMessage(`{}`),
		Customer:       json.RawMessage(`{"nif":"123456789","name":"Acme"}`),
		BillingAddress: json.RawMessage(`{}`),
		Currency:       "EUR",
		GrossTotal:     "10",
		DiscountTotal:  "0",
		NetTotal:       "10",
		TaxTotal:       "2.3",
		RoundingAdj:    "0",
		PayableTotal:   "12.3",
		Lines: []ports.FiscalLineRecord{{
			Position: 1, Description: "labor", UnitCode: "unit",
			Quantity: "1", UnitPrice: "10", GrossAmount: "10", DiscountKind: "none",
			DiscountValue: "0", DiscountAmount: "0", NetAmount: "10", TaxRate: "0.23",
			TaxAmount: "2.3", LineTotal: "12.3", TaxTreatmentCode: "normal",
		}},
	}
}

func TestDraftServiceEmployeeCRUDAndClientDenial(t *testing.T) {
	employee, _, client, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	svc := fiscalsvc.NewDraftService(repo, invoices, users)

	created, err := svc.CreateDraft(context.Background(), employee, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID,
		Kind:            domain.DocumentKindFT,
		Snapshot:        emptySnapshot(),
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalDocumentStateDraft, created.Document.State)
	require.EqualValues(t, 1, created.Document.Version)

	updated, err := svc.UpdateDraft(context.Background(), employee, created.Document.ID, 1, emptySnapshot())
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Document.Version)

	_, err = svc.CreateDraft(context.Background(), client, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID,
		Kind:            domain.DocumentKindFT,
		Snapshot:        emptySnapshot(),
	})
	require.ErrorIs(t, err, domain.ErrUnauthorizedAccess)

	_, err = svc.UpdateDraft(context.Background(), employee, created.Document.ID, 1, emptySnapshot())
	require.ErrorIs(t, err, ports.ErrFiscalVersionConflict)
}

func TestFinalizationServiceAtomicityAndAuth(t *testing.T) {
	employee, manager, _, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	draft := fiscalsvc.NewDraftService(repo, invoices, users)
	finalizer := fiscalsvc.NewFinalizationService(repo, invoices, users).WithEnabled(true)

	created, err := draft.CreateDraft(context.Background(), employee, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID,
		Kind:            domain.DocumentKindFT,
		Snapshot:        emptySnapshot(),
	})
	require.NoError(t, err)

	_, err = finalizer.Finalize(context.Background(), employee, fiscalsvc.FinalizeRequest{
		DocumentID: created.Document.ID, ExpectedVersion: 1,
		ProviderKey: "mock", ConnectionID: uuid.New(),
	})
	require.ErrorIs(t, err, domain.ErrUnauthorizedAccess)

	result, err := finalizer.Finalize(context.Background(), manager, fiscalsvc.FinalizeRequest{
		DocumentID: created.Document.ID, ExpectedVersion: 1,
		ProviderKey: "mock", ConnectionID: uuid.New(),
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalDocumentStatePending, result.Document.State)
	require.Equal(t, 1, repo.outboxCount[created.Document.ID])

	again, err := finalizer.Finalize(context.Background(), manager, fiscalsvc.FinalizeRequest{
		DocumentID: created.Document.ID, ExpectedVersion: result.Document.Version,
		ProviderKey: "mock", ConnectionID: *result.Document.ConnectionID,
	})
	require.NoError(t, err)
	assert.Equal(t, result.Document.IntentKey, again.Document.IntentKey)
	assert.Equal(t, 1, repo.outboxCount[created.Document.ID])
}

func TestFinalizationServiceOutboxFailureRollsBack(t *testing.T) {
	employee, manager, _, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	repo.failOutbox = true
	draft := fiscalsvc.NewDraftService(repo, invoices, users)
	finalizer := fiscalsvc.NewFinalizationService(repo, invoices, users).WithEnabled(true)
	created, err := draft.CreateDraft(context.Background(), employee, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID, Kind: domain.DocumentKindFT, Snapshot: emptySnapshot(),
	})
	require.NoError(t, err)

	_, err = finalizer.Finalize(context.Background(), manager, fiscalsvc.FinalizeRequest{
		DocumentID: created.Document.ID, ExpectedVersion: 1,
		ProviderKey: "mock", ConnectionID: uuid.New(),
	})
	require.Error(t, err)

	loaded, err := repo.GetAggregate(context.Background(), created.Document.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalDocumentStateDraft, loaded.Document.State)
	assert.Equal(t, 0, repo.outboxCount[created.Document.ID])
}

func TestDraftServiceRejectsUnsupportedDocumentKind(t *testing.T) {
	employee, _, _, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	svc := fiscalsvc.NewDraftService(repo, invoices, users)

	_, err := svc.CreateDraft(context.Background(), employee, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID,
		Kind:            domain.DocumentKind("NC"),
		Snapshot:        emptySnapshot(),
	})
	require.ErrorIs(t, err, ports.ErrFiscalActionNotAllowed)
	assert.Contains(t, err.Error(), "unsupported kind")
	assert.Empty(t, repo.aggs, "unsupported kind must not create a fiscal intent")
	assert.Empty(t, repo.byInvoice)
}

func TestDraftServiceStoresCanonicalDecimalsIndependentOfLegacyFloatAmount(t *testing.T) {
	employee, _, _, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	svc := fiscalsvc.NewDraftService(repo, invoices, users)

	// Legacy invoice float differs from canonical draft totals on purpose.
	invoices.byID[invoiceID].Amount = 999.99

	snap := emptySnapshot()
	snap.PayableTotal = "12.30"
	snap.GrossTotal = "10.00"
	snap.NetTotal = "10.00"
	snap.TaxTotal = "2.30"

	created, err := svc.CreateDraft(context.Background(), employee, fiscalsvc.CreateDraftRequest{
		SourceInvoiceID: invoiceID,
		Kind:            domain.DocumentKindFT,
		Snapshot:        snap,
	})
	require.NoError(t, err)
	assert.Equal(t, "12.30", created.Snapshot.PayableTotal)
	assert.Equal(t, "10.00", created.Snapshot.GrossTotal)
	assert.NotEqual(t, "999.99", created.Snapshot.PayableTotal)

	// Mutating the legacy float after draft creation must not rewrite frozen snapshot totals.
	invoices.byID[invoiceID].Amount = 1.0
	reloaded, err := repo.GetAggregate(context.Background(), created.Document.ID)
	require.NoError(t, err)
	assert.Equal(t, "12.30", reloaded.Snapshot.PayableTotal)
	assert.Equal(t, 1.0, invoices.byID[invoiceID].Amount)
}

func TestFinalizationServiceUnknownStateOnlyReconcile(t *testing.T) {
	_, manager, _, users, invoices, invoiceID := seedActors(t)
	repo := newFakeFiscalRepo()
	conn := uuid.New()
	intent := uuid.New()
	now := time.Now().UTC()
	docID := uuid.New()
	repo.aggs[docID] = &ports.FiscalDocumentAggregate{
		Document: domain.FiscalDocument{
			ID: docID, SourceInvoiceID: invoiceID, IntentSlot: "primary_sale", Kind: domain.DocumentKindFT,
			State: domain.FiscalDocumentStateOutcomeUnknown, Version: 3, ProviderKey: "mock",
			ConnectionID: &conn, IntentKey: &intent, IssueOperationKey: "fiscal:issue:" + intent.String(),
			FrozenAt: &now, CreatedBy: manager,
		},
	}
	repo.byInvoice[invoiceID] = docID
	finalizer := fiscalsvc.NewFinalizationService(repo, invoices, users).WithEnabled(true)

	_, err := finalizer.Retry(context.Background(), manager, fiscalsvc.ActionRequest{DocumentID: docID, ExpectedVersion: 3})
	require.ErrorIs(t, err, ports.ErrFiscalActionNotAllowed)

	doc, err := finalizer.Reconcile(context.Background(), manager, fiscalsvc.ActionRequest{DocumentID: docID, ExpectedVersion: 3})
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, doc.State)
	assert.Equal(t, 1, repo.outboxCount[docID])
}
