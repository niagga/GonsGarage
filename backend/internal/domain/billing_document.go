package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BillingDocumentKind classifies documents the workshop issues (billing spec).
type BillingDocumentKind string

const (
	BillingDocumentKindPayroll BillingDocumentKind = "payroll"
	BillingDocumentKindIRS     BillingDocumentKind = "irs"
	BillingDocumentKindOther   BillingDocumentKind = "other"
)

// IsValid reports whether k is one of the supported kinds.
func (k BillingDocumentKind) IsValid() bool {
	switch k {
	case BillingDocumentKindPayroll, BillingDocumentKindIRS, BillingDocumentKindOther:
		return true
	default:
		return false
	}
}

// BillingDocument is an issued staff document (payroll, IRS, or other). Client-emitted invoices are domain.Invoice.
type BillingDocument struct {
	ID          uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey"`
	Kind        BillingDocumentKind `json:"kind" gorm:"type:varchar(32);not null;index"`
	Title       string              `json:"title" gorm:"type:varchar(200);not null"`
	Amount      float64             `json:"amount" gorm:"not null"`
	CustomerID  *uuid.UUID          `json:"customerId,omitempty" gorm:"type:uuid;index"` // optional (e.g. payroll)
	Reference   string              `json:"reference" gorm:"type:varchar(120)"`
	Notes       string              `json:"notes" gorm:"type:text"`
	CreatedAt   time.Time           `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time           `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time          `json:"deletedAt,omitempty" gorm:"column:deleted_at;index"`
}

func (BillingDocument) TableName() string {
	return "billing_documents"
}

// Validate checks minimal P1 fields.
func (b *BillingDocument) Validate() error {
	if b == nil {
		return errors.New("billing document is nil")
	}
	if !b.Kind.IsValid() {
		return errors.New("invalid billing document kind")
	}
	if strings.TrimSpace(b.Title) == "" {
		return errors.New("title is required")
	}
	if b.Amount < 0 {
		return errors.New("amount must be non-negative")
	}
	return nil
}
