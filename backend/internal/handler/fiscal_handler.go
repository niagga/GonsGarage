package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
)

const (
	// MaxFiscalDraftLines bounds draft line payloads.
	MaxFiscalDraftLines = 100
	// MaxFiscalRequestBodyBytes bounds fiscal mutation request bodies.
	MaxFiscalRequestBodyBytes = 256 * 1024
	// MaxFiscalizationSummaryIDs bounds batch summary queries.
	MaxFiscalizationSummaryIDs = 100
)

// FiscalHandler exposes additive invoice fiscalization HTTP APIs.
type FiscalHandler struct {
	svc ports.FiscalizationService
}

// NewFiscalHandler builds the fiscal document HTTP handler.
func NewFiscalHandler(svc ports.FiscalizationService) *FiscalHandler {
	return &FiscalHandler{svc: svc}
}

// RegisterInvoiceAndFiscalRoutes wires legacy invoice routes plus additive fiscal nested routes.
// Static /fiscalization-summaries MUST be registered before /:id.
func RegisterInvoiceAndFiscalRoutes(api *gin.RouterGroup, invoiceH *InvoiceHandler, fiscalH *FiscalHandler) {
	invoices := api.Group("/invoices")
	{
		invoices.GET("/me", invoiceH.ListMyInvoices)

		staffInvoices := invoices.Group("")
		staffInvoices.Use(middleware.RequireAccountingAccess())
		{
			staffInvoices.POST("", invoiceH.CreateIssuedInvoice)
			staffInvoices.GET("", invoiceH.ListIssuedInvoicesStaff)
			staffInvoices.DELETE("/:id", invoiceH.DeleteIssuedInvoice)
		}

		if fiscalH != nil && fiscalH.svc != nil {
			invoices.GET("/fiscalization-summaries", fiscalH.ListFiscalizationSummaries)
		}

		invoices.GET("/:id", invoiceH.GetIssuedInvoice)
		invoices.PATCH("/:id", invoiceH.PatchIssuedInvoice)

		if fiscalH != nil && fiscalH.svc != nil {
			invoices.GET("/:id/fiscalization", fiscalH.GetFiscalization)
			invoices.PUT("/:id/fiscalization/draft", fiscalH.PutFiscalizationDraft)
			invoices.DELETE("/:id/fiscalization/draft", fiscalH.DeleteFiscalizationDraft)
			invoices.POST("/:id/fiscalization/finalize", fiscalH.FinalizeFiscalization)
			invoices.POST("/:id/fiscalization/retry", fiscalH.RetryFiscalization)
			invoices.POST("/:id/fiscalization/reconcile", fiscalH.ReconcileFiscalization)
			invoices.POST("/:id/fiscalization/void", fiscalH.VoidFiscalization)
			invoices.GET("/:id/fiscal-artifacts/:artifactId", fiscalH.DownloadFiscalArtifact)
		}
	}
}

type fiscalDraftLineBody struct {
	Position    int           `json:"position"`
	Description string        `json:"description"`
	Quantity    DecimalString `json:"quantity"`
	UnitCode    string        `json:"unitCode"`
	UnitPrice   DecimalString `json:"unitPrice"`
	Discount    *struct {
		Kind  string        `json:"kind"`
		Value DecimalString `json:"value"`
	} `json:"discount"`
	TaxTreatmentCode string        `json:"taxTreatmentCode"`
	TaxRate          DecimalString `json:"taxRate"`
	ExemptionCode    *string       `json:"exemptionCode"`
	Source           *struct {
		Type string     `json:"type"`
		ID   *uuid.UUID `json:"id"`
	} `json:"source"`
}

type fiscalDraftBody struct {
	Version         int64                 `json:"version"`
	Kind            string                `json:"kind"`
	IntentSlot      string                `json:"intentSlot"`
	IssuerProfileID *uuid.UUID            `json:"issuerProfileId"`
	PolicyKey       string                `json:"policyKey"`
	Currency        string                `json:"currency"`
	Customer        json.RawMessage       `json:"customer"`
	BillingAddress  json.RawMessage       `json:"billingAddress"`
	Lines           []fiscalDraftLineBody `json:"lines"`
	DeclaredTotals  *struct {
		PayableTotal DecimalString `json:"payableTotal"`
	} `json:"declaredTotals"`
}

type fiscalActionBody struct {
	ExpectedVersion int64     `json:"expectedVersion"`
	ProviderKey     string    `json:"providerKey"`
	ConnectionID    uuid.UUID `json:"connectionId"`
	PolicyVersionID uuid.UUID `json:"policyVersionId"`
	IssuerProfileID uuid.UUID `json:"issuerProfileId"`
}

