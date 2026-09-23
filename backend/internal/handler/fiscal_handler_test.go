package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFiscalizationService struct {
	projection       *ports.FiscalizationProjection
	summaries        []ports.FiscalizationSummary
	artifact         *ports.FiscalArtifactStream
	upsertCreated    bool
	getErr           error
	upsertErr        error
	deleteErr        error
	finalizeErr      error
	retryErr         error
	reconcileErr     error
	voidErr          error
	summariesErr     error
	artifactErr      error
	getCalls         int
	finalizeCalls    int
	retryCalls       int
	lastUpsertDraft  ports.FiscalDraftCommand
	lastSummaryIDs   []uuid.UUID
	lastArtifactMeta ports.FiscalArtifactAccessMeta
}

func (s *stubFiscalizationService) GetProjection(_ context.Context, _, _ uuid.UUID) (*ports.FiscalizationProjection, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.projection == nil {
		return &ports.FiscalizationProjection{Status: string(domain.FiscalPresentationStateLegacyUnfiscalized)}, nil
	}
	cp := *s.projection
	return &cp, nil
}

func (s *stubFiscalizationService) UpsertDraft(_ context.Context, _, _ uuid.UUID, draft ports.FiscalDraftCommand) (*ports.FiscalizationProjection, bool, error) {
	s.lastUpsertDraft = draft
	if s.upsertErr != nil {
		return nil, false, s.upsertErr
	}
	if s.projection == nil {
		return &ports.FiscalizationProjection{Status: string(domain.FiscalPresentationStateDraft), Version: 1}, s.upsertCreated, nil
	}
	cp := *s.projection
	return &cp, s.upsertCreated, nil
}

func (s *stubFiscalizationService) DeleteDraft(_ context.Context, _, _ uuid.UUID, _ int64) error {
	return s.deleteErr
}

func (s *stubFiscalizationService) Finalize(_ context.Context, _, _ uuid.UUID, _ ports.FiscalFinalizeCommand) (*ports.FiscalizationProjection, error) {
	s.finalizeCalls++
	if s.finalizeErr != nil {
		return nil, s.finalizeErr
	}
	if s.projection == nil {
		return &ports.FiscalizationProjection{Status: string(domain.FiscalPresentationStatePending)}, nil
	}
	cp := *s.projection
	return &cp, nil
}

func (s *stubFiscalizationService) Retry(_ context.Context, _, _ uuid.UUID, _ ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	s.retryCalls++
	if s.retryErr != nil {
		return nil, s.retryErr
	}
	cp := *s.projection
	return &cp, nil
}

func (s *stubFiscalizationService) Reconcile(_ context.Context, _, _ uuid.UUID, _ ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	if s.reconcileErr != nil {
		return nil, s.reconcileErr
	}
	cp := *s.projection
	return &cp, nil
}

func (s *stubFiscalizationService) Void(_ context.Context, _, _ uuid.UUID, _ ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	if s.voidErr != nil {
		return nil, s.voidErr
	}
	cp := *s.projection
	return &cp, nil
}

func (s *stubFiscalizationService) Summaries(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]ports.FiscalizationSummary, error) {
	s.lastSummaryIDs = append([]uuid.UUID(nil), ids...)
	if s.summariesErr != nil {
		return nil, s.summariesErr
	}
	return append([]ports.FiscalizationSummary(nil), s.summaries...), nil
}

func (s *stubFiscalizationService) OpenArtifact(_ context.Context, _, _, _ uuid.UUID, meta ports.FiscalArtifactAccessMeta) (*ports.FiscalArtifactStream, error) {
	s.lastArtifactMeta = meta
	if s.artifactErr != nil {
		return nil, s.artifactErr
	}
	if s.artifact == nil {
		return &ports.FiscalArtifactStream{
			Body: io.NopCloser(strings.NewReader("%PDF-1.4")), MediaType: "application/pdf",
			Filename: "fiscal.pdf", Headers: map[string]string{"Content-Disposition": `attachment; filename="fiscal.pdf"`},
		}, nil
	}
	return s.artifact, nil
}

