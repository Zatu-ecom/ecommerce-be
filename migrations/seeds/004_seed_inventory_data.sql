-- Seed: 004_seed_inventory_data.sql
-- Description: Insert demo data for inventory service (locations and inventory)
-- Created: 2025-12-13
-- Purpose: Provides test data for inventory APIs

-- ============================================================================
-- Prerequisites:
-- - 001_seed_user_data.sql (addresses, seller_profiles)
-- - 002_seed_product_data.sql (product_variants)
-- ============================================================================

-- ------------------------------
-- Insert Warehouse/Store Addresses (for locations)
-- Using IDs starting from 10 to avoid conflict with user addresses (1-7)
-- ------------------------------
INSERT INTO "address" (id, user_id, street, city, state, zip_code, country, is_default, created_at, updated_at) VALUES
-- Seller 2 (Tech Gadgets Pro) - Warehouse addresses
(10, 2, '1500 Tech Distribution Center', 'San Jose', 'CA', '95112', 'USA', false, NOW(), NOW()),
(11, 2, '2500 Retail Tech Plaza', 'San Francisco', 'CA', '94102', 'USA', false, NOW(), NOW()),

-- Seller 3 (Fashion Forward) - Warehouse addresses
(12, 3, '800 Fashion Warehouse Blvd', 'Miami', 'FL', '33125', 'USA', false, NOW(), NOW()),
(13, 3, '450 Fashion Outlet Drive', 'Orlando', 'FL', '32801', 'USA', false, NOW(), NOW()),

