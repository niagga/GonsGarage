package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureInvoicesRepairLink adds optional repair_id/car_id on invoices for internal billing.
// Safe for DBs that predate migration 012; idempotent.
func EnsureInvoicesRepairLink(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("ensure invoices repair link: database is nil")
	}
	stmts := []string{
		`ALTER TABLE invoices ADD COLUMN IF NOT EXISTS repair_id uuid`,
		`ALTER TABLE invoices ADD COLUMN IF NOT EXISTS car_id uuid`,
		`CREATE UNIQUE INDEX IF NOT EXISTS invoices_repair_id_uq ON invoices (repair_id) WHERE repair_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS invoices_car_id_idx ON invoices (car_id)`,
	}
	for _, q := range stmts {
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("invoices repair link schema: %w", err)
		}
	}
	return nil
}
