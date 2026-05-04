-- Migration: 018_create_file_storage_tables.sql
-- Description: Create file module storage provider/config/binding tables

CREATE TABLE IF NOT EXISTS storage_provider (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    adapter_type VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- storage_config matches file/entity.StorageConfig:
-- bucket_or_container + provider_id identify routing; all provider-specific settings
-- (credentials, region, endpoint, etc.) live in config_data as JSON (field-level AES in app).
CREATE TABLE IF NOT EXISTS storage_config (
    id BIGSERIAL PRIMARY KEY,
    owner_type VARCHAR(20) NOT NULL CHECK (owner_type IN ('PLATFORM', 'SELLER')),
    owner_id BIGINT,
    provider_id BIGINT NOT NULL REFERENCES storage_provider(id),
    display_name VARCHAR(150) NOT NULL,

    bucket_or_container VARCHAR(255) NOT NULL,
    config_data JSONB NOT NULL,

    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (
        (owner_type = 'PLATFORM' AND owner_id IS NULL)
        OR
        (owner_type = 'SELLER' AND owner_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_storage_config_owner ON storage_config(owner_type, owner_id);
CREATE INDEX IF NOT EXISTS idx_storage_config_provider_id ON storage_config(provider_id);
CREATE INDEX IF NOT EXISTS idx_storage_config_is_active ON storage_config(is_active);

-- Enforce a single platform default config.
CREATE UNIQUE INDEX IF NOT EXISTS uq_storage_config_platform_default
    ON storage_config(is_default)
    WHERE owner_type = 'PLATFORM' AND is_default = true;

CREATE TABLE IF NOT EXISTS seller_storage_binding (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES "user"(id),
    storage_config_id BIGINT NOT NULL REFERENCES storage_config(id),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seller_storage_binding_storage_config_id
    ON seller_storage_binding(storage_config_id);

-- One active binding per seller.
CREATE UNIQUE INDEX IF NOT EXISTS uq_seller_storage_binding_active
    ON seller_storage_binding(seller_id)
    WHERE is_active = true;

-- ─── File Object Tables (added by feature 003-upload-apis in place; migration not yet on develop) ───

-- file_object: canonical registry row for a user-uploaded file.
-- No metadata column (removed per clarification 2026-04-18).
CREATE TABLE IF NOT EXISTS file_object (
    id                  BIGSERIAL PRIMARY KEY,
    file_id             VARCHAR(80)  NOT NULL,
    seller_id           BIGINT,
    uploader_user_id    BIGINT       NOT NULL,
    owner_type          VARCHAR(20)  NOT NULL,
    owner_id            BIGINT,
    purpose             VARCHAR(40)  NOT NULL,
    visibility          VARCHAR(20)  NOT NULL DEFAULT 'PRIVATE',
    storage_config_id   BIGINT       NOT NULL REFERENCES storage_config(id),
    bucket_or_container VARCHAR(255) NOT NULL,
    object_key          VARCHAR(1000) NOT NULL,
    original_filename   VARCHAR(255) NOT NULL,
    sanitized_filename  VARCHAR(255) NOT NULL,
    mime_type           VARCHAR(150) NOT NULL,
    size_bytes          BIGINT       NOT NULL,
    etag                VARCHAR(200),
    status              VARCHAR(20)  NOT NULL DEFAULT 'UPLOADING',
    failure_reason      VARCHAR(150),
    upload_expires_at   TIMESTAMPTZ  NOT NULL,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_file_object_owner_type CHECK (owner_type IN ('SELLER', 'PLATFORM')),
    CONSTRAINT chk_file_object_status     CHECK (status IN ('UPLOADING', 'ACTIVE', 'FAILED')),
    CONSTRAINT chk_file_object_visibility CHECK (visibility IN ('PRIVATE', 'PUBLIC', 'INTERNAL')),
    -- Seller rows must have both owner_id and seller_id; platform rows must have neither.
    CONSTRAINT chk_file_object_owner_consistency CHECK (
        (owner_type = 'SELLER'   AND owner_id IS NOT NULL AND seller_id IS NOT NULL)
        OR
        (owner_type = 'PLATFORM' AND owner_id IS NULL     AND seller_id IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_file_object_file_id  ON file_object(file_id);
CREATE INDEX        IF NOT EXISTS idx_file_object_seller   ON file_object(seller_id, created_at DESC);
CREATE INDEX        IF NOT EXISTS idx_file_object_owner    ON file_object(owner_type, owner_id);
-- Supports sweeper fallback if Redis scheduler is lost.
CREATE INDEX        IF NOT EXISTS idx_file_object_expiry   ON file_object(status, upload_expires_at);
CREATE INDEX        IF NOT EXISTS idx_file_object_purpose  ON file_object(purpose, status);
-- Guards against accidental object-key collisions across configs.
CREATE UNIQUE INDEX IF NOT EXISTS uq_file_object_storage_key ON file_object(storage_config_id, object_key);

-- file_variant: derived files (thumbnails, webp).
-- Rows are created by the variant-worker feature; this feature only declares the table.
-- No metadata column.
CREATE TABLE IF NOT EXISTS file_variant (
    id                  BIGSERIAL PRIMARY KEY,
    file_object_id      BIGINT       NOT NULL REFERENCES file_object(id) ON DELETE CASCADE,
    variant_code        VARCHAR(40)  NOT NULL,
    mime_type           VARCHAR(150) NOT NULL,
    bucket_or_container VARCHAR(255) NOT NULL,
    object_key          VARCHAR(1000) NOT NULL,
    size_bytes          BIGINT       NOT NULL,
    width               INT,
    height              INT,
    status              VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX        IF NOT EXISTS idx_file_variant_file_object ON file_variant(file_object_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_file_variant_unique       ON file_variant(file_object_id, variant_code);

-- file_job: tracks async publish of file.image.process.requested command.
-- Written by complete-upload when HasVariants=true.
CREATE TABLE IF NOT EXISTS file_job (
    id              BIGSERIAL PRIMARY KEY,
    file_object_id  BIGINT       NOT NULL REFERENCES file_object(id) ON DELETE CASCADE,
    command         VARCHAR(60)  NOT NULL,
    status          VARCHAR(20)  NOT NULL,
    attempts        INT          NOT NULL DEFAULT 0,
    last_error      VARCHAR(300),
    correlation_id  VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_file_job_status CHECK (status IN ('PUBLISHED', 'FAILED_TO_PUBLISH', 'DONE'))
);

CREATE INDEX IF NOT EXISTS idx_file_job_file_object     ON file_job(file_object_id);
CREATE INDEX IF NOT EXISTS idx_file_job_command_status  ON file_job(command, status);
