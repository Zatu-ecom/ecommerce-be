-- Migration: 008_create_geo_tables.sql
-- Description: Create country, currency tables and seller_settings for multi-country/currency support
-- Created: 2025-12-28

-- ============================================================================
-- COUNTRY TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS country (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(2) NOT NULL,                    -- ISO 3166-1 alpha-2: 'US', 'IN', 'GB'
    code_alpha3 VARCHAR(3),                      -- ISO 3166-1 alpha-3: 'USA', 'IND', 'GBR'
    name VARCHAR(100) NOT NULL,                  -- 'United States', 'India'
    native_name VARCHAR(100),                    -- 'भारत' for India
    phone_code VARCHAR(10),                      -- '+1', '+91'
    region VARCHAR(50),                          -- 'Asia', 'Europe', 'Americas'
    flag_emoji VARCHAR(10),                      -- '🇺🇸', '🇮🇳'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,     -- Platform supports this country
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create unique indexes on country
CREATE UNIQUE INDEX IF NOT EXISTS idx_country_code ON country(code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_country_code_alpha3 ON country(code_alpha3) WHERE code_alpha3 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_country_is_active ON country(is_active);
CREATE INDEX IF NOT EXISTS idx_country_region ON country(region);

-- ============================================================================
-- CURRENCY TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS currency (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(3) NOT NULL,                    -- ISO 4217: 'USD', 'INR', 'EUR'
    name VARCHAR(100) NOT NULL,                  -- 'US Dollar', 'Indian Rupee'
    symbol VARCHAR(10) NOT NULL,                 -- '$', '₹', '€'
    symbol_native VARCHAR(10),                   -- Native symbol
    decimal_digits INT NOT NULL DEFAULT 2,       -- 2 for USD, 0 for JPY
    is_active BOOLEAN NOT NULL DEFAULT TRUE,     -- Currency is active
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on currency
CREATE UNIQUE INDEX IF NOT EXISTS idx_currency_code ON currency(code);
CREATE INDEX IF NOT EXISTS idx_currency_is_active ON currency(is_active);

-- ============================================================================
-- COUNTRY_CURRENCY TABLE (Many-to-Many)
-- ============================================================================
CREATE TABLE IF NOT EXISTS country_currency (
    id BIGSERIAL PRIMARY KEY,
    country_id BIGINT NOT NULL REFERENCES country(id) ON DELETE CASCADE,
    currency_id BIGINT NOT NULL REFERENCES currency(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,   -- Primary currency for this country
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on country_currency
CREATE UNIQUE INDEX IF NOT EXISTS idx_country_currency_unique ON country_currency(country_id, currency_id);
CREATE INDEX IF NOT EXISTS idx_country_currency_country_id ON country_currency(country_id);
CREATE INDEX IF NOT EXISTS idx_country_currency_currency_id ON country_currency(currency_id);

-- ============================================================================
-- SELLER_SETTINGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS seller_settings (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    business_country_id BIGINT NOT NULL REFERENCES country(id),
    base_currency_id BIGINT NOT NULL REFERENCES currency(id),
    settlement_currency_id BIGINT REFERENCES currency(id),
    display_prices_in_buyer_currency BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on seller_settings
CREATE UNIQUE INDEX IF NOT EXISTS idx_seller_settings_seller_id ON seller_settings(seller_id);
CREATE INDEX IF NOT EXISTS idx_seller_settings_business_country ON seller_settings(business_country_id);
CREATE INDEX IF NOT EXISTS idx_seller_settings_base_currency ON seller_settings(base_currency_id);

-- ============================================================================
-- ALTER USER TABLE - Add currency preference only (country is derived from default address)
-- ============================================================================
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS currency_id BIGINT REFERENCES currency(id);
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS locale VARCHAR(10) DEFAULT 'en-US';

CREATE INDEX IF NOT EXISTS idx_user_currency_id ON "user"(currency_id);

-- ============================================================================
-- ADD FK CONSTRAINT FOR ADDRESS.COUNTRY_ID (address table created in 001)
-- ============================================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_address_country'
    ) THEN
        ALTER TABLE "address" ADD CONSTRAINT fk_address_country 
        FOREIGN KEY (country_id) REFERENCES country(id);
    END IF;
END 
$$ language plpgsql;
-- ============================================================================
-- TRIGGERS FOR updated_at
-- ============================================================================
DO $$
BEGIN
    -- Country
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_country_updated_at') THEN
        CREATE TRIGGER update_country_updated_at BEFORE UPDATE ON country
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Currency
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_currency_updated_at') THEN
        CREATE TRIGGER update_currency_updated_at BEFORE UPDATE ON currency
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Seller Settings
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_seller_settings_updated_at') THEN
        CREATE TRIGGER update_seller_settings_updated_at BEFORE UPDATE ON seller_settings
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
