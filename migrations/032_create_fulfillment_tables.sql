-- ============================================================================
-- Fulfillment Module - Database Migration
-- Version: 1.0 (013-courier-fulfillment-platform)
--
-- Provider-agnostic courier tables (aggregators AND direct couriers):
--   - courier_provider catalog WITHOUT base_url (lives in adapters), WITH
--     capability flags (UI gates) and kill-switch
--   - courier_provider_field for dynamic credential forms (no migration per
--     provider)
--   - courier_provider_config hybrid creds (seller_id NULL = platform
--     default) WITH tenant automation flags (auto_book, rate_preference,
--     default_weight_grams)
--   - fulfillment_shipment one row per box, provider-generic names (no sr_*),
--     cross-module ids as plain columns (NO FK into other modules' tables),
--     pickup nickname derived at runtime (no stored alias, no mapping table)
--   - fulfillment_shipment_item split lines with Σ-guard enforced in service
--   - fulfillment_shipment_event immutable ledger (no updated_at)
--   - fulfillment_webhook_log verified-only audit + unconditional unique
--     (provider_code, event_id) idempotency
--   - fulfillment_ndr one open round per shipment (partial unique index)
-- ============================================================================

-- ============================================================================
-- 1. COURIER PROVIDER - Catalog of all supported couriers
-- ============================================================================
CREATE TABLE IF NOT EXISTS courier_provider (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    kind VARCHAR(20) NOT NULL,  -- 'aggregator', 'direct'

    -- Capability flags (UI gates; direct couriers often lack NDR).
    supports_pickup BOOLEAN NOT NULL DEFAULT TRUE,
    supports_ndr BOOLEAN NOT NULL DEFAULT TRUE,
    supports_return BOOLEAN NOT NULL DEFAULT TRUE,
    supports_cod BOOLEAN NOT NULL DEFAULT TRUE,
    webhook_supported BOOLEAN NOT NULL DEFAULT TRUE,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- UNIQUE(code) is enough; no second index on code.

-- ============================================================================
-- 2. COURIER PROVIDER FIELD - Credential form definitions per provider
-- ============================================================================
CREATE TABLE IF NOT EXISTS courier_provider_field (
    id BIGSERIAL PRIMARY KEY,
    provider_code VARCHAR(50) NOT NULL REFERENCES courier_provider(code) ON DELETE CASCADE,

    field_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) NOT NULL,

    field_type VARCHAR(50) NOT NULL,  -- 'string', 'password', 'url', 'email', 'number', 'boolean'
    description TEXT,
    placeholder VARCHAR(200),

    is_required BOOLEAN NOT NULL DEFAULT TRUE,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,

    display_order INT NOT NULL DEFAULT 0,

    validation_rules JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider_code, field_name)
);

CREATE INDEX IF NOT EXISTS idx_courier_provider_field_provider
    ON courier_provider_field(provider_code);

-- ============================================================================
-- 3. COURIER PROVIDER CONFIG - Hybrid credentials (platform default + seller
--    override) with tenant automation flags
-- ============================================================================
CREATE TABLE IF NOT EXISTS courier_provider_config (
    id BIGSERIAL PRIMARY KEY,
    -- NULL = platform default. FK keeps referential integrity for seller rows.
    seller_id BIGINT REFERENCES seller_profile(user_id) ON DELETE CASCADE,
    provider_code VARCHAR(50) NOT NULL REFERENCES courier_provider(code) ON DELETE CASCADE,

    environment VARCHAR(20) NOT NULL DEFAULT 'production',  -- 'sandbox', 'production'

    -- Encrypted credentials (JSONB, shape validated against provider fields).
    credentials JSONB NOT NULL,

    -- Tenant automation flags.
    auto_book BOOLEAN NOT NULL DEFAULT FALSE,  -- plan-only drafts vs plan + book + pickup
    rate_preference VARCHAR(20),  -- 'cheapest', 'fastest'; used only when auto_book is TRUE
    default_weight_grams INT CHECK (default_weight_grams IS NULL OR default_weight_grams > 0),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(seller_id, provider_code, environment)
);

-- Plain UNIQUE allows multiple NULL seller rows in PG; the partial index
-- below guarantees exactly one platform default per (provider, environment).
CREATE UNIQUE INDEX IF NOT EXISTS uq_courier_config_platform_default
    ON courier_provider_config (provider_code, environment)
    WHERE seller_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_courier_config_seller_provider_active
    ON courier_provider_config(seller_id, provider_code, is_active);

