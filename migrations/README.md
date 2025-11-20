# Database Migrations

Complete database migration system for the e-commerce application with **singular table names**.

---

## 🚀 Quick Start

```bash
cd /home/kushal/Work/Personal\ Codes/Ecommerce/ecommerce-be/migrations

# Run everything (create tables + insert data)
./run_migrations.sh --all

# Or create tables only
./run_migrations.sh --migrate

# Or insert data only
./run_migrations.sh --seed
```

---

## 📖 Documentation

- **[QUICK_COMMANDS.md](./QUICK_COMMANDS.md)** - Quick command cheat sheet
- **[HOW_TO_RUN.md](./HOW_TO_RUN.md)** - Complete step-by-step guide with all options

---

## 📂 Files

```
migrations/
├── run_migrations.sh                                 # Main automation script ⭐
├── 001_create_user_tables.sql                       # User service tables
├── 002_create_product_tables.sql                    # Product service tables
├── 003_alter_timestamps_to_timestamptz.sql          # Timestamp migration
├── 004_create_related_products_procedure.sql        # Related products stored 
└── seeds/
    ├── 001_seed_user_data.sql                       # User demo data
    └── 002_seed_product_data.sql                    # Product demo data
```

---

## 📋 What Gets Created

### User Service (6 tables):

- `role` - User roles
- `plan` - Subscription plans
- `"user"` - User accounts _(quoted - reserved keyword)_
- `"address"` - User addresses _(quoted - reserved keyword)_
- `seller_profile` - Seller information
- `subscription` - Active subscriptions

### Product Service (10 tables):

- `category` - Product categories
- `attribute_definition` - Reusable attributes
- `category_attribute` - Category-attribute mappings
- `product` - Product catalog
- `product_attribute` - Product attributes
- `product_option` - Variant options
- `product_variant` - Product SKUs
- `variant_option_value` - Variant-option links
- `package_option` - Bundle deals
- `product_option_values` - Option definitions

---

## ⚙️ Script Options

| Option              | Description                           |
| ------------------- | ------------------------------------- |
| `--migrate` or `-m` | Create tables only (no data)          |
| `--seed` or `-s`    | Insert demo data only                 |
| `--all` or `-a`     | Create tables + insert data (default) |
| `--reset` or `-r`   | Drop all tables and recreate ⚠️       |
| `--help` or `-h`    | Show help message                     |

---

## ✅ Features

- ✅ **Automatic discovery** - Script finds all migration files automatically
- ✅ **Idempotent** - Safe to run multiple times (`IF NOT EXISTS`)
- ✅ **Order guaranteed** - Files run in sorted order (001, 002, ...)
- ✅ **Environment-aware** - Loads config from `.env` file
- ✅ **Connection test** - Validates database before running
- ✅ **Colored output** - Easy to read progress indicators
- ✅ **Singular names** - All tables use singular naming convention
- ✅ **Reserved keywords handled** - `user` and `address` are properly quoted

---

## 🔧 Prerequisites

1. PostgreSQL must be running
2. Database must exist (or create it first)
3. `.env` file must be configured in project root

### .env Configuration

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ecommerce_test
DB_USER=postgres
DB_PASSWORD=postgres
```

---

## 📊 Demo Data Summary

**User Service:**

- 3 roles (Admin, Customer, Seller)
- 3 plans (Basic $49, Pro $99, Enterprise $199)
- 5 users (1 admin, 1 customer, 3 sellers)
- 3 seller profiles
- 3 subscriptions
- 6 addresses

**Product Service:**

- 11 categories
- 12 attribute definitions
- 9 products
- 18 product variants
- 4 package options

---

## ⚠️ Important Notes

1. **Singular Names**: All tables use singular names (`user`, not `users`)
2. **Reserved Keywords**: `user` and `address` are quoted in SQL
3. **No Soft Deletes**: No `deleted_at` fields anywhere
4. **Foreign Keys**: Properly configured with CASCADE/RESTRICT
5. **Auto-incrementing IDs**: PostgreSQL sequences managed automatically

---

## 🐛 Troubleshooting

### Database connection failed

```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Check .env file exists and has correct credentials
cat ../.env
```

### Database doesn't exist

```bash
# Create it first
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE ecommerce_test;"
```

### Permission denied

```bash
# Grant permissions
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -c "GRANT ALL ON DATABASE ecommerce_test TO postgres;"
```

---

## 📚 Additional Documentation

- **CORRECTIONS_SUMMARY.md** - Migration corrections history
- **SEED_DATA_CORRECTIONS.md** - Seed data corrections history

---

**Last Updated**: October 21, 2025  
**PostgreSQL Version**: 14+  
**Status**: ✅ Production Ready