// GetFiscalization returns the provider-neutral projection for an invoice.
// @Summary     Get invoice fiscalization projection
// @Tags        fiscalization
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Invoice UUID"
// @Success     200 {object} FiscalizationProjectionResponse
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/invoices/{id}/fiscalization [get]
func (h *FiscalHandler) GetFiscalization(c *gin.Context) {
	uid, invoiceID, ok := fiscalActorAndInvoice(c)
	if !ok {
		return
	}
	proj, err := h.svc.GetProjection(c.Request.Context(), uid, invoiceID)
	if err != nil {
		writeFiscalizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, sanitizeFiscalizationProjection(proj))
}

// PutFiscalizationDraft creates or replaces the editable draft.
func (h *FiscalHandler) PutFiscalizationDraft(c *gin.Context) {
	uid, invoiceID, ok := fiscalActorAndInvoice(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxFiscalRequestBodyBytes)
	var body fiscalDraftBody
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		field := "body"
		msg := err.Error()
		if strings.Contains(msg, "decimal") || strings.Contains(msg, "Quantity") || strings.Contains(msg, "quantity") {
			field = "quantity"
		}
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		writeFiscalFieldError(c, field, msg)
		return
	}
	if len(body.Lines) == 0 {
		writeFiscalFieldError(c, "lines", "at least one line is required")
		return
	}
	if len(body.Lines) > MaxFiscalDraftLines {
		writeFiscalFieldError(c, "lines", fmt.Sprintf("maximum %d lines allowed", MaxFiscalDraftLines))
		return
	}
	cmd, err := draftBodyToCommand(body)
	if err != nil {
		writeFiscalFieldError(c, "body", err.Error())
		return
	}
	proj, created, err := h.svc.UpsertDraft(c.Request.Context(), uid, invoiceID, cmd)
	if err != nil {
		writeFiscalizationError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, sanitizeFiscalizationProjection(proj))
}

// DeleteFiscalizationDraft deletes the current mutable draft.
func (h *FiscalHandler) DeleteFiscalizationDraft(c *gin.Context) {
	uid, invoiceID, ok := fiscalActorAndInvoice(c)
	if !ok {
		return
	}
	version, err := strconv.ParseInt(strings.TrimSpace(c.Query("expectedVersion")), 10, 64)
	if err != nil {
		writeFiscalFieldError(c, "expectedVersion", "expectedVersion query is required")
		return
	}
	if err := h.svc.DeleteDraft(c.Request.Context(), uid, invoiceID, version); err != nil {
		writeFiscalizationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// FinalizeFiscalization freezes and queues issuance.
func (h *FiscalHandler) FinalizeFiscalization(c *gin.Context) {
	h.runAction(c, func(uid, invoiceID uuid.UUID, body fiscalActionBody) (*ports.FiscalizationProjection, error) {
		return h.svc.Finalize(c.Request.Context(), uid, invoiceID, ports.FiscalFinalizeCommand{
			ExpectedVersion: body.ExpectedVersion,
			ProviderKey:     body.ProviderKey,
			ConnectionID:    body.ConnectionID,
			PolicyVersionID: body.PolicyVersionID,
			IssuerProfileID: body.IssuerProfileID,
		})
	})
}

// RetryFiscalization queues an eligible retry.
func (h *FiscalHandler) RetryFiscalization(c *gin.Context) {
	h.runAction(c, func(uid, invoiceID uuid.UUID, body fiscalActionBody) (*ports.FiscalizationProjection, error) {
		return h.svc.Retry(c.Request.Context(), uid, invoiceID, ports.FiscalActionCommand{ExpectedVersion: body.ExpectedVersion})
	})
}

// ReconcileFiscalization queues same-provider reconciliation.
func (h *FiscalHandler) ReconcileFiscalization(c *gin.Context) {
	h.runAction(c, func(uid, invoiceID uuid.UUID, body fiscalActionBody) (*ports.FiscalizationProjection, error) {
		return h.svc.Reconcile(c.Request.Context(), uid, invoiceID, ports.FiscalActionCommand{ExpectedVersion: body.ExpectedVersion})
	})
}

// VoidFiscalization queues a permitted void.
func (h *FiscalHandler) VoidFiscalization(c *gin.Context) {
	h.runAction(c, func(uid, invoiceID uuid.UUID, body fiscalActionBody) (*ports.FiscalizationProjection, error) {
		return h.svc.Void(c.Request.Context(), uid, invoiceID, ports.FiscalActionCommand{ExpectedVersion: body.ExpectedVersion})
	})
}

func (h *FiscalHandler) runAction(c *gin.Context, fn func(uuid.UUID, uuid.UUID, fiscalActionBody) (*ports.FiscalizationProjection, error)) {
	uid, invoiceID, ok := fiscalActorAndInvoice(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxFiscalRequestBodyBytes)
	var body fiscalActionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	proj, err := fn(uid, invoiceID, body)
	if err != nil {
		writeFiscalizationError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, sanitizeFiscalizationProjection(proj))
}

// ListFiscalizationSummaries returns batch provider-neutral badges.
// @Summary     Batch fiscalization summaries
// @Tags        fiscalization
// @Produce     json
// @Security    BearerAuth
// @Param       invoiceIds query string true "CSV of invoice UUIDs (max 100)"
// @Success     200 {object} FiscalizationSummariesResponse
// @Failure     422 {object} FiscalValidationError
// @Router      /api/v1/invoices/fiscalization-summaries [get]
func (h *FiscalHandler) ListFiscalizationSummaries(c *gin.Context) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	raw := strings.TrimSpace(c.Query("invoiceIds"))
	if raw == "" {
		writeFiscalFieldError(c, "invoiceIds", "invoiceIds query is required")
		return
	}
	parts := strings.Split(raw, ",")
	if len(parts) > MaxFiscalizationSummaryIDs {
		writeFiscalFieldError(c, "invoiceIds", fmt.Sprintf("maximum %d ids allowed", MaxFiscalizationSummaryIDs))
		return
	}
	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			writeFiscalFieldError(c, "invoiceIds", "invalid invoice id")
			return
		}
		ids = append(ids, id)
	}
	items, err := h.svc.Summaries(c.Request.Context(), uid, ids)
	if err != nil {
		writeFiscalizationError(c, err)
		return
	}
	safe := make([]ports.FiscalizationSummary, 0, len(items))
	for i := range items {
		safe = append(safe, sanitizeFiscalizationSummary(items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": safe})
}

// DownloadFiscalArtifact streams an archived PDF with safe headers.
func (h *FiscalHandler) DownloadFiscalArtifact(c *gin.Context) {
	uid, invoiceID, ok := fiscalActorAndInvoice(c)
	if !ok {
		return
	}
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact id"})
		return
	}
	stream, err := h.svc.OpenArtifact(c.Request.Context(), uid, invoiceID, artifactID, ports.FiscalArtifactAccessMeta{
		CorrelationID: c.GetHeader("X-Request-ID"),
		IPHash:        c.ClientIP(),
	})
	if err != nil {
		writeFiscalizationError(c, err)
		return
	}
	defer stream.Body.Close()
	for k, v := range stream.Headers {
		if strings.EqualFold(k, "Location") || strings.Contains(strings.ToLower(v), "://") {
			continue
		}
		c.Header(k, v)
	}
	if stream.MediaType != "" {
		c.Header("Content-Type", stream.MediaType)
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, stream.Body)
}

func fiscalActorAndInvoice(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false
	}
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return uuid.Nil, uuid.Nil, false
	}
	return uid, invoiceID, true
}

