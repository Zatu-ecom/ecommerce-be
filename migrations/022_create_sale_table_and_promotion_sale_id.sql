-- Migration: 022_create_sale_table_and_promotion_sale_id.sql
-- Description: Create sale table and link promotions to sales via sale_id

-- =====================================================
-- 1. SALE TABLE
-- =====================================================

CREATE TABLE IF NOT EXISTS sale (
    id BIGSERIAL PRIMARY KEY,

    seller_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,
    slug VARCHAR(255) NOT NULL,
    banner_images TEXT[] DEFAULT '{}',

    status VARCHAR(20) NOT NULL DEFAULT 'draft',

    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_seller_sale_slug UNIQUE(seller_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_sale_seller_id ON sale(seller_id);
CREATE INDEX IF NOT EXISTS idx_sale_status ON sale(status);
CREATE INDEX IF NOT EXISTS idx_sale_start_at ON sale(start_at);
CREATE INDEX IF NOT EXISTS idx_sale_end_at ON sale(end_at);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_sale_updated_at') THEN
        CREATE TRIGGER update_sale_updated_at
        BEFORE UPDATE ON sale
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =====================================================
-- 2. PROMOTION.SALE_ID
-- =====================================================

ALTER TABLE promotion
    ADD COLUMN IF NOT EXISTS sale_id BIGINT REFERENCES sale(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_promotion_sale_id ON promotion(sale_id);

COMMENT ON TABLE sale IS 'Seller-created sales campaigns that group related promotions';
COMMENT ON COLUMN promotion.sale_id IS 'Optional link to a parent sale campaign';
