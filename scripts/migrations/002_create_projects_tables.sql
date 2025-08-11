-- Migration 002: Create Projects and Analysis Tables
-- Creates the core project management and analysis system

-- Create projects table
CREATE TABLE sa3d.projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(100) NOT NULL,
    repository VARCHAR(500),
    branch VARCHAR(255) DEFAULT 'main',
    created_by UUID NOT NULL REFERENCES sa3d.users(id),
    last_analysis_id UUID, -- Will be set after analysis table is created
    
    -- Project settings
    auto_analyze BOOLEAN DEFAULT false,
    analyze_frequency VARCHAR(50) DEFAULT 'daily',
    ignore_patterns TEXT,
    max_file_size BIGINT DEFAULT 10485760, -- 10MB
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create project indexes
CREATE INDEX idx_projects_created_by ON sa3d.projects (created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_language ON sa3d.projects (language) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_name ON sa3d.projects (name) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_auto_analyze ON sa3d.projects (auto_analyze) WHERE deleted_at IS NULL AND auto_analyze = true;

-- Create project members junction table for access control
CREATE TABLE sa3d.project_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES sa3d.projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES sa3d.users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    permissions JSONB DEFAULT '{}',
    added_by UUID NOT NULL REFERENCES sa3d.users(id),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    
    UNIQUE(project_id, user_id)
);

-- Create project member role enum
CREATE TYPE sa3d.project_role AS ENUM (
    'owner',
    'maintainer', 
    'contributor',
    'analyst',
    'viewer'
);

ALTER TABLE sa3d.project_members 
    ALTER COLUMN role TYPE sa3d.project_role USING role::sa3d.project_role;

-- Create indexes for project members
CREATE INDEX idx_project_members_project_id ON sa3d.project_members (project_id);
CREATE INDEX idx_project_members_user_id ON sa3d.project_members (user_id);
CREATE INDEX idx_project_members_role ON sa3d.project_members (role);

-- Create analysis status enum
CREATE TYPE sa3d.analysis_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed',
    'cancelled'
);

-- Create analyses table
CREATE TABLE sa3d.analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES sa3d.projects(id) ON DELETE CASCADE,
    status sa3d.analysis_status NOT NULL DEFAULT 'pending',
    
    -- Analysis metadata
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    
    -- Analysis results (stored as JSONB for flexibility)
    results JSONB,
    
    -- Calculated metrics
    lines_of_code INTEGER DEFAULT 0,
    cyclomatic_complexity INTEGER DEFAULT 0,
    maintainability_index DECIMAL(5,2) DEFAULT 0,
    technical_debt DECIMAL(10,2) DEFAULT 0,
    code_smells INTEGER DEFAULT 0,
    bugs INTEGER DEFAULT 0,
    vulnerabilities INTEGER DEFAULT 0,
    security_hotspots INTEGER DEFAULT 0,
    coverage DECIMAL(5,2) DEFAULT 0,
    duplication_ratio DECIMAL(5,2) DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create analysis indexes
