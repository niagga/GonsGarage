-- Migration: Ensure appointments table has correct structure
-- File: backend/migrations/004_fix_appointments_table.up.sql

BEGIN;

-- FIX: Ensure the UUID generation function is available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; 

-- Create appointments table if it doesn't exist
CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    car_id UUID NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    service_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    scheduled_date TIMESTAMP NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_appointments_client_id ON appointments(client_id);
CREATE INDEX IF NOT EXISTS idx_appointments_car_id ON appointments(car_id);
CREATE INDEX IF NOT EXISTS idx_appointments_scheduled_date ON appointments(scheduled_date);
CREATE INDEX IF NOT EXISTS idx_appointments_status ON appointments(status);

-- Add constraints
ALTER TABLE appointments 
ADD CONSTRAINT check_appointment_status 
CHECK (status IN ('scheduled', 'confirmed', 'in-progress', 'completed', 'cancelled'));

COMMIT;