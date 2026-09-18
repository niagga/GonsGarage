package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

const sqlSelectSupplierBase = `SELECT id, name, contact_email, contact_phone, tax_id, notes, is_active, created_at, updated_at, deleted_at
FROM suppliers WHERE deleted_at IS NULL`

type postgresSupplierRepository struct {
	db   *gorm.DB
	sqlx *sqlx.DB
}

// NewPostgresSupplierRepository returns a ports.SupplierRepository backed by PostgreSQL / GORM.
func NewPostgresSupplierRepository(db *gorm.DB) ports.SupplierRepository {
	return &postgresSupplierRepository{db: db, sqlx: sqlxFromGORM(db)}
}

type SupplierModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id"`
	Name          string     `gorm:"not null" db:"name"`
	ContactEmail  string     `gorm:"column:contact_email" db:"contact_email"`
	ContactPhone  string     `gorm:"column:contact_phone" db:"contact_phone"`
	TaxID         string     `gorm:"column:tax_id" db:"tax_id"`
	Notes         string     `gorm:"column:notes" db:"notes"`
	IsActive      bool       `gorm:"column:is_active" db:"is_active"`
	CreatedAt     time.Time  `gorm:"column:created_at" db:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" db:"updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" db:"deleted_at"`
}

func (SupplierModel) TableName() string { return "suppliers" }

func (r *postgresSupplierRepository) Create(ctx context.Context, s *domain.Supplier) error {
	if s == nil {
		return fmt.Errorf("supplier is nil")
	}
	if r.sqlx != nil {
		return r.createSupplierSQLX(ctx, s)
	}
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("failed to create supplier: %w", err)
	}
	return nil
}

func (r *postgresSupplierRepository) createSupplierSQLX(ctx context.Context, s *domain.Supplier) error {
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	const q = `INSERT INTO suppliers (id, name, contact_email, contact_phone, tax_id, notes, is_active, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.sqlx.ExecContext(ctx, q,
		s.ID, s.Name, s.ContactEmail, s.ContactPhone, s.TaxID, s.Notes, s.IsActive, s.CreatedAt.UTC(), s.UpdatedAt.UTC(), nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create supplier: %w", err)
	}
	return nil
}

func (r *postgresSupplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supplier, error) {
	if r.sqlx != nil {
		var row SupplierModel
		err := r.sqlx.GetContext(ctx, &row, sqlSelectSupplierBase+` AND id = $1`, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrSupplierNotFound
			}
			return nil, fmt.Errorf("failed to get supplier: %w", err)
		}
		return r.toDomainSupplier(&row), nil
	}
	var m SupplierModel
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSupplierNotFound
		}
		return nil, fmt.Errorf("failed to get supplier: %w", err)
	}
	return r.toDomainSupplier(&m), nil
}

func (r *postgresSupplierRepository) toDomainSupplier(m *SupplierModel) *domain.Supplier {
	if m == nil {
		return nil
	}
	return &domain.Supplier{
		ID:           m.ID,
		Name:         m.Name,
		ContactEmail: m.ContactEmail,
		ContactPhone: m.ContactPhone,
		TaxID:        m.TaxID,
		Notes:        m.Notes,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}
}

func (r *postgresSupplierRepository) Update(ctx context.Context, s *domain.Supplier) error {
	if s == nil {
		return fmt.Errorf("supplier is nil")
	}
	if r.sqlx != nil {
		return r.updateSupplierSQLX(ctx, s)
	}
	res := r.db.WithContext(ctx).Model(&domain.Supplier{}).
		Where("id = ? AND deleted_at IS NULL", s.ID).
		Updates(map[string]interface{}{
			"name":          s.Name,
			"contact_email": s.ContactEmail,
			"contact_phone": s.ContactPhone,
			"tax_id":        s.TaxID,
			"notes":         s.Notes,
			"is_active":     s.IsActive,
			"updated_at":    time.Now().UTC(),
		})
	if res.Error != nil {
		return fmt.Errorf("failed to update supplier: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrSupplierNotFound
	}
	return nil
}

func (r *postgresSupplierRepository) updateSupplierSQLX(ctx context.Context, s *domain.Supplier) error {
	now := time.Now().UTC()
	const q = `UPDATE suppliers SET name = $1, contact_email = $2, contact_phone = $3, tax_id = $4, notes = $5, is_active = $6, updated_at = $7
WHERE id = $8 AND deleted_at IS NULL`
	res, err := r.sqlx.ExecContext(ctx, q, s.Name, s.ContactEmail, s.ContactPhone, s.TaxID, s.Notes, s.IsActive, now, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update supplier: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrSupplierNotFound
	}
	return nil
}

func (r *postgresSupplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	now := time.Now().UTC()
	if r.sqlx != nil {
		return r.deleteSupplierSQLX(queryCtx, id, now)
	}
	err := r.db.WithContext(queryCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.ReceivedInvoice{}).
			Where("supplier_id = ?", id).
			Updates(map[string]interface{}{
				"supplier_id": nil,
				"updated_at":  now,
			}).Error; err != nil {
			return fmt.Errorf("failed to delete supplier: %w", err)
		}
		res := tx.Model(&SupplierModel{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("deleted_at", now)
		if res.Error != nil {
			return fmt.Errorf("failed to delete supplier: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return domain.ErrSupplierNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *postgresSupplierRepository) deleteSupplierSQLX(ctx context.Context, id uuid.UUID, now time.Time) error {
	tx, err := r.sqlx.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete supplier: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	const nullInvoices = `UPDATE received_invoices SET supplier_id = NULL, updated_at = $1 WHERE supplier_id = $2`
	if _, err := tx.ExecContext(ctx, nullInvoices, now, id); err != nil {
		return fmt.Errorf("failed to delete supplier: %w", err)
	}
	const q = `UPDATE suppliers SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	res, err := tx.ExecContext(ctx, q, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete supplier: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrSupplierNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to delete supplier: %w", err)
	}
	return nil
}

func clampRepoList(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (r *postgresSupplierRepository) List(ctx context.Context, limit, offset int) ([]*domain.Supplier, int64, error) {
	limit, offset = clampRepoList(limit, offset)
	if r.sqlx != nil {
		var total int64
		if err := r.sqlx.GetContext(ctx, &total, `SELECT COUNT(*) FROM suppliers WHERE deleted_at IS NULL`); err != nil {
			return nil, 0, fmt.Errorf("failed to count suppliers: %w", err)
		}
		var rows []SupplierModel
		q := sqlSelectSupplierBase + ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		if err := r.sqlx.SelectContext(ctx, &rows, q, limit, offset); err != nil {
			return nil, 0, fmt.Errorf("failed to list suppliers: %w", err)
		}
		out := make([]*domain.Supplier, 0, len(rows))
		for i := range rows {
			out = append(out, r.toDomainSupplier(&rows[i]))
		}
		return out, total, nil
	}
	var total int64
	if err := r.db.WithContext(ctx).Model(&SupplierModel{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count suppliers: %w", err)
	}
	var rows []SupplierModel
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Model(&SupplierModel{}).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list suppliers: %w", err)
	}
	out := make([]*domain.Supplier, 0, len(rows))
	for i := range rows {
		out = append(out, r.toDomainSupplier(&rows[i]))
	}
	return out, total, nil
}

var _ ports.SupplierRepository = (*postgresSupplierRepository)(nil)
