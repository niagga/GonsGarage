package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Stubs (P1 accounting HTTP + RBAC) ---

type p1StubReceivedSvc struct {
	list []*domain.ReceivedInvoice
	tot  int64
}

func (s *p1StubReceivedSvc) Create(_ context.Context, inv *domain.ReceivedInvoice, _ uuid.UUID) (*domain.ReceivedInvoice, error) {
	id := uuid.New()
	now := time.Now().UTC()
	out := *inv
	out.ID = id
	out.CreatedAt = now
	out.UpdatedAt = now
	return &out, nil
}

func (s *p1StubReceivedSvc) Get(_ context.Context, id uuid.UUID, _ uuid.UUID) (*domain.ReceivedInvoice, error) {
	for _, r := range s.list {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, domain.ErrReceivedInvoiceNotFound
}

func (s *p1StubReceivedSvc) List(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.ReceivedInvoice, int64, error) {
	return s.list, s.tot, nil
}

func (s *p1StubReceivedSvc) Update(_ context.Context, inv *domain.ReceivedInvoice, _ uuid.UUID) (*domain.ReceivedInvoice, error) {
	now := time.Now().UTC()
	inv.UpdatedAt = now
	return inv, nil
}

func (s *p1StubReceivedSvc) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type p1StubBillingSvc struct{}

func (s *p1StubBillingSvc) Create(_ context.Context, doc *domain.BillingDocument, _ uuid.UUID) (*domain.BillingDocument, error) {
	id := uuid.New()
	now := time.Now().UTC()
	doc.ID = id
	doc.CreatedAt = now
	doc.UpdatedAt = now
	return doc, nil
}

func (s *p1StubBillingSvc) Get(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.BillingDocument, error) {
	return nil, domain.ErrBillingDocumentNotFound
}

func (s *p1StubBillingSvc) List(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.BillingDocument, int64, error) {
	return nil, 0, nil
}

func (s *p1StubBillingSvc) Update(_ context.Context, doc *domain.BillingDocument, _ uuid.UUID) (*domain.BillingDocument, error) {
	return doc, nil
}

func (s *p1StubBillingSvc) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type p1StubInvoiceSvc struct {
	myInvoices []*domain.Invoice
	byID       map[uuid.UUID]*domain.Invoice
}

func (s *p1StubInvoiceSvc) GetInvoice(_ context.Context, invoiceID uuid.UUID, requestingUserID uuid.UUID) (*domain.Invoice, error) {
	inv, ok := s.byID[invoiceID]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	if inv.CustomerID != requestingUserID {
		return nil, domain.ErrUnauthorizedAccess
	}
	return inv, nil
}

func (s *p1StubInvoiceSvc) UpdateInvoice(_ context.Context, inv *domain.Invoice, _ uuid.UUID) (*domain.Invoice, error) {
	return inv, nil
}

func (s *p1StubInvoiceSvc) ListMyInvoices(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Invoice, int64, error) {
	return s.myInvoices, int64(len(s.myInvoices)), nil
}

func (s *p1StubInvoiceSvc) CreateInvoice(_ context.Context, inv *domain.Invoice, _ uuid.UUID) (*domain.Invoice, error) {
	now := time.Now().UTC()
	out := *inv
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	out.CreatedAt = now
	out.UpdatedAt = now
	return &out, nil
}

func (s *p1StubInvoiceSvc) ListInvoicesForStaff(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Invoice, int64, error) {
	if s.myInvoices != nil {
		return s.myInvoices, int64(len(s.myInvoices)), nil
	}
	return nil, 0, nil
}

func (s *p1StubInvoiceSvc) DeleteInvoice(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func p1AccountingRouter(secret string, recv *p1StubReceivedSvc, bill *p1StubBillingSvc, inv *p1StubInvoiceSvc, sup *stubSupplierService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(middleware.GinBearerJWT(am))

	supplierH := NewSupplierHandler(sup)
	receivedH := NewReceivedInvoiceHandler(recv)
	billingH := NewBillingDocumentHandler(bill)
	invoiceH := NewInvoiceHandler(inv)

	suppliers := protected.Group("/suppliers")
	suppliers.Use(middleware.RequireWorkshopStaff())
	{
		suppliers.POST("", supplierH.CreateSupplier)
		suppliers.GET("", supplierH.ListSuppliers)
		suppliers.GET("/:id", supplierH.GetSupplier)
		suppliers.PUT("/:id", supplierH.UpdateSupplier)
		suppliers.DELETE("/:id", supplierH.DeleteSupplier)
	}

	receivedInvoices := protected.Group("/received-invoices")
	receivedInvoices.Use(middleware.RequireWorkshopStaff())
	{
		receivedInvoices.POST("", receivedH.CreateReceivedInvoice)
		receivedInvoices.GET("", receivedH.ListReceivedInvoices)
		receivedInvoices.GET("/:id", receivedH.GetReceivedInvoice)
		receivedInvoices.PUT("/:id", receivedH.UpdateReceivedInvoice)
		receivedInvoices.DELETE("/:id", receivedH.DeleteReceivedInvoice)
	}

	billingDocs := protected.Group("/billing-documents")
	billingDocs.Use(middleware.RequireWorkshopStaff())
	{
		billingDocs.POST("", billingH.CreateBillingDocument)
		billingDocs.GET("", billingH.ListBillingDocuments)
		billingDocs.GET("/:id", billingH.GetBillingDocument)
		billingDocs.PUT("/:id", billingH.UpdateBillingDocument)
		billingDocs.DELETE("/:id", billingH.DeleteBillingDocument)
	}

	invoices := protected.Group("/invoices")
	{
		invoices.GET("/me", invoiceH.ListMyInvoices)
		staffInvoices := invoices.Group("")
		staffInvoices.Use(middleware.RequireWorkshopStaff())
		{
			staffInvoices.POST("", invoiceH.CreateIssuedInvoice)
			staffInvoices.GET("", invoiceH.ListIssuedInvoicesStaff)
			staffInvoices.DELETE("/:id", invoiceH.DeleteIssuedInvoice)
		}
		invoices.GET("/:id", invoiceH.GetIssuedInvoice)
		invoices.PATCH("/:id", invoiceH.PatchIssuedInvoice)
	}
	return r
}

func TestP1Accounting_ClientGETReceivedInvoices_403(t *testing.T) {
	t.Parallel()
	secret := "p1-rbac-secret-recv"
	clientID := uuid.New()
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/received-invoices", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestP1Accounting_ClientGETSuppliers_403(t *testing.T) {
	t.Parallel()
	secret := "p1-rbac-secret-sup"
	clientID := uuid.New()
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestP1Accounting_ClientGETBillingDocuments_403(t *testing.T) {
	t.Parallel()
	secret := "p1-rbac-secret-bill"
	clientID := uuid.New()
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing-documents", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestP1Accounting_ClientPOSTInvoicesStaff_403(t *testing.T) {
	t.Parallel()
	secret := "p1-rbac-secret-inv-post"
	clientID := uuid.New()
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	body := `{"customerId":"` + uuid.New().String() + `","amount":10,"status":"open"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestP1Accounting_EmployeeGETReceivedInvoices_200(t *testing.T) {
	t.Parallel()
	secret := "p1-ok-recv-list"
	empID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()
	stub := &p1StubReceivedSvc{
		list: []*domain.ReceivedInvoice{{
			ID: invID, VendorName: "ACME", Category: "parts", Amount: 42.5,
			InvoiceDate: now, Notes: "n", CreatedAt: now, UpdatedAt: now,
		}},
		tot: 1,
	}
	r := p1AccountingRouter(secret, stub, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/received-invoices?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.EqualValues(t, 1, out["total"])
	items, ok := out["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)
}

func TestP1Accounting_EmployeePOSTReceivedInvoice_201(t *testing.T) {
	t.Parallel()
	secret := "p1-ok-recv-create"
	empID := uuid.New()
	stub := &p1StubReceivedSvc{}
	r := p1AccountingRouter(secret, stub, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	payload := map[string]interface{}{
		"vendorName":  "Vendor X",
		"category":    "fuel",
		"amount":      100.0,
		"invoiceDate": "2026-01-15",
		"notes":       "ok",
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/received-invoices", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, "Vendor X", out["vendorName"])
	assert.Equal(t, 100.0, out["amount"])
}

func TestP1Accounting_EmployeePUTReceivedInvoice_200(t *testing.T) {
	t.Parallel()
	secret := "p1-ok-recv-put"
	empID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()
	stub := &p1StubReceivedSvc{
		list: []*domain.ReceivedInvoice{{
			ID: invID, VendorName: "Old", Category: "parts", Amount: 10,
			InvoiceDate: now, Notes: "", CreatedAt: now, UpdatedAt: now,
		}},
		tot: 1,
	}
	r := p1AccountingRouter(secret, stub, &p1StubBillingSvc{}, &p1StubInvoiceSvc{}, &stubSupplierService{})

	payload := map[string]interface{}{
		"vendorName":  "NewName",
		"category":    "parts",
		"amount":      20.0,
		"invoiceDate": now.UTC().Format(time.RFC3339),
		"notes":       "updated",
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/received-invoices/"+invID.String(), bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, "NewName", out["vendorName"])
}

func TestP1Accounting_ClientGETInvoicesMe_200(t *testing.T) {
	t.Parallel()
	secret := "p1-ok-inv-me"
	clientID := uuid.New()
	invUUID := uuid.New()
	now := time.Now().UTC()
	invStub := &p1StubInvoiceSvc{
		myInvoices: []*domain.Invoice{{
			ID: invUUID, CustomerID: clientID, Amount: 55, Status: "open",
			Notes: "hello", CreatedAt: now, UpdatedAt: now,
		}},
		byID: map[uuid.UUID]*domain.Invoice{},
	}
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, invStub, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/me", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.EqualValues(t, 1, out["total"])
}

func TestP1Accounting_ClientGETIssuedInvoiceOwn_200(t *testing.T) {
	t.Parallel()
	secret := "p1-ok-inv-get"
	clientID := uuid.New()
	invUUID := uuid.New()
	now := time.Now().UTC()
	row := &domain.Invoice{
		ID: invUUID, CustomerID: clientID, Amount: 99, Status: "open",
		Notes: "", CreatedAt: now, UpdatedAt: now,
	}
	invStub := &p1StubInvoiceSvc{
		myInvoices: nil,
		byID:       map[uuid.UUID]*domain.Invoice{invUUID: row},
	}
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, invStub, &stubSupplierService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, invUUID.String(), out["id"])
	assert.Equal(t, 99.0, out["amount"])
}

func TestP1Accounting_IssuedInvoiceLegacyContractSnapshot(t *testing.T) {
	t.Parallel()
	secret := "p1-inv-legacy-contract"
	managerID := uuid.New()
	clientID := uuid.New()
	now := time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC)
	invID := uuid.New()
	invStub := &p1StubInvoiceSvc{
		byID: map[uuid.UUID]*domain.Invoice{
			invID: {ID: invID, CustomerID: clientID, Amount: 12.5, Status: "open", Notes: "n", CreatedAt: now, UpdatedAt: now},
		},
		myInvoices: []*domain.Invoice{
			{ID: invID, CustomerID: clientID, Amount: 12.5, Status: "open", Notes: "n", CreatedAt: now, UpdatedAt: now},
		},
	}
	r := p1AccountingRouter(secret, &p1StubReceivedSvc{}, &p1StubBillingSvc{}, invStub, &stubSupplierService{})

	createBody := `{"customerId":"` + clientID.String() + `","amount":12.5,"status":"open","notes":"n"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, managerID, domain.RoleManager))
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &created))
	for _, key := range []string{"id", "customerId", "amount", "status", "notes", "createdAt", "updatedAt"} {
		assert.Contains(t, created, key)
	}
	assert.NotContains(t, created, "fiscalEligibility")
	assert.NotContains(t, created, "fiscalization")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices?limit=20&offset=0", nil)
	listReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, managerID, domain.RoleManager))
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)
	var listBody map[string]any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listBody))
	assert.Contains(t, listBody, "items")
	assert.Contains(t, listBody, "total")

	ownReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invID.String(), nil)
	ownReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	ownW := httptest.NewRecorder()
	r.ServeHTTP(ownW, ownReq)
	require.Equal(t, http.StatusOK, ownW.Code)
	var own map[string]any
	require.NoError(t, json.Unmarshal(ownW.Body.Bytes(), &own))
	assert.Equal(t, "2026-06-01T09:30:00Z", own["createdAt"])
	assert.Equal(t, "2026-06-01T09:30:00Z", own["updatedAt"])

	patch := `{"notes":"only-notes"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/invoices/"+invID.String(), bytes.NewBufferString(patch))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	require.Equal(t, http.StatusOK, patchW.Code)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/invoices/"+invID.String(), nil)
	delReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, managerID, domain.RoleManager))
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)
	require.Equal(t, http.StatusNoContent, delW.Code)
}

func TestP1Accounting_FiscalSummariesDoNotChangeInvoiceEnvelope(t *testing.T) {
	t.Parallel()
	secret := "p1-fiscal-additive"
	managerID := uuid.New()
	invoiceID := uuid.New()
	invStub := &p1StubInvoiceSvc{}
	fiscalSvc := &stubFiscalizationService{
		summaries: []ports.FiscalizationSummary{{InvoiceID: invoiceID.String(), Status: "legacy_unfiscalized"}},
	}
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	RegisterInvoiceAndFiscalRoutes(api, NewInvoiceHandler(invStub), NewFiscalHandler(fiscalSvc))

	sumReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/fiscalization-summaries?invoiceIds="+invoiceID.String(), nil)
	sumReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, managerID, domain.RoleManager))
	sumW := httptest.NewRecorder()
	r.ServeHTTP(sumW, sumReq)
	require.Equal(t, http.StatusOK, sumW.Code)
	var sumBody map[string]any
	require.NoError(t, json.Unmarshal(sumW.Body.Bytes(), &sumBody))
	assert.Contains(t, sumBody, "items")
	assert.NotContains(t, sumBody, "total")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	listReq.Header.Set("Authorization", "Bearer "+testJWT(t, secret, managerID, domain.RoleManager))
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)
	var listBody map[string]any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listBody))
	assert.Contains(t, listBody, "items")
	assert.Contains(t, listBody, "total")
}
