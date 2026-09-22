package fiscal

import (
	"context"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFiscalRepo struct {
	mock.Mock
}

func (m *mockFiscalRepo) GetNextPending(ctx context.Context) (*domain.FiscalDocument, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FiscalDocument), args.Error(1)
}

func (m *mockFiscalRepo) GetAggregate(ctx context.Context, id uuid.UUID) (*ports.FiscalDocumentAggregate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ports.FiscalDocumentAggregate), args.Error(1)
}

func (m *mockFiscalRepo) EnqueueAction(ctx context.Context, cmd ports.EnqueueActionCommand) (*domain.FiscalDocument, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0).(*domain.FiscalDocument), args.Error(1)
}

// Implement other interface methods as no-ops or panics
func (m *mockFiscalRepo) HasProtectedFiscalHistory(ctx context.Context, id uuid.UUID) (bool, error) { return false, nil }
func (m *mockFiscalRepo) CreateDraft(ctx context.Context, input ports.CreateDraftInput) (*ports.FiscalDocumentAggregate, error) { return nil, nil }
func (m *mockFiscalRepo) UpdateDraft(ctx context.Context, id uuid.UUID, v int64, s ports.FiscalSnapshotRecord) (*ports.FiscalDocumentAggregate, error) { return nil, nil }
func (m *mockFiscalRepo) GetCurrentIntent(ctx context.Context, id uuid.UUID, slot string) (*ports.FiscalDocumentAggregate, error) { return nil, nil }
func (m *mockFiscalRepo) DeleteDraft(ctx context.Context, id uuid.UUID, v int64) error { return nil }
func (m *mockFiscalRepo) FinalizeDraft(ctx context.Context, cmd ports.FinalizeDraftCommand) (*ports.FinalizeDraftResult, error) { return nil, nil }

type mockFiscalProvider struct {
	mock.Mock
}

func (m *mockFiscalProvider) Issue(ctx context.Context, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(ports.FiscalIssueResult), args.Error(1)
}
func (m *mockFiscalProvider) VerifyConnection(ctx context.Context, req ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) { return ports.FiscalVerifyConnectionResult{}, nil }
func (m *mockFiscalProvider) Reconcile(ctx context.Context, req ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) { return ports.FiscalReconcileResult{}, nil }
func (m *mockFiscalProvider) Void(ctx context.Context, req ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) { return ports.FiscalVoidResult{}, nil }
func (m *mockFiscalProvider) FetchArtifact(ctx context.Context, req ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) { return ports.FiscalFetchArtifactResult{}, nil }

func TestDispatcher_ProcessNext_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFiscalRepo)
	provider := new(mockFiscalProvider)
	evaluator := NewCloudwareEnablementEvaluator(true)

	docID := uuid.New()
	connID := uuid.New()
	doc := &domain.FiscalDocument{
		ID:           docID,
		State:        domain.FiscalDocumentStatePending,
		ProviderKey:  "cloudware",
		ConnectionID: &connID,
		IntentKey:    &docID,
		IssueOperationKey: "op-123",
	}

	repo.On("GetNextPending", ctx).Return(doc, nil)
	repo.On("GetAggregate", ctx, docID).Return(&ports.FiscalDocumentAggregate{
		Document: *doc,
	}, nil)

	provider.On("Issue", ctx, mock.Anything).Return(ports.FiscalIssueResult{}, nil)
	repo.On("EnqueueAction", ctx, mock.Anything).Return(doc, nil)

	dispatcher := NewDispatcher(repo, provider, evaluator)
	err := dispatcher.ProcessNext(ctx)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	provider.AssertExpectations(t)
}
