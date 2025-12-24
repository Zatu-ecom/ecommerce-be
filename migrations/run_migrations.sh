#!/bin/bash

# Migration Runner Script
# Description: Runs SQL migration scripts in order
# Usage: ./run_migrations.sh [options]

# Note: We don't use 'set -e' because bash arithmetic ((count++)) returns 1 when count=0
# Instead, we handle errors explicitly in each function

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f ../.env ]; then
    # Source the .env file properly (handles quotes and special characters)
    set -a
    source ../.env
    set +a
    echo -e "${GREEN}✓${NC} Loaded environment from .env file"
else
    echo -e "${YELLOW}⚠${NC} .env file not found, using environment variables"
fi

# Database connection string
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ecommerce}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD}"

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to run SQL file
run_sql_file() {
    local file=$1
    local description=$2
    
    print_info "Running: $description"
    
    # Create a temporary file to capture error output
    local error_file=$(mktemp)
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" 2>"$error_file"; then
        print_success "Completed: $description"
        rm -f "$error_file"
        return 0
    else
        print_error "Failed: $description"
        echo ""
        echo -e "${RED}PostgreSQL Error Output:${NC}"
        echo "----------------------------------------"
        cat "$error_file"
        echo "----------------------------------------"
        echo ""
        rm -f "$error_file"
        return 1
    fi
}

# Function to check database connection
check_connection() {
    print_info "Checking database connection..."
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" > /dev/null 2>&1; then
        print_success "Database connection successful"
        return 0
    else
        print_error "Cannot connect to database"
        print_error "Host: $DB_HOST:$DB_PORT, Database: $DB_NAME, User: $DB_USER"
        return 1
    fi
}

# Function to create migration tracking table
create_migration_table() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
        CREATE TABLE IF NOT EXISTS schema_migration (
            id SERIAL PRIMARY KEY,
            filename VARCHAR(255) NOT NULL UNIQUE,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
            error_message TEXT,
            execution_time_ms INTEGER
        );
        
        -- Add columns if they don't exist (for existing tables)
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'schema_migration' AND column_name = 'status') THEN
                ALTER TABLE schema_migration ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS';
            END IF;
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'schema_migration' AND column_name = 'error_message') THEN
                ALTER TABLE schema_migration ADD COLUMN error_message TEXT;
            END IF;
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'schema_migration' AND column_name = 'execution_time_ms') THEN
                ALTER TABLE schema_migration ADD COLUMN execution_time_ms INTEGER;
            END IF;
        END \$\$;
    " > /dev/null 2>&1
}

# Function to check if migration was already applied successfully
# Only skips if status = 'SUCCESS', so FAILED/RUNNING scripts will be re-run
is_migration_applied() {
    local filename=$1
    local count=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT COUNT(*) FROM schema_migration WHERE filename = '$filename' AND status = 'SUCCESS';
    " 2>/dev/null | tr -d ' ')
    
    [ "$count" -gt 0 ]
}

# Function to record migration as applied
record_migration() {
    local filename=$1
    local status=${2:-SUCCESS}
    local error_message=${3:-}
    local execution_time=${4:-0}
    
    # Escape single quotes in error message
    error_message=$(echo "$error_message" | sed "s/'/''/g")
    
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
        INSERT INTO schema_migration (filename, status, error_message, execution_time_ms) 
        VALUES ('$filename', '$status', NULLIF('$error_message', ''), $execution_time) 
        ON CONFLICT (filename) DO UPDATE SET 
            status = '$status',
            error_message = NULLIF('$error_message', ''),
            execution_time_ms = $execution_time,
            applied_at = NOW();
    " > /dev/null 2>&1
}

