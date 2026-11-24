-- ============================================================================
-- Payment Module - Database Migration
-- Version: 3.0 (010-payment-gateway-platform)
-- Created: 2026-01-10 / Rewritten: 2026-09-06
--
-- Provider platform schema (final CREATE — no follow-up ALTERs):
--   - payment_gateway catalog WITHOUT country/currency arrays, webhook_url, logo_url
--   - payment_gateway_country / payment_gateway_currency FK join tables for geo
--   - payment_gateway_config WITHOUT country (UNIQUE seller, gateway, environment)
--   - payment_transaction WITH reference_*, gateway_session_id, gateway_payment_id,
--     payment_method_details, environment; WITHOUT metadata/initiated_at/
--     gateway_transaction_id; nullable gateway_fee_cents
--   - payment_transaction_event append-only ledger (folded in from 031)
--   - payment_webhook_log with event_id NOT NULL + unconditional unique
--     (gateway_id, event_id)
--   - NO payment_method table (saved cards are out of scope)
--
-- Payment is not in production: recreate dev/test databases after this rewrite.
-- ============================================================================

-- Defensive cleanup for databases created by the pre-platform 009 (arrays,
-- webhook_url, payment_method). No-ops on fresh databases.
DO $$
BEGIN
    IF to_regclass('public.payment_method') IS NOT NULL THEN
        DROP TABLE payment_method;
    END IF;
    IF to_regclass('public.payment_gateway') IS NOT NULL THEN
        ALTER TABLE payment_gateway DROP COLUMN IF EXISTS supported_countries;
        ALTER TABLE payment_gateway DROP COLUMN IF EXISTS supported_currencies;
        ALTER TABLE payment_gateway DROP COLUMN IF EXISTS webhook_url;
        ALTER TABLE payment_gateway DROP COLUMN IF EXISTS logo_url;
    END IF;
END
$$;

-- ============================================================================
-- 1. PAYMENT GATEWAY - Master table for all supported payment gateways
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    logo_file_id VARCHAR(80),
    is_active BOOLEAN DEFAULT TRUE,

    -- Supported payment methods (e.g. card, upi, wallet, netbanking).
    -- Optional filter when the client sends paymentMethodType.
    supported_payment_methods TEXT[] NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_gateway_supported_payment_methods
    ON payment_gateway USING GIN (supported_payment_methods);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_code
    ON payment_gateway(code);

