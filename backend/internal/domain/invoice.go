package domain

import (
	"time"

	"github.com/google/uuid"
)

// FiscalEligibility gates whether an invoice may host a fiscal draft.
type FiscalEligibility string

const (
	// FiscalEligibilityLegacyUnfiscalized is the fail-safe default for historical invoices.
	FiscalEligibilityLegacyUnfiscalized FiscalEligibility = "legacy_unfiscalized"
	// FiscalEligibilityEligible allows staff to prepare fiscal drafts.
	FiscalEligibilityEligible FiscalEligibility = "eligible"
)

// Invoice represents a customer invoice (client may read/update own rows — see invoice service).
type Invoice struct {
	ID                uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey"`
	CustomerID        uuid.UUID         `json:"customerId" gorm:"type:uuid;column:customer_id;not null;index"`
	Amount            float64           `json:"amount" gorm:"not null"`
	Status            string            `json:"status" gorm:"type:varchar(40);not null;default:'open'"`
	Notes             string            `json:"notes" gorm:"type:text"`
	FiscalEligibility FiscalEligibility `json:"-" gorm:"column:fiscal_eligibility;type:varchar(32);not null;default:legacy_unfiscalized"`
	CreatedAt         time.Time         `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time         `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (Invoice) TableName() string {
	return "invoices"
}
