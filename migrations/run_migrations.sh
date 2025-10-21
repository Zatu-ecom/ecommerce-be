#!/bin/bash

# Migration Runner Script
# Description: Runs SQL migration scripts in order
# Usage: ./run_migrations.sh [options]

set -e  # Exit on error

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
else
    echo -e "${RED}Error: .env file not found!${NC}"
    exit 1
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
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" > /dev/null 2>&1; then
        print_success "Completed: $description"
        return 0
    else
        print_error "Failed: $description"
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
    
    for file in "${migration_files[@]}"; do
        run_sql_file "$file" "$file"
    done
    
    echo ""
    print_success "All migrations completed successfully!"
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
    
    for file in "${seed_files[@]}"; do
        run_sql_file "$file" "$file"
    done
    
    echo ""
    print_success "All seed data inserted successfully!"
    echo ""
}

# Function to show help
show_help() {
    cat << EOF
Migration Runner Script

Usage: ./run_migrations.sh [OPTIONS]

Options:
    -h, --help          Show this help message
    -m, --migrate       Run migrations only
    -s, --seed          Run seed data only
    -a, --all           Run migrations and seed data (default)
    -r, --reset         Drop all tables and recreate (WARNING: Data loss!)

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
    -a|--all|all)
        run_migrations
        run_seeds
        ;;
    -r|--reset)
        reset_database
        ;;
    *)
        print_error "Unknown option: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
