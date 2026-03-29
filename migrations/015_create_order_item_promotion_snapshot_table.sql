-- Migration: 015_create_order_item_promotion_snapshot_table.sql
-- Description: Snapshot table for item-level applied promotions on orders
-- Notes: Preserves exact promotion allocation per order item for reporting/audits

CREATE TABLE IF NOT EXISTS order_item_applied_promotion (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    order_item_id BIGINT NOT NULL REFERENCES order_item(id) ON DELETE CASCADE,
    promotion_id BIGINT REFERENCES promotion(id) ON DELETE SET NULL,
    promotion_name VARCHAR(255) NOT NULL,
    promotion_type VARCHAR(50) NOT NULL,
    discount_cents BIGINT NOT NULL DEFAULT 0,
    original_cents BIGINT NOT NULL DEFAULT 0,
    final_cents BIGINT NOT NULL DEFAULT 0,
    free_quantity INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_item_applied_promotion_order_id
    ON order_item_applied_promotion(order_id);
CREATE INDEX IF NOT EXISTS idx_order_item_applied_promotion_order_item_id
    ON order_item_applied_promotion(order_item_id);
CREATE INDEX IF NOT EXISTS idx_order_item_applied_promotion_promotion_id
    ON order_item_applied_promotion(promotion_id);
CREATE INDEX IF NOT EXISTS idx_order_item_applied_promotion_promotion_type
    ON order_item_applied_promotion(promotion_type);

