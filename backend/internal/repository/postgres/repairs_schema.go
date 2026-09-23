package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// requiredRepairsColumns are columns that sqlx SELECT paths assume exist.
// Kept in sync with sqlSelectRepairBase / RepairModel in repair_repository.go.
var requiredRepairsColumns = []string{
	"id",
	"car_id",
	"technician_id",
	"description",
	"status",
	"cost",
	"started_at",
	"completed_at",
	"created_at",
	"updated_at",
	"deleted_at",
}

// RequiredRepairsColumns returns the column contract expected by the repairs repository.
func RequiredRepairsColumns() []string {
	out := make([]string, len(requiredRepairsColumns))
	copy(out, requiredRepairsColumns)
	return out
}

// MissingRepairsColumns returns required column names absent from present.
func MissingRepairsColumns(present map[string]bool) []string {
	missing := make([]string, 0)
	for _, col := range requiredRepairsColumns {
		if !present[col] {
			missing = append(missing, col)
		}
	}
	return missing
}

// EnsureRepairsSchema aligns legacy repairs tables with the current domain/sqlx model.
// AutoMigrate alone is not enough: tables created from older SQL (start_date/end_date,
// employee_id) keep those names, sqlx SELECT fails with SQLSTATE 42703, and startup
// previously only patched technician_id / service_job_id.
func EnsureRepairsSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("ensure repairs schema: database is nil")
	}
	if err := ensureRepairsTechnicianIDColumn(db); err != nil {
		return err
	}
	if err := ensureRepairsServiceJobIDColumn(db); err != nil {
		return err
	}
	if err := ensureRepairsTimelineColumns(db); err != nil {
		return err
	}
	if err := ensureRepairsLegacyInsertBridge(db); err != nil {
		return err
	}
	return nil
}

func ensureRepairsTechnicianIDColumn(db *gorm.DB) error {
	const qAdd = `ALTER TABLE repairs ADD COLUMN IF NOT EXISTS technician_id uuid`
	const qFillFromEmployee = `
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'employee_id'
  ) THEN
    UPDATE repairs SET technician_id = employee_id
    WHERE technician_id IS NULL AND employee_id IS NOT NULL;
  END IF;
END $$`
	const qFill = `UPDATE repairs SET technician_id = '00000000-0000-0000-0000-000000000000'::uuid WHERE technician_id IS NULL`
	const qDefault = `ALTER TABLE repairs ALTER COLUMN technician_id SET DEFAULT '00000000-0000-0000-0000-000000000000'::uuid`
	const qNotNull = `ALTER TABLE repairs ALTER COLUMN technician_id SET NOT NULL`
	for _, q := range []string{qAdd, qFillFromEmployee, qFill, qDefault, qNotNull} {
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("%s: %w", q, err)
		}
	}
	return nil
}

func ensureRepairsServiceJobIDColumn(db *gorm.DB) error {
	const q = `ALTER TABLE repairs ADD COLUMN IF NOT EXISTS service_job_id uuid`
	if err := db.Exec(q).Error; err != nil {
		return fmt.Errorf("%s: %w", q, err)
	}
	return nil
}

// ensureRepairsTimelineColumns adds started_at/completed_at and backfills from
// legacy start_date/end_date when those columns still exist.
func ensureRepairsTimelineColumns(db *gorm.DB) error {
	stmts := []string{
		`ALTER TABLE repairs ADD COLUMN IF NOT EXISTS started_at timestamptz`,
		`ALTER TABLE repairs ADD COLUMN IF NOT EXISTS completed_at timestamptz`,
		`DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'start_date'
  ) THEN
    UPDATE repairs SET started_at = start_date WHERE started_at IS NULL AND start_date IS NOT NULL;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'end_date'
  ) THEN
    UPDATE repairs SET completed_at = end_date WHERE completed_at IS NULL AND end_date IS NOT NULL;
  END IF;
END $$`,
	}
	for _, q := range stmts {
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("repairs timeline schema: %w", err)
		}
	}
	return nil
}

// ensureRepairsLegacyInsertBridge relaxes legacy NOT NULL columns (employee_id,
// start_date) that modern sqlx INSERTs omit, and installs a BEFORE INSERT/UPDATE
// trigger to keep legacy columns populated from technician_id / started_at /
// completed_at when those legacy columns still exist.
func ensureRepairsLegacyInsertBridge(db *gorm.DB) error {
	const relax = `
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'employee_id'
  ) THEN
    ALTER TABLE repairs ALTER COLUMN employee_id DROP NOT NULL;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'start_date'
  ) THEN
    ALTER TABLE repairs ALTER COLUMN start_date DROP NOT NULL;
  END IF;
END $$`

	// Build a trigger whose body only mentions columns present on this DB.
	const installTrigger = `
DO $$
DECLARE
  has_employee boolean;
  has_start boolean;
  has_end boolean;
  body text;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'employee_id'
  ) INTO has_employee;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'start_date'
  ) INTO has_start;
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'repairs' AND column_name = 'end_date'
  ) INTO has_end;

  IF NOT (has_employee OR has_start OR has_end) THEN
    RETURN;
  END IF;

  body := 'BEGIN ';
  IF has_employee THEN
    body := body || 'IF NEW.employee_id IS NULL AND NEW.technician_id IS NOT NULL THEN NEW.employee_id := NEW.technician_id; END IF; ';
  END IF;
  IF has_start THEN
    body := body || 'IF NEW.start_date IS NULL AND NEW.started_at IS NOT NULL THEN NEW.start_date := NEW.started_at; END IF; ';
  END IF;
  IF has_end THEN
    body := body || 'IF NEW.end_date IS NULL AND NEW.completed_at IS NOT NULL THEN NEW.end_date := NEW.completed_at; END IF; ';
  END IF;
  body := body || 'RETURN NEW; END;';

  EXECUTE 'CREATE OR REPLACE FUNCTION repairs_bridge_legacy_columns() RETURNS trigger AS $fn$ ' || body || ' $fn$ LANGUAGE plpgsql';
  DROP TRIGGER IF EXISTS trg_repairs_bridge_legacy ON repairs;
  EXECUTE 'CREATE TRIGGER trg_repairs_bridge_legacy BEFORE INSERT OR UPDATE ON repairs FOR EACH ROW EXECUTE FUNCTION repairs_bridge_legacy_columns()';
END $$`

	for _, q := range []string{relax, installTrigger} {
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("repairs legacy insert bridge: %w", err)
		}
	}
	return nil
}
