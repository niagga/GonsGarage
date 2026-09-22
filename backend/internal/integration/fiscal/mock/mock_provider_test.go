package mock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockProvider_FTAndFRIssuanceAreStableAndLabeled(t *testing.T) {
	t.Parallel()
	store := NewMemoryOperationStore()
	registry := NewScenarioRegistry(map[string]Scenario{
		"op-ft-ok": ScenarioSuccess,
		"op-fr-ok": ScenarioSuccess,
	})
	provider, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: store})
	require.NoError(t, err)

	ftReq := issueRequest("op-ft-ok", "FT", []byte(`{"kind":"FT","lines":1}`))
	ft1, err := provider.Issue(context.Background(), ftReq)
	require.NoError(t, err)
	require.True(t, stringsHasPrefix(ft1.ProviderReference, "MOCK-"))
	require.NotNil(t, ft1.IssuedAt)
	require.Equal(t, "FT", ft1.Metadata["documentKind"])

	ft2, err := provider.Issue(context.Background(), ftReq)
	require.NoError(t, err)
	assert.Equal(t, ft1.ProviderReference, ft2.ProviderReference)
	assert.Equal(t, ft1.ProviderNumber, ft2.ProviderNumber)
	assert.Equal(t, ft1.ArtifactID, ft2.ArtifactID)

	frReq := issueRequest("op-fr-ok", "FR", []byte(`{"kind":"FR","lines":1}`))
	fr, err := provider.Issue(context.Background(), frReq)
	require.NoError(t, err)
	assert.NotEqual(t, ft1.ProviderReference, fr.ProviderReference)
	assert.Equal(t, "FR", fr.Metadata["documentKind"])
	assert.Equal(t, "true", fr.Metadata["associatedReceiptCapability"])

	art, err := provider.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: ftReq.ConnectionID, ProviderKey: "mock", OperationKey: "op-ft-ok", CorrelationKey: "c1",
		},
		ArtifactID: ft1.ArtifactID,
	})
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", art.MediaType)
	assert.Contains(t, string(art.Content), MockPDFLabel)
	sum := sha256.Sum256(art.Content)
	assert.Equal(t, hex.EncodeToString(sum[:]), art.Sha256)
}

func TestMockProvider_ScenariosCoverValidationExpiredTransientRateLimitAmbiguousReconcileAndVoids(t *testing.T) {
	t.Parallel()
	registry := NewScenarioRegistry(map[string]Scenario{
		"op-validation": ScenarioValidation,
		"op-expired":    ScenarioExpiredConnection,
		"op-transient":  ScenarioTransient,
		"op-rate":       ScenarioRateLimit,
		"op-ambiguous":  ScenarioAmbiguous,
		"op-void-ok":    ScenarioVoidPermitted,
		"op-void-no":    ScenarioVoidRefused,
		"op-success":    ScenarioSuccess,
	})
	provider, err := NewProvider(Options{AppEnv: "test", Registry: registry, Store: NewMemoryOperationStore()})
	require.NoError(t, err)
	ctx := context.Background()

	_, err = provider.Issue(ctx, issueRequest("op-validation", "FT", []byte(`{"kind":"FT"}`)))
	assertFiscalClass(t, err, ports.FiscalErrorClassValidation)

	_, err = provider.VerifyConnection(ctx, ports.FiscalVerifyConnectionRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{ConnectionID: uuid.New(), ProviderKey: "mock", OperationKey: "op-expired"},
	})
	assertFiscalClass(t, err, ports.FiscalErrorClassUnauthorized)

	_, err = provider.Issue(ctx, issueRequest("op-transient", "FT", []byte(`{"kind":"FT"}`)))
	assertFiscalClass(t, err, ports.FiscalErrorClassTransient)

	_, err = provider.Issue(ctx, issueRequest("op-rate", "FT", []byte(`{"kind":"FT"}`)))
	assertFiscalClass(t, err, ports.FiscalErrorClassRateLimit)

	ambReq := issueRequest("op-ambiguous", "FT", []byte(`{"kind":"FT","n":1}`))
	ambRes, err := provider.Issue(ctx, ambReq)
	assertFiscalClass(t, err, ports.FiscalErrorClassAmbiguous)
	require.NotEmpty(t, ambRes.ProviderReference)
	require.Nil(t, ambRes.IssuedAt)

	rec, err := provider.Reconcile(ctx, ports.FiscalReconcileRequest{
		FiscalOperationRequest: ambReq.FiscalOperationRequest,
		DocumentID:             ambReq.DocumentID,
		ProviderReference:      ambRes.ProviderReference,
	})
	require.NoError(t, err)
	assert.True(t, rec.Matched)
	assert.Equal(t, ambRes.ProviderReference, rec.ProviderReference)
	assert.NotNil(t, rec.ResolvedAt)

	issued, err := provider.Issue(ctx, issueRequest("op-success", "FT", []byte(`{"kind":"FT"}`)))
	require.NoError(t, err)

	voided, err := provider.Void(ctx, ports.FiscalVoidRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:  "mock", OperationKey: "op-void-ok", CorrelationKey: "void-1",
		},
		DocumentID: issued.DocumentID, ProviderReference: issued.ProviderReference, Reason: "customer request",
	})
	require.NoError(t, err)
	require.NotNil(t, voided.VoidedAt)

	_, err = provider.Void(ctx, ports.FiscalVoidRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:  "mock", OperationKey: "op-void-no", CorrelationKey: "void-2",
		},
		DocumentID: issued.DocumentID, ProviderReference: issued.ProviderReference, Reason: "refused",
	})
	assertFiscalClass(t, err, ports.FiscalErrorClassPermanent)
}

