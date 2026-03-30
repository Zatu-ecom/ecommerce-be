-- ============================================================================
-- Payment Module - Database Migration
-- Version: 2.0
-- Created: 2026-01-10
-- ============================================================================

-- ============================================================================
-- 1. PAYMENT GATEWAY - Master table for all supported payment gateways
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_gateway (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Supported countries (ISO 3166-1 alpha-2 codes)
    -- NULL = supports all countries
    supported_countries VARCHAR(2)[],
    
    -- Supported currencies (ISO 4217 codes)
    supported_currencies VARCHAR(3)[] NOT NULL,
    
    -- Supported payment methods
    supported_payment_methods TEXT[] NOT NULL,
    
    webhook_url VARCHAR(500),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient array queries
CREATE INDEX IF NOT EXISTS idx_payment_gateway_supported_countries 
    ON payment_gateway USING GIN (supported_countries);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_supported_currencies 
    ON payment_gateway USING GIN (supported_currencies);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_supported_payment_methods 
    ON payment_gateway USING GIN (supported_payment_methods);
CREATE INDEX IF NOT EXISTS idx_payment_gateway_code 
    ON payment_gateway(code);

-- ============================================================================
-- 2. PAYMENT GATEWAY FIELD - Configuration fields required by each gateway
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
-- 3. PAYMENT GATEWAY CONFIG - Seller's gateway credentials
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
    country VARCHAR(2) NOT NULL,
    
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

-- ============================================================================
-- 4. PAYMENT METHOD - Saved payment methods (cards, UPI, wallets)
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_method (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id),
    
    -- Type of payment method
    type VARCHAR(50) NOT NULL,  -- 'card', 'upi', 'wallet', 'bank_account'
    
    -- Gateway references (tokens)
    gateway_customer_id VARCHAR(255),
    gateway_payment_method_id VARCHAR(255) NOT NULL,
    
    -- Display information for UI
    display_name VARCHAR(200),  -- 'Visa ending in 4242', 'UPI: user@paytm'
    
    -- All other details in JSONB
    details JSONB,  -- {"brand": "visa", "last4": "4242", "exp_month": 12, "exp_year": 2027}
    
    is_default BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_method_user_id 
    ON payment_method(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_method_gateway_payment_method_id 
    ON payment_method(gateway_payment_method_id);

-- ============================================================================
-- 5. PAYMENT TRANSACTION - Core transaction table
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_transaction (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL UNIQUE,
    
    -- Relationships
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    seller_id BIGINT NOT NULL REFERENCES seller_profile(user_id),
    gateway_id BIGINT REFERENCES payment_gateway(id),  -- NULL for COD
    
    -- Gateway reference
    gateway_transaction_id VARCHAR(255),
    
    -- Amount (in cents for precision)
    currency VARCHAR(3) NOT NULL,
    amount_cents BIGINT NOT NULL,
    gateway_fee_cents BIGINT NOT NULL,  
    
    -- Status
    status VARCHAR(30) NOT NULL,  -- 'pending', 'completed', 'failed', 'refunded', 'partially_refunded'
    failure_code VARCHAR(100),
    failure_message TEXT,
    
    -- Payment method type (for quick filtering)
    payment_method_type VARCHAR(50),  -- 'card', 'upi', 'wallet', 'cod', 'bank_transfer'
    
    -- Timestamps
    initiated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Metadata (store additional info)
    metadata JSONB,  -- ip_address, user_agent, gateway_response, fees, etc.
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_transaction_user_id 
    ON payment_transaction(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_seller_id 
    ON payment_transaction(seller_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_status 
    ON payment_transaction(status);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_transaction_id 
    ON payment_transaction(gateway_transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_created_at 
    ON payment_transaction(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_transaction_gateway_id 
    ON payment_transaction(gateway_id);

-- ============================================================================
-- 6. PAYMENT REFUND - Refund tracking
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
CREATE INDEX IF NOT EXISTS idx_payment_refund_gateway_refund_id 
    ON payment_refund(gateway_refund_id);

-- ============================================================================
-- 7. PAYMENT WEBHOOK LOG - Webhook audit trail
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_webhook_log (
    id BIGSERIAL PRIMARY KEY,
    
    gateway_id BIGINT REFERENCES payment_gateway(id),
    event_type VARCHAR(100) NOT NULL,  -- 'payment.success', 'refund.created', etc.
    event_id VARCHAR(255),  -- Gateway's event ID (for idempotency)
    
    -- Payload
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

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE payment_gateway IS 'Master table of all supported payment gateways';
COMMENT ON TABLE payment_gateway_field IS 'Configuration fields required by each gateway';
COMMENT ON TABLE payment_gateway_config IS 'Seller gateway credentials and configuration';
COMMENT ON TABLE payment_method IS 'Saved payment methods (tokenized cards, UPI, wallets)';
COMMENT ON TABLE payment_transaction IS 'Core payment transaction records';
COMMENT ON TABLE payment_refund IS 'Refund tracking';
COMMENT ON TABLE payment_webhook_log IS 'Webhook audit trail for debugging and idempotency';

COMMENT ON COLUMN payment_gateway.supported_countries IS 'NULL = supports all countries, otherwise array of ISO 3166-1 alpha-2 codes';
COMMENT ON COLUMN payment_gateway_config.credentials IS 'Encrypted gateway credentials (API keys, secrets, etc.)';
COMMENT ON COLUMN payment_transaction.amount_cents IS 'Amount in cents (e.g., 1999 = $19.99 or ₹19.99)';
COMMENT ON COLUMN payment_transaction.metadata IS 'Additional data: ip_address, user_agent, gateway_response, fees, etc.';