-- ============================================================================
-- 4. FULFILLMENT SHIPMENT - One row per box (split shipments are N rows)
-- ============================================================================
CREATE TABLE IF NOT EXISTS fulfillment_shipment (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES "order"(id) ON DELETE RESTRICT,
    seller_id BIGINT NOT NULL REFERENCES seller_profile(user_id),

    -- NULL in draft (drafts are pre-provider); set at book. No DEFAULT.
    provider_code VARCHAR(50) REFERENCES courier_provider(code),
    -- NULL until book; then the frozen config row (creds + env + flags).
    provider_config_id BIGINT REFERENCES courier_provider_config(id),

    -- NULL in draft. Value sent to the courier is this shipment's id.
    provider_order_id TEXT,
    provider_shipment_id TEXT,
    -- NULL until the AWB step. Webhook locator.
    awb TEXT,
    -- Client retry key (Idempotency-Key header).
    idempotency_key TEXT,

    -- Late-bound by aggregators (e.g. 'Delhivery Surface').
    courier_name VARCHAR(100),
    service_code VARCHAR(50),

    -- Single status vocabulary (data-model §2a allow-list enforced in service).
    status VARCHAR(32) NOT NULL DEFAULT 'draft',

    -- Logical cross-module references (NO FK: owning modules are resolved
    -- via service hooks; the shipment stores ids only, never row copies).
    pickup_location_id BIGINT,
    delivery_address_id BIGINT,
    -- Address revision stamp (from order_address.updated_at at plan time).
    delivery_address_revised_at TIMESTAMPTZ,

    weight_grams INT CHECK (weight_grams IS NULL OR weight_grams > 0),
    length_cm NUMERIC(8, 2),
    breadth_cm NUMERIC(8, 2),
    height_cm NUMERIC(8, 2),

    -- Collect-amount snapshot. 0 = prepaid. Not a remittance record.
    cod_cents BIGINT NOT NULL DEFAULT 0,
    -- Quoted freight at ship time (audit).
    rate_cents BIGINT,
    insured BOOLEAN NOT NULL DEFAULT FALSE,

    etd TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    -- Last successful provider poll or applied webhook. Not updated_at.
    last_synced_at TIMESTAMPTZ,

    -- First book-attempt marker + attempt counter (recover_drafts).
    book_requested_at TIMESTAMPTZ,
    book_attempts INT NOT NULL DEFAULT 0,

    -- Return boxes point at the original. No separate return-status table.
    return_of_shipment_id BIGINT REFERENCES fulfillment_shipment(id) ON DELETE CASCADE,

    -- Sanitized provider snapshot, no PII.
    raw_ref JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider_code, provider_order_id),
    UNIQUE(provider_code, awb),
    UNIQUE(idempotency_key),
    CHECK (return_of_shipment_id IS NULL OR return_of_shipment_id <> id)
);

CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_seller_created
    ON fulfillment_shipment(seller_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_seller_status
    ON fulfillment_shipment(seller_id, status);
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_order_id
    ON fulfillment_shipment(order_id);
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_pickup_location
    ON fulfillment_shipment(pickup_location_id);
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_status_synced
    ON fulfillment_shipment(status, last_synced_at);
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_recover
    ON fulfillment_shipment(status, book_requested_at) WHERE status = 'draft';

-- No standalone (awb) index: the unique keys already cover AWB lookup.

-- ============================================================================
-- 5. FULFILLMENT SHIPMENT ITEM - Split lines (Σ qty guard lives in service)
-- ============================================================================
CREATE TABLE IF NOT EXISTS fulfillment_shipment_item (
    id BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT NOT NULL REFERENCES fulfillment_shipment(id) ON DELETE CASCADE,
    order_item_id BIGINT NOT NULL REFERENCES order_item(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(shipment_id, order_item_id)
);

-- Unique pair covers (shipment_id, order_item_id); extra index for the sum check.
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_item_order_item
    ON fulfillment_shipment_item(order_item_id);

-- ============================================================================
-- 6. FULFILLMENT SHIPMENT EVENT - Immutable append-only ledger
-- ============================================================================
CREATE TABLE IF NOT EXISTS fulfillment_shipment_event (
    id BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT NOT NULL REFERENCES fulfillment_shipment(id) ON DELETE CASCADE,

    -- Same vocabulary as status, or 'ignore'. Raw courier text goes only in
    -- provider_event.
    event_type VARCHAR(50) NOT NULL,
    from_status VARCHAR(32),
    to_status VARCHAR(32) NOT NULL,

    provider_event VARCHAR(100),
    provider_request JSONB,
    provider_response JSONB,

    failure_code VARCHAR(100),
    failure_message TEXT,

    source VARCHAR(20) NOT NULL,  -- 'api', 'webhook', 'system', 'admin'
    actor_id BIGINT,
    actor_type VARCHAR(20),  -- 'seller', 'customer', 'admin', 'system'

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_event_shipment_created
    ON fulfillment_shipment_event(shipment_id, created_at DESC);

-- ============================================================================
-- 7. FULFILLMENT WEBHOOK LOG - Verified-only audit + idempotency
-- ============================================================================
CREATE TABLE IF NOT EXISTS fulfillment_webhook_log (
    id BIGSERIAL PRIMARY KEY,

    provider_code VARCHAR(50) NOT NULL REFERENCES courier_provider(code),
    -- Adapter-generated idempotency key, never empty.
    event_id TEXT NOT NULL,
    awb TEXT,

    -- Same vocabulary as status, or 'ignore'.
    action VARCHAR(50) NOT NULL,
    -- 'received', 'applied', 'failed', 'ignored'.
    status VARCHAR(30) NOT NULL,
    error_message TEXT,

    -- Sanitized: phones and addresses stripped. Secrets removed from headers.
    payload JSONB NOT NULL,
    headers JSONB,

    shipment_id BIGINT REFERENCES fulfillment_shipment(id),

    ip_address VARCHAR(50),
    processed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider_code, event_id)
);

-- Heal-cron index. No extra indexes on provider_code or event_id alone.
CREATE INDEX IF NOT EXISTS idx_fulfillment_webhook_log_status_created
    ON fulfillment_webhook_log(status, created_at);

-- ============================================================================
-- 8. FULFILLMENT NDR - One open round per shipment
-- ============================================================================
CREATE TABLE IF NOT EXISTS fulfillment_ndr (
    id BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT NOT NULL REFERENCES fulfillment_shipment(id) ON DELETE CASCADE,
    awb TEXT NOT NULL,

    attempt_no INT NOT NULL CHECK (attempt_no > 0),
    -- Reason label, NOT a uniqueness key: a repeat reason is a new round.
    ndr_status VARCHAR(50) NOT NULL,
    reason TEXT,

    -- 'reattempt', 'rto'. NULL = round still open.
    action_taken VARCHAR(30),
    acted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(shipment_id, attempt_no)
);

-- Exactly one open round per shipment.
CREATE UNIQUE INDEX IF NOT EXISTS uq_fulfillment_ndr_open_round
    ON fulfillment_ndr (shipment_id) WHERE action_taken IS NULL;

CREATE INDEX IF NOT EXISTS idx_fulfillment_ndr_shipment_id
    ON fulfillment_ndr(shipment_id);

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE courier_provider IS 'Catalog of supported couriers (aggregators and direct); new courier = 1 seed row';
COMMENT ON TABLE courier_provider_field IS 'Dynamic credential form per courier; new provider needs seed rows, never DDL';
COMMENT ON TABLE courier_provider_config IS 'Hybrid credentials: seller_id NULL = platform default; carries tenant automation flags';
COMMENT ON TABLE fulfillment_shipment IS 'One row per shippable box; an order with N boxes = N rows';
COMMENT ON TABLE fulfillment_shipment_item IS 'Split lines; service guards Σ quantity <= order_item.quantity';
COMMENT ON TABLE fulfillment_shipment_event IS 'Immutable ledger of every transition; never updated or deleted';
COMMENT ON TABLE fulfillment_webhook_log IS 'Verified-only webhook audit; (provider_code, event_id) is the replay guard';
COMMENT ON TABLE fulfillment_ndr IS 'NDR rounds; partial unique index enforces one open round per shipment';

COMMENT ON COLUMN courier_provider_config.credentials IS 'AES-256-GCM encrypted credential map; shape validated against courier_provider_field';
COMMENT ON COLUMN fulfillment_shipment.pickup_location_id IS 'Logical ref to inventory location (no FK); resolved via service hook';
COMMENT ON COLUMN fulfillment_shipment.delivery_address_id IS 'Logical ref to order_address (no FK); resolved via service hook';
COMMENT ON COLUMN fulfillment_shipment.provider_order_id IS 'Value sent to the courier is this shipment id, not the order id';
COMMENT ON COLUMN fulfillment_shipment.cod_cents IS 'Collect-amount snapshot; 0 = prepaid; not a remittance record';