CREATE INDEX idx_analyses_project_id ON sa3d.analyses (project_id);
CREATE INDEX idx_analyses_status ON sa3d.analyses (status);
CREATE INDEX idx_analyses_created_at ON sa3d.analyses (created_at DESC);
CREATE INDEX idx_analyses_completed_at ON sa3d.analyses (completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_analyses_project_status ON sa3d.analyses (project_id, status);

-- Create analysis files table for detailed file-level metrics
CREATE TABLE sa3d.analysis_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_id UUID NOT NULL REFERENCES sa3d.analyses(id) ON DELETE CASCADE,
    
    -- File information
    file_path VARCHAR(1000) NOT NULL,
    file_language VARCHAR(100),
    file_size BIGINT NOT NULL,
    lines_of_code INTEGER NOT NULL DEFAULT 0,
    
    -- File-level metrics
    complexity INTEGER DEFAULT 0,
    functions JSONB DEFAULT '[]',
    classes JSONB DEFAULT '[]',
    imports JSONB DEFAULT '[]',
    dependencies JSONB DEFAULT '[]',
    
    -- Quality metrics
    issues JSONB DEFAULT '[]',
    smells INTEGER DEFAULT 0,
    bugs INTEGER DEFAULT 0,
    vulnerabilities INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create analysis files indexes
CREATE INDEX idx_analysis_files_analysis_id ON sa3d.analysis_files (analysis_id);
CREATE INDEX idx_analysis_files_path ON sa3d.analysis_files (file_path);
CREATE INDEX idx_analysis_files_language ON sa3d.analysis_files (file_language);
CREATE INDEX idx_analysis_files_complexity ON sa3d.analysis_files (complexity DESC);

-- Create analysis components table for architectural components
CREATE TABLE sa3d.analysis_components (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_id UUID NOT NULL REFERENCES sa3d.analyses(id) ON DELETE CASCADE,
    
    -- Component information
    component_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- service, library, module, package, etc.
    path VARCHAR(1000),
    description TEXT,
    
    -- Component metrics
    files JSONB DEFAULT '[]',
    size_bytes BIGINT DEFAULT 0,
    complexity INTEGER DEFAULT 0,
    dependencies JSONB DEFAULT '[]',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    
    UNIQUE(analysis_id, component_id)
);

-- Create component indexes
CREATE INDEX idx_analysis_components_analysis_id ON sa3d.analysis_components (analysis_id);
CREATE INDEX idx_analysis_components_type ON sa3d.analysis_components (type);
CREATE INDEX idx_analysis_components_name ON sa3d.analysis_components (name);

-- Create component relationships table
CREATE TABLE sa3d.analysis_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_id UUID NOT NULL REFERENCES sa3d.analyses(id) ON DELETE CASCADE,
    
    -- Relationship information
    source_component VARCHAR(255) NOT NULL,
    target_component VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(100) NOT NULL, -- depends_on, calls, implements, extends
    strength INTEGER DEFAULT 1,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    
    UNIQUE(analysis_id, source_component, target_component, relationship_type)
);

-- Create relationship indexes
CREATE INDEX idx_analysis_relationships_analysis_id ON sa3d.analysis_relationships (analysis_id);
CREATE INDEX idx_analysis_relationships_source ON sa3d.analysis_relationships (source_component);
CREATE INDEX idx_analysis_relationships_target ON sa3d.analysis_relationships (target_component);
CREATE INDEX idx_analysis_relationships_type ON sa3d.analysis_relationships (relationship_type);

-- Add the foreign key constraint for last_analysis_id now that analyses table exists
ALTER TABLE sa3d.projects 
    ADD CONSTRAINT fk_projects_last_analysis 
    FOREIGN KEY (last_analysis_id) REFERENCES sa3d.analyses(id);

-- Create updated_at triggers for all new tables
CREATE TRIGGER update_projects_updated_at 
    BEFORE UPDATE ON sa3d.projects 
    FOR EACH ROW EXECUTE FUNCTION sa3d.update_updated_at_column();

CREATE TRIGGER update_project_members_updated_at 
    BEFORE UPDATE ON sa3d.project_members 
    FOR EACH ROW EXECUTE FUNCTION sa3d.update_updated_at_column();

CREATE TRIGGER update_analyses_updated_at 
    BEFORE UPDATE ON sa3d.analyses 
    FOR EACH ROW EXECUTE FUNCTION sa3d.update_updated_at_column();

-- Create audit triggers for security-sensitive operations
CREATE TRIGGER projects_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sa3d.projects
    FOR EACH ROW EXECUTE FUNCTION sa3d_audit.audit_trigger_func();

CREATE TRIGGER project_members_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sa3d.project_members
    FOR EACH ROW EXECUTE FUNCTION sa3d_audit.audit_trigger_func();

CREATE TRIGGER analyses_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sa3d.analyses
    FOR EACH ROW EXECUTE FUNCTION sa3d_audit.audit_trigger_func();

