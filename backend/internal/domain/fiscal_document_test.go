package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalDocumentStateAndPresentation(t *testing.T) {
	t.Parallel()

	states := []FiscalDocumentState{
		FiscalDocumentStateDraft,
		FiscalDocumentStatePending,
		FiscalDocumentStateDispatching,
		FiscalDocumentStateIssued,
		FiscalDocumentStateRejected,
		FiscalDocumentStateConnectionActionNeeded,
		FiscalDocumentStateRetryableFailure,
		FiscalDocumentStateOutcomeUnknown,
		FiscalDocumentStateVoidPending,
		FiscalDocumentStateVoidOutcomeUnknown,
		FiscalDocumentStateVoided,
	}
	require.Len(t, states, 11)
	for _, state := range states {
		assert.True(t, state.IsValid(), state)
	}
	assert.False(t, FiscalDocumentState("unknown").IsValid())

	assert.Equal(t, FiscalPresentationStateDraft, FiscalDocumentStateDraft.Presentation())
	assert.Equal(t, FiscalPresentationStatePending, FiscalDocumentStateDispatching.Presentation())
	assert.Equal(t, FiscalPresentationStateFinalized, FiscalDocumentStateIssued.Presentation())
	assert.Equal(t, FiscalPresentationStateVoided, FiscalDocumentStateVoided.Presentation())
	assert.Equal(t, FiscalPresentationStateUnavailable, FiscalDocumentStateRejected.Presentation())
	assert.Equal(t, FiscalPresentationStateUnavailable, FiscalDocumentStateOutcomeUnknown.Presentation())
}

func TestFiscalDocumentFinalizeFreezeAndTransitions(t *testing.T) {
	t.Parallel()

	doc := testFiscalDocumentDraft()
	connID := uuid.New()
	intentID := uuid.New()
	finalizedBy := uuid.New()
	frozenAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	frozen, err := doc.Finalize(FiscalActorRoleManager, FiscalFreezeInput{
		ProviderKey:       "mock",
		ConnectionID:      connID,
		IntentKey:         intentID,
		IssueOperationKey: "issue-1",
		FinalizedBy:       finalizedBy,
		FrozenAt:          frozenAt,
	})
	require.NoError(t, err)
	assert.Equal(t, FiscalDocumentStatePending, doc.State)
	assert.True(t, doc.ProviderFixed())
	assert.True(t, doc.IntentFixed())
	require.NotNil(t, doc.FrozenAt)
	assert.Equal(t, frozenAt, doc.FrozenAt.UTC())
	assert.True(t, frozen.HasProviderFixation())
	assert.Equal(t, connID, *frozen.ConnectionID)
	assert.Equal(t, intentID, *frozen.IntentKey)
	assert.Equal(t, finalizedBy, *frozen.FinalizedBy)

	ptr := uuid.New()
	doc.SupersedesDocumentID = &ptr
	snapshot := doc.Frozen()
	*doc.SupersedesDocumentID = uuid.New()
	require.NotNil(t, snapshot.SupersedesDocumentID)
	assert.NotEqual(t, *doc.SupersedesDocumentID, *snapshot.SupersedesDocumentID)

	_, err = doc.Finalize(FiscalActorRoleEmployee, FiscalFreezeInput{
		ProviderKey:       "mock",
		ConnectionID:      connID,
		IntentKey:         intentID,
		IssueOperationKey: "issue-2",
		FinalizedBy:       finalizedBy,
		FrozenAt:          frozenAt,
	})
	assert.ErrorIs(t, err, ErrFiscalDocumentRoleDenied)

	require.NoError(t, doc.TransitionTo(FiscalDocumentStateDispatching, frozenAt.Add(time.Minute)))
	require.NoError(t, doc.TransitionTo(FiscalDocumentStateIssued, frozenAt.Add(2*time.Minute)))
	assert.Equal(t, FiscalDocumentStateIssued, doc.State)
	assert.ErrorIs(t, doc.TransitionTo(FiscalDocumentStatePending, frozenAt.Add(3*time.Minute)), ErrFiscalDocumentTransitionNotAllowed)
	assert.ErrorIs(t, doc.TransitionTo(FiscalDocumentStateVoidPending, frozenAt.Add(4*time.Minute)), ErrFiscalDocumentFrozen)

	doc.VoidOperationKey = "void-1"
	require.NoError(t, doc.TransitionTo(FiscalDocumentStateVoidPending, frozenAt.Add(4*time.Minute)))
	require.NoError(t, doc.TransitionTo(FiscalDocumentStateVoided, frozenAt.Add(5*time.Minute)))
	assert.ErrorIs(t, doc.TransitionTo(FiscalDocumentStateDraft, frozenAt.Add(6*time.Minute)), ErrFiscalDocumentTransitionNotAllowed)
}