func TestMockProvider_ConcurrentIdenticalIssueReturnsSameReference(t *testing.T) {
	t.Parallel()
	registry := NewScenarioRegistry(map[string]Scenario{"op-concurrent": ScenarioSuccess})
	provider, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: NewMemoryOperationStore()})
	require.NoError(t, err)
	req := issueRequest("op-concurrent", "FT", []byte(`{"kind":"FT","same":true}`))

	const n = 16
	refs := make([]string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			res, err := provider.Issue(context.Background(), req)
			require.NoError(t, err)
			refs[i] = res.ProviderReference
		}(i)
	}
	wg.Wait()
	for i := 1; i < n; i++ {
		assert.Equal(t, refs[0], refs[i])
	}
}

func TestMockProvider_DifferentCanonicalInputChangesReferenceAndPDF(t *testing.T) {
	t.Parallel()
	registry := NewScenarioRegistry(map[string]Scenario{
		"op-a": ScenarioSuccess,
		"op-b": ScenarioSuccess,
	})
	provider, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: NewMemoryOperationStore()})
	require.NoError(t, err)

	a, err := provider.Issue(context.Background(), issueRequest("op-a", "FT", []byte(`{"kind":"FT","v":1}`)))
	require.NoError(t, err)
	b, err := provider.Issue(context.Background(), issueRequest("op-b", "FT", []byte(`{"kind":"FT","v":2}`)))
	require.NoError(t, err)
	assert.NotEqual(t, a.ProviderReference, b.ProviderReference)

	fetchA, err := provider.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:  "mock", OperationKey: "op-a", CorrelationKey: "fetch-a",
		},
		ArtifactID: a.ArtifactID,
	})
	require.NoError(t, err)
	fetchB, err := provider.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:  "mock", OperationKey: "op-b", CorrelationKey: "fetch-b",
		},
		ArtifactID: b.ArtifactID,
	})
	require.NoError(t, err)
	assert.NotEqual(t, fetchA.Sha256, fetchB.Sha256)
	assert.True(t, bytes.Contains(fetchA.Content, []byte(MockPDFLabel)))
	assert.True(t, bytes.Contains(fetchB.Content, []byte(MockPDFLabel)))
}

func TestMockProvider_SurvivesStoreReload(t *testing.T) {
	t.Parallel()
	shared := NewMemoryOperationStore()
	registry := NewScenarioRegistry(map[string]Scenario{"op-reload": ScenarioSuccess})
	first, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: shared})
	require.NoError(t, err)
	req := issueRequest("op-reload", "FT", []byte(`{"kind":"FT","reload":true}`))
	issued, err := first.Issue(context.Background(), req)
	require.NoError(t, err)
	pdf1, err := first.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: req.FiscalOperationRequest, ArtifactID: issued.ArtifactID,
	})
	require.NoError(t, err)

	second, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: shared})
	require.NoError(t, err)
	again, err := second.Issue(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, issued.ProviderReference, again.ProviderReference)
	pdf2, err := second.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: req.FiscalOperationRequest, ArtifactID: issued.ArtifactID,
	})
	require.NoError(t, err)
	assert.Equal(t, pdf1.Sha256, pdf2.Sha256)
	assert.Equal(t, pdf1.Content, pdf2.Content)
}

func TestMockProvider_ProductionRejectsSelectionAndLegalClassification(t *testing.T) {
	t.Parallel()
	_, err := NewProvider(Options{AppEnv: "production", Registry: NewScenarioRegistry(nil), Store: NewMemoryOperationStore()})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMockForbiddenInProduction)

	require.ErrorIs(t, AssertMockAllowed("production"), ErrMockForbiddenInProduction)
	require.NoError(t, AssertMockAllowed("development"))
	require.ErrorIs(t, RejectLegalMockClassification("production", "legal"), ErrMockLegalClassificationForbidden)
	require.ErrorIs(t, RejectLegalMockClassification("development", "legal"), ErrMockLegalClassificationForbidden)
	require.NoError(t, RejectLegalMockClassification("development", "mock"))
}

func issueRequest(operationKey, kind string, payload []byte) ports.FiscalIssueRequest {
	return ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			ProviderKey:    "mock",
			OperationKey:   operationKey,
			CorrelationKey: "corr-" + operationKey,
			Metadata:       map[string]string{"documentKind": kind},
		},
		DocumentID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Payload:    append([]byte(nil), payload...),
	}
}

func assertFiscalClass(t *testing.T, err error, class ports.FiscalErrorClass) {
	t.Helper()
	var providerErr *ports.FiscalProviderError
	require.Error(t, err)
	require.True(t, errors.As(err, &providerErr))
	assert.Equal(t, class, providerErr.Class)
	assert.True(t, providerErr.IsValid())
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
