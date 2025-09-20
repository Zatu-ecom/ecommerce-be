-- Demo Data for E-commerce Platform
-- Run this SQL in your PostgreSQL database

-- 1. Insert Roles
INSERT INTO roles (id, name, description, level, created_at, updated_at) VALUES
(1, 'ADMIN', 'System administrator with full access', 1, NOW(), NOW()),
(2, 'SELLER', 'Seller/merchant who can manage products and orders', 2, NOW(), NOW()),
(3, 'CUSTOMER', 'Regular customer who can browse and purchase', 3, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Reset sequence for roles
SELECT setval('roles_id_seq', (SELECT MAX(id) FROM roles));

-- 2. Insert Plans
INSERT INTO plans (name, description, price, currency, billing_cycle, is_popular, sort_order, trial_days, created_at, updated_at) VALUES
('Free Starter', 'Perfect for testing - up to 10 products, basic features', 0.00, 'USD', 'monthly', false, 1, 14, NOW(), NOW()),
('Basic', 'Great for small businesses - up to 100 products, email support', 29.99, 'USD', 'monthly', false, 2, 7, NOW(), NOW()),
('Professional', 'Most popular for growing businesses - unlimited products, priority support', 79.99, 'USD', 'monthly', true, 3, 7, NOW(), NOW()),
('Enterprise', 'For large businesses - custom features, dedicated support', 199.99, 'USD', 'monthly', false, 4, 14, NOW(), NOW()),
('Yearly Basic', 'Basic plan with yearly billing - 2 months free!', 299.99, 'USD', 'yearly', false, 5, 14, NOW(), NOW()),
('Yearly Pro', 'Professional plan with yearly billing - 2 months free!', 799.99, 'USD', 'yearly', false, 6, 14, NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- 3. Update existing users to have proper role assignments
-- This updates any existing users to have the CUSTOMER role
UPDATE users SET role_id = 3 WHERE role_id IS NULL OR role_id = 0;

-- 4. Insert Demo Seller Users (with hashed password 'password123')
INSERT INTO users (first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, seller_id, created_at, updated_at) VALUES
-- Admin user
('Admin', 'User', 'admin@ecommerce.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0001', '1990-01-01', 'Other', true, 1, 0, NOW(), NOW()),

-- Seller users (they will be their own sellers)
('John', 'Smith', 'john.seller@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0002', '1985-03-15', 'Male', true, 2, 0, NOW(), NOW()),
('Sarah', 'Johnson', 'sarah.seller@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0003', '1988-07-22', 'Female', true, 2, 0, NOW(), NOW()),
('Mike', 'Brown', 'mike.seller@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0004', '1982-11-30', 'Male', true, 2, 0, NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

-- 5. Update seller_id for seller users (they are their own sellers)
UPDATE users SET seller_id = id WHERE email IN ('john.seller@example.com', 'sarah.seller@example.com', 'mike.seller@example.com');

-- 6. Insert Demo Customer Users
INSERT INTO users (first_name, last_name, email, password, phone, date_of_birth, gender, is_active, role_id, seller_id, created_at, updated_at) VALUES
-- Customer users (associated with different sellers)
('Alice', 'Wilson', 'alice@customer.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0005', '1992-05-10', 'Female', true, 3, (SELECT id FROM users WHERE email = 'john.seller@example.com'), NOW(), NOW()),
('Bob', 'Davis', 'bob@customer.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0006', '1990-09-18', 'Male', true, 3, (SELECT id FROM users WHERE email = 'john.seller@example.com'), NOW(), NOW()),
('Carol', 'Garcia', 'carol@customer.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0007', '1995-12-03', 'Female', true, 3, (SELECT id FROM users WHERE email = 'sarah.seller@example.com'), NOW(), NOW()),
('David', 'Martinez', 'david@customer.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '+1-555-0008', '1987-04-25', 'Male', true, 3, (SELECT id FROM users WHERE email = 'mike.seller@example.com'), NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

-- 7. Insert Seller Profiles
INSERT INTO seller_profiles (user_id, business_name, business_logo, tax_id, is_verified, created_at, updated_at) VALUES
((SELECT id FROM users WHERE email = 'john.seller@example.com'), 'TechGadgets Pro', 'https://example.com/logos/techgadgets.png', 'TAX123456789', true, NOW(), NOW()),
((SELECT id FROM users WHERE email = 'sarah.seller@example.com'), 'Fashion Forward', 'https://example.com/logos/fashionforward.png', 'TAX987654321', true, NOW(), NOW()),
((SELECT id FROM users WHERE email = 'mike.seller@example.com'), 'Home & Garden Essentials', 'https://example.com/logos/homegarden.png', 'TAX456789123', false, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING;

-- 8. Insert Active Subscriptions
INSERT INTO subscriptions (seller_id, plan_id, status, start_date, end_date, payment_transaction_id, created_at, updated_at) VALUES
-- John Smith - Professional Plan (Active)
((SELECT id FROM users WHERE email = 'john.seller@example.com'), 
 (SELECT id FROM plans WHERE name = 'Professional'), 
 'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', 'txn_prof_john_001', NOW(), NOW()),

-- Sarah Johnson - Basic Plan (Active)
((SELECT id FROM users WHERE email = 'sarah.seller@example.com'), 
 (SELECT id FROM plans WHERE name = 'Basic'), 
 'active', NOW() - INTERVAL '10 days', NOW() + INTERVAL '20 days', 'txn_basic_sarah_001', NOW(), NOW()),

-- Mike Brown - Free Starter (Trial ending soon)
((SELECT id FROM users WHERE email = 'mike.seller@example.com'), 
 (SELECT id FROM plans WHERE name = 'Free Starter'), 
 'active', NOW() - INTERVAL '12 days', NOW() + INTERVAL '2 days', NULL, NOW(), NOW());

-- 9. Verification Queries
-- Check roles
SELECT * FROM roles ORDER BY level;

-- Check plans
SELECT name, price, currency, billing_cycle, is_popular FROM plans ORDER BY sort_order;

-- Check users with their roles
SELECT u.first_name, u.last_name, u.email, r.name as role, u.seller_id 
FROM users u 
JOIN roles r ON u.role_id = r.id 
ORDER BY r.level, u.id;

-- Check seller profiles
SELECT sp.business_name, sp.tax_id, sp.is_verified, u.email 
FROM seller_profiles sp 
JOIN users u ON sp.user_id = u.id;

-- Check active subscriptions
SELECT u.email, sp.business_name, p.name as plan, s.status, s.start_date, s.end_date
FROM subscriptions s
JOIN users u ON s.seller_id = u.id
JOIN seller_profiles sp ON s.seller_id = sp.user_id
JOIN plans p ON s.plan_id = p.id
WHERE s.status = 'active'
ORDER BY s.end_date;
