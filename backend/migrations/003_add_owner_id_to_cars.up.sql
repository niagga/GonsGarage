-- Migration: Add owner_id column to cars table
-- File: backend/migrations/003_add_owner_id_to_cars.up.sql

BEGIN;

-- 1. Add owner_id column as NULLABLE (temp) with Foreign Key constraint
ALTER TABLE cars 
ADD COLUMN owner_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- 2. Update existing cars to have a valid owner
-- This is crucial. If there are no users, this query will fail,
-- so ensure your setup migration runs first or a default user exists.
UPDATE cars 
SET owner_id = (SELECT id FROM users LIMIT 1)
WHERE owner_id IS NULL;

-- 3. Now that all existing rows have a value, apply the NOT NULL constraint
ALTER TABLE cars 
ALTER COLUMN owner_id SET NOT NULL;

-- 4. Create index for performance
CREATE INDEX idx_cars_owner_id ON cars(owner_id);

COMMIT;