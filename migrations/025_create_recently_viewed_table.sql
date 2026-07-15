-- Migration: 025_create_recently_viewed_table.sql
-- Description: Create user_recently_viewed table for tracking recently viewed products
-- Created: 2026-07-15
-- Dependencies: Requires update_updated_at_column() function from migration 003

-- ============================================================================
-- Recently Viewed Table
-- ============================================================================
-- Tracks which products a customer has viewed. One row per (user_id, product_id)
-- pair. Re-viewing a product updates the viewed_at timestamp via ON CONFLICT.
-- Keeps last 10 products per user; older entries are trimmed automatically.

CREATE TABLE IF NOT EXISTS user_recently_viewed (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    seller_id  BIGINT       NOT NULL,
    product_id BIGINT       NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    viewed_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- One row per user-product pair; re-views update via ON CONFLICT
    CONSTRAINT uq_user_recently_viewed_product UNIQUE (user_id, product_id)
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_recently_viewed_user_id
    ON user_recently_viewed (user_id);

CREATE INDEX IF NOT EXISTS idx_recently_viewed_user_viewed
    ON user_recently_viewed (user_id, viewed_at DESC);

CREATE INDEX IF NOT EXISTS idx_recently_viewed_product_id
    ON user_recently_viewed (product_id);

-- Trigger for auto-updating updated_at on row modification
CREATE TRIGGER update_user_recently_viewed_updated_at
    BEFORE UPDATE ON user_recently_viewed
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
