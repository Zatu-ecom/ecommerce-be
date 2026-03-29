-- Migration: 014_create_order_discount_snapshot_tables.sql
-- Description: Snapshot tables for order-level applied promotions and coupons
-- Notes: These tables preserve immutable discount context for reporting and audits

CREATE TABLE IF NOT EXISTS order_applied_promotion (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    promotion_id BIGINT REFERENCES promotion(id) ON DELETE SET NULL,
    promotion_name VARCHAR(255) NOT NULL,
    promotion_type VARCHAR(50) NOT NULL,
    discount_cents BIGINT NOT NULL DEFAULT 0,
    shipping_discount_cents BIGINT NOT NULL DEFAULT 0,
    is_stackable BOOLEAN,
    priority INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_applied_promotion_order_id
    ON order_applied_promotion(order_id);
CREATE INDEX IF NOT EXISTS idx_order_applied_promotion_promotion_id
    ON order_applied_promotion(promotion_id);
CREATE INDEX IF NOT EXISTS idx_order_applied_promotion_promotion_type
    ON order_applied_promotion(promotion_type);

CREATE TABLE IF NOT EXISTS order_applied_coupon (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    discount_code_id BIGINT REFERENCES discount_code(id) ON DELETE SET NULL,
    coupon_code VARCHAR(100) NOT NULL,
    coupon_title VARCHAR(255),
    discount_type VARCHAR(50),
    discount_value BIGINT,
    discount_cents BIGINT NOT NULL DEFAULT 0,
    shipping_discount_cents BIGINT NOT NULL DEFAULT 0,
    is_combinable BOOLEAN,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_applied_coupon_order_id
    ON order_applied_coupon(order_id);
CREATE INDEX IF NOT EXISTS idx_order_applied_coupon_discount_code_id
    ON order_applied_coupon(discount_code_id);
CREATE INDEX IF NOT EXISTS idx_order_applied_coupon_coupon_code
    ON order_applied_coupon(coupon_code);

