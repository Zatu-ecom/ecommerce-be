# 💳 Payment Module - Database Design & Architecture (Version 2)

> **Last Updated**: January 9, 2026  
> **Status**: Design Phase  
> **Version**: 2.0  
> **Author**: Development Team

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Key Design Considerations](#key-design-considerations)
3. [Database Schema](#database-schema)
4. [Status Enums](#status-enums)
5. [Architecture: Gateway Abstraction](#architecture-gateway-abstraction)
6. [Example Data Population](#example-data-population)
7. [Common Queries](#common-queries)
8. [MVP Strategy](#mvp-strategy)
9. [API Endpoints (Planned)](#api-endpoints-planned)
10. [Open Questions](#open-questions)

---

## 🎯 Overview

The Payment Module provides a flexible, multi-gateway payment system that supports:

- **Multiple Payment Gateways**: Stripe, Razorpay, PayU, PayPal, etc.
- **Dynamic Gateway Configuration**: Each gateway defines its own required fields
- **Country & Currency Support**: Gateways specify which countries/currencies they support
- **Multiple Payment Methods**: Cards, UPI, Wallets, Bank Transfer, BNPL, COD
- **Saved Payment Methods**: Tokenized cards and wallets
- **Refunds**: Full and partial refunds
- **Webhook Processing**: Real-time payment status updates

---

## 🎯 Key Design Considerations

### The Challenge

Payment gateways vary by region:

| Region             | Popular Gateways                  |
| ------------------ | --------------------------------- |
| **India**          | Razorpay, PayU, Cashfree, PhonePe |
| **US/Europe**      | Stripe, PayPal, Braintree, Adyen  |
| **Southeast Asia** | GrabPay, GCash, OVO               |
| **Global**         | Stripe (expanding), PayPal        |

### Strategy: Gateway Abstraction Layer

Instead of hardcoding any gateway, we design for:

1. **Gateway Configuration** - Each gateway defines its own required configuration fields
2. **Country/Currency Support** - Gateways specify which countries and currencies they support
3. **Payment Method Abstraction** - Cards, UPI, Wallets, Bank Transfer, BNPL all work the same way
4. **Gateway-Agnostic Transactions** - Store our own transaction records, link to gateway references
5. **Fallback Support** - If primary gateway fails, try secondary based on priority

---

## 📊 Database Schema

### 1. `payment_gateway` - Available Payment Gateways

Master table of all supported payment gateways in the system.

```sql
CREATE TABLE payment_gateway (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,        -- 'stripe', 'razorpay', 'payu'
    name VARCHAR(100) NOT NULL,              -- 'Stripe', 'Razorpay', 'PayU'
    description TEXT,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    
    -- Supported countries (ISO 3166-1 alpha-2 codes)
    -- NULL = supports all countries
    supported_countries VARCHAR(2)[],        -- ['IN', 'US', 'GB', 'BR']
    
    -- Supported currencies (ISO 4217 codes)
    supported_currencies VARCHAR(3)[],       -- ['USD', 'EUR', 'INR', 'BRL']
    
    -- Supported payment methods
    supported_payment_methods TEXT[],        -- ['card', 'upi', 'wallet', 'bank_transfer', 'emi']
    
    webhook_url VARCHAR(500),                -- Our endpoint for this gateway
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- GIN indexes for efficient array queries
CREATE INDEX idx_payment_gateway_supported_countries 
    ON payment_gateway USING GIN (supported_countries);
CREATE INDEX idx_payment_gateway_supported_currencies 
    ON payment_gateway USING GIN (supported_currencies);
CREATE INDEX idx_payment_gateway_supported_payment_methods 
    ON payment_gateway USING GIN (supported_payment_methods);
CREATE INDEX idx_payment_gateway_code 
    ON payment_gateway(code) WHERE deleted_at IS NULL;
```

**Example Data:**

```sql
-- PayU - India and LATAM
INSERT INTO payment_gateway (code, name, description, supported_countries, supported_currencies, supported_payment_methods) 
VALUES (
    'payu',
    'PayU',
    'PayU Payment Gateway - Popular in India and LATAM',
    ARRAY['IN', 'BR', 'MX', 'AR', 'CO', 'PE', 'CL'],
    ARRAY['INR', 'BRL', 'MXN', 'ARS', 'COP', 'PEN', 'CLP'],
    ARRAY['card', 'upi', 'wallet', 'netbanking', 'emi']
);

-- Razorpay - India only
INSERT INTO payment_gateway (code, name, description, supported_countries, supported_currencies, supported_payment_methods) 
VALUES (
    'razorpay',
    'Razorpay',
    'Leading payment solution in India',
    ARRAY['IN'],
    ARRAY['INR'],
    ARRAY['card', 'upi', 'wallet', 'netbanking', 'emi', 'cardless_emi', 'paylater']
);

-- Stripe - Global
INSERT INTO payment_gateway (code, name, description, supported_countries, supported_currencies, supported_payment_methods) 
VALUES (
    'stripe',
    'Stripe',
    'Global payment platform',
    NULL,  -- NULL = all countries
    ARRAY['USD', 'EUR', 'GBP', 'INR', 'AUD', 'CAD', 'SGD', 'JPY', 'BRL'],
    ARRAY['card', 'bank_transfer', 'wallet', 'bnpl']
);
```

---

### 2. `payment_gateway_field` - Gateway Configuration Fields

Defines what configuration fields each gateway requires from sellers.

```sql
CREATE TABLE payment_gateway_field (
    id BIGSERIAL PRIMARY KEY,
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id) ON DELETE CASCADE,
    
    -- Field identification
    field_name VARCHAR(100) NOT NULL,        -- 'merchant_key', 'auth_token', 'key_id'
    display_name VARCHAR(200) NOT NULL,      -- 'Merchant Key', 'Auth Token'
    
    -- Field properties
    field_type VARCHAR(50) NOT NULL,         -- 'string', 'number', 'boolean', 'url', 'email'
    description TEXT,                        -- Help text for sellers
    placeholder VARCHAR(200),                -- Placeholder text for input
    
    -- Field behavior
    is_required BOOLEAN DEFAULT true,        -- Is this field mandatory?
    is_sensitive BOOLEAN DEFAULT false,      -- Should this be encrypted/masked?
    
    -- Display order
    display_order INT DEFAULT 0,             -- Order to show fields in UI
    
    -- Validation rules (stored as JSONB for flexibility)
    validation_rules JSONB,                  -- {"min_length": 10, "pattern": "^[a-z]+$"}
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Ensure unique field names per gateway
    UNIQUE(gateway_id, field_name)
);

-- Index for faster lookups
CREATE INDEX idx_payment_gateway_field_gateway_id 
    ON payment_gateway_field(gateway_id) WHERE deleted_at IS NULL;
```

**Validation Rules JSONB Examples:**

```json
// String length validation
{
  "min_length": 10,
  "max_length": 100
}

// Pattern validation (regex)
{
  "pattern": "^[a-zA-Z0-9_-]+$",
  "custom_error_message": "Only alphanumeric characters, dashes, and underscores allowed"
}

// Number range validation
{
  "min_value": 0,
  "max_value": 100
}

// URL validation
{
  "type": "url",
  "allowed_protocols": ["https"]
}

// Combined validations
{
  "min_length": 20,
  "max_length": 50,
  "pattern": "^rzp_(test|live)_[a-zA-Z0-9]+$",
  "custom_error_message": "Must be a valid Razorpay key ID (e.g., rzp_live_xxxxx)"
}
```

---

### 3. `payment_gateway_config` - Seller Gateway Configuration

Stores API credentials and configuration for each gateway per seller.

```sql
CREATE TABLE payment_gateway_config (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES "user"(id),
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id),
    
    environment VARCHAR(20) NOT NULL,        -- 'sandbox', 'production'
    
    -- Actual configuration values (encrypted at application level)
    credentials JSONB NOT NULL,              -- {"merchant_key": "xxx", "auth_token": "yyy"}
    
    is_active BOOLEAN DEFAULT true,
    priority INT DEFAULT 0,                  -- Higher = preferred (for fallback)
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE(seller_id, gateway_id, environment)
);

-- Indexes
CREATE INDEX idx_payment_gateway_config_seller_id 
    ON payment_gateway_config(seller_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_gateway_config_gateway_id 
    ON payment_gateway_config(gateway_id);
CREATE INDEX idx_payment_gateway_config_active 
    ON payment_gateway_config(seller_id, is_active) WHERE deleted_at IS NULL;
```

**Credentials JSONB Structure (encrypted at application level):**

```json
// PayU credentials
{
  "merchant_key": "xxxxxx",
  "merchant_salt": "yyyyyy",
  "auth_token": "zzzzzz",
  "webhook_secret": "wwwwww"
}

// Razorpay credentials
{
  "key_id": "rzp_live_xxxxxxxxx",
  "key_secret": "yyyyyyyyyyyy",
  "webhook_secret": "zzzzzzzzzzzz"
}

// Stripe credentials
{
  "publishable_key": "pk_live_xxxxxxxxx",
  "secret_key": "sk_live_yyyyyyyy",
  "webhook_secret": "whsec_zzzzzzz"
}
```

---

### 4. `payment_method` - Saved Payment Methods

Stores tokenized payment methods (cards, UPI, wallets) for returning customers.

```sql
CREATE TABLE payment_method (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id),
    type VARCHAR(50) NOT NULL,               -- 'card', 'upi', 'wallet', 'bank_account'

    -- Gateway reference (for tokenized cards)
    gateway_customer_id VARCHAR(255),        -- Stripe customer_id, Razorpay customer_id
    gateway_payment_method_id VARCHAR(255),  -- Stripe pm_xxx, Razorpay token

    -- Display info (masked/safe to store)
    display_name VARCHAR(100),               -- 'Visa ending in 4242'
    card_brand VARCHAR(50),                  -- 'visa', 'mastercard', 'amex'
    card_last_four VARCHAR(4),
    card_exp_month INT,
    card_exp_year INT,

    -- For UPI/Wallet
    upi_id VARCHAR(100),                     -- 'user@upi'
    wallet_provider VARCHAR(50),             -- 'paytm', 'phonepe', 'googlepay'

    is_default BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,

    metadata JSONB,                          -- Additional gateway-specific data

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_payment_method_user_id 
    ON payment_method(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_method_gateway_id 
    ON payment_method(gateway_id);
CREATE INDEX idx_payment_method_gateway_payment_method_id 
    ON payment_method(gateway_payment_method_id);
```

---

### 5. `payment_transaction` - Core Transaction Table

The main table storing all payment transactions.

```sql
CREATE TABLE payment_transaction (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL UNIQUE,  -- Our internal ID: 'TXN_20260109_XXXXX'

    -- Relationships
    order_id BIGINT REFERENCES "order"(id),
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    seller_id BIGINT NOT NULL REFERENCES "user"(id),
    gateway_id BIGINT REFERENCES payment_gateway(id),
    payment_method_id BIGINT REFERENCES payment_method(id),

    -- Gateway references
    gateway_order_id VARCHAR(255),           -- Gateway's order/session ID
    gateway_payment_id VARCHAR(255),         -- Gateway's payment/charge ID
    gateway_transaction_id VARCHAR(255),     -- Gateway's transaction reference

    -- Amount details (stored in cents for precision)
    currency VARCHAR(3) NOT NULL,            -- 'USD', 'INR'
    amount_cents BIGINT NOT NULL,            -- Total amount in cents (1999 = $19.99 or ₹19.99)
    gateway_fee_cents BIGINT,                -- Fee charged by gateway in cents
    net_amount_cents BIGINT,                 -- Amount after fees in cents

    -- Status
    status VARCHAR(30) NOT NULL,             -- See status enum below
    failure_code VARCHAR(100),               -- Gateway error code
    failure_message TEXT,                    -- Human-readable error

    -- Payment details
    payment_method_type VARCHAR(50),         -- 'card', 'upi', 'wallet', 'bank_transfer', 'cod'
    payment_method_details JSONB,            -- Safe display info

    -- Timestamps
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    authorized_at TIMESTAMPTZ,
    captured_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,

    -- Metadata
    ip_address VARCHAR(50),
    user_agent TEXT,
    metadata JSONB,                          -- Any additional data
    gateway_response JSONB,                  -- Full gateway response (for debugging)

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes for common queries
CREATE INDEX idx_payment_transaction_order_id 
    ON payment_transaction(order_id);
CREATE INDEX idx_payment_transaction_user_id 
    ON payment_transaction(user_id);
CREATE INDEX idx_payment_transaction_seller_id 
    ON payment_transaction(seller_id);
CREATE INDEX idx_payment_transaction_status 
    ON payment_transaction(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_transaction_gateway_payment_id 
    ON payment_transaction(gateway_payment_id);
CREATE INDEX idx_payment_transaction_created_at 
    ON payment_transaction(created_at DESC);
CREATE INDEX idx_payment_transaction_gateway_id 
    ON payment_transaction(gateway_id);
```

**payment_method_details JSONB Example:**

```json
{
  "card_brand": "visa",
  "card_last_four": "4242",
  "card_exp_month": 12,
  "card_exp_year": 2027,
  "card_network": "visa",
  "card_type": "credit"
}
```

---

### 6. `payment_refund` - Refunds Table

Tracks all refund requests and their status.

```sql
CREATE TABLE payment_refund (
    id BIGSERIAL PRIMARY KEY,
    refund_id VARCHAR(50) NOT NULL UNIQUE,   -- 'RFD_20260109_XXXXX'

    transaction_id BIGINT NOT NULL REFERENCES payment_transaction(id),
    order_id BIGINT REFERENCES "order"(id),

    -- Gateway reference
    gateway_refund_id VARCHAR(255),          -- Gateway's refund ID

    -- Amount (stored in cents for precision)
    currency VARCHAR(3) NOT NULL,
    amount_cents BIGINT NOT NULL,            -- Refund amount in cents

    -- Status
    status VARCHAR(30) NOT NULL,             -- 'pending', 'processing', 'completed', 'failed'
    failure_reason TEXT,

    -- Reason
    reason VARCHAR(100),                     -- 'customer_request', 'order_cancelled', 'defective'
    notes TEXT,                              -- Internal notes

    -- Who initiated
    initiated_by BIGINT REFERENCES "user"(id),
    initiated_by_type VARCHAR(20),           -- 'customer', 'seller', 'admin', 'system'

    -- Timestamps
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    metadata JSONB,
    gateway_response JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_payment_refund_transaction_id 
    ON payment_refund(transaction_id);
CREATE INDEX idx_payment_refund_order_id 
    ON payment_refund(order_id);
CREATE INDEX idx_payment_refund_status 
    ON payment_refund(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_refund_gateway_refund_id 
    ON payment_refund(gateway_refund_id);
```

---

### 7. `payment_webhook_log` - Webhook Audit Trail

Stores all incoming webhooks for debugging and idempotency.

```sql
CREATE TABLE payment_webhook_log (
    id BIGSERIAL PRIMARY KEY,

    gateway_id BIGINT REFERENCES payment_gateway(id),
    event_type VARCHAR(100) NOT NULL,        -- 'payment.success', 'refund.created'
    event_id VARCHAR(255),                   -- Gateway's event ID (for idempotency)

    -- Payload
    payload JSONB NOT NULL,
    headers JSONB,

    -- Processing
    status VARCHAR(30) NOT NULL,             -- 'received', 'processed', 'failed', 'ignored'
    error_message TEXT,
    processed_at TIMESTAMPTZ,

    -- Linked records
    transaction_id BIGINT REFERENCES payment_transaction(id),
    refund_id BIGINT REFERENCES payment_refund(id),

    ip_address VARCHAR(50),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_payment_webhook_log_gateway_id 
    ON payment_webhook_log(gateway_id);
CREATE INDEX idx_payment_webhook_log_event_id 
    ON payment_webhook_log(event_id);
CREATE INDEX idx_payment_webhook_log_status 
    ON payment_webhook_log(status);
CREATE INDEX idx_payment_webhook_log_created_at 
    ON payment_webhook_log(created_at DESC);
```

---

## 📋 Status Enums

### Transaction Status Flow

```
INITIATED → PENDING → AUTHORIZED → CAPTURED → COMPLETED
                ↓           ↓          ↓
             FAILED     CANCELLED   REFUNDED (partial/full)
```

| Status               | Description                                   |
| -------------------- | --------------------------------------------- |
| `initiated`          | Payment flow started                          |
| `pending`            | Waiting for user action (redirect to gateway) |
| `authorized`         | Payment authorized but not captured           |
| `captured`           | Amount captured from user                     |
| `completed`          | Successfully processed                        |
| `failed`             | Payment failed                                |
| `cancelled`          | Cancelled by user/system                      |
| `refunded`           | Fully refunded                                |
| `partially_refunded` | Partially refunded                            |

### Refund Status

| Status       | Description                          |
| ------------ | ------------------------------------ |
| `pending`    | Refund requested, not yet processed  |
| `processing` | Refund is being processed by gateway |
| `completed`  | Refund successful                    |
| `failed`     | Refund failed                        |

### Webhook Status

| Status      | Description                         |
| ----------- | ----------------------------------- |
| `received`  | Webhook received, not yet processed |
| `processed` | Successfully processed              |
| `failed`    | Processing failed                   |
| `ignored`   | Duplicate or irrelevant event       |

---

## 🏗️ Architecture: Gateway Abstraction

### Interface Design

```go
// PaymentGateway defines the interface all gateways must implement
type PaymentGateway interface {
    // Configuration
    GetConfigSchema() GatewayConfigSchema
    ValidateCredentials(credentials map[string]interface{}) error
    
    // Core payment operations
    InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*PaymentSession, error)
    CapturePayment(ctx context.Context, transactionID string) (*Transaction, error)
    CancelPayment(ctx context.Context, transactionID string) error

    // Refunds
    RefundPayment(ctx context.Context, transactionID string, amountCents int64) (*Refund, error)

    // Webhooks
    VerifyWebhook(payload []byte, signature string) (bool, error)
    ParseWebhook(payload []byte) (*WebhookEvent, error)

    // Saved payment methods
    SavePaymentMethod(ctx context.Context, userID uint, token string) (*PaymentMethod, error)
    DeletePaymentMethod(ctx context.Context, methodID string) error

    // Info
    GetSupportedMethods() []string
    GetSupportedCurrencies() []string
    GetSupportedCountries() []string
}
```

### Gateway Configuration Schema

```go
type GatewayConfigSchema struct {
    RequiredFields []ConfigField `json:"required_fields"`
    OptionalFields []ConfigField `json:"optional_fields"`
}

type ConfigField struct {
    FieldName   string            `json:"field_name"`
    DisplayName string            `json:"display_name"`
    Type        string            `json:"type"` // string, number, boolean, url, email
    Description string            `json:"description"`
    Placeholder string            `json:"placeholder"`
    IsSensitive bool              `json:"is_sensitive"`
    Validation  *FieldValidation  `json:"validation,omitempty"`
}

type FieldValidation struct {
    MinLength int    `json:"min_length,omitempty"`
    MaxLength int    `json:"max_length,omitempty"`
    Pattern   string `json:"pattern,omitempty"`
    Required  bool   `json:"required"`
}
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Payment Service                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           PaymentGateway Interface                   │   │
│  │  • GetConfigSchema() → GatewayConfigSchema          │   │
│  │  • ValidateCredentials(creds) → error               │   │
│  │  • InitiatePayment(order, method) → PaymentSession  │   │
│  │  • CapturePayment(transactionId) → Transaction      │   │
│  │  • RefundPayment(transactionId, amount) → Refund    │   │
│  │  • VerifyWebhook(payload, signature) → bool         │   │
│  └─────────────────────────────────────────────────────┘   │
│        ▲              ▲              ▲              ▲       │
│        │              │              │              │       │
│  ┌─────┴────┐  ┌─────┴────┐  ┌─────┴────┐  ┌─────┴────┐   │
│  │  Stripe  │  │ Razorpay │  │  PayPal  │  │  PayU    │   │
│  │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Adapter  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Gateway Selection Logic

```go
func (s *PaymentService) SelectGateway(
    sellerID uint, 
    country string, 
    currency string,
) (PaymentGateway, *PaymentGatewayConfig, error) {
    // 1. Get all active gateway configs for seller
    configs, err := s.configRepo.FindBySellerID(sellerID, "production")
    if err != nil {
        return nil, nil, err
    }

    // 2. Filter gateways that support the country and currency
    var validConfigs []*PaymentGatewayConfig
    for _, config := range configs {
        gateway := config.Gateway
        
        // Check country support
        if gateway.SupportedCountries != nil && 
           !contains(gateway.SupportedCountries, country) {
            continue
        }
        
        // Check currency support
        if !contains(gateway.SupportedCurrencies, currency) {
            continue
        }
        
        validConfigs = append(validConfigs, config)
    }

    if len(validConfigs) == 0 {
        return nil, nil, errors.New("no gateway available for country/currency")
    }

    // 3. Sort by priority (highest first)
    sort.Slice(validConfigs, func(i, j int) bool {
        return validConfigs[i].Priority > validConfigs[j].Priority
    })

    // 4. Return highest priority gateway
    selectedConfig := validConfigs[0]
    gateway, err := s.gatewayFactory.GetGatewayWithConfig(
        selectedConfig.Gateway.Code,
        selectedConfig.Credentials,
    )
    
    return gateway, selectedConfig, err
}
```

---

## 📝 Example Data Population

### PayU Gateway Setup

```sql
-- 1. Insert PayU gateway
INSERT INTO payment_gateway (
    code, 
    name, 
    description, 
    supported_countries,
    supported_currencies, 
    supported_payment_methods,
    logo_url
) VALUES (
    'payu',
    'PayU',
    'PayU Payment Gateway - Popular in India and LATAM',
    ARRAY['IN', 'BR', 'MX', 'AR', 'CO', 'PE', 'CL'],
    ARRAY['INR', 'BRL', 'MXN', 'ARS', 'COP', 'PEN', 'CLP'],
    ARRAY['card', 'upi', 'wallet', 'netbanking', 'emi'],
    'https://cdn.example.com/logos/payu.png'
) RETURNING id; -- Returns id = 1

-- 2. Define PayU configuration fields
INSERT INTO payment_gateway_field (
    gateway_id, 
    field_name, 
    display_name, 
    field_type, 
    description, 
    placeholder, 
    is_required, 
    is_sensitive, 
    display_order, 
    validation_rules
) VALUES
    (1, 'merchant_key', 'Merchant Key', 'string', 
     'Your PayU merchant key provided during onboarding', 
     'Enter your merchant key', 
     true, false, 1, 
     '{"min_length": 10, "max_length": 100, "pattern": "^[a-zA-Z0-9]+$"}'::jsonb),
    
    (1, 'merchant_salt', 'Merchant Salt', 'string', 
     'Secret salt used for hash generation and transaction verification', 
     'Enter your merchant salt', 
     true, true, 2, 
     '{"min_length": 10, "max_length": 100}'::jsonb),
    
    (1, 'auth_token', 'Auth Token', 'string', 
     'API authentication token for server-to-server calls', 
     'Enter your auth token', 
     true, true, 3, 
     '{"min_length": 20}'::jsonb),
    
    (1, 'webhook_secret', 'Webhook Secret', 'string', 
     'Secret for verifying webhook signatures (optional but recommended)', 
     'Enter webhook secret', 
     false, true, 4, 
     '{"min_length": 16}'::jsonb);

-- 3. Seller configures PayU (example)
INSERT INTO payment_gateway_config (
    seller_id,
    gateway_id,
    environment,
    credentials,
    is_active,
    priority
) VALUES (
    123,  -- seller_id
    1,    -- PayU gateway_id
    'production',
    '{"merchant_key": "encrypted_key_here", "merchant_salt": "encrypted_salt_here", "auth_token": "encrypted_token_here"}'::jsonb,
    true,
    10  -- High priority
);
```

### Razorpay Gateway Setup

```sql
-- 1. Insert Razorpay gateway
INSERT INTO payment_gateway (
    code, 
    name, 
    description, 
    supported_countries,
    supported_currencies, 
    supported_payment_methods,
    logo_url
) VALUES (
    'razorpay',
    'Razorpay',
    'Leading payment solution in India with support for all major payment methods',
    ARRAY['IN'],
    ARRAY['INR'],
    ARRAY['card', 'upi', 'wallet', 'netbanking', 'emi', 'cardless_emi', 'paylater'],
    'https://cdn.example.com/logos/razorpay.png'
) RETURNING id; -- Returns id = 2

-- 2. Define Razorpay configuration fields
INSERT INTO payment_gateway_field (
    gateway_id, 
    field_name, 
    display_name, 
    field_type, 
    description, 
    placeholder, 
    is_required, 
    is_sensitive, 
    display_order, 
    validation_rules
) VALUES
    (2, 'key_id', 'Key ID', 'string', 
     'Your Razorpay Key ID (starts with rzp_test_ or rzp_live_)', 
     'rzp_live_xxxxxxxxx', 
     true, false, 1, 
     '{"pattern": "^rzp_(test|live)_[a-zA-Z0-9]+$", "custom_error_message": "Must be a valid Razorpay key ID"}'::jsonb),
    
    (2, 'key_secret', 'Key Secret', 'string', 
     'Your Razorpay Key Secret (keep this confidential)', 
     'Enter your key secret', 
     true, true, 2, 
     '{"min_length": 20}'::jsonb),
    
    (2, 'webhook_secret', 'Webhook Secret', 'string', 
     'Webhook signature secret for verifying webhook authenticity', 
     'Enter webhook secret', 
     true, true, 3, 
     '{"min_length": 16}'::jsonb),
    
    (2, 'account_id', 'Account ID', 'string', 
     'Razorpay Account ID (optional, required only for route/transfer features)', 
     'acc_xxxxx', 
     false, false, 4, 
     '{"pattern": "^acc_[a-zA-Z0-9]+$"}'::jsonb);

-- 3. Seller configures Razorpay (example)
INSERT INTO payment_gateway_config (
    seller_id,
    gateway_id,
    environment,
    credentials,
    is_active,
    priority
) VALUES (
    123,  -- seller_id
    2,    -- Razorpay gateway_id
    'production',
    '{"key_id": "rzp_live_xxxxx", "key_secret": "encrypted_secret_here", "webhook_secret": "encrypted_webhook_secret"}'::jsonb,
    true,
    5  -- Lower priority than PayU (fallback)
);
```

### Stripe Gateway Setup

```sql
-- 1. Insert Stripe gateway
INSERT INTO payment_gateway (
    code, 
    name, 
    description, 
    supported_countries,
    supported_currencies, 
    supported_payment_methods,
    logo_url
) VALUES (
    'stripe',
    'Stripe',
    'Global payment platform supporting 135+ currencies and 45+ countries',
    NULL,  -- NULL = supports all countries
    ARRAY['USD', 'EUR', 'GBP', 'INR', 'AUD', 'CAD', 'SGD', 'JPY', 'BRL', 'MXN'],
    ARRAY['card', 'bank_transfer', 'wallet', 'bnpl'],
    'https://cdn.example.com/logos/stripe.png'
) RETURNING id; -- Returns id = 3

-- 2. Define Stripe configuration fields
INSERT INTO payment_gateway_field (
    gateway_id, 
    field_name, 
    display_name, 
    field_type, 
    description, 
    placeholder, 
    is_required, 
    is_sensitive, 
    display_order, 
    validation_rules
) VALUES
    (3, 'publishable_key', 'Publishable Key', 'string', 
     'Your Stripe publishable key (safe to expose in frontend)', 
     'pk_live_xxxxxxxxx', 
     true, false, 1, 
     '{"pattern": "^pk_(test|live)_[a-zA-Z0-9]+$"}'::jsonb),
    
    (3, 'secret_key', 'Secret Key', 'string', 
     'Your Stripe secret key (keep this confidential)', 
     'sk_live_xxxxxxxxx', 
     true, true, 2, 
     '{"pattern": "^sk_(test|live)_[a-zA-Z0-9]+$"}'::jsonb),
    
    (3, 'webhook_secret', 'Webhook Secret', 'string', 
     'Webhook signing secret for verifying webhook events', 
     'whsec_xxxxxxxxx', 
     true, true, 3, 
     '{"pattern": "^whsec_[a-zA-Z0-9]+$"}'::jsonb);
```

---

## 🔍 Common Queries

### 1. Find gateways that support a specific country

```sql
-- Find all gateways supporting India
SELECT 
    code,
    name,
    supported_currencies,
    supported_payment_methods
FROM payment_gateway
WHERE 
    (supported_countries IS NULL OR 'IN' = ANY(supported_countries))
    AND is_active = true
    AND deleted_at IS NULL;
```

### 2. Find gateways that support a specific currency

```sql
-- Find all gateways supporting INR
SELECT 
    code,
    name,
    supported_countries,
    supported_payment_methods
FROM payment_gateway
WHERE 
    'INR' = ANY(supported_currencies)
    AND is_active = true
    AND deleted_at IS NULL;
```

### 3. Find gateways for country + currency combination

```sql
-- Find gateways for India + INR
SELECT 
    pg.code,
    pg.name,
    pg.supported_payment_methods
FROM payment_gateway pg
WHERE 
    (pg.supported_countries IS NULL OR 'IN' = ANY(pg.supported_countries))
    AND 'INR' = ANY(pg.supported_currencies)
    AND pg.is_active = true
    AND pg.deleted_at IS NULL;
```

### 4. Get seller's configured gateways for a country/currency

```sql
-- Get all gateways configured by seller for India/INR, ordered by priority
SELECT 
    pg.code,
    pg.name,
    pgc.environment,
    pgc.is_active,
    pgc.priority
FROM payment_gateway_config pgc
JOIN payment_gateway pg ON pgc.gateway_id = pg.id
WHERE 
    pgc.seller_id = 123
    AND (pg.supported_countries IS NULL OR 'IN' = ANY(pg.supported_countries))
    AND 'INR' = ANY(pg.supported_currencies)
    AND pgc.is_active = true
    AND pgc.deleted_at IS NULL
ORDER BY pgc.priority DESC;
```

### 5. Get configuration fields for a gateway

```sql
-- Get all configuration fields for PayU
SELECT 
    field_name,
    display_name,
    field_type,
    description,
    placeholder,
    is_required,
    is_sensitive,
    validation_rules
FROM payment_gateway_field pgf
JOIN payment_gateway pg ON pgf.gateway_id = pg.id
WHERE 
    pg.code = 'payu'
    AND pgf.deleted_at IS NULL
ORDER BY pgf.display_order;
```

### 6. Validate seller configuration completeness

```sql
-- Check if seller has provided all required fields for a gateway
SELECT 
    pgf.field_name,
    pgf.display_name,
    pgf.is_required,
    CASE 
        WHEN pgc.credentials ? pgf.field_name THEN true 
        ELSE false 
    END as is_configured
FROM payment_gateway_field pgf
LEFT JOIN payment_gateway_config pgc 
    ON pgf.gateway_id = pgc.gateway_id 
    AND pgc.seller_id = 123
    AND pgc.environment = 'production'
WHERE 
    pgf.gateway_id = 1  -- PayU
    AND pgf.is_required = true
    AND pgf.deleted_at IS NULL;
```

### 7. Get payment transactions with gateway details

```sql
-- Get recent transactions with gateway information
SELECT 
    pt.transaction_id,
    pt.amount_cents,
    pt.currency,
    pt.status,
    pg.name as gateway_name,
    pg.code as gateway_code,
    pt.payment_method_type,
    pt.created_at
FROM payment_transaction pt
JOIN payment_gateway pg ON pt.gateway_id = pg.id
WHERE 
    pt.seller_id = 123
    AND pt.deleted_at IS NULL
ORDER BY pt.created_at DESC
LIMIT 50;
```

### 8. Get refund summary for a transaction

```sql
-- Get all refunds for a transaction
SELECT 
    pr.refund_id,
    pr.amount_cents,
    pr.currency,
    pr.status,
    pr.reason,
    pr.initiated_by_type,
    pr.initiated_at,
    pr.completed_at
FROM payment_refund pr
WHERE 
    pr.transaction_id = 12345
    AND pr.deleted_at IS NULL
ORDER BY pr.created_at DESC;
```

---

## 🚀 MVP Strategy

### Phase 1: MVP (Week 1-2)

**Goal**: Get payments working with ONE gateway

- [ ] Implement database schema (all tables)
- [ ] Single Gateway Integration (Razorpay OR PayU for India, Stripe for global)
- [ ] Payment Methods: Card only
- [ ] Basic webhook handling
- [ ] Admin: Manual gateway configuration via SQL

**Endpoints:**

- `POST /api/payments/initiate` - Start payment
- `POST /api/payments/:id/capture` - Capture authorized payment
- `POST /api/payments/webhook/:gateway` - Handle webhooks
- `GET /api/payments/:id` - Get transaction details

### Phase 2: Multi-Gateway (Week 3-4)

- [ ] Add second gateway for fallback
- [ ] Gateway selection by country/currency
- [ ] Admin UI to configure gateways
- [ ] Seller UI to configure gateway credentials
- [ ] Gateway priority and fallback logic

### Phase 3: Full Features (Week 5+)

- [ ] Saved payment methods
- [ ] Multiple payment methods (UPI, wallets, BNPL)
- [ ] Automatic gateway selection
- [ ] Retry logic with fallback gateways
- [ ] COD (Cash on Delivery) support
- [ ] Split payments (multi-seller)
- [ ] Payment analytics dashboard

---

## 🔌 API Endpoints (Planned)

### Customer APIs

| Method   | Endpoint                    | Description                     |
| -------- | --------------------------- | ------------------------------- |
| `POST`   | `/api/payments/initiate`    | Initiate a payment for an order |
| `GET`    | `/api/payments/:id`         | Get transaction details         |
| `POST`   | `/api/payments/:id/cancel`  | Cancel a pending payment        |
| `GET`    | `/api/payments/methods`     | Get saved payment methods       |
| `POST`   | `/api/payments/methods`     | Save a new payment method       |
| `DELETE` | `/api/payments/methods/:id` | Delete a saved payment method   |

### Seller APIs

| Method | Endpoint                          | Description               |
| ------ | --------------------------------- | ------------------------- |
| `GET`  | `/api/seller/payments`            | List all transactions     |
| `GET`  | `/api/seller/payments/:id`        | Get transaction details   |
| `POST` | `/api/seller/payments/:id/refund` | Initiate a refund         |
| `GET`  | `/api/seller/payments/summary`    | Payment summary/analytics |
| `GET`  | `/api/seller/payment-gateways`    | List available gateways   |
| `POST` | `/api/seller/payment-gateways`    | Configure a gateway       |
| `PUT`  | `/api/seller/payment-gateways/:id`| Update gateway config     |

### Admin APIs

| Method | Endpoint                                      | Description                    |
| ------ | --------------------------------------------- | ------------------------------ |
| `GET`  | `/api/admin/payment-gateways`                 | List all gateways              |
| `POST` | `/api/admin/payment-gateways`                 | Add a new gateway              |
| `PUT`  | `/api/admin/payment-gateways/:id`             | Update gateway                 |
| `GET`  | `/api/admin/payment-gateways/:code/schema`    | Get gateway configuration schema|
| `GET`  | `/api/admin/payment-gateway-configs`          | List all seller configs        |

### Webhook Endpoints

| Method | Endpoint                 | Description              |
| ------ | ------------------------ | ------------------------ |
| `POST` | `/api/webhooks/stripe`   | Stripe webhook handler   |
| `POST` | `/api/webhooks/razorpay` | Razorpay webhook handler |
| `POST` | `/api/webhooks/payu`     | PayU webhook handler     |
| `POST` | `/api/webhooks/paypal`   | PayPal webhook handler   |

---

## ❓ Open Questions

1. **Which country/region are you launching first?**
   - This determines the initial gateway choice.

2. **Do you need COD (Cash on Delivery)?**
   - Common in India/Southeast Asia.

3. **Do you want to support subscriptions?**
   - Recurring payments for memberships, etc.

4. **Multi-seller payouts?**
   - Split payments directly to sellers vs. platform collects all.

5. **Currency handling?**
   - Single currency or multi-currency support?

6. **Payment capture strategy?**
   - Auto-capture or manual capture (authorize first, capture later)?

---

## 📚 References

- [Stripe API Documentation](https://stripe.com/docs/api)
- [Razorpay API Documentation](https://razorpay.com/docs/api/)
- [PayU API Documentation](https://docs.payu.in/)
- [PayPal REST API](https://developer.paypal.com/docs/api/overview/)

---

## 📝 Change Log

| Date       | Version | Changes                                                      |
| ---------- | ------- | ------------------------------------------------------------ |
| 2026-01-09 | 2.0     | Removed `country_code` from `payment_gateway_config`, removed `payment_gateway_field_mapping`, moved country/currency support to `payment_gateway` table using arrays |
| 2025-12-28 | 1.0     | Initial design document                                       |

---

**Next Steps**: 
1. Review and approve this design
2. Create database migrations
3. Implement entities and repositories
4. Build service layer with gateway abstraction
5. Implement first gateway adapter (Razorpay or Stripe)
