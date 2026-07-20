-- Migration: 027_guest_cart_support.sql
-- Description: Add device_id to cart for guest cart support.
-- user_id becomes nullable; device_id added as alternative owner.
-- Enforces that exactly one of user_id or device_id is set (CHECK constraint).

-- Step 1: Make user_id nullable (guest carts have no user)
ALTER TABLE cart ALTER COLUMN user_id DROP NOT NULL;

-- Step 2: Add device_id for guest cart identification
ALTER TABLE cart ADD COLUMN IF NOT EXISTS device_id VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_cart_device_id ON cart(device_id);

-- Step 3: Enforce exactly-one-owner constraint
-- Uses idempotent DO-block pattern (see 016_add_cart_status_and_order_history.sql)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_cart_owner'
    ) THEN
        ALTER TABLE cart ADD CONSTRAINT chk_cart_owner
            CHECK (user_id IS NOT NULL OR device_id IS NOT NULL);
    END IF;
END $$;

-- Step 4: Replace active-cart uniqueness
-- Old: one active cart per user_id
DROP INDEX IF EXISTS idx_cart_user_id_active;

-- New: one active cart per user (when user_id is set)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_user_id_active
    ON cart(user_id) WHERE status = 'active' AND user_id IS NOT NULL;

-- New: one active cart per device (when device_id is set)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_device_id_active
    ON cart(device_id) WHERE status = 'active' AND device_id IS NOT NULL;