# Function to run migrations
run_migrations() {
    echo ""
    echo "========================================="
    echo "  Running Database Migrations"
    echo "========================================="
    echo ""
    
    # Check connection first
    if ! check_connection; then
        exit 1
    fi
    
    # Create migration tracking table if not exists
    create_migration_table
    
    echo ""
    print_info "Starting migrations..."
    echo ""
    
    # Automatically discover and run all migration files in order (sorted by filename)
    local migration_files=($(ls -1 [0-9][0-9][0-9]_*.sql 2>/dev/null | sort))
    
    if [ ${#migration_files[@]} -eq 0 ]; then
        print_warning "No migration files found matching pattern: [0-9][0-9][0-9]_*.sql"
        return 1
    fi
    
    print_info "Found ${#migration_files[@]} migration file(s)"
    echo ""
    
    local applied_count=0
    local skipped_count=0
    local failed_count=0
    local applied_files=()
    local failed_files=()
    
    for file in "${migration_files[@]}"; do
        if is_migration_applied "$file"; then
            print_warning "Skipped (already applied): $file"
            skipped_count=$((skipped_count + 1))
        else
            # Record as RUNNING before execution
            record_migration "$file" "RUNNING" "" 0
            
            # Track execution time
            local start_time=$(date +%s%3N)
            
            # Create a temporary file to capture error output
            local error_file=$(mktemp)
            
            # Use ON_ERROR_STOP=1 to make psql exit with error on SQL failures
            if PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" 2>"$error_file"; then
                local end_time=$(date +%s%3N)
                local execution_time=$((end_time - start_time))
                
                print_success "Completed: $file (${execution_time}ms)"
                record_migration "$file" "SUCCESS" "" "$execution_time"
                rm -f "$error_file"
                applied_count=$((applied_count + 1))
            else
                local end_time=$(date +%s%3N)
                local execution_time=$((end_time - start_time))
                local error_msg=$(cat "$error_file")
                
                print_error "Failed: $file"
                echo ""
                echo -e "${RED}PostgreSQL Error Output:${NC}"
                echo "----------------------------------------"
                cat "$error_file"
                echo "----------------------------------------"
                echo ""
                
                record_migration "$file" "FAILED" "$error_msg" "$execution_time"
                rm -f "$error_file"
                failed_count=$((failed_count + 1))
                failed_files+=("$file")
                print_error "Migration failed, stopping."
                exit 1
            fi
        fi
    done
    
    echo ""
    print_success "Migrations completed! Applied: $applied_count, Skipped: $skipped_count, Failed: $failed_count"
    if [ ${#failed_files[@]} -gt 0 ]; then
        echo "  Failed files:"
        for f in "${failed_files[@]}"; do
            echo "    - $f"
        done
    fi
    echo ""
}

# Function to run seed data
run_seeds() {
    echo ""
    echo "========================================="
    echo "  Running Seed Data Scripts"
    echo "========================================="
    echo ""
    
    # Check connection first
    if ! check_connection; then
        exit 1
    fi
    
    # Create migration tracking table if not exists
    create_migration_table
    
    echo ""
    print_info "Starting seed data insertion..."
    echo ""
    
    # Automatically discover and run all seed files in order (sorted by filename)
    local seed_files=($(ls -1 seeds/[0-9][0-9][0-9]_*.sql 2>/dev/null | sort))
    
    if [ ${#seed_files[@]} -eq 0 ]; then
        print_warning "No seed files found in seeds/ directory matching pattern: [0-9][0-9][0-9]_*.sql"
        return 1
    fi
    
    print_info "Found ${#seed_files[@]} seed file(s)"
    echo ""
    
    local applied_count=0
    local skipped_count=0
    local failed_count=0
    local applied_files=()
    local failed_files=()
    
    for file in "${seed_files[@]}"; do
        if is_migration_applied "$file"; then
            print_warning "Skipped (already applied): $file"
            skipped_count=$((skipped_count + 1))
        else
            # Record as RUNNING before execution
            record_migration "$file" "RUNNING" "" 0
            
            # Track execution time
            local start_time=$(date +%s%3N)
            
            # Create a temporary file to capture error output
            local error_file=$(mktemp)
            
            # Use ON_ERROR_STOP=1 to make psql exit with error on SQL failures
            if PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" 2>"$error_file"; then
                local end_time=$(date +%s%3N)
                local execution_time=$((end_time - start_time))
                
                print_success "Completed: $file (${execution_time}ms)"
                record_migration "$file" "SUCCESS" "" "$execution_time"
                rm -f "$error_file"
                applied_count=$((applied_count + 1))
            else
                local end_time=$(date +%s%3N)
                local execution_time=$((end_time - start_time))
                local error_msg=$(cat "$error_file")
                
                print_error "Failed: $file"
                echo ""
                echo -e "${RED}PostgreSQL Error Output:${NC}"
                echo "----------------------------------------"
                cat "$error_file"
                echo "----------------------------------------"
                echo ""
                
                record_migration "$file" "FAILED" "$error_msg" "$execution_time"
                rm -f "$error_file"
                failed_count=$((failed_count + 1))
                failed_files+=("$file")
                print_error "Seed failed, stopping."
                exit 1
            fi
        fi
    done
    
    echo ""
    print_success "Seeds completed! Applied: $applied_count, Skipped: $skipped_count, Failed: $failed_count"
    if [ ${#failed_files[@]} -gt 0 ]; then
        echo "  Failed files:"
        for f in "${failed_files[@]}"; do
            echo "    - $f"
        done
    fi
    echo ""
}

# Function to run seed data (force - ignores tracking, always runs)
run_seeds_force() {
    echo ""
    echo "========================================="
    echo "  Running Seed Data Scripts (FORCE)"
    echo "========================================="
    echo ""
    
    print_warning "Force mode: Seeds will run even if previously applied"
    print_warning "This may cause duplicate data if seeds are not idempotent!"
    echo ""
    
    # Check connection first
    if ! check_connection; then
        exit 1
    fi
    
    # Create migration tracking table if not exists
    create_migration_table
    
    echo ""
    print_info "Starting seed data insertion (force mode)..."
    echo ""
    
    # Automatically discover and run all seed files in order (sorted by filename)
    local seed_files=($(ls -1 seeds/[0-9][0-9][0-9]_*.sql 2>/dev/null | sort))
    
    if [ ${#seed_files[@]} -eq 0 ]; then
        print_warning "No seed files found in seeds/ directory matching pattern: [0-9][0-9][0-9]_*.sql"
        return 1
    fi
    
    print_info "Found ${#seed_files[@]} seed file(s)"
    echo ""
    
    local applied_count=0
    local failed_count=0
    local applied_files=()
    local failed_files=()
    
    for file in "${seed_files[@]}"; do
        # Record as RUNNING before execution
        record_migration "$file" "RUNNING" "" 0
        
        # Track execution time
        local start_time=$(date +%s%3N)
        
        # Create a temporary file to capture error output
        local error_file=$(mktemp)
        
        # Use ON_ERROR_STOP=1 to make psql exit with error on SQL failures
        if PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" 2>"$error_file"; then
            local end_time=$(date +%s%3N)
            local execution_time=$((end_time - start_time))
            
            print_success "Completed: $file (${execution_time}ms)"
            record_migration "$file" "SUCCESS" "" "$execution_time"
            rm -f "$error_file"
            applied_count=$((applied_count + 1))
        else
            local end_time=$(date +%s%3N)
            local execution_time=$((end_time - start_time))
            local error_msg=$(cat "$error_file")
            
            print_error "Failed: $file"
            echo ""
            echo -e "${RED}PostgreSQL Error Output:${NC}"
            echo "----------------------------------------"
            cat "$error_file"
            echo "----------------------------------------"
            echo ""
            
            record_migration "$file" "FAILED" "$error_msg" "$execution_time"
            rm -f "$error_file"
            failed_count=$((failed_count + 1))
            failed_files+=("$file")
            print_error "Seed failed, stopping."
            exit 1
        fi
    done
    
    echo ""
    print_success "Seeds completed (force mode)! Applied: $applied_count, Failed: $failed_count"
    if [ ${#failed_files[@]} -gt 0 ]; then
        echo "  Failed files:"
        for f in "${failed_files[@]}"; do
            echo "    - $f"
        done
    fi
    echo ""
}

# Function to show help
show_help() {
    cat << EOF
Migration Runner Script

Usage: ./run_migrations.sh [OPTIONS]

Options:
    -h, --help          Show this help message
    -m, --migrate       Run migrations only (skips already applied)
    -s, --seed          Run seed data only (skips already applied)
    -fs, --force-seed   Run ALL seed data (ignores tracking, re-runs everything)
    -a, --all           Run migrations and seed data (default)
    -r, --reset         Drop all tables and recreate (WARNING: Data loss!)
    --status            Show migration status history

Examples:
    ./run_migrations.sh                 # Run all (migrations + seeds)
    ./run_migrations.sh --migrate       # Run migrations only
    ./run_migrations.sh --seed          # Run seeds only
    ./run_migrations.sh --all           # Run all (explicit)

Environment Variables (from .env):
    DB_HOST             Database host (default: localhost)
    DB_PORT             Database port (default: 5432)
    DB_NAME             Database name (default: ecommerce)
    DB_USER             Database user (default: postgres)
    DB_PASSWORD         Database password (required)

EOF
}

# Function to reset database
reset_database() {
    echo ""
    echo "========================================="
    echo "  ⚠️  RESETTING DATABASE ⚠️"
    echo "========================================="
    echo ""
    
    print_warning "This will DROP ALL TABLES and recreate them!"
    print_warning "ALL DATA WILL BE LOST!"
    echo ""
    read -p "Are you sure? Type 'yes' to continue: " confirm
    
    if [ "$confirm" != "yes" ]; then
        print_info "Reset cancelled."
        exit 0
    fi
    
    # Check connection first
    if ! check_connection; then
        exit 1
    fi
    
    echo ""
    print_info "Dropping all tables..."
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO $DB_USER; GRANT ALL ON SCHEMA public TO public;" > /dev/null 2>&1; then
        print_success "All tables dropped"
    else
        print_error "Failed to drop tables"
        exit 1
    fi
    
    # Run migrations and seeds
    run_migrations
    run_seeds
    
    echo ""
    print_success "Database reset complete!"
    echo ""
}

# Function to show migration status
show_status() {
    echo ""
    echo "========================================="
    echo "  Migration Status History"
    echo "========================================="
    echo ""
    
    # Check connection first
    if ! check_connection; then
        exit 1
    fi
    
    # Create migration tracking table if not exists
    create_migration_table
    
    echo ""
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
        SELECT 
            filename,
            status,
            execution_time_ms || 'ms' as execution_time,
            to_char(applied_at, 'YYYY-MM-DD HH24:MI:SS') as applied_at,
            CASE WHEN error_message IS NOT NULL THEN LEFT(error_message, 50) || '...' ELSE NULL END as error_preview
        FROM schema_migration 
        ORDER BY applied_at DESC;
    "
    
    echo ""
    echo "Summary:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
        SELECT 
            'Total: ' || COUNT(*) || 
            ', Success: ' || COUNT(*) FILTER (WHERE status = 'SUCCESS') ||
            ', Failed: ' || COUNT(*) FILTER (WHERE status = 'FAILED') ||
            ', Running: ' || COUNT(*) FILTER (WHERE status = 'RUNNING')
        FROM schema_migration;
    "
    echo ""
}

# Main script logic
case "${1:-all}" in
    -h|--help)
        show_help
        ;;
    -m|--migrate)
        run_migrations
        ;;
    -s|--seed)
        run_seeds
        ;;
    -fs|--force-seed)
        run_seeds_force
        ;;
    -a|--all|all)
        run_migrations
        run_seeds
        ;;
    -r|--reset)
        reset_database
        ;;
    --status)
        show_status
        ;;
    *)
        print_error "Unknown option: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
