package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalArtifactStateClassificationAndValidation(t *testing.T) {
	t.Parallel()

	statuses := []FiscalArtifactStatus{
		FiscalArtifactStatusPending,
		FiscalArtifactStatusAvailable,
		FiscalArtifactStatusUnavailable,
		FiscalArtifactStatusCompromised,
	}
	require.Len(t, statuses, 4)
	for _, status := range statuses {
		assert.True(t, status.IsValid(), status)
	}
	assert.False(t, FiscalArtifactStatus("unknown").IsValid())
	assert.True(t, FiscalArtifactClassificationLegal.IsValid())
	assert.True(t, FiscalArtifactClassificationMock.IsValid())
	assert.False(t, FiscalArtifactClassification("unknown").IsValid())
	assert.True(t, FiscalArtifactClassificationLegal.IsProductionLegal())
	assert.False(t, FiscalArtifactClassificationMock.IsProductionLegal())

	available := testFiscalArtifact(FiscalArtifactStatusAvailable, FiscalArtifactClassificationLegal)
	require.NoError(t, available.Validate())
	assert.True(t, available.CanServeInProduction())

	mock := testFiscalArtifact(FiscalArtifactStatusAvailable, FiscalArtifactClassificationMock)
	require.NoError(t, mock.Validate())
	assert.False(t, mock.CanServeInProduction())

	broken := available
	broken.StorageKey = ""
	assert.Error(t, broken.Validate())
}

func TestFiscalArtifactRoleActionMatrixAndIndependence(t *testing.T) {
	t.Parallel()

	artifact := testFiscalArtifact(FiscalArtifactStatusAvailable, FiscalArtifactClassificationLegal)
	assert.ElementsMatch(t, []FiscalArtifactAction{FiscalArtifactActionView}, artifact.AllowedActions(FiscalActorRoleAdmin, false))
	assert.ElementsMatch(t, []FiscalArtifactAction{FiscalArtifactActionView}, artifact.AllowedActions(FiscalActorRoleManager, false))
	assert.ElementsMatch(t, []FiscalArtifactAction{FiscalArtifactActionView}, artifact.AllowedActions(FiscalActorRoleEmployee, false))
	assert.ElementsMatch(t, []FiscalArtifactAction{FiscalArtifactActionView}, artifact.AllowedActions(FiscalActorRoleClient, true))
	assert.Empty(t, artifact.AllowedActions(FiscalActorRoleClient, false))
	assert.True(t, artifact.CanAct(FiscalActorRoleAdmin, false, FiscalArtifactActionView))
	assert.False(t, artifact.CanAct(FiscalActorRoleClient, false, FiscalArtifactActionView))

	blocked := testFiscalArtifact(FiscalArtifactStatusUnavailable, FiscalArtifactClassificationLegal)
	assert.False(t, blocked.CanAct(FiscalActorRoleAdmin, false, FiscalArtifactActionView))

	doc := testFiscalDocumentState(FiscalDocumentStateIssued)
	doc.VoidOperationKey = "void-1"
	before := artifact.Status
	require.NoError(t, doc.TransitionTo(FiscalDocumentStateVoidPending, time.Date(2026, 1, 2, 13, 0, 0, 0, time.UTC)))
	assert.Equal(t, before, artifact.Status)
	require.NoError(t, doc.TransitionTo(FiscalDocumentStateVoided, time.Date(2026, 1, 2, 13, 5, 0, 0, time.UTC)))
	assert.Equal(t, before, artifact.Status)
}

func testFiscalArtifact(status FiscalArtifactStatus, classification FiscalArtifactClassification) FiscalArtifact {
	availableAt := time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC)
	verifiedAt := time.Date(2026, 1, 2, 14, 5, 0, 0, time.UTC)
	return FiscalArtifact{
		ID:               uuid.New(),
		FiscalDocumentID:  uuid.New(),
		SourceInvoiceID:   uuid.New(),
		Kind:             FiscalArtifactKindProviderPDF,
		Status:           status,
		Classification:   classification,
		StorageKey:       "fiscal/default/doc-1/provider.pdf",
		MediaType:        "application/pdf",
		ByteSize:         1024,
		SHA256:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ProviderReference: "MOCK-REF",
		ProviderVersion:  "v1",
		CreatedAt:        time.Unix(1, 0).UTC(),
		AvailableAt:      &availableAt,
		LastVerifiedAt:   &verifiedAt,
		LastErrorCode:    "",
		LastErrorMessage: "",
	}
}
