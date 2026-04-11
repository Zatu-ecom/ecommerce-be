-- Seed: 003_seed_storage_providers.sql
-- Description: Core storage provider master data for file module
-- Environment: ALL (core data)

INSERT INTO storage_provider (code, name, adapter_type, is_active, created_at, updated_at)
VALUES
    ('aws_s3', 'AWS S3', 's3_compatible', true, NOW(), NOW()),
    ('gcs', 'Google Cloud Storage', 'gcs', true, NOW(), NOW()),
    ('azure_blob', 'Azure Blob Storage', 'azure', true, NOW(), NOW()),
    ('r2', 'Cloudflare R2', 's3_compatible', true, NOW(), NOW()),
    ('minio', 'MinIO', 's3_compatible', true, NOW(), NOW()),
    ('b2', 'Backblaze B2', 's3_compatible', true, NOW(), NOW())
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    adapter_type = EXCLUDED.adapter_type,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
