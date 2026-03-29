-- Migration: 016_add_cart_status_and_order_history.sql
-- Description: Add cart lifecycle fields and create order history audit table

-- Add cart lifecycle fields
ALTER TABLE cart
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

ALTER TABLE cart
    ADD COLUMN IF NOT EXISTS order_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_cart_order_id'
    ) THEN
        ALTER TABLE cart
            ADD CONSTRAINT fk_cart_order_id
            FOREIGN KEY (order_id) REFERENCES "order"(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Replace old "one cart per user" uniqueness with "one active cart per user"
ALTER TABLE cart DROP CONSTRAINT IF EXISTS uq_cart_user;
DROP INDEX IF EXISTS idx_cart_user_id;
DROP INDEX IF EXISTS idx_cart_user_id_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_user_id_active
    ON cart(user_id) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_cart_status_updated_at
    ON cart(status, updated_at);

CREATE INDEX IF NOT EXISTS idx_cart_order_id
    ON cart(order_id);

-- Order status transition audit log
CREATE TABLE IF NOT EXISTS order_history (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
    from_status VARCHAR(32),
    to_status VARCHAR(32) NOT NULL,
    changed_by_user_id BIGINT,
    changed_by_role VARCHAR(32),
    transaction_id VARCHAR(255),
    failure_reason TEXT,
    note TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_history_order_id ON order_history(order_id);
CREATE INDEX IF NOT EXISTS idx_order_history_created_at ON order_history(created_at);