-- ============================================================================
-- 2. PAYMENT GATEWAY COUNTRY - Gateway <-> country geo membership (FK joins)
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway_country (
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id) ON DELETE CASCADE,
    country_id BIGINT NOT NULL REFERENCES country(id),

    UNIQUE(gateway_id, country_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_gateway_country_gateway_id
    ON payment_gateway_country(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_country_country_id
    ON payment_gateway_country(country_id);

-- ============================================================================
-- 3. PAYMENT GATEWAY CURRENCY - Gateway <-> currency geo membership (FK joins)
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway_currency (
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id) ON DELETE CASCADE,
    currency_id BIGINT NOT NULL REFERENCES currency(id),

    UNIQUE(gateway_id, currency_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_gateway_currency_gateway_id
    ON payment_gateway_currency(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_currency_currency_id
    ON payment_gateway_currency(currency_id);

-- ============================================================================
-- 4. PAYMENT GATEWAY FIELD - Configuration fields required by each gateway
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway_field (
    id BIGSERIAL PRIMARY KEY,
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id) ON DELETE CASCADE,

    -- Field identification
    field_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) NOT NULL,

    -- Field properties
    field_type VARCHAR(50) NOT NULL,  -- 'string', 'number', 'boolean', 'url', 'email'
    description TEXT,
    placeholder VARCHAR(200),

    -- Field behavior
    is_required BOOLEAN DEFAULT TRUE,
    is_sensitive BOOLEAN DEFAULT FALSE,

    -- Display order in UI
    display_order INT DEFAULT 0,

    -- Validation rules (JSONB for flexibility)
    validation_rules JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(gateway_id, field_name)
);

CREATE INDEX IF NOT EXISTS idx_payment_gateway_field_gateway_id
    ON payment_gateway_field(gateway_id);

-- ============================================================================
-- 5. PAYMENT GATEWAY CONFIG - Seller's per-environment gateway credentials
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway_config (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES seller_profile(user_id),
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id),

    environment VARCHAR(20) NOT NULL,  -- 'sandbox', 'production'

    -- Encrypted credentials (JSONB)
    credentials JSONB NOT NULL,

    is_active BOOLEAN DEFAULT TRUE,
    priority INT DEFAULT 0,  -- Higher = preferred (for fallback)

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(seller_id, gateway_id, environment)
);

CREATE INDEX IF NOT EXISTS idx_payment_gateway_config_seller_id
    ON payment_gateway_config(seller_id);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_config_gateway_id
    ON payment_gateway_config(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_config_active
    ON payment_gateway_config(seller_id, is_active);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_config_seller_gateway_env
    ON payment_gateway_config(seller_id, gateway_id, environment);

-- ============================================================================
-- 6. PAYMENT TRANSACTION - Core transaction table
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_transaction (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL UNIQUE,

    -- Relationships
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    seller_id BIGINT NOT NULL REFERENCES seller_profile(user_id),
    gateway_id BIGINT REFERENCES payment_gateway(id),  -- NULL for COD

    -- Polymorphic owner (no FK): order today, subscription later
    reference_type VARCHAR(20),  -- 'order', 'subscription'
    reference_id BIGINT,

    -- Gateway references
    gateway_session_id VARCHAR(255),  -- Provider checkout/order/intent id
    gateway_payment_id VARCHAR(255),  -- Provider captured payment/charge id

    -- Amount (in cents for precision)
    currency VARCHAR(3) NOT NULL,
    amount_cents BIGINT NOT NULL,
    gateway_fee_cents BIGINT,  -- Unknown at initiation; set on capture

    -- Status
    status VARCHAR(30) NOT NULL,  -- 'pending', 'completed', 'failed', 'refunded', 'partially_refunded'
    failure_code VARCHAR(100),
    failure_message TEXT,

    -- Payment method type (for quick filtering)
    payment_method_type VARCHAR(50),  -- 'card', 'upi', 'wallet', 'cod', 'bank_transfer'

    -- Provider method snapshot (filled on capture from webhook payload)
    payment_method_details JSONB,

    -- Frozen copy of seller_settings.payments_environment at initiate time.
    -- Refunds, webhooks and reconciliation use THIS, not the current toggle.
    environment VARCHAR(20) NOT NULL,  -- 'sandbox', 'production'

    -- Timestamps
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_transaction_user_id
    ON payment_transaction(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_seller_id
    ON payment_transaction(seller_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_status_created_at
    ON payment_transaction(status, created_at);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_id
    ON payment_transaction(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_reference
    ON payment_transaction(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_session_id
    ON payment_transaction(gateway_session_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_payment_id
    ON payment_transaction(gateway_payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_created_at
    ON payment_transaction(created_at DESC);

-- ============================================================================
-- 7. PAYMENT TRANSACTION EVENT - Append-only history ledger (folded in from 031)
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_transaction_event (
    id               BIGSERIAL PRIMARY KEY,
    transaction_id   BIGINT NOT NULL REFERENCES payment_transaction(id) ON DELETE CASCADE,
    event_type       VARCHAR(50)  NOT NULL,  -- initiated, gateway_session_created,
                                             -- authorized, captured, completed, failed,
                                             -- refund_initiated, refund_completed, refund_failed
    from_status      VARCHAR(30),
    to_status        VARCHAR(30)  NOT NULL,
    gateway_event_id VARCHAR(255),
    gateway_request  JSONB,
    gateway_response JSONB,
    failure_code     VARCHAR(100),
    failure_message  TEXT,
    source           VARCHAR(20)  NOT NULL,  -- api, webhook, system, admin
    actor_id         BIGINT,
    actor_type       VARCHAR(20),            -- customer, seller, admin, system
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_txn_event_transaction_id ON payment_transaction_event(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_txn_event_created_at     ON payment_transaction_event(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_txn_event_type           ON payment_transaction_event(event_type);

-- ============================================================================
-- 8. PAYMENT REFUND - Refund tracking
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_refund (
    id BIGSERIAL PRIMARY KEY,
    refund_id VARCHAR(50) NOT NULL UNIQUE,

    transaction_id BIGINT NOT NULL REFERENCES payment_transaction(id),

    -- Gateway reference
    gateway_refund_id VARCHAR(255),

    -- Amount (in cents)
    currency VARCHAR(3) NOT NULL,
    amount_cents BIGINT NOT NULL,

    -- Status
    status VARCHAR(30) NOT NULL,  -- 'pending', 'processing', 'completed', 'failed'
    failure_reason TEXT,

    -- Reason for refund
    reason VARCHAR(100),  -- 'customer_request', 'order_cancelled', 'defective', 'duplicate'
    notes TEXT,

    -- Who initiated the refund
    initiated_by BIGINT REFERENCES "user"(id),
    initiated_by_type VARCHAR(20),  -- 'customer', 'seller', 'admin', 'system'

    -- Timestamps
    completed_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_refund_transaction_id
    ON payment_refund(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_refund_status
    ON payment_refund(status);
CREATE INDEX IF NOT EXISTS idx_payment_refund_status_created_at
    ON payment_refund(status, created_at);
CREATE INDEX IF NOT EXISTS idx_payment_refund_gateway_refund_id
    ON payment_refund(gateway_refund_id);

-- ============================================================================
-- 9. PAYMENT WEBHOOK LOG - Webhook audit trail
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_webhook_log (
    id BIGSERIAL PRIMARY KEY,

    gateway_id BIGINT REFERENCES payment_gateway(id),
    event_type VARCHAR(100) NOT NULL,  -- Provider event name (audit only)
    -- Adapter-generated idempotency key; always set (composite fallback
    -- {providerEvent}:{entityId}) so the unique index below is effective.
    event_id VARCHAR NOT NULL,

    -- Payload (stored AFTER signature verification only; never unverified bodies)
    payload JSONB NOT NULL,
    headers JSONB,

    -- Processing status
    status VARCHAR(30) NOT NULL,  -- 'received', 'processed', 'failed', 'ignored'
    error_message TEXT,
    processed_at TIMESTAMPTZ,

    -- Linked records
    transaction_id BIGINT REFERENCES payment_transaction(id),
    refund_id BIGINT REFERENCES payment_refund(id),

    ip_address VARCHAR(50),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_webhook_log_gateway_id
    ON payment_webhook_log(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payment_webhook_log_event_id
    ON payment_webhook_log(event_id);
CREATE INDEX IF NOT EXISTS idx_payment_webhook_log_status
    ON payment_webhook_log(status);
CREATE INDEX IF NOT EXISTS idx_payment_webhook_log_created_at
    ON payment_webhook_log(created_at DESC);

-- Webhook idempotency: one processing per (gateway, event). Unconditional
-- (no WHERE event_id IS NOT NULL) because event_id is NOT NULL.
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_webhook_log_gateway_event
    ON payment_webhook_log (gateway_id, event_id);

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE payment_gateway IS 'Master table of all supported payment gateways';
COMMENT ON TABLE payment_gateway_country IS 'Gateway geo support: which countries a gateway serves';
COMMENT ON TABLE payment_gateway_currency IS 'Gateway geo support: which currencies a gateway settles';
COMMENT ON TABLE payment_gateway_field IS 'Configuration fields required by each gateway';
COMMENT ON TABLE payment_gateway_config IS 'Seller gateway credentials per environment (sandbox/production)';
COMMENT ON TABLE payment_transaction IS 'Core payment transaction records';
COMMENT ON TABLE payment_transaction_event IS 'Append-only ledger of every payment state change, incl. provider request/response';
COMMENT ON TABLE payment_refund IS 'Refund tracking';
COMMENT ON TABLE payment_webhook_log IS 'Webhook audit trail for debugging and idempotency';

COMMENT ON COLUMN payment_gateway_config.credentials IS 'Encrypted gateway credentials (API keys, secrets, etc.)';
COMMENT ON COLUMN payment_transaction.amount_cents IS 'Amount in cents (e.g., 1999 = $19.99 or ₹19.99)';
COMMENT ON COLUMN payment_transaction.environment IS 'Frozen copy of seller_settings.payments_environment at initiate time';
COMMENT ON COLUMN payment_transaction.gateway_session_id IS 'Provider checkout/order/intent id (e.g. Razorpay order.id)';
COMMENT ON COLUMN payment_transaction.gateway_payment_id IS 'Provider captured payment/charge id (e.g. Razorpay payment.id)';
COMMENT ON COLUMN payment_transaction.reference_type IS 'Polymorphic owner type: order, subscription (no FK)';
COMMENT ON COLUMN payment_transaction.reference_id IS 'Polymorphic owner id (no FK)';
COMMENT ON INDEX uq_payment_webhook_log_gateway_event IS 'Webhook idempotency: one processing per (gateway, event)';
