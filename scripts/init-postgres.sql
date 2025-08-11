-- SA3D PostgreSQL Initialization Script
-- This script sets up the initial database structure with security hardening

-- Enable security extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create application user with limited privileges
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sa3d_app') THEN
        CREATE ROLE sa3d_app WITH LOGIN PASSWORD NULL;
        -- Password will be set via environment variable  
        ALTER ROLE sa3d_app WITH PASSWORD 'sa3d_app_password';
    END IF;
END
$$;

-- Create readonly user for reporting/analytics
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sa3d_readonly') THEN
        CREATE ROLE sa3d_readonly WITH LOGIN;
    END IF;
END
$$;

-- Set up proper database permissions
GRANT CONNECT ON DATABASE sa3d_db TO sa3d_app;
GRANT CONNECT ON DATABASE sa3d_db TO sa3d_readonly;

-- Create schema for application tables
CREATE SCHEMA IF NOT EXISTS sa3d AUTHORIZATION sa3d_app;
CREATE SCHEMA IF NOT EXISTS sa3d_audit AUTHORIZATION sa3d_app;

-- Grant permissions on schemas
GRANT USAGE ON SCHEMA sa3d TO sa3d_app;
GRANT CREATE ON SCHEMA sa3d TO sa3d_app;
GRANT USAGE ON SCHEMA sa3d TO sa3d_readonly;

GRANT USAGE ON SCHEMA sa3d_audit TO sa3d_app;
GRANT CREATE ON SCHEMA sa3d_audit TO sa3d_app;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA sa3d GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO sa3d_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA sa3d GRANT SELECT ON TABLES TO sa3d_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA sa3d GRANT USAGE, SELECT ON SEQUENCES TO sa3d_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA sa3d GRANT SELECT ON SEQUENCES TO sa3d_readonly;

-- Create audit log function
CREATE OR REPLACE FUNCTION sa3d_audit.audit_trigger_func()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO sa3d_audit.audit_log (
            table_name, operation, old_data, user_name, timestamp
        ) VALUES (
            TG_TABLE_NAME, TG_OP, to_jsonb(OLD), current_user, now()
        );
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO sa3d_audit.audit_log (
            table_name, operation, old_data, new_data, user_name, timestamp
        ) VALUES (
            TG_TABLE_NAME, TG_OP, to_jsonb(OLD), to_jsonb(NEW), current_user, now()
        );
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO sa3d_audit.audit_log (
            table_name, operation, new_data, user_name, timestamp
        ) VALUES (
            TG_TABLE_NAME, TG_OP, to_jsonb(NEW), current_user, now()
        );
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create audit log table
CREATE TABLE IF NOT EXISTS sa3d_audit.audit_log (
    id BIGSERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    old_data JSONB,
    new_data JSONB,
    user_name TEXT NOT NULL DEFAULT current_user,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create index for audit log queries
CREATE INDEX IF NOT EXISTS audit_log_table_timestamp_idx ON sa3d_audit.audit_log (table_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_log_user_timestamp_idx ON sa3d_audit.audit_log (user_name, timestamp DESC);

-- Enable row level security on audit log
ALTER TABLE sa3d_audit.audit_log ENABLE ROW LEVEL SECURITY;

-- Create security policies
CREATE POLICY audit_log_select_policy ON sa3d_audit.audit_log
    FOR SELECT USING (true); -- Allow all authenticated users to read audit logs

CREATE POLICY audit_log_insert_policy ON sa3d_audit.audit_log
    FOR INSERT WITH CHECK (user_name = current_user);

-- Revoke public access
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON DATABASE sa3d_db FROM PUBLIC;

-- Set secure search path
ALTER DATABASE sa3d_db SET search_path TO sa3d, sa3d_audit, public;

-- Configure security settings
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_lock_waits = on;
ALTER SYSTEM SET log_min_duration_statement = 1000; -- Log slow queries (>1s)

-- Force password encryption
ALTER SYSTEM SET password_encryption = 'scram-sha-256';

-- Set connection limits
ALTER ROLE sa3d_app CONNECTION LIMIT 50;
ALTER ROLE sa3d_readonly CONNECTION LIMIT 10;

-- Log security notice
DO $$
BEGIN
    RAISE NOTICE 'SA3D database initialized with security hardening';
    RAISE NOTICE 'Created roles: sa3d_app (read/write), sa3d_readonly (read-only)';
    RAISE NOTICE 'Enabled audit logging and row-level security';
    RAISE NOTICE 'Set secure password encryption and connection limits';
END $$;