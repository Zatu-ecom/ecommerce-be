-- Seed: 001_seed_user_data.sql
-- Description: Insert demo data for user service (matches entity structure)
-- Created: 2025-10-23
-- Updated: 2025-10-23 - Cleaned for testcontainers

-- ------------------------------
-- Insert Roles
-- ------------------------------
INSERT INTO role (id, name, description, level, created_at, updated_at) VALUES
(1, 'ADMIN', 'System administrator with full access', 1, NOW(), NOW()),
(2, 'SELLER', 'Seller/merchant who can manage products and orders', 2, NOW(), NOW()),
(3, 'CUSTOMER', 'Regular customer who can browse and purchase', 3, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    level = EXCLUDED.level,
    updated_at = NOW();

SELECT setval('role_id_seq', (SELECT MAX(id) FROM role));

-- ------------------------------
-- Insert Plans
-- ------------------------------
INSERT INTO plan (id, name, description, price, currency, billing_cycle, is_popular, sort_order, trial_days, created_at, updated_at) VALUES
(1, 'Free Starter', 'Up to 10 products, basic features', 0.00, 'USD', 'monthly', false, 1, 14, NOW(), NOW()),
(2, 'Basic', 'Up to 100 products, email support', 29.99, 'USD', 'monthly', false, 2, 7, NOW(), NOW()),
(3, 'Professional', 'Unlimited products, priority support', 79.99, 'USD', 'monthly', true, 3, 7, NOW(), NOW()),
(4, 'Enterprise', 'Custom features, dedicated support', 199.99, 'USD', 'monthly', false, 4, 14, NOW(), NOW()),
(5, 'Yearly Basic', 'Basic plan with yearly billing', 299.99, 'USD', 'yearly', false, 5, 14, NOW(), NOW()),
(6, 'Yearly Pro', 'Professional plan with yearly billing', 799.99, 'USD', 'yearly', false, 6, 14, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    is_popular = EXCLUDED.is_popular,
    trial_days = EXCLUDED.trial_days,
    updated_at = NOW();

SELECT setval('plan_id_seq', (SELECT MAX(id) FROM plan));

-- ------------------------------
-- Insert Users
-- ------------------------------
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, created_at, updated_at) VALUES
-- Admin
(1, 'Admin', 'User', 'admin@ecommerce.com', '$2a$12$iD08YgeBSaoVwXujHrWrBeQM4YwsnRfA8p.J0r8GCQLNZ6G4G8Kta', '+1234567890', NULL, NULL, true, 1, NOW(), NOW()),
-- Sellers
(2, 'John', 'Seller', 'john.seller@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567891', '1985-05-15', 'male', true, 2, NOW(), NOW()),
(3, 'Jane', 'Merchant', 'jane.merchant@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567892', '1990-08-22', 'female', true, 2, NOW(), NOW()),
(4, 'Bob', 'Store', 'bob.store@example.com', '$2a$12$UgEs7psVsJ1/urkc0gb1M.yDs8BPY3iuWrL33iIbGPApBMOgeUUqS', '+1234567893', '1988-03-10', 'male', true, 2, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    date_of_birth = EXCLUDED.date_of_birth,
    gender = EXCLUDED.gender,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    updated_at = NOW();

SELECT setval('user_id_seq', (SELECT MAX(id) FROM "user"));

-- ------------------------------
-- Insert Seller Profiles
-- ------------------------------
INSERT INTO seller_profile (user_id, business_name, business_logo, tax_id, is_verified, created_at, updated_at) VALUES
(2, 'Tech Gadgets Pro', 'https://example.com/logos/tech-gadgets-pro.png', 'TAX-001-2024', true, NOW(), NOW()),
(3, 'Fashion Forward', 'https://example.com/logos/fashion-forward.png', 'TAX-002-2024', true, NOW(), NOW()),
(4, 'Home & Living Store', 'https://example.com/logos/home-living.png', 'TAX-003-2024', true, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET
    business_name = EXCLUDED.business_name,
    business_logo = EXCLUDED.business_logo,
    tax_id = EXCLUDED.tax_id,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- Update seller_id in user table
UPDATE "user" SET seller_id = 2 WHERE id = 2;
UPDATE "user" SET seller_id = 3 WHERE id = 3;
UPDATE "user" SET seller_id = 4 WHERE id = 4;


-- ------------------------------
-- Insert Customers
-- ------------------------------
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, seller_id, created_at, updated_at) VALUES
(5, 'Alice', 'Johnson', 'alice.j@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567894', '1992-11-05', 'female', true, 3, 2, NOW(), NOW()),
(6, 'Michael', 'Smith', 'michael.s@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567895', '1995-07-18', 'male', true, 3, 3, NOW(), NOW()),
(7, 'Sarah', 'Williams', 'sarah.w@example.com', '$2a$12$KChTvYVo4KtWdT.4N1ijbuL/Wm.lv.nndgndCalCwDrf03L1D9LLy', '+1234567896', '1993-02-28', 'female', true, 3, 4, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    date_of_birth = EXCLUDED.date_of_birth,
    gender = EXCLUDED.gender,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    seller_id = EXCLUDED.seller_id,
    updated_at = NOW();


-- ------------------------------
-- Insert Subscriptions
-- ------------------------------
INSERT INTO subscription (id, seller_id, plan_id, status, start_date, end_date, payment_transaction_id, created_at, updated_at) VALUES
(1, 2, 3, 'active', NOW() - INTERVAL '30 days', NOW() + INTERVAL '11 months', 'TXN-001-2024', NOW(), NOW()),
(2, 3, 4, 'active', NOW() - INTERVAL '60 days', NOW() + INTERVAL '10 months', 'TXN-002-2024', NOW(), NOW()),
(3, 4, 2, 'trialing', NOW() - INTERVAL '5 days', NOW() + INTERVAL '1 month', NULL, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    plan_id = EXCLUDED.plan_id,
    status = EXCLUDED.status,
    updated_at = NOW();

SELECT setval('subscription_id_seq', (SELECT MAX(id) FROM subscription));

-- ------------------------------
-- Insert Addresses
-- ------------------------------
INSERT INTO "address" (id, user_id, street, city, state, zip_code, country, is_default, created_at, updated_at) VALUES
-- Customer addresses
(1, 5, '123 Main Street, Apt 4B', 'New York', 'NY', '10001', 'USA', true, NOW(), NOW()),
(2, 5, '456 Oak Avenue', 'Brooklyn', 'NY', '11201', 'USA', false, NOW(), NOW()),
(3, 6, '789 Pine Road, Suite 200', 'Los Angeles', 'CA', '90001', 'USA', true, NOW(), NOW()),
(4, 7, '321 Elm Street', 'Chicago', 'IL', '60601', 'USA', true, NOW(), NOW()),
-- Seller addresses
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

SELECT setval('address_id_seq', (SELECT MAX(id) FROM "address"));

-- ------------------------------
-- Final Summary Notice
-- ------------------------------
DO $$
BEGIN
    RAISE NOTICE 'Seed data completed successfully!';
    RAISE NOTICE 'Roles: %', (SELECT COUNT(*) FROM role);
    RAISE NOTICE 'Plans: %', (SELECT COUNT(*) FROM plan);
    RAISE NOTICE 'Users: %', (SELECT COUNT(*) FROM "user");
    RAISE NOTICE 'Seller Profiles: %', (SELECT COUNT(*) FROM seller_profile);
    RAISE NOTICE 'Subscriptions: %', (SELECT COUNT(*) FROM subscription);
    RAISE NOTICE 'Addresses: %', (SELECT COUNT(*) FROM "address");
END $$;
