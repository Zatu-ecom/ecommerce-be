-- Migration: 005_create_promotion_tables.sql
-- Description: Create all promotion service related tables (Consolidated)
-- Created: 2025-11-25
-- Updated: 2025-11-26 (Consolidated customer segments and cleanup)

-- =====================================================
-- 1. CUSTOMER SEGMENTS (Dynamic Rules)
-- =====================================================

CREATE TABLE IF NOT EXISTS customer_segments (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    seller_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_customer_segments_seller_id ON customer_segments(seller_id);

-- =====================================================
-- 2. PROMOTION (Sales/Offers created by sellers)
-- =====================================================

CREATE TABLE IF NOT EXISTS promotion (
    id BIGSERIAL PRIMARY KEY,
    
    -- OWNER
    seller_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    
    -- Promotion Info
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    slug VARCHAR(255),
    description TEXT,
    
    -- Promotion Mechanics
    promotion_type VARCHAR(50) NOT NULL,
    discount_config JSONB NOT NULL,
    
    -- Scope
    applies_to VARCHAR(50) NOT NULL DEFAULT 'specific_products',
    
    -- Conditions
    min_purchase_amount_cents BIGINT DEFAULT 0,
    min_quantity INT DEFAULT 1,
    max_discount_amount_cents BIGINT,
    
    -- Customer Eligibility
    eligible_for VARCHAR(50) DEFAULT 'everyone',
    customer_segment_id INTEGER REFERENCES customer_segments(id),
    
    -- Usage Limits
    usage_limit_total INT,
    usage_limit_per_customer INT DEFAULT 1,
    current_usage_count INT DEFAULT 0,
    
    -- Date Range
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    
    -- Automatic Start/Stop
    auto_start BOOLEAN DEFAULT TRUE,
    auto_end BOOLEAN DEFAULT TRUE,
    
    -- Status
    status VARCHAR(50) DEFAULT 'draft',
    
    -- Stacking Rules
    can_stack_with_other_promotions BOOLEAN DEFAULT FALSE,
    can_stack_with_coupons BOOLEAN DEFAULT TRUE,
    
    -- Display Settings
    show_on_storefront BOOLEAN DEFAULT TRUE,
    badge_text VARCHAR(50),
    badge_color VARCHAR(20),
    
    -- Priority
    priority INT DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_seller_promotion_slug UNIQUE(seller_id, slug)
);

-- Indexes for promotion
CREATE INDEX IF NOT EXISTS idx_promotion_seller_id ON promotion(seller_id);
CREATE INDEX IF NOT EXISTS idx_promotion_status ON promotion(status);
CREATE INDEX IF NOT EXISTS idx_promotion_starts_at ON promotion(starts_at);
CREATE INDEX IF NOT EXISTS idx_promotion_ends_at ON promotion(ends_at);
CREATE INDEX IF NOT EXISTS idx_promotion_customer_segment_id ON promotion(customer_segment_id);

-- =====================================================
-- 3. PROMOTION SCOPE TABLES
-- =====================================================

-- Products
CREATE TABLE IF NOT EXISTS promotion_product (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES promotion(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    variant_id BIGINT REFERENCES product_variant(id) ON DELETE CASCADE,
    override_discount_config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_promotion_product_variant UNIQUE(promotion_id, product_id, variant_id)
);

CREATE INDEX IF NOT EXISTS idx_promotion_product_promotion_id ON promotion_product(promotion_id);

-- Categories
CREATE TABLE IF NOT EXISTS promotion_category (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES promotion(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES category(id) ON DELETE CASCADE,
    include_subcategories BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_promotion_category UNIQUE(promotion_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_promotion_category_promotion_id ON promotion_category(promotion_id);

-- Collections
CREATE TABLE IF NOT EXISTS promotion_collection (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES promotion(id) ON DELETE CASCADE,
    collection_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_promotion_collection UNIQUE(promotion_id, collection_id)
);

CREATE INDEX IF NOT EXISTS idx_promotion_collection_promotion_id ON promotion_collection(promotion_id);

-- =====================================================
-- 4. DISCOUNT CODES
-- =====================================================

CREATE TABLE IF NOT EXISTS discount_code (
    id BIGSERIAL PRIMARY KEY,
    
    -- OWNER
    seller_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    
    -- Code
    code VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    
    -- Discount Type
    discount_type VARCHAR(50) NOT NULL,
    value BIGINT NOT NULL,
    
    -- Applies To
    applies_to VARCHAR(50) NOT NULL DEFAULT 'all_products',
    
    -- Requirements
    min_purchase_amount_cents BIGINT,
    min_quantity INT,
    
    -- Customer Eligibility
    customer_eligibility VARCHAR(50) DEFAULT 'everyone',
    customer_segment_id INTEGER REFERENCES customer_segments(id),
    
    -- Usage Limits
    usage_limit_total INT,
    usage_limit_per_customer INT DEFAULT 1,
    current_usage_count INT DEFAULT 0,
    
    -- Combinations
    can_combine_with_other_discounts BOOLEAN DEFAULT FALSE,
    
    -- Date Range
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_seller_discount_code UNIQUE(seller_id, code)
);

-- Indexes for discount_code
CREATE INDEX IF NOT EXISTS idx_discount_code_seller_id ON discount_code(seller_id);
CREATE INDEX IF NOT EXISTS idx_discount_code_code ON discount_code(code);
CREATE INDEX IF NOT EXISTS idx_discount_code_customer_segment_id ON discount_code(customer_segment_id);

-- =====================================================
-- 5. DISCOUNT CODE SCOPE TABLES
-- =====================================================

-- Products
CREATE TABLE IF NOT EXISTS discount_code_product (
    id BIGSERIAL PRIMARY KEY,
    discount_code_id BIGINT NOT NULL REFERENCES discount_code(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    variant_id BIGINT REFERENCES product_variant(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_discount_code_product_variant UNIQUE(discount_code_id, product_id, variant_id)
);

CREATE INDEX IF NOT EXISTS idx_discount_code_product_discount_code_id ON discount_code_product(discount_code_id);

-- Categories
CREATE TABLE IF NOT EXISTS discount_code_category (
    id BIGSERIAL PRIMARY KEY,
    discount_code_id BIGINT NOT NULL REFERENCES discount_code(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES category(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_discount_code_category UNIQUE(discount_code_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_discount_code_category_discount_code_id ON discount_code_category(discount_code_id);

-- Collections
CREATE TABLE IF NOT EXISTS discount_code_collection (
    id BIGSERIAL PRIMARY KEY,
    discount_code_id BIGINT NOT NULL REFERENCES discount_code(id) ON DELETE CASCADE,
    collection_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_discount_code_collection UNIQUE(discount_code_id, collection_id)
);

CREATE INDEX IF NOT EXISTS idx_discount_code_collection_discount_code_id ON discount_code_collection(discount_code_id);

-- =====================================================
-- 6. USAGE TRACKING TABLES
-- =====================================================

-- Promotion Usage
CREATE TABLE IF NOT EXISTS promotion_usage (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES promotion(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL,
    discount_amount_cents BIGINT NOT NULL,
    original_amount_cents BIGINT NOT NULL,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_promotion_usage_promotion_id ON promotion_usage(promotion_id);

-- Discount Code Usage
CREATE TABLE IF NOT EXISTS discount_code_usage (
    id BIGSERIAL PRIMARY KEY,
    discount_code_id BIGINT NOT NULL REFERENCES discount_code(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL,
    discount_amount_cents BIGINT NOT NULL,
    original_amount_cents BIGINT NOT NULL,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_discount_code_usage_discount_code_id ON discount_code_usage(discount_code_id);

-- =====================================================
-- 7. TRIGGERS
-- =====================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_promotion_updated_at') THEN
        CREATE TRIGGER update_promotion_updated_at BEFORE UPDATE ON promotion
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_discount_code_updated_at') THEN
        CREATE TRIGGER update_discount_code_updated_at BEFORE UPDATE ON discount_code
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_customer_segments_updated_at') THEN
        CREATE TRIGGER update_customer_segments_updated_at BEFORE UPDATE ON customer_segments
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
