-- Migration: Add performance indexes and data integrity constraints
-- File: backend/migrations/005_add_indexes_and_constraints.up.sql
BEGIN;

-- Users table improvements
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Cars table improvements  
CREATE INDEX IF NOT EXISTS idx_cars_license_plate ON cars(license_plate);
CREATE INDEX IF NOT EXISTS idx_cars_make_model ON cars(make, model);

-- Add updated_at trigger function (only if it doesn't exist)
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc 
        WHERE proname = 'update_updated_at_column'
    ) THEN
        CREATE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $func$
        BEGIN
            NEW.updated_at = CURRENT_TIMESTAMP;
            RETURN NEW;
        END;
        $func$ LANGUAGE plpgsql;
    END IF;
END $$;

-- Add triggers for updated_at (only for existing tables)
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_cars_updated_at ON cars;
CREATE TRIGGER update_cars_updated_at
    BEFORE UPDATE ON cars
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Removido: appointments não existe ainda
-- DROP TRIGGER IF EXISTS update_appointments_updated_at ON appointments;
-- CREATE TRIGGER update_appointments_updated_at
--     BEFORE UPDATE ON appointments
--     FOR EACH ROW
--     EXECUTE FUNCTION update_updated_at_column();

COMMIT;