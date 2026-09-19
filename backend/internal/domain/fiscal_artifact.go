package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrFiscalArtifactStateInvalid is returned when an artifact status is unknown.
	ErrFiscalArtifactStateInvalid = errors.New("invalid fiscal artifact state")
	// ErrFiscalArtifactClassificationInvalid is returned when the artifact classification is unknown.
	ErrFiscalArtifactClassificationInvalid = errors.New("invalid fiscal artifact classification")
)

// FiscalArtifactKind identifies the persisted artifact kind.
type FiscalArtifactKind string

const (
	FiscalArtifactKindProviderPDF FiscalArtifactKind = "provider_pdf"
)

// IsValid reports whether the kind is recognized.
func (k FiscalArtifactKind) IsValid() bool {
	return k == FiscalArtifactKindProviderPDF
}

// FiscalArtifactStatus identifies the artifact availability lifecycle.
type FiscalArtifactStatus string

const (
	FiscalArtifactStatusPending     FiscalArtifactStatus = "pending"
	FiscalArtifactStatusAvailable   FiscalArtifactStatus = "available"
	FiscalArtifactStatusUnavailable FiscalArtifactStatus = "unavailable"
	FiscalArtifactStatusCompromised FiscalArtifactStatus = "compromised"
)

// IsValid reports whether the status is recognized.
func (s FiscalArtifactStatus) IsValid() bool {
	switch s {
	case FiscalArtifactStatusPending, FiscalArtifactStatusAvailable, FiscalArtifactStatusUnavailable, FiscalArtifactStatusCompromised:
		return true
	default:
		return false
	}
}

// IsAvailable reports whether the artifact may be served.
func (s FiscalArtifactStatus) IsAvailable() bool {
	return s == FiscalArtifactStatusAvailable
}

// FiscalArtifactClassification distinguishes legal from mock evidence.
type FiscalArtifactClassification string

const (
	FiscalArtifactClassificationLegal FiscalArtifactClassification = "legal"
	FiscalArtifactClassificationMock  FiscalArtifactClassification = "mock"
)

// IsValid reports whether the classification is recognized.
func (c FiscalArtifactClassification) IsValid() bool {
	switch c {
	case FiscalArtifactClassificationLegal, FiscalArtifactClassificationMock:
		return true
	default:
		return false
	}
}

// IsProductionLegal reports whether the artifact may be exposed in a legal production flow.
func (c FiscalArtifactClassification) IsProductionLegal() bool {
	return c == FiscalArtifactClassificationLegal
}

// FiscalArtifactAction identifies the artifact access matrix.
type FiscalArtifactAction string

const (
	FiscalArtifactActionView FiscalArtifactAction = "view"
)

// IsValid reports whether the action is recognized.
func (a FiscalArtifactAction) IsValid() bool {
	return a == FiscalArtifactActionView
}

// FiscalArtifact is the private PDF evidence aggregate.
type FiscalArtifact struct {
	ID                uuid.UUID                    `json:"id" gorm:"type:uuid;primaryKey"`
	FiscalDocumentID  uuid.UUID                    `json:"fiscalDocumentId" gorm:"type:uuid;not null;index"`
	SourceInvoiceID   uuid.UUID                    `json:"sourceInvoiceId" gorm:"type:uuid;not null;index"`
	Kind              FiscalArtifactKind           `json:"kind" gorm:"type:varchar(24);not null;index"`
	Status            FiscalArtifactStatus         `json:"status" gorm:"type:varchar(16);not null;index"`
	Classification    FiscalArtifactClassification `json:"classification" gorm:"type:varchar(16);not null;index"`
	StorageKey        string                       `json:"storageKey,omitempty" gorm:"column:storage_key;type:text;uniqueIndex"`
	MediaType         string                       `json:"mediaType,omitempty" gorm:"column:media_type;type:varchar(100)"`
	ByteSize          int64                        `json:"byteSize,omitempty" gorm:"column:byte_size"`
	SHA256            string                       `json:"sha256,omitempty" gorm:"type:char(64)"`
	ProviderReference string                       `json:"providerReference,omitempty" gorm:"column:provider_reference;type:varchar(255)"`
	ProviderVersion   string                       `json:"providerVersion,omitempty" gorm:"column:provider_version;type:varchar(80)"`
	CreatedAt         time.Time                    `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	AvailableAt       *time.Time                   `json:"availableAt,omitempty" gorm:"column:available_at"`
	LastVerifiedAt    *time.Time                   `json:"lastVerifiedAt,omitempty" gorm:"column:last_verified_at"`
	LastErrorCode     string                       `json:"lastErrorCode,omitempty" gorm:"column:last_error_code;type:varchar(80)"`
	LastErrorMessage  string                       `json:"lastErrorMessage,omitempty" gorm:"column:last_error_message;type:text"`
}

func (FiscalArtifact) TableName() string {
	return "fiscal_artifacts"
}

// Validate checks the structural invariants used by the domain slice.
func (a *FiscalArtifact) Validate() error {
	if a == nil {
		return errors.New("fiscal artifact is nil")
	}
	if !a.Kind.IsValid() {
		return errors.New("invalid fiscal artifact kind")
	}
	if !a.Status.IsValid() {
		return ErrFiscalArtifactStateInvalid
	}
	if !a.Classification.IsValid() {
		return ErrFiscalArtifactClassificationInvalid
	}
	if a.Status.IsAvailable() {
		if strings.TrimSpace(a.StorageKey) == "" || strings.TrimSpace(a.MediaType) != "application/pdf" || a.ByteSize <= 0 || strings.TrimSpace(a.SHA256) == "" || a.AvailableAt == nil || a.AvailableAt.IsZero() {
			return fmt.Errorf("%w: available artifact is missing storage metadata", ErrFiscalArtifactStateInvalid)
		}
	}
	return nil
}

// AllowedActions returns the role/action matrix for artifact access.
func (a FiscalArtifact) AllowedActions(role FiscalActorRole, owns bool) []FiscalArtifactAction {
	if !a.Status.IsAvailable() {
		return nil
	}
	if role.IsStaff() {
		return []FiscalArtifactAction{FiscalArtifactActionView}
	}
	if role == FiscalActorRoleClient && owns {
		return []FiscalArtifactAction{FiscalArtifactActionView}
	}
	return nil
}

// CanAct reports whether the role may perform the requested action.
func (a FiscalArtifact) CanAct(role FiscalActorRole, owns bool, action FiscalArtifactAction) bool {
	for _, allowed := range a.AllowedActions(role, owns) {
		if allowed == action {
			return true
		}
	}
	return false
}

// CanServeInProduction reports whether the artifact may be exposed through a legal production flow.
func (a FiscalArtifact) CanServeInProduction() bool {
	return a.Status.IsAvailable() && a.Classification.IsProductionLegal()
}
