-- Migration: 028_align_discount_code_and_cart_coupon.sql
-- Description: Align discount_code with Go entity; create cart_applied_coupon and
--              cart_item_promotion; add updated_at on discount-code scope tables;
--              add indexes for seller/active, usage, and cart junctions.
-- Notes: Idempotent ALTER/CREATE IF NOT EXISTS; SingularTable naming.

-- =====================================================
-- 1. ALTER discount_code — columns present on entity, missing from 005
-- =====================================================

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS description TEXT;

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS max_discount_amount_cents BIGINT;

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS usage_reset_time_type VARCHAR(20) NOT NULL DEFAULT 'none';

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS usage_reset_amount INT;

CREATE INDEX IF NOT EXISTS idx_discount_code_seller_id_is_active
    ON discount_code (seller_id, is_active);

-- =====================================================
-- 2. discount_code_usage — per-customer usage lookup
-- =====================================================

CREATE INDEX IF NOT EXISTS idx_discount_code_usage_code_user
    ON discount_code_usage (discount_code_id, user_id);

CREATE INDEX IF NOT EXISTS idx_discount_code_usage_user_id
    ON discount_code_usage (user_id);

-- =====================================================
-- 3. discount-code scope tables — updated_at for BaseEntity
-- =====================================================

ALTER TABLE discount_code_product
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE discount_code_category
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE discount_code_collection
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_discount_code_product_updated_at') THEN
        CREATE TRIGGER update_discount_code_product_updated_at
            BEFORE UPDATE ON discount_code_product
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_discount_code_category_updated_at') THEN
        CREATE TRIGGER update_discount_code_category_updated_at
            BEFORE UPDATE ON discount_code_category
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_discount_code_collection_updated_at') THEN
        CREATE TRIGGER update_discount_code_collection_updated_at
            BEFORE UPDATE ON discount_code_collection
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =====================================================
-- 4. cart_applied_coupon — reference-only coupon attach
-- =====================================================

CREATE TABLE IF NOT EXISTS cart_applied_coupon (
    id BIGSERIAL PRIMARY KEY,
    cart_id BIGINT NOT NULL REFERENCES cart(id) ON DELETE CASCADE,
    discount_code_id BIGINT NOT NULL REFERENCES discount_code(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_cart_applied_coupon UNIQUE (cart_id, discount_code_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_applied_coupon_cart_id
    ON cart_applied_coupon (cart_id);

CREATE INDEX IF NOT EXISTS idx_cart_applied_coupon_discount_code_id
    ON cart_applied_coupon (discount_code_id);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_cart_applied_coupon_updated_at') THEN
        CREATE TRIGGER update_cart_applied_coupon_updated_at
            BEFORE UPDATE ON cart_applied_coupon
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =====================================================
-- 5. cart_item_promotion — reference-only promo attach per line
-- =====================================================

CREATE TABLE IF NOT EXISTS cart_item_promotion (
    id BIGSERIAL PRIMARY KEY,
    cart_item_id BIGINT NOT NULL REFERENCES cart_item(id) ON DELETE CASCADE,
    promotion_id BIGINT NOT NULL REFERENCES promotion(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_cart_item_promotion UNIQUE (cart_item_id, promotion_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_item_promotion_cart_item_id
    ON cart_item_promotion (cart_item_id);

CREATE INDEX IF NOT EXISTS idx_cart_item_promotion_promotion_id
    ON cart_item_promotion (promotion_id);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_cart_item_promotion_updated_at') THEN
        CREATE TRIGGER update_cart_item_promotion_updated_at
            BEFORE UPDATE ON cart_item_promotion
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
