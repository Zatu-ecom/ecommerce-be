-- ============================================================================
-- Migration: 031_create_razorpay_payment_integration.sql
-- Description: Razorpay payment gateway integration
--   - Reshape payment_transaction (add polymorphic reference, gateway refs, drop redundant columns)
--   - Replace payment_gateway.logo_url with logo_file_id (file service reference)
--   - Create payment_transaction_event (append-only history ledger)
--   - Add webhook idempotency unique index on payment_webhook_log
-- Created: 2026-08-15
-- ============================================================================

-- ============================================================================
-- 1. PAYMENT TRANSACTION - reshape (existing table, created in 009)
-- ============================================================================
ALTER TABLE payment_transaction
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS initiated_at,
    DROP COLUMN IF EXISTS gateway_transaction_id,
    ADD COLUMN IF NOT EXISTS reference_type       VARCHAR(20),
    ADD COLUMN IF NOT EXISTS reference_id         BIGINT,          -- no FK (polymorphic)
    ADD COLUMN IF NOT EXISTS gateway_session_id   VARCHAR(255),
    ADD COLUMN IF NOT EXISTS gateway_payment_id   VARCHAR(255),
    ADD COLUMN IF NOT EXISTS payment_method_details JSONB;

-- gateway_fee_cents is unknown at initiation; set on capture by webhook
ALTER TABLE payment_transaction ALTER COLUMN gateway_fee_cents DROP NOT NULL;
ALTER TABLE payment_transaction ALTER COLUMN gateway_fee_cents SET DEFAULT 0;

DROP INDEX IF EXISTS idx_payment_transaction_gateway_transaction_id;

CREATE INDEX IF NOT EXISTS idx_payment_transaction_reference
    ON payment_transaction(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_session_id
    ON payment_transaction(gateway_session_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_payment_id
    ON payment_transaction(gateway_payment_id);

-- ============================================================================
-- 2. PAYMENT GATEWAY - replace logo_url with file service reference
-- ============================================================================
ALTER TABLE payment_gateway
    ADD COLUMN IF NOT EXISTS logo_file_id VARCHAR(80),
    DROP COLUMN IF EXISTS logo_url;

-- ============================================================================
-- 3. PAYMENT TRANSACTION EVENT - append-only history ledger
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
-- 4. PAYMENT WEBHOOK LOG - idempotency (unique per gateway + event)
-- ============================================================================
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_webhook_log_gateway_event
    ON payment_webhook_log (gateway_id, event_id)
    WHERE event_id IS NOT NULL;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON COLUMN payment_transaction.reference_type IS 'Polymorphic owner type: order, subscription (no FK)';
COMMENT ON COLUMN payment_transaction.reference_id IS 'Polymorphic owner id (no FK)';
COMMENT ON COLUMN payment_transaction.gateway_session_id IS 'Provider checkout/order/intent id (e.g. Razorpay order.id)';
COMMENT ON COLUMN payment_transaction.gateway_payment_id IS 'Provider captured payment/charge id (e.g. Razorpay payment.id)';
COMMENT ON TABLE payment_transaction_event IS 'Append-only ledger of every payment state change, incl. provider request/response';
COMMENT ON INDEX uq_payment_webhook_log_gateway_event IS 'Webhook idempotency: one processing per (gateway, event)';
