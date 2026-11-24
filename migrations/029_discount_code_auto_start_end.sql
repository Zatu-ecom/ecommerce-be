-- Migration: 029_discount_code_auto_start_end.sql
-- Description: Add auto_start / auto_end to discount_code for promotion-parity cron sweeps.

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS auto_start BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE discount_code
    ADD COLUMN IF NOT EXISTS auto_end BOOLEAN NOT NULL DEFAULT TRUE;
