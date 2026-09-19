package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyConnectionScenarios(t *testing.T) {
	t.Parallel()
	provider := NewProvider()
	baseReq := ports.FiscalVerifyConnectionRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			ProviderKey:    "mock",
			OperationKey:   "verify-normalized",
			CorrelationKey: "corr-verify-1",
			Metadata:       map[string]string{"scenario": "success"},
		},
	}

	first, err := provider.VerifyConnection(context.Background(), baseReq)
	require.NoError(t, err)
	second, err := provider.VerifyConnection(context.Background(), baseReq)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.True(t, first.State.IsReady())
	assert.NotNil(t, first.AccessExpiresAt)
	assert.NotEmpty(t, first.GrantedScopes)

	transientReq := baseReq
	transientReq.OperationKey = "verify-transient"
	transientReq.Metadata = map[string]string{}
	_, err = provider.VerifyConnection(context.Background(), transientReq)
	require.Error(t, err)
	assertFiscalProviderError(t, err, ports.FiscalErrorClassTransient)
}

func TestIssueScenarios(t *testing.T) {
	t.Parallel()
	provider := NewProvider()
	req := ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:    "mock",
			OperationKey:   "issue-permanent",
			CorrelationKey: "corr-issue-1",
			Metadata:       map[string]string{},
		},
		DocumentID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Payload:    []byte("document-payload"),
	}

	_, err := provider.Issue(context.Background(), req)
	require.Error(t, err)
	assertFiscalProviderError(t, err, ports.FiscalErrorClassPermanent)

	req.OperationKey = "issue-success"
	req.Metadata = map[string]string{"scenario": "success"}
	res, err := provider.Issue(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, req.DocumentID, res.DocumentID)
	assert.NotEqual(t, uuid.Nil, res.ArtifactID)
	assert.NotEmpty(t, res.ProviderNumber)
	assert.NotNil(t, res.IssuedAt)
}

func TestReconcileAmbiguousScenario(t *testing.T) {
	t.Parallel()
	provider := NewProvider()
	req := ports.FiscalReconcileRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			ProviderKey:    "mock",
			OperationKey:   "reconcile-normalized",
			CorrelationKey: "corr-reconcile-1",
			Metadata:       map[string]string{"scenario": "ambiguous"},
		},
		DocumentID:        uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		ProviderReference: "provider-ref-1",
	}

	res, err := provider.Reconcile(context.Background(), req)
	require.Error(t, err)
	assertFiscalProviderError(t, err, ports.FiscalErrorClassConflict)
	assert.False(t, res.Matched)
	assert.NotNil(t, res.ResolvedAt)
}

func TestVoidAndFetchScenarios(t *testing.T) {
	t.Parallel()
	provider := NewProvider()

	voidReq := ports.FiscalVoidRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			ProviderKey:    "mock",
			OperationKey:   "void-transient",
			CorrelationKey: "corr-void-1",
			Metadata:       map[string]string{},
		},
		DocumentID:        uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		ProviderReference: "provider-ref-2",
		Reason:            "customer request",
	}
	_, err := provider.Void(context.Background(), voidReq)
	require.Error(t, err)
	assertFiscalProviderError(t, err, ports.FiscalErrorClassTransient)

	fetchReq := ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			ProviderKey:    "mock",
			OperationKey:   "fetch-success",
			CorrelationKey: "corr-fetch-1",
			Metadata:       map[string]string{"scenario": "success"},
		},
		ArtifactID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
	}
	res, err := provider.FetchArtifact(context.Background(), fetchReq)
	require.NoError(t, err)
	assert.Equal(t, fetchReq.ArtifactID, res.ArtifactID)
	assert.NotEmpty(t, res.Content)
	assert.NotEmpty(t, res.Sha256)
	assert.NotNil(t, res.FetchedAt)

	fetchReq.OperationKey = "fetch-permanent"
	fetchReq.Metadata = map[string]string{}
	_, err = provider.FetchArtifact(context.Background(), fetchReq)
	require.Error(t, err)
	assertFiscalProviderError(t, err, ports.FiscalErrorClassPermanent)
}

func assertFiscalProviderError(t *testing.T, err error, class ports.FiscalErrorClass) {
	t.Helper()
	var providerErr *ports.FiscalProviderError
	require.True(t, errors.As(err, &providerErr))
	assert.Equal(t, class, providerErr.Class)
	assert.True(t, providerErr.IsValid())
}
