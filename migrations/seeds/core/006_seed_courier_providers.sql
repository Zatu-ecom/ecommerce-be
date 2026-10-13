-- Seed: 006_seed_courier_providers.sql
-- Description: Core courier provider catalog for the fulfillment module
-- Environment: ALL (core data) — the provider catalog must exist whenever the backend runs
-- Notes: Idempotent upserts keyed on natural unique columns.
--   013-courier-fulfillment-platform: provider-agnostic catalog; Shiprocket
--   first adapter. No per-provider DDL is ever needed (creds are JSONB,
--   forms come from courier_provider_field rows below).

-- ============================================================================
-- 1. COURIER PROVIDER - Shiprocket (aggregator)
-- ============================================================================
INSERT INTO courier_provider (
    code, name, kind,
    supports_pickup, supports_ndr, supports_return, supports_cod, webhook_supported,
    is_active, created_at, updated_at
)
VALUES (
    'shiprocket',
    'Shiprocket',
    'aggregator',
    TRUE, TRUE, TRUE, TRUE, TRUE,
    TRUE,
    NOW(), NOW()
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    kind = EXCLUDED.kind,
    supports_pickup = EXCLUDED.supports_pickup,
    supports_ndr = EXCLUDED.supports_ndr,
    supports_return = EXCLUDED.supports_return,
    supports_cod = EXCLUDED.supports_cod,
    webhook_supported = EXCLUDED.webhook_supported,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

-- ============================================================================
-- 2. COURIER PROVIDER FIELD - Shiprocket credential form
-- ============================================================================
INSERT INTO courier_provider_field (
    provider_code, field_name, display_name, field_type, description, placeholder,
    is_required, is_sensitive, display_order, validation_rules,
    created_at, updated_at
)
VALUES
(
    'shiprocket', 'api_email', 'API email', 'email',
    'Shiprocket API user email (Settings > API > API user; differs from the panel login).',
    'api-user@shop.com',
    TRUE, FALSE, 1, NULL,
    NOW(), NOW()
),
(
    'shiprocket', 'api_password', 'API password', 'password',
    'Password emailed when the API user was created. Stored encrypted; never displayed.',
    '••••••••',
    TRUE, TRUE, 2, NULL,
    NOW(), NOW()
),
(
    'shiprocket', 'webhook_secret', 'Webhook secret', 'password',
    'Optional security token configured at Settings > API > Webhooks; verified on every push.',
    '••••••••',
    FALSE, TRUE, 3, NULL,
    NOW(), NOW()
)
ON CONFLICT (provider_code, field_name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    field_type = EXCLUDED.field_type,
    description = EXCLUDED.description,
    placeholder = EXCLUDED.placeholder,
    is_required = EXCLUDED.is_required,
    is_sensitive = EXCLUDED.is_sensitive,
    display_order = EXCLUDED.display_order,
    validation_rules = EXCLUDED.validation_rules,
    updated_at = NOW();

-- ------------------------------
-- Summary
-- ------------------------------
DO $$
BEGIN
    RAISE NOTICE 'Courier provider catalog seeded: % provider(s), % field(s).',
        (SELECT COUNT(*) FROM courier_provider),
        (SELECT COUNT(*) FROM courier_provider_field);
END $$;
