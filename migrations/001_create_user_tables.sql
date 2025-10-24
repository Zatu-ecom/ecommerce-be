-- Migration: 001_create_user_tables.sql
-- Description: Create all user service related tables (matches Go entities exactly)
-- Created: 2025-10-21
-- Updated: 2025-10-21 - Fixed to match actual entity definitions (using singular table names)

-- Create role table if not exists
CREATE TABLE IF NOT EXISTS role (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    level INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on role
CREATE INDEX IF NOT EXISTS idx_role_name ON role(name);
CREATE INDEX IF NOT EXISTS idx_role_level ON role(level);

-- Create plan table if not exists
CREATE TABLE IF NOT EXISTS plan (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    price DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
    is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    trial_days INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on plan
CREATE INDEX IF NOT EXISTS idx_plan_name ON plan(name);
CREATE INDEX IF NOT EXISTS idx_plan_billing_cycle ON plan(billing_cycle);

-- Create user table if not exists
CREATE TABLE IF NOT EXISTS "user" (
    id BIGSERIAL PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    phone VARCHAR(255),
    date_of_birth VARCHAR(255),
    gender VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    role_id BIGINT NOT NULL DEFAULT 3 REFERENCES role(id) ON DELETE RESTRICT,
    seller_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on user
CREATE INDEX IF NOT EXISTS idx_user_email ON "user"(email);
CREATE INDEX IF NOT EXISTS idx_user_role_id ON "user"(role_id);
CREATE INDEX IF NOT EXISTS idx_user_seller_id ON "user"(seller_id);
CREATE INDEX IF NOT EXISTS idx_user_is_active ON "user"(is_active);

-- Create address table if not exists
CREATE TABLE IF NOT EXISTS "address" (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    street VARCHAR(255) NOT NULL,
    city VARCHAR(255) NOT NULL,
    state VARCHAR(255) NOT NULL,
    zip_code VARCHAR(255) NOT NULL,
    country VARCHAR(255) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on address
CREATE INDEX IF NOT EXISTS idx_address_user_id ON "address"(user_id);
CREATE INDEX IF NOT EXISTS idx_address_is_default ON "address"(is_default);

-- Create subscription table if not exists
CREATE TABLE IF NOT EXISTS subscription (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL,
    plan_id BIGINT NOT NULL REFERENCES plan(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    payment_transaction_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on subscription
CREATE INDEX IF NOT EXISTS idx_subscription_seller_id ON subscription(seller_id);
CREATE INDEX IF NOT EXISTS idx_subscription_plan_id ON subscription(plan_id);
CREATE INDEX IF NOT EXISTS idx_subscription_status ON subscription(status);

-- Create seller_profile table if not exists
CREATE TABLE IF NOT EXISTS seller_profile (
    user_id BIGINT PRIMARY KEY REFERENCES "user"(id) ON DELETE CASCADE,
    business_name VARCHAR(255) NOT NULL,
    business_logo VARCHAR(255) NOT NULL,
    tax_id VARCHAR(255) UNIQUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on seller_profile
CREATE INDEX IF NOT EXISTS idx_seller_profile_user_id ON seller_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_seller_profile_is_verified ON seller_profile(is_verified);
CREATE INDEX IF NOT EXISTS idx_seller_profile_tax_id ON seller_profile(tax_id);

-- Add foreign key constraint for user.seller_id referencing seller_profile
-- This needs to be done after seller_profile table is created
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_user_seller_id'
    ) THEN
        ALTER TABLE "user" ADD CONSTRAINT fk_user_seller_id 
        FOREIGN KEY (seller_id) REFERENCES seller_profile(user_id) ON DELETE SET NULL;
    END IF;
END 
$$ language plpgsql;

-- Create updated_at trigger function if not exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language plpgsql;

-- Create triggers for updated_at on all tables
DO $$
BEGIN
    -- Role
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_role_updated_at') THEN
        CREATE TRIGGER update_role_updated_at BEFORE UPDATE ON role
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Plan
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_plan_updated_at') THEN
        CREATE TRIGGER update_plan_updated_at BEFORE UPDATE ON plan
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- User
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_user_updated_at') THEN
        CREATE TRIGGER update_user_updated_at BEFORE UPDATE ON "user"
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Address
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_address_updated_at') THEN
        CREATE TRIGGER update_address_updated_at BEFORE UPDATE ON "address"
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Subscription
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_subscription_updated_at') THEN
        CREATE TRIGGER update_subscription_updated_at BEFORE UPDATE ON subscription
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Seller Profile
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_seller_profile_updated_at') THEN
        CREATE TRIGGER update_seller_profile_updated_at BEFORE UPDATE ON seller_profile
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END 
$$ language plpgsql;
