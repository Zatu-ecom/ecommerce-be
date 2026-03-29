-- Migration: 017_add_fulfillment_type_to_order.sql
-- Description: Add fulfillment type column to order table for order placement flow

ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS fulfillment_type VARCHAR(32) NOT NULL DEFAULT 'directship';

CREATE INDEX IF NOT EXISTS idx_order_fulfillment_type ON "order"(fulfillment_type);

