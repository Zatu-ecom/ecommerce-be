-- Seed: 007_seed_oracle_cloud_storage_provider.sql
-- Description: Oracle Cloud Infrastructure Object Storage (S3-compatible) provider entry
-- Environment: ALL (core data)
--
-- Configure with the OCI S3-compatible endpoint:
--   https://<namespace>.compat.objectstorage.<region>.oci.customer-oci.com
-- and force_path_style=true.

INSERT INTO storage_provider (code, name, adapter_type, is_active, created_at, updated_at)
VALUES
    ('oci', 'Oracle Cloud Object Storage', 's3_compatible', true, NOW(), NOW())
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    adapter_type = EXCLUDED.adapter_type,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
