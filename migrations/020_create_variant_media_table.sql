-- Migration 020: Replace product_variants.images TEXT[] with a normalised
-- variant_media join table so all binary assets are managed by the File module.

-- 1. Drop the raw URL column from product_variants.
ALTER TABLE product_variant
    DROP COLUMN IF EXISTS images;

-- 2. Create the variant_media association table.
CREATE TABLE IF NOT EXISTS variant_media (
    id            BIGSERIAL    PRIMARY KEY,
    variant_id    BIGINT       NOT NULL
                               REFERENCES product_variant(id) ON DELETE CASCADE,
    file_id       TEXT         NOT NULL,
    is_primary    BOOLEAN      NOT NULL DEFAULT false,
    display_order INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_variant_media UNIQUE (variant_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_variant_media_variant_id
    ON variant_media (variant_id);
