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

CREATE TABLE IF NOT EXISTS storage_config (
    id BIGSERIAL PRIMARY KEY,
    owner_type VARCHAR(20) NOT NULL CHECK (owner_type IN ('PLATFORM', 'SELLER')),
    owner_id BIGINT,
    provider_id BIGINT NOT NULL REFERENCES storage_provider(id),
    display_name VARCHAR(150) NOT NULL,

    bucket_or_container VARCHAR(255) NOT NULL,
    region VARCHAR(100),
    endpoint VARCHAR(500),
    base_path VARCHAR(500),
    force_path_style BOOLEAN NOT NULL DEFAULT false,

    credentials_encrypted BYTEA NOT NULL,
    config_json JSONB,

    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_validated_at TIMESTAMPTZ,
    validation_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

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
