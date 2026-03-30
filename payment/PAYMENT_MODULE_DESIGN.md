# 💳 Payment Module - Database Design & Architecture

> **Last Updated**: December 28, 2025  
> **Status**: Design Phase  
> **Author**: Development Team

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Key Design Considerations](#key-design-considerations)
3. [Database Schema](#database-schema)
4. [Status Enums](#status-enums)
5. [Architecture: Gateway Abstraction](#architecture-gateway-abstraction)
6. [MVP Strategy](#mvp-strategy)
7. [API Endpoints (Planned)](#api-endpoints-planned)
8. [Open Questions](#open-questions)

---

## 🎯 Overview

The Payment Module provides a flexible, multi-gateway payment system that supports:

- **Multiple Payment Gateways**: Stripe, Razorpay, PayPal, etc.
- **Region-Based Configuration**: Different gateways for different countries
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

1. **Gateway Configuration** - Admin can enable/disable gateways per country/region
2. **Payment Method Abstraction** - Cards, UPI, Wallets, Bank Transfer, BNPL all work the same way
3. **Gateway-Agnostic Transactions** - Store our own transaction records, link to gateway references
4. **Fallback Support** - If primary gateway fails, try secondary

---

## 📊 Database Schema

### 1. `payment_gateway` - Available Payment Gateways

Master table of all supported payment gateways in the system.

```sql
CREATE TABLE payment_gateway (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,        -- 'stripe', 'razorpay', 'paypal'
    name VARCHAR(100) NOT NULL,              -- 'Stripe', 'Razorpay'
    description TEXT,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    supported_currencies TEXT[],             -- ['USD', 'EUR', 'INR']
    supported_payment_methods TEXT[],        -- ['card', 'upi', 'wallet', 'bank_transfer']
    webhook_url VARCHAR(500),                -- Our endpoint for this gateway
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

**Example Data:**

```sql
INSERT INTO payment_gateway (code, name, supported_currencies, supported_payment_methods) VALUES
('stripe', 'Stripe', ARRAY['USD', 'EUR', 'GBP', 'INR'], ARRAY['card', 'bank_transfer']),
('razorpay', 'Razorpay', ARRAY['INR'], ARRAY['card', 'upi', 'wallet', 'bank_transfer']),
('paypal', 'PayPal', ARRAY['USD', 'EUR', 'GBP'], ARRAY['wallet', 'card']);
```

---

### 2. `payment_gateway_config` - Gateway Credentials per Seller/Region

Stores API credentials and configuration for each gateway per seller and region.

```sql
CREATE TABLE payment_gateway_config (
    id BIGSERIAL PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES "user"(id),
    gateway_id BIGINT NOT NULL REFERENCES payment_gateway(id),
    country_code VARCHAR(3),                 -- 'US', 'IN', 'GB' (NULL = all countries)
    environment VARCHAR(20) NOT NULL,        -- 'sandbox', 'production'
    credentials JSONB NOT NULL,              -- Encrypted: {"api_key": "...", "secret": "..."}
    is_active BOOLEAN DEFAULT true,
    priority INT DEFAULT 0,                  -- Higher = preferred (for fallback)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(seller_id, gateway_id, country_code, environment)
);
```

**Credentials JSONB Structure (encrypted at application level):**

```json
{
  "api_key": "sk_live_xxxxx",
  "secret_key": "whsec_xxxxx",
  "webhook_secret": "whsec_xxxxx",
  "merchant_id": "merchant_xxxxx"
}
```

---

### 3. `payment_method` - Saved Payment Methods

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
    wallet_provider VARCHAR(50),             -- 'paytm', 'phonepe'

    is_default BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,

    metadata JSONB,                          -- Additional gateway-specific data

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_payment_method_user_id ON payment_method(user_id);
CREATE INDEX idx_payment_method_gateway_id ON payment_method(gateway_id);
```

---

### 4. `payment_transaction` - Core Transaction Table

The main table storing all payment transactions.

```sql
CREATE TABLE payment_transaction (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(50) NOT NULL UNIQUE,  -- Our internal ID: 'TXN_20251228_XXXXX'

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
    amount_cents BIGINT NOT NULL,            -- Total amount in cents (1999 = $19.99)
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
CREATE INDEX idx_payment_transaction_order_id ON payment_transaction(order_id);
CREATE INDEX idx_payment_transaction_user_id ON payment_transaction(user_id);
CREATE INDEX idx_payment_transaction_seller_id ON payment_transaction(seller_id);
CREATE INDEX idx_payment_transaction_status ON payment_transaction(status);
CREATE INDEX idx_payment_transaction_gateway_payment_id ON payment_transaction(gateway_payment_id);
CREATE INDEX idx_payment_transaction_created_at ON payment_transaction(created_at);
```

**payment_method_details JSONB Example:**

```json
{
  "card_brand": "visa",
  "card_last_four": "4242",
  "card_exp_month": 12,
  "card_exp_year": 2027
}
```

---

### 5. `payment_refund` - Refunds Table

Tracks all refund requests and their status.

```sql
CREATE TABLE payment_refund (
    id BIGSERIAL PRIMARY KEY,
    refund_id VARCHAR(50) NOT NULL UNIQUE,   -- 'RFD_20251228_XXXXX'

    transaction_id BIGINT NOT NULL REFERENCES payment_transaction(id),
    order_id BIGINT REFERENCES "order"(id),

    -- Gateway reference
    gateway_refund_id VARCHAR(255),          -- Gateway's refund ID

    -- Amount (stored in cents for precision)
    currency VARCHAR(3) NOT NULL,
    amount_cents BIGINT NOT NULL,            -- Refund amount in cents (1999 = $19.99)

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
CREATE INDEX idx_payment_refund_transaction_id ON payment_refund(transaction_id);
CREATE INDEX idx_payment_refund_order_id ON payment_refund(order_id);
CREATE INDEX idx_payment_refund_status ON payment_refund(status);
```

---

### 6. `payment_webhook_log` - Webhook Audit Trail

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
CREATE INDEX idx_payment_webhook_log_gateway_id ON payment_webhook_log(gateway_id);
CREATE INDEX idx_payment_webhook_log_event_id ON payment_webhook_log(event_id);
CREATE INDEX idx_payment_webhook_log_status ON payment_webhook_log(status);
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
}
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Payment Service                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           PaymentGateway Interface                   │   │
│  │  • InitiatePayment(order, method) → PaymentSession  │   │
│  │  • CapturePayment(transactionId) → Transaction      │   │
│  │  • RefundPayment(transactionId, amount) → Refund    │   │
│  │  • VerifyWebhook(payload, signature) → bool         │   │
│  │  • GetPaymentMethods(userId) → []PaymentMethod      │   │
│  └─────────────────────────────────────────────────────┘   │
│        ▲              ▲              ▲              ▲       │
│        │              │              │              │       │
│  ┌─────┴────┐  ┌─────┴────┐  ┌─────┴────┐  ┌─────┴────┐   │
│  │  Stripe  │  │ Razorpay │  │  PayPal  │  │  Future  │   │
│  │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Gateway  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Gateway Selection Logic

```go
func (s *PaymentService) SelectGateway(sellerID uint, country string, currency string) (PaymentGateway, error) {
    // 1. Get active gateway configs for seller + country
    configs, err := s.configRepo.FindBySellerAndCountry(sellerID, country)

    // 2. Filter by supported currency
    // 3. Sort by priority
    // 4. Return highest priority gateway

    // 5. Fallback to default gateway if no config found
}
```

---

## 🚀 MVP Strategy

### Phase 1: MVP (Week 1-2)

**Goal**: Get payments working with ONE gateway

- [ ] Single Gateway Integration (Stripe OR Razorpay)
- [ ] Tables: `payment_gateway`, `payment_transaction`, `payment_refund`, `payment_webhook_log`
- [ ] Payment Methods: Card only
- [ ] Admin: Hardcode gateway config (no UI)
- [ ] Basic webhook handling

**Endpoints:**

- `POST /api/payments/initiate` - Start payment
- `POST /api/payments/:id/capture` - Capture authorized payment
- `POST /api/payments/webhook/:gateway` - Handle webhooks
- `GET /api/payments/:id` - Get transaction details

### Phase 2: Multi-Gateway (Week 3-4)

- [ ] Add `payment_gateway_config` table
- [ ] Admin UI to configure gateways per region
- [ ] Add second gateway for fallback
- [ ] Gateway selection by country/currency

### Phase 3: Full Features (Week 5+)

- [ ] Saved payment methods (`payment_method` table)
- [ ] Multiple payment methods (UPI, wallets, BNPL)
- [ ] Automatic gateway selection by country
- [ ] Retry logic with fallback gateways
- [ ] COD (Cash on Delivery) support
- [ ] Split payments (multi-seller)

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

### Admin APIs

| Method | Endpoint                             | Description          |
| ------ | ------------------------------------ | -------------------- |
| `GET`  | `/api/admin/payment-gateways`        | List all gateways    |
| `POST` | `/api/admin/payment-gateways`        | Add a new gateway    |
| `PUT`  | `/api/admin/payment-gateways/:id`    | Update gateway       |
| `GET`  | `/api/admin/payment-gateway-configs` | List gateway configs |
| `POST` | `/api/admin/payment-gateway-configs` | Add gateway config   |

### Webhook Endpoints

| Method | Endpoint                 | Description              |
| ------ | ------------------------ | ------------------------ |
| `POST` | `/api/webhooks/stripe`   | Stripe webhook handler   |
| `POST` | `/api/webhooks/razorpay` | Razorpay webhook handler |
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

---

## 📚 References

- [Stripe API Documentation](https://stripe.com/docs/api)
- [Razorpay API Documentation](https://razorpay.com/docs/api/)
- [PayPal REST API](https://developer.paypal.com/docs/api/overview/)

---

## 📝 Change Log

| Date       | Version | Changes                 |
| ---------- | ------- | ----------------------- |
| 2025-12-28 | 0.1     | Initial design document |

---

**Next Steps**: Finalize gateway choice → Create migration → Implement entities → Build service layer
