-- Seed: 000_seed_plans.sql
-- Description: Core subscription plans data
-- Environment: ALL (core data required for system)
-- Created: 2026-01-25

-- ============================================================================
-- SUBSCRIPTION PLANS
-- ============================================================================

-- Insert basic plans for subscription system
INSERT INTO plan (id, name, description, price, currency, billing_cycle, is_popular, sort_order, trial_days, created_at, updated_at) VALUES
(1, 'Free', 'Basic free plan with limited features', 0, 'USD', 'monthly', FALSE, 1, 0, NOW(), NOW()),
(2, 'Starter', 'Starter plan for small businesses', 9.99, 'USD', 'monthly', FALSE, 2, 14, NOW(), NOW()),
(3, 'Professional', 'Professional plan for growing businesses', 29.99, 'USD', 'monthly', TRUE, 3, 14, NOW(), NOW()),
(4, 'Enterprise', 'Enterprise plan for large businesses', 99.99, 'USD', 'monthly', FALSE, 4, 30, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    currency = EXCLUDED.currency,
    billing_cycle = EXCLUDED.billing_cycle,
    is_popular = EXCLUDED.is_popular,
    sort_order = EXCLUDED.sort_order,
    trial_days = EXCLUDED.trial_days,
    updated_at = NOW();

SELECT setval('plan_id_seq', (SELECT MAX(id) FROM plan));

-- ------------------------------
-- Summary
-- ------------------------------
DO $$
BEGIN
    RAISE NOTICE 'Core plan data seeded successfully!';
    RAISE NOTICE 'Plans: %', (SELECT COUNT(*) FROM plan);
END $$;
