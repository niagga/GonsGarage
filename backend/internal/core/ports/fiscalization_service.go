package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// FiscalizationService is the invoice-scoped fiscal HTTP application port.
type FiscalizationService interface {
	GetProjection(ctx context.Context, actorID, invoiceID uuid.UUID) (*FiscalizationProjection, error)
	UpsertDraft(ctx context.Context, actorID, invoiceID uuid.UUID, draft FiscalDraftCommand) (*FiscalizationProjection, bool, error)
	DeleteDraft(ctx context.Context, actorID, invoiceID uuid.UUID, expectedVersion int64) error
	Finalize(ctx context.Context, actorID, invoiceID uuid.UUID, cmd FiscalFinalizeCommand) (*FiscalizationProjection, error)
	Retry(ctx context.Context, actorID, invoiceID uuid.UUID, cmd FiscalActionCommand) (*FiscalizationProjection, error)
	Reconcile(ctx context.Context, actorID, invoiceID uuid.UUID, cmd FiscalActionCommand) (*FiscalizationProjection, error)
	Void(ctx context.Context, actorID, invoiceID uuid.UUID, cmd FiscalActionCommand) (*FiscalizationProjection, error)
	Summaries(ctx context.Context, actorID uuid.UUID, invoiceIDs []uuid.UUID) ([]FiscalizationSummary, error)
	OpenArtifact(ctx context.Context, actorID, invoiceID, artifactID uuid.UUID, meta FiscalArtifactAccessMeta) (*FiscalArtifactStream, error)
}

// FiscalizationProjection is the provider-neutral HTTP-safe document view.
type FiscalizationProjection struct {
	InvoiceID      string   `json:"invoiceId"`
	DocumentID     *string  `json:"documentId,omitempty"`
	Kind           string   `json:"kind,omitempty"`
	Lifecycle      string   `json:"lifecycle,omitempty"`
	Status         string   `json:"status"`
	Version        int64    `json:"version,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	PayableTotal   string   `json:"payableTotal,omitempty"`
	GrossTotal     string   `json:"grossTotal,omitempty"`
	TaxTotal       string   `json:"taxTotal,omitempty"`
	AllowedActions []string `json:"allowedActions"`
	ArtifactStatus string   `json:"artifactStatus,omitempty"`
	LastErrorCode  string   `json:"lastErrorCode,omitempty"`
	LastErrorSafe  string   `json:"lastErrorMessage,omitempty"`
	Readiness      []string `json:"readinessIssues,omitempty"`
}

// FiscalizationSummary is a compact list-page badge projection.
type FiscalizationSummary struct {
	InvoiceID      string   `json:"invoiceId"`
	Status         string   `json:"status"`
	Lifecycle      string   `json:"lifecycle,omitempty"`
	AllowedActions []string `json:"allowedActions,omitempty"`
}

// FiscalDraftCommand is a validated draft upsert payload (decimals as strings).
type FiscalDraftCommand struct {
	ExpectedVersion int64
	Kind            string
	IntentSlot      string
	IssuerProfileID *uuid.UUID
	PolicyKey       string
	Currency        string
	Customer        JSONObject
	BillingAddress  JSONObject
	Lines           []FiscalDraftLineCommand
	DeclaredPayable string
}

// FiscalDraftLineCommand is one draft line with string decimals.
type FiscalDraftLineCommand struct {
	Position         int
	Description      string
	Quantity         string
	UnitCode         string
	UnitPrice        string
	DiscountKind     string
	DiscountValue    string
	TaxTreatmentCode string
	TaxRate          string
	ExemptionCode    string
	SourceType       string
	SourceID         *uuid.UUID
}

// FiscalFinalizeCommand freezes the current draft for an invoice.
type FiscalFinalizeCommand struct {
	ExpectedVersion int64
	ProviderKey     string
	ConnectionID    uuid.UUID
	PolicyVersionID uuid.UUID
	IssuerProfileID uuid.UUID
}

// FiscalActionCommand is shared by retry/reconcile/void.
type FiscalActionCommand struct {
	ExpectedVersion int64
}

// FiscalArtifactAccessMeta carries download audit fields.
type FiscalArtifactAccessMeta struct {
	CorrelationID string
	IPHash        string
	ActorRole     string
}

// FiscalArtifactStream is a safe PDF download (no storage keys/URLs).
type FiscalArtifactStream struct {
	Body      io.ReadCloser
	MediaType string
	ByteSize  int64
	Filename  string
	Headers   map[string]string
}

// JSONObject holds opaque JSON object bytes without exposing provider internals.
type JSONObject []byte
