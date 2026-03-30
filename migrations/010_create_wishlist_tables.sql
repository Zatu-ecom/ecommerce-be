-- Migration: 010_create_wishlist_tables.sql
-- Description: Create wishlist and wishlist_item tables for user wishlists
-- Created: 2026-01-25

-- ============================================================================
-- Wishlist Table
-- ============================================================================
-- Stores user wishlists. Each user can have multiple wishlists with one default.

CREATE TABLE IF NOT EXISTS wishlist (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT 'default',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on wishlist
CREATE INDEX IF NOT EXISTS idx_wishlist_user_id ON wishlist(user_id);
CREATE INDEX IF NOT EXISTS idx_wishlist_user_id_is_default ON wishlist(user_id, is_default);

-- ============================================================================
-- Wishlist Item Table
-- ============================================================================
-- Stores items (variants) in a wishlist. Links wishlist to product variants.

CREATE TABLE IF NOT EXISTS wishlist_item (
    id BIGSERIAL PRIMARY KEY,
    wishlist_id BIGINT NOT NULL REFERENCES wishlist(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variant(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(wishlist_id, variant_id)
);

-- Create indexes on wishlist_item
CREATE INDEX IF NOT EXISTS idx_wishlist_item_wishlist_id ON wishlist_item(wishlist_id);
CREATE INDEX IF NOT EXISTS idx_wishlist_item_variant_id ON wishlist_item(variant_id);

-- Composite index for efficient wishlist check queries (isWishlisted feature)
CREATE INDEX IF NOT EXISTS idx_wishlist_item_variant_wishlist ON wishlist_item(variant_id, wishlist_id);
