package cloudware_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/cloudware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapFrozenFTAndFROnly(t *testing.T) {
	t.Parallel()

	ft, err := cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{
		Kind:              "FT",
		ExternalReference: "inv-100",
		SeriesPrefix:      "FT2025",
		CustomerNIF:       "123456789",
		CustomerName:      "Cliente SA",
		Lines: []cloudware.FrozenLineInput{{
			Description: "Reparação",
			Quantity:    "1.000",
			UnitPrice:   "100.00",
			TaxCode:     "NOR",
			TaxPercent:  "23.00",
		}},
		Finalize:  true,
		ReturnPDF: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "FT", ft.DocumentType)
	assert.Equal(t, "inv-100", ft.ExternalReference)
	assert.True(t, ft.Finalize)
	assert.True(t, ft.ReturnPDF)
	assert.False(t, ft.AssociatedReceipt)
	require.Len(t, ft.Lines, 1)

	fr, err := cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{
		Kind:              "FR",
		ExternalReference: "inv-101",
		SeriesPrefix:      "FR2025",
		CustomerNIF:       "123456789",
		CustomerName:      "Cliente SA",
		Lines: []cloudware.FrozenLineInput{{
			Description: "Peça",
			Quantity:    "2.000",
			UnitPrice:   "10.00",
			TaxCode:     "NOR",
			TaxPercent:  "23.00",
		}},
		Finalize: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "FR", fr.DocumentType)
	assert.True(t, fr.AssociatedReceipt, "FR associated-receipt is Cloudware FR behavior, not a local receipt workflow")
}

func TestMapRejectsUnsupportedKindsAndReceiptWorkflows(t *testing.T) {
	t.Parallel()

	_, err := cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{Kind: "FS", ExternalReference: "x", Lines: []cloudware.FrozenLineInput{{Description: "a", Quantity: "1", UnitPrice: "1", TaxCode: "NOR"}}})
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrUnsupportedDocumentKind)

	_, err = cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{Kind: "NC", ExternalReference: "x", Lines: []cloudware.FrozenLineInput{{Description: "a", Quantity: "1", UnitPrice: "1", TaxCode: "NOR"}}})
	require.Error(t, err)

	err = cloudware.ValidateWorkflow(cloudware.WorkflowStandaloneReceipt)
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrUnsupportedWorkflow)

	err = cloudware.ValidateWorkflow(cloudware.WorkflowPartialReceipt)
	require.Error(t, err)

	err = cloudware.ValidateWorkflow(cloudware.WorkflowAutomatedCorrection)
	require.Error(t, err)

	err = cloudware.ValidateWorkflow(cloudware.WorkflowFTFRIssue)
	require.NoError(t, err)
}

