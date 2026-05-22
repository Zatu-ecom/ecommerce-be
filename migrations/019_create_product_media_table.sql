-- Migration: 019_create_product_media_table.sql
-- Description: Create product_media association table linking products to file-module assets

CREATE TABLE IF NOT EXISTS product_media (
    id            BIGSERIAL    PRIMARY KEY,
    product_id    BIGINT       NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    file_id       VARCHAR(80)  NOT NULL,
    is_primary    BOOLEAN      NOT NULL DEFAULT false,
    display_order INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX        IF NOT EXISTS idx_product_media_product_id   ON product_media(product_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_media_product_file  ON product_media(product_id, file_id);
