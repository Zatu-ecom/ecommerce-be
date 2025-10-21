-- Seed: 001_seed_user_data.sql
-- Description: Insert demo data for user service (matches corrected entity structure)
-- Created: 2025-10-21
-- Updated: 2025-10-21 - Fixed to match actual entity definitions

-- Insert Roles (if not exists)
INSERT INTO role (id, name, description, level, created_at, updated_at) VALUES
(1, 'ADMIN', 'System administrator with full access', 1, NOW(), NOW()),
(2, 'SELLER', 'Seller/merchant who can manage products and orders', 2, NOW(), NOW()),
(3, 'CUSTOMER', 'Regular customer who can browse and purchase', 3, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    level = EXCLUDED.level,
    updated_at = NOW();

-- Reset sequence for roles
SELECT setval('role_id_seq', (SELECT MAX(id) FROM role));

-- Insert Plans (if not exists)
INSERT INTO plan (id, name, description, price, currency, billing_cycle, is_popular, sort_order, trial_days, created_at, updated_at) VALUES
(1, 'Free Starter', 'Perfect for testing - up to 10 products, basic features', 0.00, 'USD', 'monthly', false, 1, 14, NOW(), NOW()),
(2, 'Basic', 'Great for small businesses - up to 100 products, email support', 29.99, 'USD', 'monthly', false, 2, 7, NOW(), NOW()),
(3, 'Professional', 'Most popular for growing businesses - unlimited products, priority support', 79.99, 'USD', 'monthly', true, 3, 7, NOW(), NOW()),
(4, 'Enterprise', 'For large businesses - custom features, dedicated support', 199.99, 'USD', 'monthly', false, 4, 14, NOW(), NOW()),
(5, 'Yearly Basic', 'Basic plan with yearly billing - 2 months free!', 299.99, 'USD', 'yearly', false, 5, 14, NOW(), NOW()),
(6, 'Yearly Pro', 'Professional plan with yearly billing - 2 months free!', 799.99, 'USD', 'yearly', false, 6, 14, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    is_popular = EXCLUDED.is_popular,
    trial_days = EXCLUDED.trial_days,
    updated_at = NOW();

-- Reset sequence for plans
SELECT setval('plan_id_seq', (SELECT MAX(id) FROM plan));

-- Insert Demo Admin User (password: admin123)
-- Note: Hash generated with bcrypt cost 10
INSERT INTO "user" (id, first_name, last_name, email, password, phone, is_active, role_id, created_at, updated_at) VALUES
(1, 'Admin', 'User', 'admin@ecommerce.com', '$2a$12$iD08YgeBSaoVwXujHrWrBeQM4YwsnRfA8p.J0r8GCQLNZ6G4G8Kta', '+1234567890', true, 1, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    updated_at = NOW();

-- Insert Demo Seller Users (password: seller123)
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, created_at, updated_at) VALUES
(2, 'John', 'Seller', 'john.seller@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567891', '1985-05-15', 'male', true, 2, NOW(), NOW()),
(3, 'Jane', 'Merchant', 'jane.merchant@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567892', '1990-08-22', 'female', true, 2, NOW(), NOW()),
(4, 'Bob', 'Store', 'bob.store@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567893', '1988-03-10', 'male', true, 2, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    updated_at = NOW();

-- Insert Demo Customer Users (password: customer123)
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, created_at, updated_at) VALUES
(5, 'Alice', 'Johnson', 'alice.j@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567894', '1992-11-05', 'female', true, 3, NOW(), NOW()),
(6, 'Michael', 'Smith', 'michael.s@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567895', '1995-07-18', 'male', true, 3, NOW(), NOW()),
(7, 'Sarah', 'Williams', 'sarah.w@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567896', '1993-02-28', 'female', true, 3, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    updated_at = NOW();

-- Reset sequence for users
SELECT setval('"user"_id_seq', (SELECT MAX(id) FROM "user"));

-- Insert Seller Profiles (user_id is PRIMARY KEY, not separate id)
INSERT INTO seller_profile (user_id, business_name, business_logo, tax_id, is_verified, created_at, updated_at) VALUES
(2, 'Tech Gadgets Pro', 'https://example.com/logos/tech-gadgets-pro.png', 'TAX-001-2024', true, NOW(), NOW()),
(3, 'Fashion Forward', 'https://example.com/logos/fashion-forward.png', 'TAX-002-2024', true, NOW(), NOW()),
(4, 'Home & Living Store', 'https://example.com/logos/home-living.png', 'TAX-003-2024', true, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET
    business_name = EXCLUDED.business_name,
    business_logo = EXCLUDED.business_logo,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- Update users with seller_id (references seller_profiles.user_id)
UPDATE users SET seller_id = 2 WHERE id = 2;
UPDATE users SET seller_id = 3 WHERE id = 3;
UPDATE users SET seller_id = 4 WHERE id = 4;

-- Insert Subscriptions for Sellers (uses seller_id, not user_id)
INSERT INTO subscription (id, seller_id, plan_id, status, start_date, end_date, payment_transaction_id, created_at, updated_at) VALUES
(1, 2, 3, 'active', NOW() - INTERVAL '30 days', NOW() + INTERVAL '11 months', 'TXN-001-2024', NOW(), NOW()),
(2, 3, 4, 'active', NOW() - INTERVAL '60 days', NOW() + INTERVAL '10 months', 'TXN-002-2024', NOW(), NOW()),
(3, 4, 2, 'trialing', NOW() - INTERVAL '5 days', NOW() + INTERVAL '1 month', NULL, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    plan_id = EXCLUDED.plan_id,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Reset sequence for subscriptions
SELECT setval('subscription_id_seq', (SELECT MAX(id) FROM subscription));

-- Insert Demo Addresses (corrected field names)
INSERT INTO "address" (id, user_id, street, city, state, zip_code, country, is_default, created_at, updated_at) VALUES
-- Customer addresses
(1, 5, '123 Main Street, Apt 4B', 'New York', 'NY', '10001', 'USA', true, NOW(), NOW()),
(2, 5, '456 Oak Avenue', 'Brooklyn', 'NY', '11201', 'USA', false, NOW(), NOW()),
(3, 6, '789 Pine Road, Suite 200', 'Los Angeles', 'CA', '90001', 'USA', true, NOW(), NOW()),
(4, 7, '321 Elm Street', 'Chicago', 'IL', '60601', 'USA', true, NOW(), NOW()),
-- Seller business addresses
(5, 2, '1000 Tech Park Drive, Building A', 'San Jose', 'CA', '95110', 'USA', true, NOW(), NOW()),
(6, 3, '500 Fashion Boulevard, Floor 3', 'Miami', 'FL', '33101', 'USA', true, NOW(), NOW()),
(7, 4, '750 Home Avenue', 'Seattle', 'WA', '98101', 'USA', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    street = EXCLUDED.street,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip_code = EXCLUDED.zip_code,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

-- Reset sequence for addresses
SELECT setval('"address"_id_seq', (SELECT MAX(id) FROM "address"));

-- Display summary
DO $$
BEGIN
    RAISE NOTICE 'User seed data completed successfully!';
    RAISE NOTICE 'Created % roles', (SELECT COUNT(*) FROM role);
    RAISE NOTICE 'Created % plans', (SELECT COUNT(*) FROM plan);
    RAISE NOTICE 'Created % users', (SELECT COUNT(*) FROM "user");
    RAISE NOTICE 'Created % seller profiles', (SELECT COUNT(*) FROM seller_profile);
    RAISE NOTICE 'Created % subscriptions', (SELECT COUNT(*) FROM subscription);
    RAISE NOTICE 'Created % addresses', (SELECT COUNT(*) FROM "address");
END $$;
(2, 3, 4, 'active', NOW() - INTERVAL '60 days', NOW() + INTERVAL '10 months', NULL, true, NOW(), NOW()),
(3, 4, 2, 'trial', NOW() - INTERVAL '5 days', NOW() + INTERVAL '1 month', NOW() + INTERVAL '2 days', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    plan_id = EXCLUDED.plan_id,
    status = EXCLUDED.status,
    auto_renew = EXCLUDED.auto_renew,
    updated_at = NOW();

-- Reset sequence for subscriptions
SELECT setval('subscription_id_seq', (SELECT MAX(id) FROM subscription));

-- Insert Demo Addresses
INSERT INTO "address" (id, user_id, address_line1, address_line2, city, state, postal_code, country, is_default, address_type, created_at, updated_at) VALUES
-- Customer addresses
(1, 5, '123 Main Street', 'Apt 4B', 'New York', 'NY', '10001', 'USA', true, 'both', NOW(), NOW()),
(2, 5, '456 Oak Avenue', NULL, 'Brooklyn', 'NY', '11201', 'USA', false, 'shipping', NOW(), NOW()),
(3, 6, '789 Pine Road', 'Suite 200', 'Los Angeles', 'CA', '90001', 'USA', true, 'both', NOW(), NOW()),
(4, 7, '321 Elm Street', NULL, 'Chicago', 'IL', '60601', 'USA', true, 'both', NOW(), NOW()),
-- Seller business addresses
(5, 2, '1000 Tech Park Drive', 'Building A', 'San Jose', 'CA', '95110', 'USA', true, 'billing', NOW(), NOW()),
(6, 3, '500 Fashion Boulevard', 'Floor 3', 'Miami', 'FL', '33101', 'USA', true, 'billing', NOW(), NOW()),
(7, 4, '750 Home Avenue', NULL, 'Seattle', 'WA', '98101', 'USA', true, 'billing', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    address_line1 = EXCLUDED.address_line1,
    address_line2 = EXCLUDED.address_line2,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    postal_code = EXCLUDED.postal_code,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

-- Reset sequence for addresses
SELECT setval('"address"_id_seq', (SELECT MAX(id) FROM "address"));

-- Display summary
DO $$
BEGIN
    RAISE NOTICE 'User seed data completed successfully!';
    RAISE NOTICE 'Created % roles', (SELECT COUNT(*) FROM role);
    RAISE NOTICE 'Created % plans', (SELECT COUNT(*) FROM plan);
    RAISE NOTICE 'Created % users', (SELECT COUNT(*) FROM "user");
    RAISE NOTICE 'Created % seller profiles', (SELECT COUNT(*) FROM seller_profile);
    RAISE NOTICE 'Created % subscriptions', (SELECT COUNT(*) FROM subscription);
    RAISE NOTICE 'Created % addresses', (SELECT COUNT(*) FROM "address");
END $$;
