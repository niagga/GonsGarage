-- backend/migrations/002_create_employees_table.sql
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_code VARCHAR(50) UNIQUE NOT NULL,
    position VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    hire_date DATE NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    hours_per_week INTEGER DEFAULT 40,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_employees_user_id ON employees(user_id);
CREATE INDEX idx_employees_department ON employees(department);
CREATE INDEX idx_employees_is_active ON employees(is_active);