func draftBodyToCommand(body fiscalDraftBody) (ports.FiscalDraftCommand, error) {
	lines := make([]ports.FiscalDraftLineCommand, 0, len(body.Lines))
	for _, line := range body.Lines {
		if strings.TrimSpace(string(line.Quantity)) == "" || strings.TrimSpace(string(line.UnitPrice)) == "" {
			return ports.FiscalDraftCommand{}, fmt.Errorf("line decimals are required")
		}
		cmd := ports.FiscalDraftLineCommand{
			Position: line.Position, Description: line.Description,
			Quantity: string(line.Quantity), UnitCode: line.UnitCode, UnitPrice: string(line.UnitPrice),
			TaxTreatmentCode: line.TaxTreatmentCode, TaxRate: string(line.TaxRate),
		}
		if line.Discount != nil {
			cmd.DiscountKind = line.Discount.Kind
			cmd.DiscountValue = string(line.Discount.Value)
		} else {
			cmd.DiscountKind = "none"
			cmd.DiscountValue = "0"
		}
		if line.ExemptionCode != nil {
			cmd.ExemptionCode = *line.ExemptionCode
		}
		if line.Source != nil {
			cmd.SourceType = line.Source.Type
			cmd.SourceID = line.Source.ID
		}
		lines = append(lines, cmd)
	}
	declared := ""
	if body.DeclaredTotals != nil {
		declared = string(body.DeclaredTotals.PayableTotal)
	}
	return ports.FiscalDraftCommand{
		ExpectedVersion: body.Version, Kind: body.Kind, IntentSlot: body.IntentSlot,
		IssuerProfileID: body.IssuerProfileID, PolicyKey: body.PolicyKey, Currency: body.Currency,
		Customer: ports.JSONObject(body.Customer), BillingAddress: ports.JSONObject(body.BillingAddress),
		Lines: lines, DeclaredPayable: declared,
	}, nil
}
