-- Seed: 003_seed_related_products_test_data.sql
-- Description: Comprehensive test data for GetRelatedProducts API testing
-- Purpose: Create diverse product catalog for testing all 8 matching strategies
-- Created: 2025-11-11

-- This seed file creates 50+ products for Seller 2 (Tech Gadgets Pro) to test:
-- 1. Same Category Strategy - Multiple products in same categories
-- 2. Same Brand Strategy - Same brands across different categories
-- 3. Sibling Category Strategy - Products in sibling categories
-- 4. Parent Category Strategy - Products in parent categories
-- 5. Child Category Strategy - Products in child categories
-- 6. Tag Matching Strategy - Products with overlapping tags
-- 7. Price Range Strategy - Products in similar price ranges
-- 8. Seller Popular Strategy - Recent products from same seller

-- First, ensure we have the necessary category structure
-- Add more electronics subcategories for sibling testing
INSERT INTO category (id, name, parent_id, description, is_global, seller_id, created_at, updated_at) VALUES
-- Existing: 1=Electronics, 4=Smartphones, 5=Laptops, 6=Headphones
(12, 'Tablets', 1, 'Tablet computers and accessories', true, NULL, NOW(), NOW()),
(13, 'Smartwatches', 1, 'Wearable smart devices', true, NULL, NOW(), NOW()),
(14, 'Cameras', 1, 'Digital cameras and photography equipment', true, NULL, NOW(), NOW()),
(15, 'Gaming', 1, 'Gaming consoles and accessories', true, NULL, NOW(), NOW()),
(16, 'Audio Systems', 1, 'Speakers and audio equipment', true, NULL, NOW(), NOW()),
-- Add child categories under Smartphones
(17, 'Android Phones', 4, 'Android smartphones', true, NULL, NOW(), NOW()),
(18, 'iOS Phones', 4, 'Apple iPhones', true, NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

SELECT setval('category_id_seq', (SELECT MAX(id) FROM category));

-- Insert 50+ products for Seller 2 to enable comprehensive testing
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES

-- SMARTPHONES (Category 4) - 10 products for same category testing
(101, 'iPhone 14', 4, 'Apple', 'IPHONE-14', 'Previous gen Apple phone', 'Reliable iPhone with great performance', ARRAY['smartphone', 'apple', 'ios'], 2, NOW() - INTERVAL '60 days', NOW()),
(102, 'iPhone 13', 4, 'Apple', 'IPHONE-13', 'Affordable Apple phone', 'Great value iPhone with solid features', ARRAY['smartphone', 'apple', 'ios', 'budget'], 2, NOW() - INTERVAL '90 days', NOW()),
(103, 'Samsung Galaxy S23', 4, 'Samsung', 'SAMSUNG-S23', 'Previous flagship', 'Powerful Android flagship phone', ARRAY['smartphone', 'samsung', 'android', 'flagship'], 2, NOW() - INTERVAL '120 days', NOW()),
(104, 'Samsung Galaxy A54', 4, 'Samsung', 'SAMSUNG-A54', 'Mid-range Samsung', 'Affordable Samsung with great features', ARRAY['smartphone', 'samsung', 'android', 'midrange'], 2, NOW() - INTERVAL '45 days', NOW()),
(105, 'Google Pixel 8', 4, 'Google', 'PIXEL-8', 'Latest Pixel phone', 'Pure Android experience with best camera', ARRAY['smartphone', 'google', 'android', 'camera'], 2, NOW() - INTERVAL '30 days', NOW()),
(106, 'Google Pixel 7', 4, 'Google', 'PIXEL-7', 'Previous Pixel', 'Great value Pixel phone', ARRAY['smartphone', 'google', 'android', 'budget'], 2, NOW() - INTERVAL '150 days', NOW()),
(107, 'OnePlus 11', 4, 'OnePlus', 'ONEPLUS-11', 'Flagship killer', 'Premium features at competitive price', ARRAY['smartphone', 'oneplus', 'android', 'premium'], 2, NOW() - INTERVAL '75 days', NOW()),
(108, 'Xiaomi 13 Pro', 4, 'Xiaomi', 'XIAOMI-13', 'Feature-packed phone', 'Excellent specs and camera', ARRAY['smartphone', 'xiaomi', 'android', 'premium'], 2, NOW() - INTERVAL '50 days', NOW()),
(109, 'Motorola Edge 40', 4, 'Motorola', 'MOTO-EDGE-40', 'Sleek design phone', 'Stylish phone with good performance', ARRAY['smartphone', 'motorola', 'android'], 2, NOW() - INTERVAL '40 days', NOW()),
(110, 'Nothing Phone 2', 4, 'Nothing', 'NOTHING-2', 'Unique design phone', 'Transparent design with glyph interface', ARRAY['smartphone', 'nothing', 'android', 'unique'], 2, NOW() - INTERVAL '20 days', NOW()),

-- LAPTOPS (Category 5) - 10 products for sibling category testing
(111, 'MacBook Air M2', 5, 'Apple', 'MBA-M2', 'Lightweight laptop', 'Ultra-portable laptop for everyday use', ARRAY['laptop', 'apple', 'portable', 'student'], 2, NOW() - INTERVAL '35 days', NOW()),
(112, 'MacBook Pro 14"', 5, 'Apple', 'MBP-14', 'Compact pro laptop', 'Professional laptop in compact size', ARRAY['laptop', 'apple', 'professional', 'creator'], 2, NOW() - INTERVAL '25 days', NOW()),
(113, 'Dell XPS 15', 5, 'Dell', 'DELL-XPS-15', 'Premium Windows laptop', 'High-performance Windows laptop', ARRAY['laptop', 'dell', 'windows', 'professional'], 2, NOW() - INTERVAL '55 days', NOW()),
(114, 'Dell Inspiron 15', 5, 'Dell', 'DELL-INS-15', 'Budget laptop', 'Affordable laptop for basic tasks', ARRAY['laptop', 'dell', 'windows', 'budget'], 2, NOW() - INTERVAL '100 days', NOW()),
(115, 'HP Spectre x360', 5, 'HP', 'HP-SPECTRE', '2-in-1 laptop', 'Versatile convertible laptop', ARRAY['laptop', 'hp', 'windows', 'convertible'], 2, NOW() - INTERVAL '70 days', NOW()),
(116, 'Lenovo ThinkPad X1', 5, 'Lenovo', 'LENOVO-X1', 'Business laptop', 'Premium business laptop', ARRAY['laptop', 'lenovo', 'windows', 'business'], 2, NOW() - INTERVAL '80 days', NOW()),
(117, 'ASUS ROG Zephyrus', 5, 'ASUS', 'ASUS-ROG', 'Gaming laptop', 'Powerful gaming laptop', ARRAY['laptop', 'asus', 'windows', 'gaming'], 2, NOW() - INTERVAL '45 days', NOW()),
(118, 'Acer Swift 3', 5, 'Acer', 'ACER-SWIFT-3', 'Lightweight laptop', 'Affordable and portable', ARRAY['laptop', 'acer', 'windows', 'portable'], 2, NOW() - INTERVAL '90 days', NOW()),
(119, 'Microsoft Surface Laptop 5', 5, 'Microsoft', 'SURFACE-5', 'Premium Surface', 'Elegant Windows laptop', ARRAY['laptop', 'microsoft', 'windows', 'premium'], 2, NOW() - INTERVAL '60 days', NOW()),
(120, 'Razer Blade 15', 5, 'Razer', 'RAZER-BLADE-15', 'Gaming powerhouse', 'Top-tier gaming laptop', ARRAY['laptop', 'razer', 'windows', 'gaming', 'premium'], 2, NOW() - INTERVAL '30 days', NOW()),

-- TABLETS (Category 12) - 8 products for sibling category testing
(121, 'iPad Pro 12.9"', 12, 'Apple', 'IPAD-PRO-129', 'Pro tablet', 'Most powerful iPad', ARRAY['tablet', 'apple', 'ios', 'professional', 'creator'], 2, NOW() - INTERVAL '40 days', NOW()),
(122, 'iPad Air', 12, 'Apple', 'IPAD-AIR', 'Mid-tier iPad', 'Balanced iPad for most users', ARRAY['tablet', 'apple', 'ios'], 2, NOW() - INTERVAL '50 days', NOW()),
(123, 'iPad Mini', 12, 'Apple', 'IPAD-MINI', 'Compact tablet', 'Portable iPad', ARRAY['tablet', 'apple', 'ios', 'portable'], 2, NOW() - INTERVAL '70 days', NOW()),
(124, 'Samsung Galaxy Tab S9', 12, 'Samsung', 'TAB-S9', 'Premium Android tablet', 'High-end Android tablet', ARRAY['tablet', 'samsung', 'android', 'premium'], 2, NOW() - INTERVAL '35 days', NOW()),
(125, 'Samsung Galaxy Tab A8', 12, 'Samsung', 'TAB-A8', 'Budget tablet', 'Affordable Android tablet', ARRAY['tablet', 'samsung', 'android', 'budget'], 2, NOW() - INTERVAL '80 days', NOW()),
(126, 'Microsoft Surface Pro 9', 12, 'Microsoft', 'SURFACE-PRO-9', '2-in-1 tablet', 'Tablet that replaces laptop', ARRAY['tablet', 'microsoft', 'windows', 'convertible'], 2, NOW() - INTERVAL '55 days', NOW()),
(127, 'Lenovo Tab P11', 12, 'Lenovo', 'LENOVO-TAB-P11', 'Entertainment tablet', 'Great for media consumption', ARRAY['tablet', 'lenovo', 'android'], 2, NOW() - INTERVAL '90 days', NOW()),
(128, 'Amazon Fire HD 10', 12, 'Amazon', 'FIRE-HD-10', 'Budget tablet', 'Affordable tablet for basics', ARRAY['tablet', 'amazon', 'android', 'budget'], 2, NOW() - INTERVAL '120 days', NOW()),

-- SMARTWATCHES (Category 13) - 6 products
(129, 'Apple Watch Series 9', 13, 'Apple', 'WATCH-9', 'Latest Apple Watch', 'Most advanced Apple Watch', ARRAY['smartwatch', 'apple', 'ios', 'fitness', 'health'], 2, NOW() - INTERVAL '25 days', NOW()),
(130, 'Apple Watch SE', 13, 'Apple', 'WATCH-SE', 'Affordable Apple Watch', 'Essential Apple Watch features', ARRAY['smartwatch', 'apple', 'ios', 'fitness', 'budget'], 2, NOW() - INTERVAL '60 days', NOW()),
(131, 'Samsung Galaxy Watch 6', 13, 'Samsung', 'WATCH-6', 'Premium Android watch', 'Top Android smartwatch', ARRAY['smartwatch', 'samsung', 'android', 'fitness'], 2, NOW() - INTERVAL '40 days', NOW()),
(132, 'Garmin Forerunner 265', 13, 'Garmin', 'GARMIN-265', 'Running watch', 'Advanced GPS running watch', ARRAY['smartwatch', 'garmin', 'fitness', 'running', 'sports'], 2, NOW() - INTERVAL '50 days', NOW()),
(133, 'Fitbit Versa 4', 13, 'Fitbit', 'FITBIT-VERSA-4', 'Fitness tracker', 'Health and fitness focused', ARRAY['smartwatch', 'fitbit', 'fitness', 'health'], 2, NOW() - INTERVAL '70 days', NOW()),
(134, 'Amazfit GTR 4', 13, 'Amazfit', 'AMAZFIT-GTR-4', 'Budget smartwatch', 'Affordable feature-packed watch', ARRAY['smartwatch', 'amazfit', 'fitness', 'budget'], 2, NOW() - INTERVAL '85 days', NOW()),

-- HEADPHONES (Category 6) - 8 products
(135, 'AirPods Pro 2', 6, 'Apple', 'AIRPODS-PRO-2', 'Premium earbuds', 'Best ANC earbuds', ARRAY['headphones', 'apple', 'wireless', 'earbuds', 'noise-cancelling'], 2, NOW() - INTERVAL '30 days', NOW()),
(136, 'AirPods Max', 6, 'Apple', 'AIRPODS-MAX', 'Over-ear headphones', 'Premium over-ear ANC', ARRAY['headphones', 'apple', 'wireless', 'noise-cancelling', 'premium'], 2, NOW() - INTERVAL '45 days', NOW()),
(137, 'Sony WH-1000XM4', 6, 'Sony', 'SONY-XM4', 'Previous gen ANC', 'Excellent noise cancelling', ARRAY['headphones', 'sony', 'wireless', 'noise-cancelling'], 2, NOW() - INTERVAL '150 days', NOW()),
(138, 'Bose QuietComfort 45', 6, 'Bose', 'BOSE-QC45', 'Bose ANC headphones', 'Legendary Bose comfort', ARRAY['headphones', 'bose', 'wireless', 'noise-cancelling'], 2, NOW() - INTERVAL '100 days', NOW()),
(139, 'Sennheiser Momentum 4', 6, 'Sennheiser', 'SENNHEISER-M4', 'Audiophile headphones', 'Superior sound quality', ARRAY['headphones', 'sennheiser', 'wireless', 'audiophile'], 2, NOW() - INTERVAL '65 days', NOW()),
(140, 'Jabra Elite 85h', 6, 'Jabra', 'JABRA-85H', 'Business headphones', 'Great for calls', ARRAY['headphones', 'jabra', 'wireless', 'business'], 2, NOW() - INTERVAL '120 days', NOW()),
(141, 'Beats Studio Pro', 6, 'Beats', 'BEATS-STUDIO-PRO', 'Stylish headphones', 'Fashion meets function', ARRAY['headphones', 'beats', 'wireless', 'style'], 2, NOW() - INTERVAL '55 days', NOW()),
(142, 'Anker Soundcore Q30', 6, 'Anker', 'ANKER-Q30', 'Budget ANC', 'Affordable noise cancelling', ARRAY['headphones', 'anker', 'wireless', 'budget'], 2, NOW() - INTERVAL '90 days', NOW()),

-- CAMERAS (Category 14) - 5 products
(143, 'Canon EOS R6', 14, 'Canon', 'CANON-R6', 'Professional mirrorless', 'Full-frame mirrorless camera', ARRAY['camera', 'canon', 'mirrorless', 'professional'], 2, NOW() - INTERVAL '40 days', NOW()),
(144, 'Sony A7 IV', 14, 'Sony', 'SONY-A7-4', 'Hybrid camera', 'Versatile full-frame camera', ARRAY['camera', 'sony', 'mirrorless', 'professional'], 2, NOW() - INTERVAL '50 days', NOW()),
(145, 'Nikon Z6 II', 14, 'Nikon', 'NIKON-Z6-2', 'All-rounder camera', 'Great all-around mirrorless', ARRAY['camera', 'nikon', 'mirrorless'], 2, NOW() - INTERVAL '75 days', NOW()),
(146, 'Fujifilm X-T5', 14, 'Fujifilm', 'FUJI-XT5', 'Retro design camera', 'Classic looks, modern tech', ARRAY['camera', 'fujifilm', 'mirrorless', 'retro'], 2, NOW() - INTERVAL '60 days', NOW()),
(147, 'GoPro Hero 12', 14, 'GoPro', 'GOPRO-12', 'Action camera', 'Ultimate action cam', ARRAY['camera', 'gopro', 'action', 'sports'], 2, NOW() - INTERVAL '35 days', NOW()),

-- TODO: Add stock testing when inventory service is integrated
-- Products for edge case testing (older products)
(148, 'iPhone 12 (Older Model)', 18, 'Apple', 'IPHONE-12-OLD', 'Older model', 'Previous generation iPhone', ARRAY['smartphone', 'apple', 'ios'], 2, NOW() - INTERVAL '365 days', NOW()),
(149, 'Samsung Note 20', 17, 'Samsung', 'NOTE-20', 'Samsung Note 20', 'Previous flagship phone', ARRAY['smartphone', 'samsung', 'android'], 2, NOW() - INTERVAL '200 days', NOW()),

-- Products with extreme prices for price range testing
(150, 'Budget Phone Lite', 17, 'Generic', 'BUDGET-PHONE', 'Ultra budget', 'Basic smartphone', ARRAY['smartphone', 'android', 'budget'], 2, NOW() - INTERVAL '180 days', NOW()),
(151, 'Ultra Premium Fold Phone', 18, 'Samsung', 'FOLD-ULTRA', 'Foldable luxury', 'Most expensive phone', ARRAY['smartphone', 'samsung', 'android', 'luxury', 'premium'], 2, NOW() - INTERVAL '15 days', NOW())

ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    short_description = EXCLUDED.short_description,
    long_description = EXCLUDED.long_description,
    tags = EXCLUDED.tags,
    updated_at = NOW();

SELECT setval('product_id_seq', (SELECT MAX(id) FROM product));

-- Add product options for variant support (required for response structure)
INSERT INTO product_option (product_id, name, display_name, position) VALUES
-- Add options to a few products for variant preview testing
(101, 'Color', 'Color', 1),
(101, 'Storage', 'Storage Capacity', 2),
(105, 'Color', 'Color', 1),
(105, 'Storage', 'Storage Capacity', 2),
(111, 'Color', 'Color', 1),
(111, 'RAM', 'RAM', 2),
(121, 'Color', 'Color', 1),
(121, 'Storage', 'Storage Capacity', 2),
(129, 'Size', 'Case Size', 1),
(129, 'Band', 'Band Type', 2)
ON CONFLICT DO NOTHING;

-- Add option values
INSERT INTO product_option_value (option_id, value, position) 
SELECT id, 'Black', 1 FROM product_option WHERE product_id IN (101, 105, 111, 121, 129) AND name = 'Color'
UNION ALL
SELECT id, 'White', 2 FROM product_option WHERE product_id IN (101, 105, 111, 121, 129) AND name = 'Color'
UNION ALL
SELECT id, '128GB', 1 FROM product_option WHERE product_id IN (101, 105, 121) AND name = 'Storage'
UNION ALL
SELECT id, '256GB', 2 FROM product_option WHERE product_id IN (101, 105, 121) AND name = 'Storage'
UNION ALL
SELECT id, '8GB', 1 FROM product_option WHERE product_id = 111 AND name = 'RAM'
UNION ALL
SELECT id, '16GB', 2 FROM product_option WHERE product_id = 111 AND name = 'RAM'
UNION ALL
SELECT id, '41mm', 1 FROM product_option WHERE product_id = 129 AND name = 'Size'
UNION ALL
SELECT id, '45mm', 2 FROM product_option WHERE product_id = 129 AND name = 'Size'
UNION ALL
SELECT id, 'Sport', 1 FROM product_option WHERE product_id = 129 AND name = 'Band'
UNION ALL
SELECT id, 'Leather', 2 FROM product_option WHERE product_id = 129 AND name = 'Band'
ON CONFLICT DO NOTHING;

-- Create product variants for testing variant preview
-- TODO: Add stock column when inventory service is integrated
INSERT INTO product_variant (product_id, sku, price, is_default, created_at, updated_at)
SELECT 
    101, 
    'IPHONE-14-BLACK-128GB',
    799.00,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'IPHONE-14-BLACK-128GB')
