-- Seed: 001_seed_roles.sql
-- Description: Core role data required for system to function
-- Environment: ALL (core data)
-- Created: 2025-10-23

-- ------------------------------
-- Insert Roles (Required for all environments)
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
