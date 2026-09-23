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
	// ErrFiscalOutboxEmpty reports that no claimable outbox event is available.
	ErrFiscalOutboxEmpty = errors.New("fiscal outbox empty")
	// ErrFiscalLeaseLost reports that the worker no longer owns the lease.
	ErrFiscalLeaseLost = errors.New("fiscal outbox lease lost")
	// ErrFiscalAttemptNotStarted reports an unexpected attempt lifecycle state.
	ErrFiscalAttemptNotStarted = errors.New("fiscal attempt is not started")
)

// FiscalOutboxEventType is a durable worker dispatch kind.
type FiscalOutboxEventType string

const (
	FiscalOutboxIssue           FiscalOutboxEventType = "issue"
	FiscalOutboxReconcileIssue  FiscalOutboxEventType = "reconcile_issue"
	FiscalOutboxVoid            FiscalOutboxEventType = "void"
	FiscalOutboxReconcileVoid   FiscalOutboxEventType = "reconcile_void"
	FiscalOutboxRecoverArtifact FiscalOutboxEventType = "recover_artifact"
)

// FiscalOutboxStatus is the claim lifecycle of an outbox event.
type FiscalOutboxStatus string

const (
	FiscalOutboxReady     FiscalOutboxStatus = "ready"
	FiscalOutboxLeased    FiscalOutboxStatus = "leased"
	FiscalOutboxCompleted FiscalOutboxStatus = "completed"
	FiscalOutboxDead      FiscalOutboxStatus = "dead"
)

// FiscalAttemptStatus is the attempt row lifecycle.
type FiscalAttemptStatus string

const (
	FiscalAttemptStarted   FiscalAttemptStatus = "started"
	FiscalAttemptSucceeded FiscalAttemptStatus = "succeeded"
	FiscalAttemptFailed    FiscalAttemptStatus = "failed"
	FiscalAttemptUnknown   FiscalAttemptStatus = "unknown"
)

// FiscalOutboxEvent is one durable dispatch unit.
type FiscalOutboxEvent struct {
	ID               uuid.UUID
	FiscalDocumentID uuid.UUID
	EventType        FiscalOutboxEventType
	OperationKey     string
	SequenceNo       int
	Status           FiscalOutboxStatus
	AvailableAt      time.Time
	LeaseOwner       string
	LeaseToken       uuid.UUID
	LeaseExpiresAt   *time.Time
	ClaimCount       int
	LastSafeError    string
	ArtifactID       *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FiscalProviderAttempt is a durable redacted provider attempt.
type FiscalProviderAttempt struct {
	ID                uuid.UUID
	FiscalDocumentID  uuid.UUID
	OutboxEventID     uuid.UUID
	ProviderKey       string
	Operation         FiscalOutboxEventType
	OperationKey      string
	AttemptNo         int
	Status            FiscalAttemptStatus
	Classification    FiscalErrorClass
	Definitive        bool
	RequestSHA256     string
	SafeProviderCode  string
	SafeProviderMsg   string
	Diagnostics       json.RawMessage
	ProviderReference string
	StartedAt         time.Time
	FinishedAt        *time.Time
}

// ClaimedOutboxEvent is a leased event ready for worker processing.
type ClaimedOutboxEvent struct {
	Event             FiscalOutboxEvent
	Document          domain.FiscalDocument
	CanonicalBytes    []byte
	HasStartedAttempt bool
	StartedAttemptID  uuid.UUID
}

// BeginLegalCallCommand records dispatching + started attempt before a provider call.
type BeginLegalCallCommand struct {
	EventID           uuid.UUID
	DocumentID        uuid.UUID
	LeaseOwner        string
	LeaseToken        uuid.UUID
	ProviderKey       string
	Operation         FiscalOutboxEventType
	OperationKey      string
	RequestSHA256     string
	NextDocumentState domain.FiscalDocumentState
}

// RecoverStaleAttemptCommand marks an unfinished started attempt as unknown.
type RecoverStaleAttemptCommand struct {
	EventID      uuid.UUID
	DocumentID   uuid.UUID
	AttemptID    uuid.UUID
	LeaseOwner   string
	LeaseToken   uuid.UUID
	UnknownState domain.FiscalDocumentState
	SafeError    string
}

// CompleteOutboxCallCommand applies the normalized provider outcome under lease fencing.
type CompleteOutboxCallCommand struct {
	EventID           uuid.UUID
	DocumentID        uuid.UUID
	AttemptID         uuid.UUID
	LeaseOwner        string
	LeaseToken        uuid.UUID
	AttemptStatus     FiscalAttemptStatus
	Classification    FiscalErrorClass
	Definitive        bool
	SafeProviderCode  string
	SafeProviderMsg   string
	Diagnostics       map[string]any
	ProviderReference string
	ProviderNumber    string
	NextDocumentState domain.FiscalDocumentState
	IssuedAt          *time.Time
	VoidedAt          *time.Time
	LastSafeError     string
	EnqueueRecovery   bool
	ArtifactID        *uuid.UUID
}

// FiscalOutboxRepository owns claim/lease/attempt/completion transactions.
type FiscalOutboxRepository interface {
	ClaimNext(ctx context.Context, owner string, leaseFor time.Duration) (*ClaimedOutboxEvent, error)
	BeginLegalCall(ctx context.Context, cmd BeginLegalCallCommand) (*FiscalProviderAttempt, error)
	RecoverStaleAttempt(ctx context.Context, cmd RecoverStaleAttemptCommand) error
	CompleteCall(ctx context.Context, cmd CompleteOutboxCallCommand) error
}
