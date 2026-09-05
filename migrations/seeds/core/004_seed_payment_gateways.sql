-- Seed: 004_seed_payment_gateways.sql
-- Description: Core payment gateway catalog for the payment module
-- Environment: ALL (core data) — the gateway catalog must exist whenever the backend runs
-- Notes: Idempotent upserts keyed on the natural unique columns.

-- ============================================================================
-- 1. PAYMENT GATEWAY - Razorpay
-- ============================================================================
INSERT INTO payment_gateway (
    code, name, description, logo_file_id, is_active,
    supported_countries, supported_currencies, supported_payment_methods,
    webhook_url, created_at, updated_at
)
VALUES (
    'razorpay',
    'Razorpay',
    'Leading payment solution in India with support for cards, UPI, wallets, netbanking, EMI and Pay Later.',
    NULL,  -- logo uploaded later via file service
    TRUE,
    ARRAY['IN'],
    ARRAY['INR'],
    ARRAY['card', 'upi', 'wallet', 'netbanking', 'emi', 'cardless_emi', 'paylater'],
    NULL,
    NOW(),
    NOW()
)
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    logo_file_id = EXCLUDED.logo_file_id,
    is_active = EXCLUDED.is_active,
    supported_countries = EXCLUDED.supported_countries,
    supported_currencies = EXCLUDED.supported_currencies,
    supported_payment_methods = EXCLUDED.supported_payment_methods,
    webhook_url = EXCLUDED.webhook_url,
    updated_at = NOW();

-- ============================================================================
-- 2. PAYMENT GATEWAY FIELD - Razorpay configuration fields
-- ============================================================================
INSERT INTO payment_gateway_field (
    gateway_id, field_name, display_name, field_type, description, placeholder,
    is_required, is_sensitive, display_order, validation_rules,
    created_at, updated_at
)
SELECT
    g.id, v.field_name, v.display_name, v.field_type, v.description, v.placeholder,
    v.is_required, v.is_sensitive, v.display_order, v.validation_rules,
    NOW(), NOW()
FROM payment_gateway g
JOIN (
    VALUES
        ('key_id', 'Key ID', 'string',
         'Your Razorpay Key ID (starts with rzp_test_ or rzp_live_)',
         'rzp_live_xxxxxxxxx', TRUE, FALSE, 1,
         '{"pattern": "^rzp_(test|live)_[a-zA-Z0-9]+$", "custom_error_message": "Must be a valid Razorpay key ID"}'::jsonb),
        ('key_secret', 'Key Secret', 'string',
         'Your Razorpay Key Secret (keep this confidential)',
         'Enter your key secret', TRUE, TRUE, 2,
         '{"min_length": 20}'::jsonb),
        ('webhook_secret', 'Webhook Secret', 'string',
         'Webhook signature secret for verifying webhook authenticity',
         'Enter webhook secret', TRUE, TRUE, 3,
         '{"min_length": 16}'::jsonb),
        ('account_id', 'Account ID', 'string',
         'Razorpay Account ID (optional, required only for route/transfer features)',
         'acc_xxxxx', FALSE, FALSE, 4,
         '{"pattern": "^acc_[a-zA-Z0-9]+$"}'::jsonb)
) AS v(field_name, display_name, field_type, description, placeholder, is_required, is_sensitive, display_order, validation_rules)
    ON v.field_name = v.field_name
WHERE g.code = 'razorpay'
ON CONFLICT (gateway_id, field_name) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    field_type = EXCLUDED.field_type,
    description = EXCLUDED.description,
    placeholder = EXCLUDED.placeholder,
    is_required = EXCLUDED.is_required,
    is_sensitive = EXCLUDED.is_sensitive,
    display_order = EXCLUDED.display_order,
    validation_rules = EXCLUDED.validation_rules,
    updated_at = NOW();