-- Enable Row Level Security
ALTER TABLE sa3d.projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.analyses ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.analysis_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.analysis_components ENABLE ROW LEVEL SECURITY;
ALTER TABLE sa3d.analysis_relationships ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for projects
CREATE POLICY projects_select_policy ON sa3d.projects
    FOR SELECT USING (
        -- Project creator, members, or admins can see projects
        created_by = (current_setting('app.current_user_id', true))::uuid
        OR
        id IN (
            SELECT project_id FROM sa3d.project_members 
            WHERE user_id = (current_setting('app.current_user_id', true))::uuid
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

CREATE POLICY projects_insert_policy ON sa3d.projects
    FOR INSERT WITH CHECK (
        -- Users can create projects, admins can create for others
        created_by = (current_setting('app.current_user_id', true))::uuid
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

CREATE POLICY projects_update_policy ON sa3d.projects
    FOR UPDATE USING (
        -- Project owner, maintainers, or admins can update
        created_by = (current_setting('app.current_user_id', true))::uuid
        OR
        id IN (
            SELECT project_id FROM sa3d.project_members 
            WHERE user_id = (current_setting('app.current_user_id', true))::uuid
            AND role IN ('owner', 'maintainer')
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

-- Create RLS policies for project members
CREATE POLICY project_members_select_policy ON sa3d.project_members
    FOR SELECT USING (
        -- Members can see all members of their projects
        user_id = (current_setting('app.current_user_id', true))::uuid
        OR
        project_id IN (
            SELECT project_id FROM sa3d.project_members pm 
            WHERE pm.user_id = (current_setting('app.current_user_id', true))::uuid
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

CREATE POLICY project_members_insert_policy ON sa3d.project_members
    FOR INSERT WITH CHECK (
        -- Project owners/maintainers can add members
        project_id IN (
            SELECT project_id FROM sa3d.project_members 
            WHERE user_id = (current_setting('app.current_user_id', true))::uuid
            AND role IN ('owner', 'maintainer')
        )
        OR
        project_id IN (
            SELECT id FROM sa3d.projects 
            WHERE created_by = (current_setting('app.current_user_id', true))::uuid
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

-- Create RLS policies for analyses
CREATE POLICY analyses_select_policy ON sa3d.analyses
    FOR SELECT USING (
        -- Project members can see analyses
        project_id IN (
            SELECT project_id FROM sa3d.project_members 
            WHERE user_id = (current_setting('app.current_user_id', true))::uuid
        )
        OR
        project_id IN (
            SELECT id FROM sa3d.projects 
            WHERE created_by = (current_setting('app.current_user_id', true))::uuid
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

-- Create policies for analysis detail tables (inherit from analyses)
CREATE POLICY analysis_files_policy ON sa3d.analysis_files
    FOR ALL USING (
        analysis_id IN (
            SELECT id FROM sa3d.analyses a
            WHERE a.project_id IN (
                SELECT project_id FROM sa3d.project_members 
                WHERE user_id = (current_setting('app.current_user_id', true))::uuid
            )
            OR a.project_id IN (
                SELECT id FROM sa3d.projects 
                WHERE created_by = (current_setting('app.current_user_id', true))::uuid
            )
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

CREATE POLICY analysis_components_policy ON sa3d.analysis_components
    FOR ALL USING (
        analysis_id IN (
            SELECT id FROM sa3d.analyses a
            WHERE a.project_id IN (
                SELECT project_id FROM sa3d.project_members 
                WHERE user_id = (current_setting('app.current_user_id', true))::uuid
            )
            OR a.project_id IN (
                SELECT id FROM sa3d.projects 
                WHERE created_by = (current_setting('app.current_user_id', true))::uuid
            )
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

CREATE POLICY analysis_relationships_policy ON sa3d.analysis_relationships
    FOR ALL USING (
        analysis_id IN (
            SELECT id FROM sa3d.analyses a
            WHERE a.project_id IN (
                SELECT project_id FROM sa3d.project_members 
                WHERE user_id = (current_setting('app.current_user_id', true))::uuid
            )
            OR a.project_id IN (
                SELECT id FROM sa3d.projects 
                WHERE created_by = (current_setting('app.current_user_id', true))::uuid
            )
        )
        OR
        current_setting('app.current_user_role', true) IN ('admin', 'super_admin')
    );

-- Grant permissions to application roles
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.projects TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.project_members TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.analyses TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.analysis_files TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.analysis_components TO sa3d_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sa3d.analysis_relationships TO sa3d_app;

-- Grant readonly access
GRANT SELECT ON sa3d.projects TO sa3d_readonly;
GRANT SELECT ON sa3d.project_members TO sa3d_readonly;
GRANT SELECT ON sa3d.analyses TO sa3d_readonly;
GRANT SELECT ON sa3d.analysis_files TO sa3d_readonly;
GRANT SELECT ON sa3d.analysis_components TO sa3d_readonly;
GRANT SELECT ON sa3d.analysis_relationships TO sa3d_readonly;

RAISE NOTICE 'Migration 002 completed: Project and analysis system created';
RAISE NOTICE 'Features: Project access control, detailed analysis tracking, RLS policies';