-- Seller 4 (Home & Living Store) - Warehouse addresses
(14, 4, '3000 Furniture Distribution Hub', 'Seattle', 'WA', '98108', 'USA', false, NOW(), NOW()),
(15, 4, '1200 Home Store Main Street', 'Portland', 'OR', '97201', 'USA', false, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    street = EXCLUDED.street,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip_code = EXCLUDED.zip_code,
    updated_at = NOW();

SELECT setval('address_id_seq', (SELECT MAX(id) FROM "address"));

-- ------------------------------
-- Insert Locations
-- ------------------------------
INSERT INTO "location" (id, seller_id, name, type, is_active, priority, address_id, created_at, updated_at) VALUES
-- Seller 2 (Tech Gadgets Pro) - 2 locations
(1, 2, 'Tech Main Warehouse', 'WAREHOUSE', true, 1, 10, NOW(), NOW()),
(2, 2, 'Tech Downtown Store', 'STORE', true, 2, 11, NOW(), NOW()),

-- Seller 3 (Fashion Forward) - 2 locations
(3, 3, 'Fashion Central Warehouse', 'WAREHOUSE', true, 1, 12, NOW(), NOW()),
(4, 3, 'Fashion Outlet Store', 'STORE', true, 2, 13, NOW(), NOW()),

-- Seller 4 (Home & Living Store) - 2 locations
(5, 4, 'Home Distribution Center', 'WAREHOUSE', true, 1, 14, NOW(), NOW()),
(6, 4, 'Home Living Showroom', 'STORE', true, 2, 15, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    is_active = EXCLUDED.is_active,
    priority = EXCLUDED.priority,
    updated_at = NOW();

SELECT setval('location_id_seq', (SELECT MAX(id) FROM "location"));

-- ------------------------------
-- Insert Inventory Records
-- Mapping variants to locations with realistic stock levels
-- ------------------------------

-- ============================================================================
-- SELLER 2 (Tech Gadgets Pro) - Products 1, 2, 3, 4
-- Locations: 1 (Main Warehouse), 2 (Downtown Store)
-- ============================================================================

INSERT INTO inventory (id, variant_id, location_id, quantity, reserved_quantity, threshold, bin_location, created_at, updated_at) VALUES
-- iPhone 15 Pro variants (1-4) at Tech Main Warehouse (location 1)
(1, 1, 1, 100, 5, 15, 'A-01-01', NOW(), NOW()),   -- NAT-128: Good stock
(2, 2, 1, 75, 3, 10, 'A-01-02', NOW(), NOW()),    -- NAT-256: Good stock
(3, 3, 1, 50, 2, 10, 'A-01-03', NOW(), NOW()),    -- BLU-128: Good stock
(4, 4, 1, 8, 0, 10, 'A-01-04', NOW(), NOW()),     -- BLU-256: Low stock (8 <= 10)

-- iPhone 15 Pro variants at Tech Downtown Store (location 2)
(5, 1, 2, 25, 3, 5, 'S-01-01', NOW(), NOW()),     -- NAT-128: Good stock
(6, 2, 2, 15, 2, 5, 'S-01-02', NOW(), NOW()),     -- NAT-256: Good stock
(7, 3, 2, 0, 0, 5, 'S-01-03', NOW(), NOW()),      -- BLU-128: Out of stock
(8, 4, 2, 5, 1, 5, 'S-01-04', NOW(), NOW()),      -- BLU-256: Low stock (5 = 5)

-- Samsung S24 variants (5-6) at Tech Main Warehouse
(9, 5, 1, 80, 10, 15, 'A-02-01', NOW(), NOW()),   -- BLK-128: Good stock
(10, 6, 1, 60, 5, 10, 'A-02-02', NOW(), NOW()),   -- BLK-256: Good stock

-- Samsung S24 at Tech Downtown Store
(11, 5, 2, 20, 5, 5, 'S-02-01', NOW(), NOW()),    -- BLK-128: Good stock
(12, 6, 2, 10, 2, 5, 'S-02-02', NOW(), NOW()),    -- BLK-256: Good stock

-- MacBook Pro variants (7-8) at Tech Main Warehouse
(13, 7, 1, 30, 2, 5, 'A-03-01', NOW(), NOW()),    -- Space Black: Good stock
(14, 8, 1, 25, 1, 5, 'A-03-02', NOW(), NOW()),    -- Silver: Good stock

-- MacBook Pro at Tech Downtown Store
(15, 7, 2, 5, 1, 3, 'S-03-01', NOW(), NOW()),     -- Space Black: Low stock
(16, 8, 2, 3, 0, 3, 'S-03-02', NOW(), NOW()),     -- Silver: Low stock

-- Sony Headphones variants (19-20) at Tech Main Warehouse
(17, 19, 1, 150, 20, 25, 'A-04-01', NOW(), NOW()), -- Black: Good stock
(18, 20, 1, 120, 15, 20, 'A-04-02', NOW(), NOW()), -- Silver: Good stock

-- Sony Headphones at Tech Downtown Store
(19, 19, 2, 30, 5, 10, 'S-04-01', NOW(), NOW()),  -- Black: Good stock
(20, 20, 2, 0, 0, 10, 'S-04-02', NOW(), NOW()),   -- Silver: Out of stock

-- ============================================================================
-- SELLER 3 (Fashion Forward) - Products 5, 6, 7
-- Locations: 3 (Central Warehouse), 4 (Outlet Store)
-- ============================================================================

-- T-Shirt variants (9-11) at Fashion Central Warehouse (location 3)
(21, 9, 3, 500, 50, 100, 'F-01-01', NOW(), NOW()),  -- BLK-M: Good stock
(22, 10, 3, 450, 30, 100, 'F-01-02', NOW(), NOW()), -- WHT-M: Good stock
(23, 11, 3, 200, 20, 50, 'F-01-03', NOW(), NOW()),  -- BLK-L: Good stock

-- T-Shirt at Fashion Outlet Store (location 4)
(24, 9, 4, 100, 10, 20, 'O-01-01', NOW(), NOW()),   -- BLK-M: Good stock
(25, 10, 4, 80, 5, 20, 'O-01-02', NOW(), NOW()),    -- WHT-M: Good stock
(26, 11, 4, 15, 0, 20, 'O-01-03', NOW(), NOW()),    -- BLK-L: Low stock

-- Summer Dress variants (12-13) at Fashion Central Warehouse
(27, 12, 3, 200, 25, 40, 'F-02-01', NOW(), NOW()),  -- BLUE-M: Good stock
(28, 13, 3, 150, 15, 30, 'F-02-02', NOW(), NOW()),  -- PINK-M: Good stock

-- Summer Dress at Fashion Outlet Store
(29, 12, 4, 40, 5, 10, 'O-02-01', NOW(), NOW()),    -- BLUE-M: Good stock
(30, 13, 4, 0, 0, 10, 'O-02-02', NOW(), NOW()),     -- PINK-M: Out of stock

-- Running Shoes variants (14-15) at Fashion Central Warehouse
(31, 14, 3, 100, 10, 20, 'F-03-01', NOW(), NOW()),  -- BW-9: Good stock
(32, 15, 3, 80, 8, 15, 'F-03-02', NOW(), NOW()),    -- BW-10: Good stock

-- Running Shoes at Fashion Outlet Store
(33, 14, 4, 25, 3, 5, 'O-03-01', NOW(), NOW()),     -- BW-9: Good stock
(34, 15, 4, 5, 0, 5, 'O-03-02', NOW(), NOW()),      -- BW-10: Low stock (5 = 5)

-- ============================================================================
-- SELLER 4 (Home & Living Store) - Products 8, 9
-- Locations: 5 (Distribution Center), 6 (Showroom)
-- ============================================================================

-- Sofa variants (16-17) at Home Distribution Center (location 5)
(35, 16, 5, 15, 2, 5, 'H-01-01', NOW(), NOW()),     -- Gray-Fabric: Good stock
(36, 17, 5, 10, 1, 3, 'H-01-02', NOW(), NOW()),     -- Beige-Fabric: Good stock

-- Sofa at Home Living Showroom (location 6)
(37, 16, 6, 3, 1, 2, 'S-01-01', NOW(), NOW()),      -- Gray-Fabric: Low stock
(38, 17, 6, 0, 0, 2, 'S-01-02', NOW(), NOW()),      -- Beige-Fabric: Out of stock

-- Mattress variant (18) at Home Distribution Center
(39, 18, 5, 50, 5, 10, 'H-02-01', NOW(), NOW()),    -- Queen-Foam: Good stock

-- Mattress at Home Living Showroom
(40, 18, 6, 8, 2, 5, 'S-02-01', NOW(), NOW())       -- Queen-Foam: Good stock
ON CONFLICT (id) DO UPDATE SET
    quantity = EXCLUDED.quantity,
    reserved_quantity = EXCLUDED.reserved_quantity,
    threshold = EXCLUDED.threshold,
    bin_location = EXCLUDED.bin_location,
    updated_at = NOW();

SELECT setval('inventory_id_seq', (SELECT MAX(id) FROM inventory));

-- ------------------------------
-- Insert Sample Inventory Transactions (recent activity)
-- ------------------------------
INSERT INTO inventory_transaction (id, inventory_id, type, quantity, before_quantity, after_quantity, performed_by, reference_id, reference_type, reason, note, created_at, updated_at) VALUES
-- Recent transactions for Tech Main Warehouse (Seller 2)
(1, 1, 'PURCHASE', 50, 50, 100, 2, 'PO-2024-001', 'PURCHASE_ORDER', 'Restocking iPhone 15 Pro Natural 128GB', 'Holiday season preparation', NOW() - INTERVAL '5 days', NOW()),
(2, 1, 'RESERVED', 5, 100, 100, 2, 'ORD-2024-1001', 'ORDER', 'Reserved for customer order', NULL, NOW() - INTERVAL '2 days', NOW()),
(3, 4, 'SALE', 2, 10, 8, 2, 'ORD-2024-1002', 'ORDER', 'Sold iPhone 15 Pro Blue 256GB', 'Walk-in customer', NOW() - INTERVAL '1 day', NOW()),

-- Transactions for Fashion Central Warehouse (Seller 3)
(4, 21, 'PURCHASE', 200, 300, 500, 3, 'PO-2024-050', 'PURCHASE_ORDER', 'Bulk t-shirt restock', 'New collection arrival', NOW() - INTERVAL '7 days', NOW()),
(5, 27, 'RESERVED', 25, 200, 200, 3, 'ORD-2024-2001', 'ORDER', 'Reserved for wholesale order', 'Corporate client order', NOW() - INTERVAL '3 days', NOW()),
(6, 30, 'SALE', 10, 10, 0, 3, 'ORD-2024-2002', 'ORDER', 'Sold last units of pink dress', 'End of season sale', NOW() - INTERVAL '1 day', NOW()),

-- Transactions for Home Distribution Center (Seller 4)
(7, 35, 'TRANSFER_OUT', 2, 17, 15, 4, 'TRF-2024-001', 'TRANSFER', 'Transfer to showroom', NULL, NOW() - INTERVAL '4 days', NOW()),
(8, 37, 'TRANSFER_IN', 2, 1, 3, 4, 'TRF-2024-001', 'TRANSFER', 'Received from warehouse', NULL, NOW() - INTERVAL '4 days', NOW()),
(9, 38, 'SALE', 2, 2, 0, 4, 'ORD-2024-3001', 'ORDER', 'Sold last beige sofas', 'Customer picked up from showroom', NOW() - INTERVAL '2 days', NOW())
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    quantity = EXCLUDED.quantity,
    before_quantity = EXCLUDED.before_quantity,
    after_quantity = EXCLUDED.after_quantity,
    reason = EXCLUDED.reason,
    updated_at = NOW();

SELECT setval('inventory_transaction_id_seq', (SELECT MAX(id) FROM inventory_transaction));

-- ------------------------------
-- Final Summary Notice
-- ------------------------------
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Inventory Seed Data Summary:';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Location Addresses: %', (SELECT COUNT(*) FROM "address" WHERE id >= 10);
    RAISE NOTICE 'Locations: %', (SELECT COUNT(*) FROM "location");
    RAISE NOTICE '  - Seller 2 (Tech): %', (SELECT COUNT(*) FROM "location" WHERE seller_id = 2);
    RAISE NOTICE '  - Seller 3 (Fashion): %', (SELECT COUNT(*) FROM "location" WHERE seller_id = 3);
    RAISE NOTICE '  - Seller 4 (Home): %', (SELECT COUNT(*) FROM "location" WHERE seller_id = 4);
    RAISE NOTICE 'Inventory Records: %', (SELECT COUNT(*) FROM inventory);
    RAISE NOTICE '  - In Stock: %', (SELECT COUNT(*) FROM inventory WHERE quantity > threshold);
    RAISE NOTICE '  - Low Stock: %', (SELECT COUNT(*) FROM inventory WHERE quantity > 0 AND quantity <= threshold);
    RAISE NOTICE '  - Out of Stock: %', (SELECT COUNT(*) FROM inventory WHERE quantity = 0);
    RAISE NOTICE 'Transactions: %', (SELECT COUNT(*) FROM inventory_transaction);
    RAISE NOTICE '========================================';
    RAISE NOTICE 'All inventory seed data inserted!';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE 'Test Credentials:';
    RAISE NOTICE '  Seller 2 (Tech): john.seller@example.com / seller123';
    RAISE NOTICE '  Seller 3 (Fashion): jane.merchant@example.com / seller123';
    RAISE NOTICE '  Seller 4 (Home): bob.store@example.com / seller123';
    RAISE NOTICE '';
END $$;
