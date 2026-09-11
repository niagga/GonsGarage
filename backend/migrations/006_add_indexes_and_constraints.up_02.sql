-- Rollback: Remove indexes and constraints
-- File: backend/migrations/005_add_indexes_and_constraints.down.sql
BEGIN;

-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_cars_updated_at ON cars;
-- Removido: appointments não existe
-- DROP TRIGGER IF EXISTS update_appointments_updated_at ON appointments;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_cars_license_plate;
DROP INDEX IF EXISTS idx_cars_make_model;

COMMIT;