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

type invoiceDeleteStub struct {
	p1StubInvoiceSvc
	deleteErr error
	deleted   uuid.UUID
}

func (s *invoiceDeleteStub) DeleteInvoice(_ context.Context, invoiceID uuid.UUID, _ uuid.UUID) error {
	s.deleted = invoiceID
	return s.deleteErr
}

func TestInvoiceHandler_DeleteProtectedFiscalHistory_409(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "inv-fiscal-protect"
	managerID := uuid.New()
	invoiceID := uuid.New()
	stub := &invoiceDeleteStub{deleteErr: ports.ErrFiscalHistoryProtected}
	h := NewInvoiceHandler(stub)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(middleware.NewAuthMiddleware(secret)))
	staff := api.Group("/invoices")
	staff.Use(middleware.RequireWorkshopStaff())
	staff.DELETE("/:id", h.DeleteIssuedInvoice)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoices/"+invoiceID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "fiscal")
	assert.Equal(t, invoiceID, stub.deleted)
}

func TestInvoiceHandler_LegacyContractSnapshot(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "inv-legacy-snap"
	clientID := uuid.New()
	invoiceID := uuid.New()
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	row := &domain.Invoice{
		ID: invoiceID, CustomerID: clientID, Amount: 77.5, Status: "open",
		Notes: "keep", CreatedAt: now, UpdatedAt: now,
	}
	stub := &p1StubInvoiceSvc{
		myInvoices: []*domain.Invoice{row},
		byID:       map[uuid.UUID]*domain.Invoice{invoiceID: row},
	}
	h := NewInvoiceHandler(stub)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	invoices := api.Group("/invoices")
	invoices.GET("/me", h.ListMyInvoices)
	invoices.GET("/:id", h.GetIssuedInvoice)
	invoices.PATCH("/:id", h.PatchIssuedInvoice)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/me?limit=20&offset=0", nil)
	listReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, clientID, domain.RoleClient))
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)
	var listBody map[string]any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listBody))
	assert.Contains(t, listBody, "items")
	assert.Contains(t, listBody, "total")
	items := listBody["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	for _, key := range []string{"id", "customerId", "amount", "status", "notes", "createdAt", "updatedAt"} {
		assert.Contains(t, item, key)
	}
	assert.NotContains(t, item, "fiscalEligibility")
	assert.Equal(t, "2026-03-15T12:00:00Z", item["createdAt"])

	patch := `{"notes":"client-only"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/invoices/"+invoiceID.String(), bytes.NewBufferString(patch))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, clientID, domain.RoleClient))
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	require.Equal(t, http.StatusOK, patchW.Code)
	var patched map[string]any
	require.NoError(t, json.Unmarshal(patchW.Body.Bytes(), &patched))
	assert.Equal(t, "client-only", patched["notes"])
	assert.Equal(t, 77.5, patched["amount"])
}
