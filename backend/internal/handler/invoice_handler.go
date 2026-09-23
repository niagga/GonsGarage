package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// InvoiceHandler exposes customer-issued invoices (client: own; staff: CRUD list).
type InvoiceHandler struct {
	svc ports.InvoiceService
}

func NewInvoiceHandler(svc ports.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// IssuedInvoiceResponse JSON for customer invoices (emitidas al cliente).
type IssuedInvoiceResponse struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customerId"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

func issuedInvoiceToResponse(inv *domain.Invoice) IssuedInvoiceResponse {
	if inv == nil {
		return IssuedInvoiceResponse{}
	}
	return IssuedInvoiceResponse{
		ID:         inv.ID.String(),
		CustomerID: inv.CustomerID.String(),
		Amount:     inv.Amount,
		Status:     inv.Status,
		Notes:      inv.Notes,
		CreatedAt:  inv.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  inv.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// CreateIssuedInvoiceRequest body for staff POST /invoices.
type CreateIssuedInvoiceRequest struct {
	CustomerID string  `json:"customerId"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes"`
}

// PatchIssuedInvoiceRequest body for PATCH /invoices/:id (client notes; staff may send status/amount).
type PatchIssuedInvoiceRequest struct {
	Notes  *string  `json:"notes,omitempty"`
	Status *string  `json:"status,omitempty"`
	Amount *float64 `json:"amount,omitempty"`
}

func writeInvoiceServiceError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrUnauthorizedAccess) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return true
	}
	if errors.Is(err, domain.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return true
	}
	if errors.Is(err, domain.ErrInvoiceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return true
	}
	if errors.Is(err, ports.ErrFiscalHistoryProtected) {
		c.JSON(http.StatusConflict, gin.H{"error": "invoice has protected fiscal history"})
		return true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return true
}

// ListMyInvoices GET /api/v1/invoices/me
// @Summary     Listar mis facturas (cliente)
// @Tags        invoices
// @Security    BearerAuth
// @Produce     json
// @Param       limit query int false "Límite (default 20, max 100)"
// @Param       offset query int false "Offset"
// @Success     200 {object} map[string]interface{} "{items,total}"
// @Router      /api/v1/invoices/me [get]
func (h *InvoiceHandler) ListMyInvoices(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	limit, offset := QueryLimitOffset(c, 20, 100)
	list, total, err := h.svc.ListMyInvoices(c.Request.Context(), uid, limit, offset)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	items := make([]IssuedInvoiceResponse, 0, len(list))
	for _, inv := range list {
		items = append(items, issuedInvoiceToResponse(inv))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// GetIssuedInvoice GET /api/v1/invoices/:id
// @Summary     Obtener factura emitida al cliente
// @Tags        invoices
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "UUID"
// @Success     200 {object} IssuedInvoiceResponse
// @Router      /api/v1/invoices/{id} [get]
func (h *InvoiceHandler) GetIssuedInvoice(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	out, err := h.svc.GetInvoice(c.Request.Context(), id, uid)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, issuedInvoiceToResponse(out))
}

// PatchIssuedInvoice PATCH /api/v1/invoices/:id
// @Summary     Actualizar factura (notas cliente; staff: estado/importe)
// @Tags        invoices
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID"
// @Param       body body PatchIssuedInvoiceRequest true "Campos"
// @Success     200 {object} IssuedInvoiceResponse
// @Router      /api/v1/invoices/{id} [patch]
func (h *InvoiceHandler) PatchIssuedInvoice(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req PatchIssuedInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	existing, err := h.svc.GetInvoice(c.Request.Context(), id, uid)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	patch := *existing
	if req.Notes != nil {
		patch.Notes = *req.Notes
	}
	if req.Status != nil {
		patch.Status = *req.Status
	}
	if req.Amount != nil {
		patch.Amount = *req.Amount
	}
	out, err := h.svc.UpdateInvoice(c.Request.Context(), &patch, uid)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, issuedInvoiceToResponse(out))
}

// CreateIssuedInvoice POST /api/v1/invoices (staff only — use RequireWorkshopStaff on route group)
// @Summary     Crear factura emitida a cliente
// @Tags        invoices
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body CreateIssuedInvoiceRequest true "Datos"
// @Success     201 {object} IssuedInvoiceResponse
// @Router      /api/v1/invoices [post]
func (h *InvoiceHandler) CreateIssuedInvoice(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req CreateIssuedInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customerId"})
		return
	}
	inv := &domain.Invoice{
		CustomerID: custID,
		Amount:     req.Amount,
		Status:     req.Status,
		Notes:      req.Notes,
	}
	out, err := h.svc.CreateInvoice(c.Request.Context(), inv, uid)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, issuedInvoiceToResponse(out))
}

// ListIssuedInvoicesStaff GET /api/v1/invoices (staff only)
// @Summary     Listar todas las facturas emitidas (taller)
// @Tags        invoices
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]interface{} "{items,total}"
// @Router      /api/v1/invoices [get]
func (h *InvoiceHandler) ListIssuedInvoicesStaff(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	limit, offset := QueryLimitOffset(c, 20, 100)
	list, total, err := h.svc.ListInvoicesForStaff(c.Request.Context(), uid, limit, offset)
	if err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	items := make([]IssuedInvoiceResponse, 0, len(list))
	for _, inv := range list {
		items = append(items, issuedInvoiceToResponse(inv))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// DeleteIssuedInvoice DELETE /api/v1/invoices/:id (staff only)
// @Summary     Eliminar factura emitida
// @Tags        invoices
// @Security    BearerAuth
// @Param       id path string true "UUID"
// @Success     204
// @Router      /api/v1/invoices/{id} [delete]
func (h *InvoiceHandler) DeleteIssuedInvoice(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteInvoice(c.Request.Context(), id, uid); err != nil {
		writeInvoiceServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
