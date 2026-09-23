-- Internal invoice ↔ repair link (operational billing; not fiscal/Cloudware).
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS repair_id uuid;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS car_id uuid;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoices_repair_id_fkey'
  ) THEN
    ALTER TABLE invoices
      ADD CONSTRAINT invoices_repair_id_fkey
      FOREIGN KEY (repair_id) REFERENCES repairs(id) ON DELETE SET NULL;
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoices_car_id_fkey'
  ) THEN
    ALTER TABLE invoices
      ADD CONSTRAINT invoices_car_id_fkey
      FOREIGN KEY (car_id) REFERENCES cars(id) ON DELETE SET NULL;
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS invoices_repair_id_uq ON invoices (repair_id) WHERE repair_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS invoices_car_id_idx ON invoices (car_id);
