-- Migration: 006_create_collection_tables.sql
-- Description: Create collection tables in product service for custom product groupings
-- Created: 2025-11-25
-- Author: Product Service Team

-- =====================================================
-- 1. COLLECTION TABLE
-- =====================================================

CREATE TABLE IF NOT EXISTS collection (
    id BIGSERIAL PRIMARY KEY,
    
    -- Owner (Seller)
    seller_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    
    -- Collection Info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255),
    description TEXT,
    
    -- Display
    image TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_seller_collection_slug UNIQUE(seller_id, slug)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_collection_seller_id ON collection(seller_id);
CREATE INDEX IF NOT EXISTS idx_collection_slug ON collection(slug);
CREATE INDEX IF NOT EXISTS idx_collection_is_active ON collection(is_active);

-- =====================================================
-- 2. COLLECTION_PRODUCT TABLE (Junction table)
-- =====================================================

CREATE TABLE IF NOT EXISTS collection_product (
    id BIGSERIAL PRIMARY KEY,
    
    -- References
    collection_id BIGINT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    
    -- Display order within collection
    position INT DEFAULT 0,
    
    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_collection_product UNIQUE(collection_id, product_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_collection_product_collection_id ON collection_product(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_product_product_id ON collection_product(product_id);
CREATE INDEX IF NOT EXISTS idx_collection_product_position ON collection_product(position);

-- =====================================================
-- 3. TRIGGERS FOR UPDATED_AT
-- =====================================================

DO $$
BEGIN
    -- Collection
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_collection_updated_at') THEN
        CREATE TRIGGER update_collection_updated_at BEFORE UPDATE ON collection
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =====================================================
-- 4. ADD FOREIGN KEY TO PROMOTION TABLES (if they exist)
-- =====================================================

-- Add foreign key constraint to promotion_collection table
DO $$
BEGIN
    -- Check if promotion_collection table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'promotion_collection') THEN
        -- Check if foreign key doesn't already exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_promotion_collection_collection_id'
        ) THEN
            ALTER TABLE promotion_collection 
            ADD CONSTRAINT fk_promotion_collection_collection_id 
            FOREIGN KEY (collection_id) REFERENCES collection(id) ON DELETE CASCADE;
        END IF;
    END IF;
    
    -- Check if discount_code_collection table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'discount_code_collection') THEN
        -- Check if foreign key doesn't already exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_discount_code_collection_collection_id'
        ) THEN
            ALTER TABLE discount_code_collection 
            ADD CONSTRAINT fk_discount_code_collection_collection_id 
            FOREIGN KEY (collection_id) REFERENCES collection(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- =====================================================
-- 5. COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON TABLE collection IS 'Seller-created product collections for grouping products (e.g., "Summer Sale", "Best Sellers")';
COMMENT ON TABLE collection_product IS 'Junction table linking collections to products';

-- =====================================================
-- END OF MIGRATION
-- =====================================================
