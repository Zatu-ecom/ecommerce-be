-- Migration: 024_widen_order_item_image_url.sql
-- Description: Widen order_item.image_url for long storage object paths

ALTER TABLE order_item ALTER COLUMN image_url TYPE TEXT;