func fiscalRouter(secret string, svc ports.FiscalizationService, inv ports.InvoiceService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	invoiceH := NewInvoiceHandler(inv)
	fiscalH := NewFiscalHandler(svc)
	RegisterInvoiceAndFiscalRoutes(api, invoiceH, fiscalH)
	return r
}

func TestFiscalHandler_SummaryRouteRegisteredBeforeID(t *testing.T) {
	t.Parallel()
	secret := "fiscal-route-order"
	managerID := uuid.New()
	invA := uuid.New()
	svc := &stubFiscalizationService{
		summaries: []ports.FiscalizationSummary{{
			InvoiceID: invA.String(),
			Status:    string(domain.FiscalPresentationStateLegacyUnfiscalized),
		}},
	}
	invStub := &p1StubInvoiceSvc{byID: map[uuid.UUID]*domain.Invoice{}}
	r := fiscalRouter(secret, svc, invStub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/fiscalization-summaries?invoiceIds="+invA.String(), nil)
	req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "static summaries must not be captured by /:id")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	assert.Equal(t, []uuid.UUID{invA}, svc.lastSummaryIDs)
}

func TestFiscalHandler_StrictDecimalJSONRejected(t *testing.T) {
	t.Parallel()
	secret := "fiscal-decimal"
	empID := uuid.New()
	invoiceID := uuid.New()
	svc := &stubFiscalizationService{}
	r := fiscalRouter(secret, svc, &p1StubInvoiceSvc{})

	body := `{
		"version":1,
		"kind":"FT",
		"currency":"EUR",
		"customer":{"legalName":"A","taxIdentifier":"PT1","countryCode":"PT"},
		"billingAddress":{"line1":"Rua","postalCode":"1000","city":"Lisboa","countryCode":"PT"},
		"lines":[{"position":1,"description":"oil","quantity":1.000,"unitCode":"unit","unitPrice":"10.00","discount":{"kind":"none","value":"0"},"taxTreatmentCode":"VAT23","taxRate":"0.23"}]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/draft", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "quantity")
	assert.Equal(t, ports.FiscalDraftCommand{}, svc.lastUpsertDraft)
}

func TestFiscalHandler_BodyAndLineLimits(t *testing.T) {
	t.Parallel()
	secret := "fiscal-limits"
	empID := uuid.New()
	invoiceID := uuid.New()
	svc := &stubFiscalizationService{}
	r := fiscalRouter(secret, svc, &p1StubInvoiceSvc{})

	lines := make([]map[string]any, MaxFiscalDraftLines+1)
	for i := range lines {
		lines[i] = map[string]any{
			"position": i + 1, "description": "line", "quantity": "1.000", "unitCode": "unit",
			"unitPrice": "1.00", "discount": map[string]any{"kind": "none", "value": "0"},
			"taxTreatmentCode": "VAT23", "taxRate": "0.23",
		}
	}
	payload := map[string]any{
		"version": 1, "kind": "FT", "currency": "EUR",
		"customer":       map[string]any{"legalName": "A", "taxIdentifier": "PT1", "countryCode": "PT"},
		"billingAddress": map[string]any{"line1": "Rua", "postalCode": "1000", "city": "Lisboa", "countryCode": "PT"},
		"lines":          lines,
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/draft", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "lines")

	huge := strings.Repeat("a", MaxFiscalRequestBodyBytes+10)
	oversized := `{"version":1,"kind":"FT","currency":"EUR","customer":{"legalName":"` + huge + `","taxIdentifier":"PT1","countryCode":"PT"},"billingAddress":{"line1":"x","postalCode":"1","city":"y","countryCode":"PT"},"lines":[{"position":1,"description":"d","quantity":"1.000","unitCode":"u","unitPrice":"1.00","discount":{"kind":"none","value":"0"},"taxTreatmentCode":"VAT23","taxRate":"0.23"}]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/draft", strings.NewReader(oversized))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, empID, domain.RoleEmployee))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.True(t, w2.Code == http.StatusRequestEntityTooLarge || w2.Code == http.StatusBadRequest || w2.Code == http.StatusUnprocessableEntity)
}

func TestFiscalHandler_StatusMappingsAndRepeatedFinalize(t *testing.T) {
	t.Parallel()
	secret := "fiscal-status"
	managerID := uuid.New()
	employeeID := uuid.New()
	clientID := uuid.New()
	invoiceID := uuid.New()
	docID := uuid.New().String()

	pending := &ports.FiscalizationProjection{
		InvoiceID: invoiceID.String(), DocumentID: &docID, Kind: "FT",
		Lifecycle: string(domain.FiscalDocumentStatePending),
		Status:    string(domain.FiscalPresentationStatePending),
		Version:   2, PayableTotal: "12.34",
		AllowedActions: []string{string(domain.FiscalDocumentActionView)},
	}
	svc := &stubFiscalizationService{projection: pending}
	r := fiscalRouter(secret, svc, &p1StubInvoiceSvc{})

	finBody := `{"expectedVersion":2,"providerKey":"mock","connectionId":"` + uuid.New().String() + `"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/finalize", strings.NewReader(finBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusAccepted, w.Code)
		assert.NotContains(t, w.Body.String(), "credential")
		assert.NotContains(t, w.Body.String(), "storageKey")
		assert.NotContains(t, w.Body.String(), "https://")
	}
	assert.Equal(t, 2, svc.finalizeCalls)

	svc.finalizeErr = domain.ErrUnauthorizedAccess
	reqEmp := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/finalize", strings.NewReader(finBody))
	reqEmp.Header.Set("Content-Type", "application/json")
	reqEmp.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, employeeID, domain.RoleEmployee))
	wEmp := httptest.NewRecorder()
	r.ServeHTTP(wEmp, reqEmp)
	assert.Equal(t, http.StatusForbidden, wEmp.Code)

	svc.finalizeErr = nil
	svc.getErr = domain.ErrInvoiceNotFound
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization", nil)
	req404.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, clientID, domain.RoleClient))
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)
	assert.Equal(t, http.StatusNotFound, w404.Code)

	svc.getErr = nil
	svc.upsertErr = ports.ErrFiscalVersionConflict
	okDraft := `{
		"version":1,"kind":"FT","currency":"EUR",
		"customer":{"legalName":"A","taxIdentifier":"PT1","countryCode":"PT"},
		"billingAddress":{"line1":"Rua","postalCode":"1000","city":"Lisboa","countryCode":"PT"},
		"lines":[{"position":1,"description":"oil","quantity":"1.000","unitCode":"unit","unitPrice":"10.00","discount":{"kind":"none","value":"0"},"taxTreatmentCode":"VAT23","taxRate":"0.23"}]
	}`
	req409 := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/draft", strings.NewReader(okDraft))
	req409.Header.Set("Content-Type", "application/json")
	req409.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, employeeID, domain.RoleEmployee))
	w409 := httptest.NewRecorder()
	r.ServeHTTP(w409, req409)
	assert.Equal(t, http.StatusConflict, w409.Code)

	svc.upsertErr = ports.ErrFiscalNotReady
	req422 := httptest.NewRequest(http.MethodPut, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/draft", strings.NewReader(okDraft))
	req422.Header.Set("Content-Type", "application/json")
	req422.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, employeeID, domain.RoleEmployee))
	w422 := httptest.NewRecorder()
	r.ServeHTTP(w422, req422)
	assert.Equal(t, http.StatusUnprocessableEntity, w422.Code)
}

func TestFiscalHandler_ClientOwnOnlyProjectionAndArtifact(t *testing.T) {
	t.Parallel()
	secret := "fiscal-own"
	ownerID := uuid.New()
	otherID := uuid.New()
	invoiceID := uuid.New()
	artifactID := uuid.New()
	docID := uuid.New().String()

	svc := &stubFiscalizationService{
		projection: &ports.FiscalizationProjection{
			InvoiceID: invoiceID.String(), DocumentID: &docID, Kind: "FT",
			Lifecycle: string(domain.FiscalDocumentStateIssued),
			Status:    string(domain.FiscalPresentationStateFinalized),
			Version:   5, PayableTotal: "20.00",
			AllowedActions: []string{string(domain.FiscalDocumentActionView)},
			ArtifactStatus: "available",
		},
		artifact: &ports.FiscalArtifactStream{
			Body: io.NopCloser(strings.NewReader("%PDF-1.4 mock")), MediaType: "application/pdf",
			Filename: "doc.pdf", Headers: map[string]string{
				"Content-Type": "application/pdf", "Content-Disposition": `attachment; filename="doc.pdf"`,
			},
		},
	}
	r := fiscalRouter(secret, svc, &p1StubInvoiceSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization", nil)
	req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, ownerID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "storageKey")
	assert.NotContains(t, w.Body.String(), "rawProvider")
	assert.NotContains(t, w.Body.String(), "credential")

	artReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String()+"/fiscal-artifacts/"+artifactID.String(), nil)
	artReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, ownerID, domain.RoleClient))
	artW := httptest.NewRecorder()
	r.ServeHTTP(artW, artReq)
	require.Equal(t, http.StatusOK, artW.Code)
	assert.Equal(t, "application/pdf", artW.Header().Get("Content-Type"))
	assert.Contains(t, artW.Header().Get("Content-Disposition"), "attachment")
	assert.NotContains(t, artW.Body.String(), "s3://")
	assert.NotContains(t, artW.Header().Get("Location"), "http")

	svc.getErr = domain.ErrInvoiceNotFound
	svc.artifactErr = ports.ErrFiscalArtifactDenied
	deny := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization", nil)
	deny.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, otherID, domain.RoleClient))
	denyW := httptest.NewRecorder()
	r.ServeHTTP(denyW, deny)
	assert.Equal(t, http.StatusNotFound, denyW.Code)
}

func TestSanitizeFiscalizationProjection_OmitsSecrets(t *testing.T) {
	t.Parallel()
	docID := uuid.New().String()
	proj := sanitizeFiscalizationProjection(&ports.FiscalizationProjection{
		InvoiceID: uuid.New().String(), DocumentID: &docID, Status: "pending",
		AllowedActions: []string{"view"},
		LastErrorSafe:  "provider said Bearer secret-token https://provider.example/callback",
	})
	raw, err := json.Marshal(proj)
	require.NoError(t, err)
	assert.True(t, projectionJSONOmitsSecrets(raw))
	assert.Equal(t, "[redacted]", proj.LastErrorSafe)
	assert.NotContains(t, string(raw), "credentialCiphertext")
	assert.NotContains(t, string(raw), "storageKey")
}

func TestFiscalHandler_ManagerAdminOnlyLegalActions(t *testing.T) {
	t.Parallel()
	secret := "fiscal-legal"
	managerID := uuid.New()
	employeeID := uuid.New()
	invoiceID := uuid.New()
	docID := uuid.New().String()
	proj := &ports.FiscalizationProjection{
		InvoiceID: invoiceID.String(), DocumentID: &docID,
		Status: string(domain.FiscalPresentationStatePending), Version: 3,
	}
	svc := &stubFiscalizationService{projection: proj}
	r := fiscalRouter(secret, svc, &p1StubInvoiceSvc{})

	actionBody := `{"expectedVersion":3}`
	for _, path := range []string{"retry", "reconcile", "void"} {
		svc.retryErr = nil
		svc.reconcileErr = nil
		svc.voidErr = nil
		req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/"+path, strings.NewReader(actionBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusAccepted, w.Code, path)

		svc.retryErr = domain.ErrUnauthorizedAccess
		svc.reconcileErr = domain.ErrUnauthorizedAccess
		svc.voidErr = domain.ErrUnauthorizedAccess
		reqEmp := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/fiscalization/"+path, strings.NewReader(actionBody))
		reqEmp.Header.Set("Content-Type", "application/json")
		reqEmp.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, employeeID, domain.RoleEmployee))
		wEmp := httptest.NewRecorder()
		r.ServeHTTP(wEmp, reqEmp)
		assert.Equal(t, http.StatusForbidden, wEmp.Code, path)
	}
}
