-- Migration 001: Create Users Table with Security Features
-- Creates the user authentication system with proper security measures

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create users table in sa3d schema
CREATE TABLE sa3d.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    last_login TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance and security
CREATE INDEX idx_users_email ON sa3d.users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username ON sa3d.users (username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON sa3d.users (role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_active ON sa3d.users (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_locked ON sa3d.users (locked_until) WHERE locked_until IS NOT NULL;

-- Create user roles enum type
CREATE TYPE sa3d.user_role AS ENUM (
    'super_admin',
    'admin', 
    'project_manager',
    'developer',
    'analyst',
    'user',
    'viewer'
);

-- Alter users table to use the enum
ALTER TABLE sa3d.users 
    ALTER COLUMN role TYPE sa3d.user_role USING role::sa3d.user_role,
    ALTER COLUMN role SET DEFAULT 'user';

-- Create user sessions table for JWT blacklisting and session management
CREATE TABLE sa3d.user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES sa3d.users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token VARCHAR(255) UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create indexes for sessions
CREATE INDEX idx_user_sessions_user_id ON sa3d.user_sessions (user_id);
CREATE INDEX idx_user_sessions_token ON sa3d.user_sessions (session_token);
CREATE INDEX idx_user_sessions_refresh ON sa3d.user_sessions (refresh_token);
CREATE INDEX idx_user_sessions_expires ON sa3d.user_sessions (expires_at);
CREATE INDEX idx_user_sessions_active ON sa3d.user_sessions (is_active, expires_at);

-- Create password reset tokens table
CREATE TABLE sa3d.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES sa3d.users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create indexes for password reset tokens
CREATE INDEX idx_password_reset_user_id ON sa3d.password_reset_tokens (user_id);
CREATE INDEX idx_password_reset_token ON sa3d.password_reset_tokens (token);
CREATE INDEX idx_password_reset_expires ON sa3d.password_reset_tokens (expires_at);

-- Create email verification tokens table
CREATE TABLE sa3d.email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES sa3d.users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create indexes for email verification
CREATE INDEX idx_email_verification_user_id ON sa3d.email_verification_tokens (user_id);
CREATE INDEX idx_email_verification_token ON sa3d.email_verification_tokens (token);

-- Create user login attempts table for security monitoring
CREATE TABLE sa3d.login_attempts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255),
    ip_address INET NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(255),
    attempted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create indexes for login attempts
CREATE INDEX idx_login_attempts_email ON sa3d.login_attempts (email, attempted_at DESC);
CREATE INDEX idx_login_attempts_ip ON sa3d.login_attempts (ip_address, attempted_at DESC);
CREATE INDEX idx_login_attempts_success ON sa3d.login_attempts (success, attempted_at DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION sa3d.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON sa3d.users 
    FOR EACH ROW EXECUTE FUNCTION sa3d.update_updated_at_column();

CREATE TRIGGER update_user_sessions_updated_at 
    BEFORE UPDATE ON sa3d.user_sessions 
    FOR EACH ROW EXECUTE FUNCTION sa3d.update_updated_at_column();

-- Create audit triggers for security-sensitive tables
CREATE TRIGGER users_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sa3d.users
    FOR EACH ROW EXECUTE FUNCTION sa3d_audit.audit_trigger_func();

CREATE TRIGGER user_sessions_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sa3d.user_sessions
    FOR EACH ROW EXECUTE FUNCTION sa3d_audit.audit_trigger_func();

-- Enable Row Level Security
ALTER TABLE sa3d.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.password_reset_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.email_verification_tokens ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for users table
CREATE POLICY users_select_policy ON sa3d.users
    FOR SELECT USING (
        -- Users can see their own data, admins can see all
        id = (current_setting('app.current_user_id', true))::uuid 
        OR 
        (current_setting('app.current_user_role', true) IN ('admin', 'super_admin'))
    );

CREATE POLICY users_update_policy ON sa3d.users
    FOR UPDATE USING (
        -- Users can update their own data (except role), admins can update all
        (id = (current_setting('app.current_user_id', true))::uuid AND role = OLD.role)
        OR 
        (current_setting('app.current_user_role', true) IN ('admin', 'super_admin'))
    );

CREATE POLICY users_insert_policy ON sa3d.users
    FOR INSERT WITH CHECK (
        -- Only admins can create users directly, or system during registration
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin', 'system')
    );

CREATE POLICY users_delete_policy ON sa3d.users
    FOR DELETE USING (
        -- Only super_admins can delete users
        current_setting('app.current_user_role', true) = 'super_admin'
    );

-- Create RLS policies for user sessions
CREATE POLICY user_sessions_policy ON sa3d.user_sessions
    FOR ALL USING (
        user_id = (current_setting('app.current_user_id', true))::uuid
        OR 
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

-- Create password validation function
CREATE OR REPLACE FUNCTION sa3d.validate_password(password TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- Password must be at least 8 characters
    IF length(password) < 8 THEN
        RETURN FALSE;
    END IF;
    
    -- Password must contain at least one uppercase letter
    IF password !~ '[A-Z]' THEN
        RETURN FALSE;
    END IF;
    
    -- Password must contain at least one lowercase letter
    IF password !~ '[a-z]' THEN
        RETURN FALSE;
    END IF;
    
    -- Password must contain at least one number
    IF password !~ '[0-9]' THEN
        RETURN FALSE;
    END IF;
    
    -- Password must contain at least one special character
    IF password !~ '[!@#$%^&*(),.?":{}|<>]' THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to hash passwords
CREATE OR REPLACE FUNCTION sa3d.hash_password(password TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Validate password strength first
    IF NOT sa3d.validate_password(password) THEN
        RAISE EXCEPTION 'Password does not meet security requirements';
    END IF;
    
    -- Hash password with bcrypt (cost factor 12)
    RETURN crypt(password, gen_salt('bf', 12));
END;
$$ LANGUAGE plpgsql;

-- Create function to verify passwords
CREATE OR REPLACE FUNCTION sa3d.verify_password(password TEXT, hash TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN hash = crypt(password, hash);
END;
$$ LANGUAGE plpgsql;

-- Create function to check if account is locked
CREATE OR REPLACE FUNCTION sa3d.is_account_locked(user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    user_record RECORD;
BEGIN
    SELECT locked_until, failed_login_attempts 
    INTO user_record 
    FROM sa3d.users 
    WHERE id = user_id;
    
    -- If no lock time set, account is not locked
    IF user_record.locked_until IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- If lock time has passed, unlock the account
    IF user_record.locked_until <= now() THEN
        UPDATE sa3d.users 
        SET locked_until = NULL, failed_login_attempts = 0 
        WHERE id = user_id;
        RETURN FALSE;
    END IF;
    
    -- Account is currently locked
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to handle failed login attempts
CREATE OR REPLACE FUNCTION sa3d.handle_failed_login(user_email VARCHAR(255))
RETURNS VOID AS $$
DECLARE
    max_attempts CONSTANT INTEGER := 5;
    lockout_duration CONSTANT INTERVAL := '15 minutes';
    current_attempts INTEGER;
BEGIN
    -- Increment failed login attempts
    UPDATE sa3d.users 
    SET 
        failed_login_attempts = failed_login_attempts + 1,
        updated_at = now()
    WHERE email = user_email
    RETURNING failed_login_attempts INTO current_attempts;
    
    -- Lock account if max attempts reached
    IF current_attempts >= max_attempts THEN
        UPDATE sa3d.users 
        SET 
            locked_until = now() + lockout_duration,
            updated_at = now()
        WHERE email = user_email;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to handle successful login
CREATE OR REPLACE FUNCTION sa3d.handle_successful_login(user_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE sa3d.users 
    SET 
        failed_login_attempts = 0,
        locked_until = NULL,
        last_login = now(),
        updated_at = now()
    WHERE id = user_id;
END;
$$ LANGUAGE plpgsql;

-- Create default admin user (password will need to be changed on first login)
DO $$
DECLARE
    admin_id UUID;
    default_password TEXT := 'Admin123!ChangeMe';
BEGIN
    -- Only create if no admin users exist
    IF NOT EXISTS (SELECT 1 FROM sa3d.users WHERE role = 'super_admin') THEN
        INSERT INTO sa3d.users (
            email, 
            username, 
            password_hash, 
            first_name, 
            last_name, 
            role, 
            is_active, 
            is_verified
        ) VALUES (
            'admin@sa3d.local',
            'admin',
            sa3d.hash_password(default_password),
            'System',
            'Administrator', 
            'super_admin',
            true,
            true
        ) RETURNING id INTO admin_id;
        
        RAISE NOTICE 'Created default admin user: admin@sa3d.local with password: %', default_password;
        RAISE NOTICE 'SECURITY WARNING: Change the default admin password immediately!';
    END IF;
END $$;

-- Grant appropriate permissions to application role
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.users TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.user_sessions TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.password_reset_tokens TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.email_verification_tokens TO sa3d_app;
GRANT INSERT ON sa3d.login_attempts TO sa3d_app;

-- Grant readonly access to readonly role
GRANT SELECT ON sa3d.users TO sa3d_readonly;
GRANT SELECT ON sa3d.user_sessions TO sa3d_readonly;
GRANT SELECT ON sa3d.login_attempts TO sa3d_readonly;

-- Grant usage on sequences
GRANT USAGE, SELECT ON SEQUENCE sa3d.login_attempts_id_seq TO sa3d_app;

RAISE NOTICE 'Migration 001 completed: User authentication system created with security features';
RAISE NOTICE 'Features enabled: Password validation, account lockout, audit logging, RLS policies';
RAISE NOTICE 'Default admin created: admin@sa3d.local (change password immediately!)';