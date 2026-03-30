-- Seed: 005_seed_wishlist_data.sql
-- Description: Sample wishlist data for development and testing
-- Environment: MOCK (development/testing only)
-- Created: 2026-01-25

-- ============================================================================
-- Prerequisites:
-- - Users must exist (from user seeds)
-- - Product variants must exist (from product seeds)
-- ============================================================================

-- ------------------------------
-- Wishlists for Customer Users
-- ------------------------------

-- Customer user ID 5 (alice.j@example.com) - Default wishlist (seller_id = 2)
INSERT INTO wishlist (id, user_id, name, is_default, created_at, updated_at) VALUES
(1, 5, 'My Wishlist', TRUE, NOW(), NOW()),
(2, 5, 'Gift Ideas', FALSE, NOW(), NOW()),
(3, 5, 'Tech Upgrades', FALSE, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

-- Customer user ID 6 (michael.s@example.com) - Default wishlist (seller_id = 3)
INSERT INTO wishlist (id, user_id, name, is_default, created_at, updated_at) VALUES
(4, 6, 'My Wishlist', TRUE, NOW(), NOW()),
(5, 6, 'Birthday List', FALSE, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

-- Customer user ID 7 (sarah.w@example.com) - Single default wishlist (seller_id = 4)
INSERT INTO wishlist (id, user_id, name, is_default, created_at, updated_at) VALUES
(6, 7, 'My Wishlist', TRUE, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    is_default = EXCLUDED.is_default,
    updated_at = NOW();

-- Reset sequence
SELECT setval('wishlist_id_seq', (SELECT MAX(id) FROM wishlist));

-- ------------------------------
-- Wishlist Items
-- ------------------------------
-- Note: variant_id references product_variant table
-- Ensure these variants exist in your product seeds

-- Alice's default wishlist items (wishlist_id = 1, user_id = 5)
-- Variants 1, 2 are from Product 1 (iPhone) - seller_id = 2
-- Variant 5 is from Product 2 (Samsung) - seller_id = 2
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(1, 1, 1, NOW(), NOW()),   -- iPhone variant 1 (IPHONE-15-PRO-NAT-128)
(2, 1, 2, NOW(), NOW()),   -- iPhone variant 2 (IPHONE-15-PRO-NAT-256)
(3, 1, 5, NOW(), NOW())    -- Samsung variant 5 (SAMSUNG-S24-BLK-128)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Alice's Gift Ideas wishlist items (wishlist_id = 2)
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(4, 2, 3, NOW(), NOW()),   -- iPhone variant 3 (IPHONE-15-PRO-BLU-128)
(5, 2, 4, NOW(), NOW())    -- iPhone variant 4 (IPHONE-15-PRO-BLU-256)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Alice's Tech Upgrades wishlist items (wishlist_id = 3)
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(6, 3, 6, NOW(), NOW()),   -- Samsung variant 6 (SAMSUNG-S24-BLK-256)
(7, 3, 7, NOW(), NOW())    -- MacBook variant 7 (MBP-16-M3-SB-16-512)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Michael's default wishlist items (wishlist_id = 4, user_id = 6)
-- Variant 9 is from Product 5 (T-Shirt) - seller_id = 3
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(8, 4, 9, NOW(), NOW()),   -- T-Shirt variant 9 (NIKE-TSHIRT-BLK-M)
(9, 4, 10, NOW(), NOW())   -- T-Shirt variant 10 (NIKE-TSHIRT-WHT-M)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Michael's Birthday List items (wishlist_id = 5)
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(10, 5, 12, NOW(), NOW()), -- Dress variant 12 (ZARA-DRESS-BLUE-M)
(11, 5, 14, NOW(), NOW())  -- Running Shoes variant 14 (ADIDAS-RUN-BW-9)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Sarah's wishlist items (wishlist_id = 6, user_id = 7)
-- Variant 16 is from Product 8 (Sofa) - seller_id = 4
INSERT INTO wishlist_item (id, wishlist_id, variant_id, created_at, updated_at) VALUES
(12, 6, 16, NOW(), NOW()), -- Sofa variant 16 (IKEA-SOFA-GRAY-FAB)
(13, 6, 17, NOW(), NOW()), -- Sofa variant 17 (IKEA-SOFA-BEIGE-FAB)
(14, 6, 18, NOW(), NOW())  -- Mattress variant 18 (CASPER-MATTRESS-Q-FOAM)
ON CONFLICT (wishlist_id, variant_id) DO UPDATE SET
    updated_at = NOW();

-- Reset sequence
SELECT setval('wishlist_item_id_seq', (SELECT MAX(id) FROM wishlist_item));
