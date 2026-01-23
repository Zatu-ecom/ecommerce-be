-- Seed: 001_seed_users.sql
-- Description: Mock user data for development and testing
-- Environment: DEV/STAGING ONLY (mock data)
-- Created: 2025-10-23

-- ============================================================================
-- MOCK USERS (Admin, Sellers, Customers)
-- ============================================================================

-- ------------------------------
-- Insert Admin User
-- ------------------------------
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, created_at, updated_at) VALUES
(1, 'Admin', 'User', 'admin@ecommerce.com', '$2a$12$iD08YgeBSaoVwXujHrWrBeQM4YwsnRfA8p.J0r8GCQLNZ6G4G8Kta', '+1234567890', NULL, NULL, true, 1, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    is_active = EXCLUDED.is_active,
    role_id = EXCLUDED.role_id,
    updated_at = NOW();

-- ------------------------------
-- Insert Sellers
-- ------------------------------
INSERT INTO "user" (id, first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, created_at, updated_at) VALUES
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

SELECT setval('user_id_seq', (SELECT MAX(id) FROM "user"));

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
INSERT INTO "address" (id, user_id, type, address, landmark, city, state, zip_code, country_id, latitude, longitude, is_default, created_at, updated_at) VALUES
-- Customer addresses
(1, 5, 'HOME', '123 Main Street, Apt 4B', 'Near Central Park', 'New York', 'NY', '10001', 1, 40.7128, -74.0060, true, NOW(), NOW()),
(2, 5, 'WORK', '456 Oak Avenue, Floor 12', 'Empire State Building', 'Brooklyn', 'NY', '11201', 1, 40.6782, -73.9442, false, NOW(), NOW()),
(3, 6, 'HOME', '789 Pine Road, Suite 200', 'Next to Whole Foods', 'Los Angeles', 'CA', '90001', 1, 34.0522, -118.2437, true, NOW(), NOW()),
(4, 7, 'HOME', '321 Elm Street', 'Near Willis Tower', 'Chicago', 'IL', '60601', 1, 41.8781, -87.6298, true, NOW(), NOW()),
-- Seller addresses (office/work)
(5, 2, 'WORK', '1000 Tech Park Drive, Building A', 'Tech Park Campus', 'San Jose', 'CA', '95110', 1, 37.3382, -121.8863, true, NOW(), NOW()),
(6, 3, 'WORK', '500 Fashion Boulevard, Floor 3', 'Fashion District', 'Miami', 'FL', '33101', 1, 25.7617, -80.1918, true, NOW(), NOW()),
(7, 4, 'WORK', '750 Home Avenue', 'Near Pike Place Market', 'Seattle', 'WA', '98101', 1, 47.6062, -122.3321, true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    address = EXCLUDED.address,
    landmark = EXCLUDED.landmark,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip_code = EXCLUDED.zip_code,
    country_id = EXCLUDED.country_id,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

SELECT setval('address_id_seq', (SELECT MAX(id) FROM "address"));

-- ------------------------------
-- Insert Seller Settings (for country/currency)
-- ------------------------------
INSERT INTO seller_settings (seller_id, business_country_id, base_currency_id, settlement_currency_id, display_prices_in_buyer_currency)
SELECT 2, 21, 4, 4, FALSE  -- Seller ID 2, India, INR
WHERE EXISTS (SELECT 1 FROM "user" WHERE id = 2)
  AND NOT EXISTS (SELECT 1 FROM seller_settings WHERE seller_id = 2);

INSERT INTO seller_settings (seller_id, business_country_id, base_currency_id, settlement_currency_id, display_prices_in_buyer_currency)
SELECT 3, 1, 1, 1, FALSE  -- Seller ID 3, USA, USD
WHERE EXISTS (SELECT 1 FROM "user" WHERE id = 3)
  AND NOT EXISTS (SELECT 1 FROM seller_settings WHERE seller_id = 3);

INSERT INTO seller_settings (seller_id, business_country_id, base_currency_id, settlement_currency_id, display_prices_in_buyer_currency)
SELECT 4, 4, 3, 3, FALSE  -- Seller ID 4, UK, GBP
WHERE EXISTS (SELECT 1 FROM "user" WHERE id = 4)
  AND NOT EXISTS (SELECT 1 FROM seller_settings WHERE seller_id = 4);

-- ------------------------------
-- Summary
-- ------------------------------
DO $$
BEGIN
    RAISE NOTICE 'Mock user data seeded successfully!';
    RAISE NOTICE 'Users: %', (SELECT COUNT(*) FROM "user");
    RAISE NOTICE 'Seller Profiles: %', (SELECT COUNT(*) FROM seller_profile);
    RAISE NOTICE 'Subscriptions: %', (SELECT COUNT(*) FROM subscription);
    RAISE NOTICE 'Addresses: %', (SELECT COUNT(*) FROM "address");
    RAISE NOTICE 'Seller Settings: %', (SELECT COUNT(*) FROM seller_settings);
END $$;