UNION ALL
SELECT 101, 'IPHONE-14-BLACK-256GB', 899.00, false, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'IPHONE-14-BLACK-256GB')
UNION ALL
SELECT 101, 'IPHONE-14-WHITE-128GB', 799.00, false, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'IPHONE-14-WHITE-128GB')
UNION ALL
SELECT 105, 'PIXEL-8-BLACK-128GB', 699.00, true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'PIXEL-8-BLACK-128GB')
UNION ALL
SELECT 105, 'PIXEL-8-WHITE-256GB', 799.00, false, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'PIXEL-8-WHITE-256GB')
UNION ALL
-- TODO: Add stock testing when inventory service is integrated
SELECT 148, 'IPHONE-12-DISC-BLACK', 599.00, true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'IPHONE-12-DISC-BLACK')
UNION ALL
SELECT 149, 'NOTE-20-OOS-BLACK', 899.00, true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'NOTE-20-OOS-BLACK')
UNION ALL
-- Extreme price variants
SELECT 150, 'BUDGET-PHONE-BLACK', 99.00, true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'BUDGET-PHONE-BLACK')
UNION ALL
SELECT 151, 'FOLD-ULTRA-BLACK', 2499.00, true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM product_variant WHERE sku = 'FOLD-ULTRA-BLACK');

-- Display summary
DO $$
BEGIN
    RAISE NOTICE '=== Related Products Test Data Seed Completed ===';
    RAISE NOTICE 'Total products in database: %', (SELECT COUNT(*) FROM product);
    RAISE NOTICE 'Products for Seller 2: %', (SELECT COUNT(*) FROM product WHERE seller_id = 2);
    RAISE NOTICE 'Smartphones: %', (SELECT COUNT(*) FROM product WHERE category_id IN (4, 17, 18));
    RAISE NOTICE 'Laptops: %', (SELECT COUNT(*) FROM product WHERE category_id = 5);
    RAISE NOTICE 'Tablets: %', (SELECT COUNT(*) FROM product WHERE category_id = 12);
    RAISE NOTICE 'Smartwatches: %', (SELECT COUNT(*) FROM product WHERE category_id = 13);
    RAISE NOTICE 'Headphones: %', (SELECT COUNT(*) FROM product WHERE category_id = 6);
    RAISE NOTICE 'Cameras: %', (SELECT COUNT(*) FROM product WHERE category_id = 14);
    RAISE NOTICE 'Total variants: %', (SELECT COUNT(*) FROM product_variant);
    RAISE NOTICE '=== Ready for Related Products Testing ===';
END $$;
