-- Migration: 002_create_product_tables.sql
-- Description: Create all product service related tables (matches Go entities exactly)
-- Created: 2025-10-21
-- Updated: 2025-10-21 - Fixed to match actual entity definitions

-- Create categories table if not exists
CREATE TABLE IF NOT EXISTS category (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    parent_id BIGINT REFERENCES category(id) ON DELETE RESTRICT,
    description TEXT,
    is_global BOOLEAN NOT NULL DEFAULT False,
    seller_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on categories
CREATE INDEX IF NOT EXISTS idx_category_parent_id ON category(parent_id);
CREATE INDEX IF NOT EXISTS idx_category_seller_id ON category(seller_id);
CREATE INDEX IF NOT EXISTS idx_category_is_global ON category(is_global);
CREATE INDEX IF NOT EXISTS idx_category_name ON category(name);

-- Create attribute_definitions table if not exists
CREATE TABLE IF NOT EXISTS attribute_definition (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    unit VARCHAR(255),
    allowed_values TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on attribute_definitions
CREATE INDEX IF NOT EXISTS idx_attribute_definition_key ON attribute_definition(key);
CREATE INDEX IF NOT EXISTS idx_attribute_definition_name ON attribute_definition(name);

-- Create category_attributes table if not exists (junction table)
CREATE TABLE IF NOT EXISTS category_attribute (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES category(id) ON DELETE CASCADE,
    attribute_definition_id BIGINT NOT NULL REFERENCES attribute_definition(id) ON DELETE RESTRICT,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_searchable BOOLEAN NOT NULL DEFAULT FALSE,
    is_filterable BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(category_id, attribute_definition_id)
);

-- Create indexes on category_attributes
CREATE INDEX IF NOT EXISTS idx_category_attribute_category_id ON category_attribute(category_id);
CREATE INDEX IF NOT EXISTS idx_category_attribute_attribute_definition_id ON category_attribute(attribute_definition_id);

-- Create products table if not exists
CREATE TABLE IF NOT EXISTS product (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category_id BIGINT NOT NULL REFERENCES category(id) ON DELETE RESTRICT,
    brand VARCHAR(255),
    base_sku VARCHAR(255) UNIQUE,
    short_description TEXT,
    long_description TEXT,
    tags TEXT[],
    seller_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on products
CREATE INDEX IF NOT EXISTS idx_product_category_id ON product(category_id);
CREATE INDEX IF NOT EXISTS idx_product_seller_id ON product(seller_id);
CREATE INDEX IF NOT EXISTS idx_product_base_sku ON product(base_sku);
CREATE INDEX IF NOT EXISTS idx_product_brand ON product(brand);
CREATE INDEX IF NOT EXISTS idx_product_name ON product USING gin(to_tsvector('english', name));
CREATE INDEX IF NOT EXISTS idx_product_tags ON product USING gin(tags);

-- Create product_attributes table if not exists
CREATE TABLE IF NOT EXISTS product_attribute (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    attribute_definition_id BIGINT NOT NULL REFERENCES attribute_definition(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, attribute_definition_id)
);

-- Create indexes on product_attributes
CREATE INDEX IF NOT EXISTS idx_product_attribute_product_id ON product_attribute(product_id);
CREATE INDEX IF NOT EXISTS idx_product_attribute_attribute_definition_id ON product_attribute(attribute_definition_id);
CREATE INDEX IF NOT EXISTS idx_product_attribute_value ON product_attribute(value);

-- Create product_options table if not exists (e.g., Color, Size)
CREATE TABLE IF NOT EXISTS product_option (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, name)
);

-- Create indexes on product_options
CREATE INDEX IF NOT EXISTS idx_product_option_product_id ON product_option(product_id);

-- Create product_option_value table if not exists (e.g., Red, Blue, Small, Large)
CREATE TABLE IF NOT EXISTS product_option_value (
    id BIGSERIAL PRIMARY KEY,
    option_id BIGINT NOT NULL REFERENCES product_option(id) ON DELETE CASCADE,
    value VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    color_code VARCHAR(255),
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(option_id, value)
);

-- Create indexes on product_option_value
CREATE INDEX IF NOT EXISTS idx_product_option_value_option_id ON product_option_value(option_id);

-- Create product_variants table if not exists
CREATE TABLE IF NOT EXISTS product_variant (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    sku VARCHAR(255) UNIQUE,
    price DOUBLE PRECISION NOT NULL,
    images TEXT[],
    allow_purchase BOOLEAN NOT NULL DEFAULT TRUE,
    is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on product_variants
CREATE INDEX IF NOT EXISTS idx_product_variant_product_id ON product_variant(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variant_sku ON product_variant(sku);
CREATE INDEX IF NOT EXISTS idx_product_variant_allow_purchase ON product_variant(allow_purchase);
CREATE INDEX IF NOT EXISTS idx_product_variant_is_popular ON product_variant(is_popular);

-- Create variant_option_values table if not exists (junction table for variant-option relationships)
CREATE TABLE IF NOT EXISTS variant_option_value (
    id BIGSERIAL PRIMARY KEY,
    variant_id BIGINT NOT NULL REFERENCES product_variant(id) ON DELETE CASCADE,
    option_id BIGINT NOT NULL REFERENCES product_option(id) ON DELETE CASCADE,
    option_value_id BIGINT NOT NULL REFERENCES product_option_value(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(variant_id, option_id, option_value_id)
);

-- Create indexes on variant_option_values
CREATE INDEX IF NOT EXISTS idx_variant_option_value_variant_id ON variant_option_value(variant_id);
CREATE INDEX IF NOT EXISTS idx_variant_option_value_option_id ON variant_option_value(option_id);
CREATE INDEX IF NOT EXISTS idx_variant_option_value_option_value_id ON variant_option_value(option_value_id);

-- Create package_options table if not exists
CREATE TABLE IF NOT EXISTS package_option (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DOUBLE PRECISION NOT NULL,
    quantity INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on package_options
CREATE INDEX IF NOT EXISTS idx_package_option_product_id ON package_option(product_id);

-- Create triggers for updated_at on all product tables
DO $$
BEGIN
    -- Categories
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_category_updated_at') THEN
        CREATE TRIGGER update_category_updated_at BEFORE UPDATE ON category
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Attribute Definitions
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_attribute_definition_updated_at') THEN
        CREATE TRIGGER update_attribute_definition_updated_at BEFORE UPDATE ON attribute_definition
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Category Attributes
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_category_attribute_updated_at') THEN
        CREATE TRIGGER update_category_attribute_updated_at BEFORE UPDATE ON category_attribute
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Products
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_product_updated_at') THEN
        CREATE TRIGGER update_product_updated_at BEFORE UPDATE ON product
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Product Attributes
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_product_attribute_updated_at') THEN
        CREATE TRIGGER update_product_attribute_updated_at BEFORE UPDATE ON product_attribute
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Product Options
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_product_option_updated_at') THEN
        CREATE TRIGGER update_product_option_updated_at BEFORE UPDATE ON product_option
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Product Option Values
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_product_option_value_updated_at') THEN
        CREATE TRIGGER update_product_option_value_updated_at BEFORE UPDATE ON product_option_value
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Product Variants
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_product_variant_updated_at') THEN
        CREATE TRIGGER update_product_variant_updated_at BEFORE UPDATE ON product_variant
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Variant Option Values
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_variant_option_value_updated_at') THEN
        CREATE TRIGGER update_variant_option_value_updated_at BEFORE UPDATE ON variant_option_value
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Package Options
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_package_option_updated_at') THEN
        CREATE TRIGGER update_package_option_updated_at BEFORE UPDATE ON package_option
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