func TestFiscalDocumentRoleActionMatrixAndSupersession(t *testing.T) {
	t.Parallel()

	draft := testFiscalDocumentDraft()
	assert.ElementsMatch(t, []FiscalDocumentAction{FiscalDocumentActionView, FiscalDocumentActionEdit, FiscalDocumentActionDelete}, draft.AllowedActions(FiscalActorRoleEmployee, false))
	assert.ElementsMatch(t, []FiscalDocumentAction{FiscalDocumentActionView, FiscalDocumentActionEdit, FiscalDocumentActionDelete, FiscalDocumentActionFinalize}, draft.AllowedActions(FiscalActorRoleManager, false))
	assert.Empty(t, draft.AllowedActions(FiscalActorRoleClient, true))
	assert.True(t, draft.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionFinalize))
	assert.False(t, draft.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionFinalize))

	rejected := testFiscalDocumentRejected()
	assert.ElementsMatch(t, []FiscalDocumentAction{FiscalDocumentActionView, FiscalDocumentActionSupersede}, rejected.AllowedActions(FiscalActorRoleEmployee, false))
	assert.ElementsMatch(t, []FiscalDocumentAction{FiscalDocumentActionView, FiscalDocumentActionSupersede}, rejected.AllowedActions(FiscalActorRoleAdmin, false))

	clientIssued := testFiscalDocumentState(FiscalDocumentStateIssued)
	assert.ElementsMatch(t, []FiscalDocumentAction{FiscalDocumentActionView}, clientIssued.AllowedActions(FiscalActorRoleClient, true))
	assert.Empty(t, clientIssued.AllowedActions(FiscalActorRoleClient, false))

	retryable := testFiscalDocumentState(FiscalDocumentStateRetryableFailure)
	assert.True(t, retryable.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionRetry))
	assert.False(t, retryable.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionRetry))

	outcomeUnknown := testFiscalDocumentState(FiscalDocumentStateOutcomeUnknown)
	assert.True(t, outcomeUnknown.CanAct(FiscalActorRoleAdmin, false, FiscalDocumentActionReconcile))
	assert.False(t, outcomeUnknown.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionReconcile))

	issued := testFiscalDocumentState(FiscalDocumentStateIssued)
	assert.True(t, issued.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionVoid))
	assert.False(t, issued.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionVoid))

	replacement := testFiscalDocumentDraft()
	replacement.SourceInvoiceID = rejected.SourceInvoiceID
	replacement.IntentSlot = rejected.IntentSlot
	replacement.Kind = rejected.Kind
	linkAt := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	require.NoError(t, replacement.SupersedeRejected(&rejected, FiscalActorRoleAdmin, linkAt))
	require.NotNil(t, replacement.SupersedesDocumentID)
	assert.Equal(t, rejected.ID, *replacement.SupersedesDocumentID)
	require.NotNil(t, rejected.SupersededAt)
	assert.Equal(t, linkAt, rejected.SupersededAt.UTC())
	assert.ErrorIs(t, replacement.SupersedeRejected(&issued, FiscalActorRoleAdmin, linkAt), ErrFiscalDocumentSupersessionNotAllowed)
	assert.ErrorIs(t, replacement.SupersedeRejected(&rejected, FiscalActorRoleClient, linkAt), ErrFiscalDocumentRoleDenied)
}

func TestFiscalDocumentProviderFixation(t *testing.T) {
	t.Parallel()

	doc := testFiscalDocumentDraft()
	connID := uuid.New()
	require.NoError(t, doc.FixProvider("mock", connID))
	assert.True(t, doc.ProviderFixed())
	require.NoError(t, doc.FixProvider("mock", connID))
	assert.ErrorIs(t, doc.FixProvider("other", connID), ErrFiscalDocumentFrozen)
	assert.ErrorIs(t, doc.FixProvider("mock", uuid.Nil), ErrFiscalDocumentFrozen)
}