func TestEnablementEvaluatorBlocksMutationBeforeHTTP(t *testing.T) {
	t.Parallel()

	var httpCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"doc-1"}`))
	}))
	t.Cleanup(server.Close)

	gates := cloudware.NewGateCatalog("production")
	for _, key := range cloudware.MandatoryGateKeys() {
		gates.Set(key, cloudware.GatePending, "", "", "", "")
	}
	gates.Set(cloudware.GateWebhooks, cloudware.GateNotApplicable, "", "", "", "webhooks unused in first release")

	client, err := cloudware.NewHTTPClient(cloudware.HTTPClientConfig{
		BaseURL:          server.URL,
		AllowedHosts:     []string{strings.TrimPrefix(strings.TrimPrefix(server.URL, "https://"), "http://")},
		Timeout:          2 * time.Second,
		MaxResponseBytes: 1 << 20,
	})
	require.NoError(t, err)

	adapter := cloudware.NewAdapter(client, gates, nil)
	_, err = adapter.Issue(context.Background(), ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{ProviderKey: "cloudware", OperationKey: "op-1"},
		DocumentID:             uuid.New(),
		Payload: mustJSON(t, cloudware.FrozenDocumentInput{
			Kind: "FT", ExternalReference: "inv", Lines: []cloudware.FrozenLineInput{{Description: "x", Quantity: "1", UnitPrice: "1", TaxCode: "NOR"}},
		}),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrEnablementIncomplete)
	assert.Equal(t, 0, httpCalls, "incomplete gates must block before any Cloudware HTTP")

	// Satisfy every mandatory gate — still blocked without credentialed acceptance marker for production mutations.
	for _, key := range cloudware.MandatoryGateKeys() {
		gates.Set(key, cloudware.GateSatisfied, "evidence://"+key, "decision://"+key, "acceptance://"+key, "")
	}
	_, err = adapter.Issue(context.Background(), ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{ProviderKey: "cloudware", OperationKey: "op-2"},
		DocumentID:             uuid.New(),
		Payload: mustJSON(t, cloudware.FrozenDocumentInput{
			Kind: "FT", ExternalReference: "inv", Lines: []cloudware.FrozenLineInput{{Description: "x", Quantity: "1", UnitPrice: "1", TaxCode: "NOR"}},
		}),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrEnablementIncomplete)
	assert.Equal(t, 0, httpCalls)
}

func TestGateNotApplicableRequiresRationale(t *testing.T) {
	t.Parallel()

	gates := cloudware.NewGateCatalog("production")
	err := gates.Set(cloudware.GateWebhooks, cloudware.GateNotApplicable, "", "", "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrGateRationaleRequired)

	err = gates.Set(cloudware.GateWebhooks, cloudware.GateNotApplicable, "", "", "", "not used; no webhook routes registered")
	require.NoError(t, err)

	err = gates.Set(cloudware.GateOAuthSecurity, cloudware.GateNotApplicable, "", "", "", "should not be optional")
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrGateNotOptional)
}

func TestNormalizeErrorsRefreshLicenseAmbiguous(t *testing.T) {
	t.Parallel()

	refresh := cloudware.NormalizeHTTPError(http.StatusUnauthorized, []byte(`{"error":"invalid_grant","error_description":"refresh rejected"}`))
	require.NotNil(t, refresh)
	assert.Equal(t, ports.FiscalErrorClassUnauthorized, refresh.Class)
	assert.Equal(t, "refresh_failed", refresh.Code)
	assert.NotContains(t, refresh.Message, "invalid_grant")

	license := cloudware.NormalizeHTTPError(http.StatusForbidden, []byte(`{"code":"active_license_required","message":"GC license missing"}`))
	require.NotNil(t, license)
	assert.Equal(t, ports.FiscalErrorClassAuthorization, license.Class)
	assert.Equal(t, "active_license_required", license.Code)
	assert.Contains(t, license.Message, "license")

	unknown := cloudware.NormalizeHTTPError(http.StatusOK, []byte(`{"weird":true}`))
	require.NotNil(t, unknown)
	assert.Equal(t, ports.FiscalErrorClassAmbiguous, unknown.Class)
	assert.False(t, unknown.IsRetryable())
}

func TestContractFixturesFinalizeVoidPDFAndFRReceipt(t *testing.T) {
	t.Parallel()

	finalizeBody, err := cloudware.BuildFinalizeRequest("cw-doc-1")
	require.NoError(t, err)
	assert.Contains(t, string(finalizeBody), "finalize")

	voidBody, err := cloudware.BuildVoidRequest("cw-doc-1", "customer request")
	require.NoError(t, err)
	assert.Contains(t, string(voidBody), "void")

	pdfBody, err := cloudware.BuildPDFRequest("cw-doc-1")
	require.NoError(t, err)
	assert.Contains(t, string(pdfBody), "return_pdf")

	fr, err := cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{
		Kind: "FR", ExternalReference: "inv-fr", SeriesPrefix: "FR", CustomerNIF: "123456789", CustomerName: "C",
		Lines:    []cloudware.FrozenLineInput{{Description: "svc", Quantity: "1", UnitPrice: "5", TaxCode: "NOR", TaxPercent: "23"}},
		Finalize: true, ReturnPDF: true,
	})
	require.NoError(t, err)
	assert.True(t, fr.AssociatedReceipt)
	assert.True(t, fr.Finalize)
	assert.True(t, fr.ReturnPDF)
}

func TestHTTPClientAllowlistDeadlinesAndNoMutationRetry(t *testing.T) {
	t.Parallel()

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"boom"}`)
	}))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(strings.TrimPrefix(server.URL, "https://"), "http://")
	client, err := cloudware.NewHTTPClient(cloudware.HTTPClientConfig{
		BaseURL:          server.URL,
		AllowedHosts:     []string{host},
		Timeout:          2 * time.Second,
		MaxResponseBytes: 4096,
	})
	require.NoError(t, err)

	_, err = client.DoJSON(context.Background(), http.MethodPost, "/v1/commercial_sales_documents", map[string]string{"Authorization": "Bearer secret-token"}, map[string]any{"document_type": "FT"})
	require.Error(t, err)
	assert.Equal(t, 1, hits, "transport must not retry mutations")

	_, err = client.DoJSON(context.Background(), http.MethodGet, "https://evil.example/steal", nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, cloudware.ErrBaseURLNotAllowed)
}

