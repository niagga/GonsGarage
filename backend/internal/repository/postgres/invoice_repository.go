package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

type postgresInvoiceRepository struct {
	db *gorm.DB
}

// NewPostgresInvoiceRepository returns ports.InvoiceRepository.
func NewPostgresInvoiceRepository(db *gorm.DB) ports.InvoiceRepository {
	return &postgresInvoiceRepository{db: db}
}

func (r *postgresInvoiceRepository) Create(ctx context.Context, invoice *domain.Invoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is nil")
	}
	if invoice.FiscalEligibility == "" {
		invoice.FiscalEligibility = domain.FiscalEligibilityEligible
	}
	if err := r.db.WithContext(ctx).Create(invoice).Error; err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}
	return nil
}

func (r *postgresInvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	return &inv, nil
}

func (r *postgresInvoiceRepository) GetByRepairID(ctx context.Context, repairID uuid.UUID) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).Where("repair_id = ?", repairID).First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice by repair: %w", err)
	}
	return &inv, nil
}

func (r *postgresInvoiceRepository) Update(ctx context.Context, invoice *domain.Invoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is nil")
	}
	res := r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("id = ?", invoice.ID).
		Updates(map[string]interface{}{
			"customer_id": invoice.CustomerID,
			"repair_id":   invoice.RepairID,
			"car_id":      invoice.CarID,
			"amount":      invoice.Amount,
			"status":      invoice.Status,
			"notes":       invoice.Notes,
		})
	if res.Error != nil {
		return fmt.Errorf("failed to update invoice: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrInvoiceNotFound
	}
	return nil
}

func (r *postgresInvoiceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Invoice{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete invoice: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrInvoiceNotFound
	}
	return nil
}

func (r *postgresInvoiceRepository) ListByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error) {
	limit, offset = clampRepoList(limit, offset)
	var total int64
	q := r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("customer_id = ?", customerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %w", err)
	}
	var rows []domain.Invoice
	if err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Model(&domain.Invoice{}).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}
	out := make([]*domain.Invoice, 0, len(rows))
	for i := range rows {
		row := rows[i]
		out = append(out, &row)
	}
	return out, total, nil
}

func (r *postgresInvoiceRepository) ListForStaff(ctx context.Context, limit, offset int) ([]*domain.Invoice, int64, error) {
	limit, offset = clampRepoList(limit, offset)
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.Invoice{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %w", err)
	}
	var rows []domain.Invoice
	if err := r.db.WithContext(ctx).Model(&domain.Invoice{}).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}
	out := make([]*domain.Invoice, 0, len(rows))
	for i := range rows {
		row := rows[i]
		out = append(out, &row)
	}
	return out, total, nil
}

var _ ports.InvoiceRepository = (*postgresInvoiceRepository)(nil)