func TestFiscalDocumentUnknownRetryProhibitionAndFailedVoidReturnsIssued(t *testing.T) {
	t.Parallel()

	unknown := testFiscalDocumentState(FiscalDocumentStateOutcomeUnknown)
	assert.False(t, unknown.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionRetry))
	assert.False(t, unknown.CanAct(FiscalActorRoleAdmin, false, FiscalDocumentActionRetry))
	assert.True(t, unknown.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionReconcile))
	assert.ErrorIs(t, unknown.TransitionTo(FiscalDocumentStatePending, time.Unix(10, 0).UTC()), ErrFiscalDocumentTransitionNotAllowed)
	assert.ErrorIs(t, unknown.TransitionTo(FiscalDocumentStateDispatching, time.Unix(11, 0).UTC()), ErrFiscalDocumentTransitionNotAllowed)

	assert.False(t, unknown.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionReconcile))
	assert.False(t, unknown.CanAct(FiscalActorRoleClient, true, FiscalDocumentActionReconcile))
	assert.False(t, unknown.CanAct(FiscalActorRoleClient, true, FiscalDocumentActionRetry))
	assert.False(t, unknown.CanAct(FiscalActorRoleClient, true, FiscalDocumentActionVoid))

	issued := testFiscalDocumentState(FiscalDocumentStateIssued)
	assert.False(t, issued.CanAct(FiscalActorRoleEmployee, false, FiscalDocumentActionVoid))
	assert.False(t, issued.CanAct(FiscalActorRoleClient, true, FiscalDocumentActionVoid))
	issued.VoidOperationKey = "void-failed-1"
	at := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	require.NoError(t, issued.TransitionTo(FiscalDocumentStateVoidPending, at))
	require.NoError(t, issued.TransitionTo(FiscalDocumentStateIssued, at.Add(time.Minute)))
	assert.Equal(t, FiscalDocumentStateIssued, issued.State)
	assert.True(t, issued.CanAct(FiscalActorRoleManager, false, FiscalDocumentActionVoid))
}

func TestFiscalDocumentCloudwareProviderKeyIsOpaqueValue(t *testing.T) {
	t.Parallel()

	doc := testFiscalDocumentDraft()
	connID := uuid.New()
	require.NoError(t, doc.FixProvider("cloudware", connID))
	assert.Equal(t, "cloudware", doc.ProviderKey)
	assert.True(t, doc.ProviderFixed())
}

func testFiscalDocumentDraft() FiscalDocument {
	return FiscalDocument{
		ID:              uuid.New(),
		SourceInvoiceID: uuid.New(),
		IntentSlot:      "primary_sale",
		Kind:            DocumentKindFT,
		State:           FiscalDocumentStateDraft,
		Version:         1,
		CreatedBy:       uuid.New(),
		CreatedAt:       time.Unix(1, 0).UTC(),
		UpdatedAt:       time.Unix(1, 0).UTC(),
	}
}

func testFiscalDocumentRejected() FiscalDocument {
	doc := testFiscalDocumentDraft()
	doc.State = FiscalDocumentStateRejected
	doc.ProviderKey = "mock"
	connID := uuid.New()
	intentID := uuid.New()
	doc.ConnectionID = &connID
	doc.IntentKey = &intentID
	doc.IssueOperationKey = "issue-rejected"
	frozenAt := time.Unix(2, 0).UTC()
	doc.FrozenAt = &frozenAt
	return doc
}

func testFiscalDocumentState(state FiscalDocumentState) FiscalDocument {
	doc := testFiscalDocumentDraft()
	doc.State = state
	doc.ProviderKey = "mock"
	connID := uuid.New()
	intentID := uuid.New()
	doc.ConnectionID = &connID
	doc.IntentKey = &intentID
	doc.IssueOperationKey = "issue-state"
	frozenAt := time.Unix(3, 0).UTC()
	doc.FrozenAt = &frozenAt
	if state == FiscalDocumentStateVoidPending || state == FiscalDocumentStateVoidOutcomeUnknown || state == FiscalDocumentStateVoided {
		doc.VoidOperationKey = "void-state"
	}
	return doc
}
