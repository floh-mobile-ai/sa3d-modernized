#!/bin/bash

# SA3D Database Migration Runner
# This script runs database migrations in order

set -euo pipefail

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sa3d_db}"
DB_USER="${DB_USER:-sa3d}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if psql is available
check_psql() {
    if ! command -v psql &> /dev/null; then
        log_error "psql command not found. Please install PostgreSQL client."
        exit 1
    fi
}

# Function to check database connection
check_connection() {
    log_info "Testing database connection..."
    
    if [ -z "$DB_PASSWORD" ]; then
        log_error "DB_PASSWORD environment variable is required"
        exit 1
    fi
    
    export PGPASSWORD="$DB_PASSWORD"
    
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
        log_error "Failed to connect to database"
        log_error "Connection details: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
        exit 1
    fi
    
    log_success "Database connection successful"
}

# Function to create migrations table if it doesn't exist
create_migrations_table() {
    log_info "Creating migrations tracking table..."
    
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
CREATE TABLE IF NOT EXISTS sa3d.schema_migrations (
    id SERIAL PRIMARY KEY,
    migration_name VARCHAR(255) UNIQUE NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    checksum VARCHAR(64)
);

-- Grant permissions
GRANT SELECT, INSERT ON sa3d.schema_migrations TO sa3d_app;
GRANT SELECT ON sa3d.schema_migrations TO sa3d_readonly;
EOF

    log_success "Migrations table ready"
}

# Function to calculate file checksum
calculate_checksum() {
    if command -v sha256sum &> /dev/null; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum &> /dev/null; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        # Fallback - just use file size and name
        echo "$(basename "$1")_$(wc -c < "$1")"
    fi
}

# Function to check if migration was already applied
is_migration_applied() {
    local migration_name="$1"
    local count
    
    count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
        "SELECT COUNT(*) FROM sa3d.schema_migrations WHERE migration_name = '$migration_name';" | tr -d ' ')
    
    [ "$count" -gt 0 ]
}

# Function to run a single migration
run_migration() {
    local migration_file="$1"
    local migration_name
    local checksum
    
    migration_name=$(basename "$migration_file" .sql)
    checksum=$(calculate_checksum "$migration_file")
    
    log_info "Processing migration: $migration_name"
    
    if is_migration_applied "$migration_name"; then
        log_warning "Migration $migration_name already applied, skipping"
        return 0
    fi
    
    log_info "Applying migration: $migration_name"
    
    # Run the migration in a transaction
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
BEGIN;

-- Run the migration
\i $migration_file

-- Record the migration
INSERT INTO sa3d.schema_migrations (migration_name, checksum) 
VALUES ('$migration_name', '$checksum');

COMMIT;
EOF
    then
        log_success "Migration $migration_name applied successfully"
    else
        log_error "Migration $migration_name failed"
        exit 1
    fi
}

# Function to run all migrations
run_all_migrations() {
    local migrations_dir="$(dirname "$0")/migrations"
    
    if [ ! -d "$migrations_dir" ]; then
        log_error "Migrations directory not found: $migrations_dir"
        exit 1
    fi
    
    log_info "Looking for migrations in: $migrations_dir"
    
    # Find all SQL files and sort them numerically
    local migration_files
    migration_files=$(find "$migrations_dir" -name "*.sql" | sort -V)
    
    if [ -z "$migration_files" ]; then
        log_warning "No migration files found"
        return 0
    fi
    
    local count=0
    for migration_file in $migration_files; do
        run_migration "$migration_file"
        ((count++))
    done
    
    log_success "Applied $count migrations"
}

# Function to show migration status
show_status() {
    log_info "Migration Status:"
    echo
    
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
SELECT 
    migration_name,
    applied_at,
    checksum
FROM sa3d.schema_migrations 
ORDER BY id;
EOF
}

# Function to show help
show_help() {
    cat << EOF
SA3D Database Migration Runner

Usage: $0 [OPTIONS] [COMMAND]

Commands:
    migrate     Run all pending migrations (default)
    status      Show migration status
    help        Show this help message

Options:
    -h, --help  Show help message

Environment Variables:
    DB_HOST     Database host (default: localhost)
    DB_PORT     Database port (default: 5432)
    DB_NAME     Database name (default: sa3d_db)
    DB_USER     Database user (default: sa3d)
    DB_PASSWORD Database password (required)

Examples:
    # Run all migrations
    ./run-migrations.sh
    
    # Check migration status
    ./run-migrations.sh status
    
    # With custom database settings
    DB_HOST=prod-db.example.com DB_PASSWORD=secret ./run-migrations.sh

Security Notes:
    - Always backup your database before running migrations
    - Use strong passwords and secure connections in production
    - Review migration files before applying them
EOF
}

# Main function
main() {
    local command="${1:-migrate}"
    
    case "$command" in
        "migrate")
            log_info "🚀 Starting SA3D Database Migrations"
            echo "=================================="
            check_psql
            check_connection
            create_migrations_table
            run_all_migrations
            echo
            log_success "🎉 All migrations completed successfully!"
            ;;
        "status")
            check_psql
            check_connection
            show_status
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"