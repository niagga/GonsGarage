package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

var (
	// ErrFiscalDocumentNotFound reports a missing fiscal document.
	ErrFiscalDocumentNotFound = errors.New("fiscal document not found")
	// ErrFiscalVersionConflict reports an optimistic version mismatch.
	ErrFiscalVersionConflict = errors.New("fiscal document version conflict")
	// ErrFiscalIntentConflict reports more than one current intent for a slot.
	ErrFiscalIntentConflict = errors.New("fiscal current intent conflict")
	// ErrFiscalInvoiceNotEligible reports draft creation against a non-eligible invoice.
	ErrFiscalInvoiceNotEligible = errors.New("invoice is not eligible for fiscal drafts")
	// ErrFiscalHistoryProtected reports invoice deletion blocked by frozen fiscal history.
	ErrFiscalHistoryProtected = errors.New("invoice has protected fiscal history")
	// ErrFiscalActionNotAllowed reports a guarded lifecycle/action denial.
	ErrFiscalActionNotAllowed = errors.New("fiscal action is not allowed")
	// ErrFiscalNotReady reports missing readiness for finalization.
	ErrFiscalNotReady = errors.New("fiscal document is not ready")
)

// FiscalProtectionReader is the narrow invoice-delete seam.
type FiscalProtectionReader interface {
	HasProtectedFiscalHistory(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}

// FiscalLineRecord is a persisted draft/frozen line.
type FiscalLineRecord struct {
	ID               uuid.UUID
	Position         int
	Description      string
	UnitCode         string
	Quantity         string
	UnitPrice        string
	GrossAmount      string
	DiscountKind     string
	DiscountValue    string
	DiscountAmount   string
	NetAmount        string
	TaxRate          string
	TaxAmount        string
	LineTotal        string
	TaxTreatmentCode string
	ExemptionCode    string
	ExemptionReason  string
	SourceType       string
	SourceID         *uuid.UUID
}

// FiscalSnapshotRecord is the snapshot attached to a fiscal document.
type FiscalSnapshotRecord struct {
	ID              uuid.UUID
	PolicyVersionID *uuid.UUID
	IssuerProfileID *uuid.UUID
	Issuer          json.RawMessage
	Customer        json.RawMessage
	BillingAddress  json.RawMessage
	Currency        string
	GrossTotal      string
	DiscountTotal   string
	NetTotal        string
	TaxTotal        string
	RoundingAdj     string
	PayableTotal    string
	CanonicalBytes  []byte
	CanonicalSHA256 string
	FrozenAt        *time.Time
	Lines           []FiscalLineRecord
}

// FiscalDocumentAggregate is the document plus editable/frozen snapshot.
type FiscalDocumentAggregate struct {
	Document domain.FiscalDocument
	Snapshot FiscalSnapshotRecord
}

// CreateDraftInput seeds a mutable draft aggregate.
type CreateDraftInput struct {
	ID              uuid.UUID
	SourceInvoiceID uuid.UUID
	IntentSlot      string
	Kind            domain.DocumentKind
	CreatedBy       uuid.UUID
	Snapshot        FiscalSnapshotRecord
}

// FinalizeDraftCommand freezes a draft and enqueues the initial issue outbox event.
type FinalizeDraftCommand struct {
	DocumentID      uuid.UUID
	ExpectedVersion int64
	ActorID         uuid.UUID
	ProviderKey     string
	ConnectionID    uuid.UUID
	IntentKey       uuid.UUID
	IssueOpKey      string
	FrozenAt        time.Time
	PolicyVersionID uuid.UUID
	IssuerProfileID uuid.UUID
	Issuer          json.RawMessage
	Customer        json.RawMessage
	BillingAddress  json.RawMessage
	Currency        string
	GrossTotal      string
	DiscountTotal   string
	NetTotal        string
	TaxTotal        string
	RoundingAdj     string
	PayableTotal    string
	CanonicalBytes  []byte
	CanonicalSHA256 string
	Lines           []FiscalLineRecord
}

// FinalizeDraftResult is the frozen projection after a successful finalize.
type FinalizeDraftResult struct {
	Document domain.FrozenFiscalDocument
	OutboxID uuid.UUID
	Sequence int
}

// EnqueueActionCommand inserts the next outbox sequence for retry/reconcile/void.
type EnqueueActionCommand struct {
	DocumentID       uuid.UUID
	ExpectedVersion  int64
	ActorID          uuid.UUID
	EventType        string
	OperationKey     string
	NextState        domain.FiscalDocumentState
	VoidOperationKey string
}

// FiscalRepository persists fiscal aggregates with atomic finalization semantics.
type FiscalRepository interface {
	FiscalProtectionReader

	CreateDraft(ctx context.Context, input CreateDraftInput) (*FiscalDocumentAggregate, error)
	UpdateDraft(ctx context.Context, documentID uuid.UUID, expectedVersion int64, snapshot FiscalSnapshotRecord) (*FiscalDocumentAggregate, error)
	GetAggregate(ctx context.Context, documentID uuid.UUID) (*FiscalDocumentAggregate, error)
	GetCurrentIntent(ctx context.Context, invoiceID uuid.UUID, intentSlot string) (*FiscalDocumentAggregate, error)
	DeleteDraft(ctx context.Context, documentID uuid.UUID, expectedVersion int64) error

	FinalizeDraft(ctx context.Context, cmd FinalizeDraftCommand) (*FinalizeDraftResult, error)
	EnqueueAction(ctx context.Context, cmd EnqueueActionCommand) (*domain.FiscalDocument, error)
	GetNextPending(ctx context.Context) (*domain.FiscalDocument, error)
}
