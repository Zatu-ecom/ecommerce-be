-- Migration: 003_alter_timestamps_to_timestamptz.sql
-- Description: Convert all TIMESTAMP columns to TIMESTAMPTZ for proper UTC handling
-- Created: 2025-10-25
-- Reason: Ensure all timestamps are stored and returned in UTC timezone

-- ============================================================================
-- USER SERVICE TABLES
-- ============================================================================

-- Alter user table timestamps
ALTER TABLE "user" 
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter address table timestamps  
ALTER TABLE address
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter subscription table timestamps
ALTER TABLE subscription
    ALTER COLUMN start_date TYPE TIMESTAMPTZ USING start_date AT TIME ZONE 'UTC',
    ALTER COLUMN end_date TYPE TIMESTAMPTZ USING end_date AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- PRODUCT SERVICE TABLES
-- ============================================================================

-- Alter category table timestamps
ALTER TABLE category
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter attribute_definition table timestamps
ALTER TABLE attribute_definition
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter category_attribute table timestamps
ALTER TABLE category_attribute
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter product table timestamps
ALTER TABLE product
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter product_attribute table timestamps
ALTER TABLE product_attribute
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter product_variant table timestamps
ALTER TABLE product_variant
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter package_option table timestamps
ALTER TABLE package_option
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Alter product_option table timestamps
ALTER TABLE product_option
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- NOTES
-- ============================================================================
-- TIMESTAMPTZ stores all timestamps in UTC internally
-- When inserting: automatically converts from any timezone to UTC
-- When selecting: returns in UTC by default (or converts to session timezone if set)
-- This ensures consistent UTC handling across the entire application