func TestAdapterRedactsSecretsFromDiagnostics(t *testing.T) {
	t.Parallel()

	redacted := cloudware.RedactDiagnostics("Authorization: Bearer abc.def.ghi refresh_token=xyz access_token=aaa")
	assert.NotContains(t, redacted, "abc.def.ghi")
	assert.NotContains(t, redacted, "xyz")
	assert.NotContains(t, redacted, "aaa")
	assert.Contains(t, redacted, "[REDACTED]")
}

func TestCloudwareImportIsolation(t *testing.T) {
	t.Parallel()

	// Compile-time isolation is enforced by keeping Cloudware DTOs adapter-local.
	// Runtime check: domain package path must not appear as an importer of cloudware symbols in this test binary's dependency sense —
	// the public ports package must remain free of Cloudware request DTOs.
	var _ ports.FiscalProvider = (*cloudware.Adapter)(nil)

	req, err := cloudware.MapFrozenDocument(cloudware.FrozenDocumentInput{
		Kind: "FT", ExternalReference: "iso", Lines: []cloudware.FrozenLineInput{{Description: "d", Quantity: "1", UnitPrice: "1", TaxCode: "NOR"}},
	})
	require.NoError(t, err)
	raw, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "document_type")
	assert.NotContains(t, string(raw), "FiscalIssueRequest")
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestReadinessReportSurfacesIncompleteGates(t *testing.T) {
	t.Parallel()

	gates := cloudware.NewGateCatalog("production")
	for _, key := range cloudware.MandatoryGateKeys() {
		_ = gates.Set(key, cloudware.GatePending, "", "", "", "")
	}
	_ = gates.Set(cloudware.GateWebhooks, cloudware.GateNotApplicable, "", "", "", "unused")

	report := gates.Readiness()
	assert.False(t, report.Ready)
	require.NotEmpty(t, report.Gates)
	var webhook *cloudware.GateView
	for i := range report.Gates {
		if report.Gates[i].Key == cloudware.GateWebhooks {
			webhook = &report.Gates[i]
		}
	}
	require.NotNil(t, webhook)
	assert.Equal(t, cloudware.GateNotApplicable, webhook.Status)
	assert.Equal(t, "unused", webhook.Rationale)

	incomplete := gates.IncompleteMandatory()
	assert.NotEmpty(t, incomplete)
	assert.NotContains(t, incomplete, cloudware.GateWebhooks)
}

// Compile guard: NormalizeHTTPError must return a typed provider error.
func TestNormalizeHTTPErrorType(t *testing.T) {
	t.Parallel()
	err := cloudware.NormalizeHTTPError(499, []byte(`{}`))
	var pe *ports.FiscalProviderError
	require.True(t, errors.As(err, &pe))
}
