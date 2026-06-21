-- Migration: 023_file_id_references.sql
-- Description: Replace raw URL storage with file_id references across modules
-- Created: 2026-06-14

-- Collection: single image file reference
ALTER TABLE collection ADD COLUMN IF NOT EXISTS image_file_id VARCHAR(80);
ALTER TABLE collection DROP COLUMN IF EXISTS image;

-- Sale: banner file ID array replaces URL array
ALTER TABLE sale ADD COLUMN IF NOT EXISTS banner_file_ids TEXT[] DEFAULT '{}';
ALTER TABLE sale DROP COLUMN IF EXISTS banner_images;

-- Seller profile: business logo file reference
ALTER TABLE seller_profile ADD COLUMN IF NOT EXISTS business_logo_file_id VARCHAR(80);
ALTER TABLE seller_profile DROP COLUMN IF EXISTS business_logo;

-- Order item: optional audit link to source file (image_url snapshot retained)
ALTER TABLE order_item ADD COLUMN IF NOT EXISTS image_file_id VARCHAR(80);
