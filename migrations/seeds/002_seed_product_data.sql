-- Seed: 002_seed_product_data.sql
-- Description: Insert demo data for product service (matches corrected entity structure)
-- Created: 2025-10-21
-- Updated: 2025-10-21 - Fixed to match actual entity definitions

-- Insert Categories (hierarchical structure with is_global and seller_id)
INSERT INTO category (id, name, parent_id, description, is_global, seller_id, created_at, updated_at) VALUES
-- Level 1 Global Categories
(1, 'Electronics', NULL, 'Electronic devices and accessories', true, NULL, NOW(), NOW()),
(2, 'Fashion', NULL, 'Clothing, shoes, and accessories', true, NULL, NOW(), NOW()),
(3, 'Home & Living', NULL, 'Furniture and home decor', true, NULL, NOW(), NOW()),

-- Level 2 Global Categories - Electronics
(4, 'Smartphones', 1, 'Mobile phones and accessories', true, NULL, NOW(), NOW()),
(5, 'Laptops', 1, 'Laptops and notebook computers', true, NULL, NOW(), NOW()),
(6, 'Headphones', 1, 'Audio devices and headphones', true, NULL, NOW(), NOW()),

-- Level 2 Global Categories - Fashion
(7, 'Men''s Clothing', 2, 'Clothing for men', true, NULL, NOW(), NOW()),
(8, 'Women''s Clothing', 2, 'Clothing for women', true, NULL, NOW(), NOW()),
(9, 'Footwear', 2, 'Shoes and sandals', true, NULL, NOW(), NOW()),

-- Level 2 Global Categories - Home & Living
(10, 'Furniture', 3, 'Home and office furniture', true, NULL, NOW(), NOW()),
(11, 'Bedding', 3, 'Bedsheets, pillows, and mattresses', true, NULL, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    is_global = EXCLUDED.is_global,
    updated_at = NOW();

-- Reset sequence for categories
SELECT setval('category_id_seq', (SELECT MAX(id) FROM category));

-- Insert Attribute Definitions (using key, name, unit, allowed_values)
INSERT INTO attribute_definition (id, key, name, unit, allowed_values, created_at, updated_at) VALUES
-- Common attributes
(1, 'color', 'Color', NULL, ARRAY['Red', 'Blue', 'Green', 'Black', 'White', 'Silver', 'Gold'], NOW(), NOW()),
(2, 'brand', 'Brand', NULL, NULL, NOW(), NOW()),
(3, 'material', 'Material', NULL, ARRAY['Cotton', 'Polyester', 'Leather', 'Wood', 'Metal', 'Plastic'], NOW(), NOW()),

-- Electronics specific
(4, 'screen_size', 'Screen Size', 'inches', NULL, NOW(), NOW()),
(5, 'storage', 'Storage Capacity', 'GB', ARRAY['64', '128', '256', '512', '1024'], NOW(), NOW()),
(6, 'ram', 'RAM', 'GB', ARRAY['4', '8', '16', '32', '64'], NOW(), NOW()),
(7, 'processor', 'Processor', NULL, NULL, NOW(), NOW()),
(8, 'battery', 'Battery Capacity', 'mAh', NULL, NOW(), NOW()),

-- Fashion specific
(9, 'size', 'Size', NULL, ARRAY['XS', 'S', 'M', 'L', 'XL', 'XXL'], NOW(), NOW()),
(10, 'fit', 'Fit Type', NULL, ARRAY['Slim', 'Regular', 'Loose'], NOW(), NOW()),

-- Furniture specific
(11, 'dimensions', 'Dimensions', 'cm', NULL, NOW(), NOW()),
(12, 'weight_capacity', 'Weight Capacity', 'kg', NULL, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    allowed_values = EXCLUDED.allowed_values,
    updated_at = NOW();

-- Reset sequence for attribute_definitions
SELECT setval('attribute_definition_id_seq', (SELECT MAX(id) FROM attribute_definition));

-- Link Attributes to Categories (using attribute_definition_id, is_searchable, is_filterable)
INSERT INTO category_attribute (category_id, attribute_definition_id, is_required, is_searchable, is_filterable, default_value) VALUES
-- Smartphones (category 4)
(4, 2, false, true, true, NULL),    -- brand
(4, 1, true, false, true, NULL),    -- color
(4, 5, true, true, true, NULL),     -- storage
(4, 6, false, true, true, NULL),    -- ram
(4, 4, false, true, false, NULL),   -- screen_size
(4, 8, false, true, false, NULL),   -- battery

-- Laptops (category 5)
(5, 2, false, true, true, NULL),    -- brand
(5, 6, true, true, true, NULL),     -- ram
(5, 5, true, true, true, NULL),     -- storage
(5, 7, false, true, false, NULL),   -- processor
(5, 1, false, false, true, NULL),   -- color

-- Men's Clothing (category 7)
(7, 9, true, false, true, NULL),    -- size
(7, 1, true, false, true, NULL),    -- color
(7, 3, false, true, true, NULL),    -- material
(7, 10, false, false, true, NULL),  -- fit

-- Women's Clothing (category 8)
(8, 9, true, false, true, NULL),    -- size
(8, 1, true, false, true, NULL),    -- color
(8, 3, false, true, true, NULL),    -- material

-- Furniture (category 10)
(10, 3, false, true, true, NULL),   -- material
(10, 1, false, false, true, NULL),  -- color
(10, 11, false, true, false, NULL), -- dimensions
(10, 12, false, true, false, NULL)  -- weight_capacity
ON CONFLICT (category_id, attribute_definition_id) DO UPDATE SET
    is_required = EXCLUDED.is_required,
    is_searchable = EXCLUDED.is_searchable,
    is_filterable = EXCLUDED.is_filterable,
    updated_at = NOW();

-- Insert Products (seller_id 2, 3, 4 from user seeds)
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
-- Electronics by Seller 2 (Tech Gadgets Pro)
(1, 'iPhone 15 Pro', 4, 'Apple', 'IPHONE-15-PRO', 'Latest Apple smartphone', 'Premium smartphone with A17 Pro chip, advanced camera system, and titanium design', ARRAY['smartphone', 'apple', 'flagship', 'premium'], 2, NOW(), NOW()),
(2, 'Samsung Galaxy S24', 4, 'Samsung', 'SAMSUNG-S24', 'Flagship Android smartphone', 'Premium Android phone with AI features, excellent camera, and long battery life', ARRAY['smartphone', 'samsung', 'android', 'ai'], 2, NOW(), NOW()),
(3, 'MacBook Pro 16"', 5, 'Apple', 'MBP-16-M3', 'Professional laptop', 'Powerful laptop for professionals with M3 chip, stunning display, and all-day battery', ARRAY['laptop', 'apple', 'professional', 'creator'], 2, NOW(), NOW()),
(4, 'Sony WH-1000XM5', 6, 'Sony', 'SONY-WH1000XM5', 'Premium noise cancelling headphones', 'Industry-leading noise cancellation with exceptional sound quality', ARRAY['headphones', 'sony', 'wireless', 'noise-cancelling'], 2, NOW(), NOW()),

-- Fashion by Seller 3 (Fashion Forward)
(5, 'Classic Cotton T-Shirt', 7, 'Nike', 'NIKE-TSHIRT-001', 'Comfortable everyday t-shirt', 'Premium cotton t-shirt perfect for casual wear', ARRAY['tshirt', 'casual', 'cotton', 'everyday'], 3, NOW(), NOW()),
(6, 'Summer Dress', 8, 'Zara', 'ZARA-DRESS-001', 'Elegant floral summer dress', 'Beautiful summer dress with floral pattern, perfect for warm weather', ARRAY['dress', 'summer', 'casual', 'floral'], 3, NOW(), NOW()),
(7, 'Running Shoes', 9, 'Adidas', 'ADIDAS-RUN-001', 'Lightweight running shoes', 'Professional running shoes with excellent cushioning and support', ARRAY['shoes', 'sports', 'running', 'athletic'], 3, NOW(), NOW()),

-- Home & Living by Seller 4 (Home & Living Store)
(8, 'Modern Sofa Set', 10, 'IKEA', 'IKEA-SOFA-001', 'Contemporary 3-seater sofa', 'Stylish and comfortable sofa perfect for any living room', ARRAY['furniture', 'sofa', 'living-room', 'modern'], 4, NOW(), NOW()),
(9, 'Memory Foam Mattress', 11, 'Casper', 'CASPER-MATTRESS-Q', 'Queen size memory foam mattress', 'Premium memory foam mattress for the best sleep experience', ARRAY['mattress', 'bedroom', 'comfort', 'sleep'], 4, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    short_description = EXCLUDED.short_description,
    long_description = EXCLUDED.long_description,
    tags = EXCLUDED.tags,
    updated_at = NOW();

-- Reset sequence for products
SELECT setval('product_id_seq', (SELECT MAX(id) FROM product));

-- Insert Product Attributes (using attribute_definition_id and sort_order)
INSERT INTO product_attribute (product_id, attribute_definition_id, value, sort_order) VALUES
-- iPhone 15 Pro attributes
(1, 2, 'Apple', 1),
(1, 4, '6.1', 2),
(1, 8, '4852', 3),

-- Samsung S24 attributes
(2, 2, 'Samsung', 1),
(2, 4, '6.2', 2),
(2, 8, '5000', 3),

-- MacBook Pro attributes
(3, 2, 'Apple', 1),
(3, 4, '16', 2),
(3, 7, 'Apple M3 Pro', 3),

-- Sony Headphones
(4, 2, 'Sony', 1),
(4, 3, 'Synthetic Leather', 2),

-- T-Shirt
(5, 2, 'Nike', 1),
(5, 3, 'Cotton', 2),
(5, 10, 'Regular', 3),

-- Summer Dress
(6, 2, 'Zara', 1),
(6, 3, 'Cotton', 2),

-- Running Shoes
(7, 2, 'Adidas', 1),
(7, 3, 'Mesh', 2),

-- Sofa
(8, 2, 'IKEA', 1),
(8, 3, 'Fabric', 2),
(8, 11, '220 x 90 x 85', 3),
(8, 12, '300', 4),

-- Mattress
(9, 2, 'Casper', 1),
(9, 3, 'Memory Foam', 2),
(9, 11, '152 x 203 x 25', 3)
ON CONFLICT (product_id, attribute_definition_id) DO UPDATE SET
    value = EXCLUDED.value,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- Display summary
DO $$
BEGIN
    RAISE NOTICE 'Product seed data completed successfully!';
    RAISE NOTICE 'Created % categories', (SELECT COUNT(*) FROM category);
    RAISE NOTICE 'Created % attribute definitions', (SELECT COUNT(*) FROM attribute_definition);
    RAISE NOTICE 'Created % category-attribute mappings', (SELECT COUNT(*) FROM category_attribute);
    RAISE NOTICE 'Created % products', (SELECT COUNT(*) FROM product);
    RAISE NOTICE 'Created % product attributes', (SELECT COUNT(*) FROM product_attribute);
END $$;

-- Insert Product Options (name, display_name, position)
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
-- iPhone 15 Pro options
(1, 1, 'Color', 'Color', 1),
(2, 1, 'Storage', 'Storage Capacity', 2),

-- Samsung S24 options
(3, 2, 'Color', 'Color', 1),
(4, 2, 'Storage', 'Storage Capacity', 2),

-- MacBook Pro options
(5, 3, 'Color', 'Color', 1),
(6, 3, 'Memory', 'RAM Memory', 2),
(7, 3, 'Storage', 'Storage Capacity', 3),

-- T-Shirt options
(8, 5, 'Size', 'Size', 1),
(9, 5, 'Color', 'Color', 2),

-- Summer Dress options
(10, 6, 'Size', 'Size', 1),
(11, 6, 'Color', 'Color', 2),

-- Running Shoes options
(12, 7, 'Size', 'Shoe Size', 1),
(13, 7, 'Color', 'Color', 2),

-- Sony Headphones options (Product 4)
(16, 4, 'Color', 'Color', 1),

-- Sofa options
(14, 8, 'Color', 'Color', 1),
(15, 8, 'Material', 'Material Type', 2)
ON CONFLICT (product_id, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    position = EXCLUDED.position,
    updated_at = NOW();

-- Reset sequence for product_options
SELECT setval('product_option_id_seq', (SELECT MAX(id) FROM product_option));

-- Insert Product Option Values (value, display_name, position)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
-- iPhone 15 Pro - Color option values
(1, 1, 'Natural Titanium', 'Natural Titanium', '#F5E6D3', 1),
(2, 1, 'Blue Titanium', 'Blue Titanium', '#5B8DBE', 2),
(3, 1, 'White Titanium', 'White Titanium', '#F0F0F0', 3),
(4, 1, 'Black Titanium', 'Black Titanium', '#2C2C2C', 4),

-- iPhone 15 Pro - Storage option values
(5, 2, '128GB', '128GB', NULL, 1),
(6, 2, '256GB', '256GB', NULL, 2),
(7, 2, '512GB', '512GB', NULL, 3),
(8, 2, '1TB', '1TB', NULL, 4),

-- Samsung S24 - Color option values
(9, 3, 'Onyx Black', 'Onyx Black', '#000000', 1),
(10, 3, 'Marble Gray', 'Marble Gray', '#808080', 2),
(11, 3, 'Cobalt Violet', 'Cobalt Violet', '#5E4FA2', 3),

-- Samsung S24 - Storage option values
(12, 4, '128GB', '128GB', NULL, 1),
(13, 4, '256GB', '256GB', NULL, 2),
(14, 4, '512GB', '512GB', NULL, 3),

-- MacBook Pro - Color option values
(15, 5, 'Space Black', 'Space Black', '#1C1C1E', 1),
(16, 5, 'Silver', 'Silver', '#C0C0C0', 2),

-- MacBook Pro - Memory option values
(17, 6, '16GB', '16GB', NULL, 1),
(18, 6, '32GB', '32GB', NULL, 2),
(19, 6, '64GB', '64GB', NULL, 3),

-- MacBook Pro - Storage option values
(20, 7, '512GB', '512GB', NULL, 1),
(21, 7, '1TB', '1TB', NULL, 2),
(22, 7, '2TB', '2TB', NULL, 3),

-- T-Shirt - Size option values
(23, 8, 'S', 'Small', NULL, 1),
(24, 8, 'M', 'Medium', NULL, 2),
(25, 8, 'L', 'Large', NULL, 3),
(26, 8, 'XL', 'Extra Large', NULL, 4),
(27, 8, 'XXL', '2X Large', NULL, 5),

-- T-Shirt - Color option values
(28, 9, 'Black', 'Black', '#000000', 1),
(29, 9, 'White', 'White', '#FFFFFF', 2),
(30, 9, 'Navy', 'Navy Blue', '#000080', 3),
(31, 9, 'Gray', 'Gray', '#808080', 4),

-- Summer Dress - Size option values
(32, 10, 'XS', 'Extra Small', NULL, 1),
(33, 10, 'S', 'Small', NULL, 2),
(34, 10, 'M', 'Medium', NULL, 3),
(35, 10, 'L', 'Large', NULL, 4),
(36, 10, 'XL', 'Extra Large', NULL, 5),

-- Summer Dress - Color option values
(37, 11, 'Floral Blue', 'Floral Blue', '#4169E1', 1),
(38, 11, 'Floral Pink', 'Floral Pink', '#FFB6C1', 2),
(39, 11, 'Solid White', 'Solid White', '#FFFFFF', 3),

-- Running Shoes - Size option values
(40, 12, '7', 'Size 7', NULL, 1),
(41, 12, '8', 'Size 8', NULL, 2),
(42, 12, '9', 'Size 9', NULL, 3),
(43, 12, '10', 'Size 10', NULL, 4),
(44, 12, '11', 'Size 11', NULL, 5),
(45, 12, '12', 'Size 12', NULL, 6),

-- Running Shoes - Color option values
(46, 13, 'Black/White', 'Black/White', NULL, 1),
(47, 13, 'Blue/Orange', 'Blue/Orange', NULL, 2),
(48, 13, 'All Black', 'All Black', '#000000', 3),

-- Sony Headphones - Color option values (option_id 16)
(55, 16, 'Black', 'Midnight Black', '#000000', 1),
(56, 16, 'Silver', 'Silver', '#C0C0C0', 2),

-- Sofa - Color option values
(49, 14, 'Gray', 'Gray', '#808080', 1),
(50, 14, 'Beige', 'Beige', '#F5F5DC', 2),
(51, 14, 'Navy Blue', 'Navy Blue', '#000080', 3),

-- Sofa - Material option values
(52, 15, 'Fabric', 'Fabric', NULL, 1),
(53, 15, 'Velvet', 'Velvet', NULL, 2),
(54, 15, 'Leather', 'Leather', NULL, 3)
ON CONFLICT (option_id, value) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    color_code = EXCLUDED.color_code,
    position = EXCLUDED.position,
    updated_at = NOW();

-- Reset sequence for product_option_value
SELECT setval('product_option_value_id_seq', (SELECT MAX(id) FROM product_option_value));

-- Insert Product Variants (sku, price, images, allow_purchase, is_default)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
-- iPhone 15 Pro variants (2 colors × 2 storage = 4 variants)
(1, 1, 'IPHONE-15-PRO-NAT-128', 999.00, ARRAY['https://example.com/iphone-15-pro-natural-128.jpg'], true, true, true),
(2, 1, 'IPHONE-15-PRO-NAT-256', 1099.00, ARRAY['https://example.com/iphone-15-pro-natural-256.jpg'], true, false, false),
(3, 1, 'IPHONE-15-PRO-BLU-128', 999.00, ARRAY['https://example.com/iphone-15-pro-blue-128.jpg'], true, false, false),
(4, 1, 'IPHONE-15-PRO-BLU-256', 1099.00, ARRAY['https://example.com/iphone-15-pro-blue-256.jpg'], true, false, false),

-- Samsung S24 variants
(5, 2, 'SAMSUNG-S24-BLK-128', 799.00, ARRAY['https://example.com/samsung-s24-black-128.jpg'], true, true, true),
(6, 2, 'SAMSUNG-S24-BLK-256', 899.00, ARRAY['https://example.com/samsung-s24-black-256.jpg'], true, false, false),

-- MacBook Pro variants
(7, 3, 'MBP-16-M3-SB-16-512', 2499.00, ARRAY['https://example.com/mbp-16-space-black.jpg'], true, true, true),
(8, 3, 'MBP-16-M3-SLV-16-512', 2499.00, ARRAY['https://example.com/mbp-16-silver.jpg'], true, false, false),

-- Sony Headphones variants
(19, 4, 'SONY-WH1000XM5-BLK', 399.99, ARRAY['https://example.com/sony-wh1000xm5-black.jpg'], true, true, true),
(20, 4, 'SONY-WH1000XM5-SLV', 399.99, ARRAY['https://example.com/sony-wh1000xm5-silver.jpg'], true, false, false),

-- T-Shirt variants
(9, 5, 'NIKE-TSHIRT-BLK-M', 29.99, ARRAY['https://example.com/nike-tshirt-black-m.jpg'], true, true, true),
(10, 5, 'NIKE-TSHIRT-WHT-M', 29.99, ARRAY['https://example.com/nike-tshirt-white-m.jpg'], true, false, false),
(11, 5, 'NIKE-TSHIRT-BLK-L', 29.99, ARRAY['https://example.com/nike-tshirt-black-l.jpg'], true, false, false),

-- Summer Dress variants
(12, 6, 'ZARA-DRESS-BLUE-M', 49.99, ARRAY['https://example.com/zara-dress-blue-m.jpg'], true, true, true),
(13, 6, 'ZARA-DRESS-PINK-M', 49.99, ARRAY['https://example.com/zara-dress-pink-m.jpg'], true, false, false),

-- Running Shoes variants
(14, 7, 'ADIDAS-RUN-BW-9', 89.99, ARRAY['https://example.com/adidas-run-blackwhite-9.jpg'], true, true, true),
(15, 7, 'ADIDAS-RUN-BW-10', 89.99, ARRAY['https://example.com/adidas-run-blackwhite-10.jpg'], true, false, false),

-- Sofa variants
(16, 8, 'IKEA-SOFA-GRAY-FAB', 899.00, ARRAY['https://example.com/ikea-sofa-gray-fabric.jpg'], true, true, true),
(17, 8, 'IKEA-SOFA-BEIGE-FAB', 899.00, ARRAY['https://example.com/ikea-sofa-beige-fabric.jpg'], true, false, false),

-- Mattress (single variant)
(18, 9, 'CASPER-MATTRESS-Q-FOAM', 799.00, ARRAY['https://example.com/casper-mattress-queen.jpg'], true, true, true)
ON CONFLICT (id) DO UPDATE SET
    price = EXCLUDED.price,
    is_popular = EXCLUDED.is_popular,
    updated_at = NOW();

-- Reset sequence for product_variants
SELECT setval('product_variant_id_seq', (SELECT MAX(id) FROM product_variant));

-- Insert Variant Option Values (links variants to their option values)
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
-- iPhone 15 Pro variant 1: Natural Titanium + 128GB
(1, 1, 1), (1, 2, 5),
-- iPhone 15 Pro variant 2: Natural Titanium + 256GB
(2, 1, 1), (2, 2, 6),
-- iPhone 15 Pro variant 3: Blue Titanium + 128GB
(3, 1, 2), (3, 2, 5),
-- iPhone 15 Pro variant 4: Blue Titanium + 256GB
(4, 1, 2), (4, 2, 6),

-- Samsung S24 variant 5: Onyx Black + 128GB
(5, 3, 9), (5, 4, 12),
-- Samsung S24 variant 6: Onyx Black + 256GB
(6, 3, 9), (6, 4, 13),

-- MacBook Pro variant 7: Space Black + 16GB + 512GB
(7, 5, 15), (7, 6, 17), (7, 7, 20),
-- MacBook Pro variant 8: Silver + 16GB + 512GB
(8, 5, 16), (8, 6, 17), (8, 7, 20),

-- Sony Headphones variant 19: Black
(19, 16, 55),
-- Sony Headphones variant 20: Silver
(20, 16, 56),

-- T-Shirt variant 9: Black + M
(9, 8, 24), (9, 9, 28),
-- T-Shirt variant 10: White + M
(10, 8, 24), (10, 9, 29),
-- T-Shirt variant 11: Black + L
(11, 8, 25), (11, 9, 28),

-- Summer Dress variant 12: Floral Blue + M
(12, 10, 34), (12, 11, 37),
-- Summer Dress variant 13: Floral Pink + M
(13, 10, 34), (13, 11, 38),

-- Running Shoes variant 14: Size 9 + Black/White
(14, 12, 42), (14, 13, 46),
-- Running Shoes variant 15: Size 10 + Black/White
(15, 12, 43), (15, 13, 46),

-- Sofa variant 16: Gray + Fabric
(16, 14, 49), (16, 15, 52),
-- Sofa variant 17: Beige + Fabric
(17, 14, 50), (17, 15, 52)
ON CONFLICT (variant_id, option_id, option_value_id) DO UPDATE SET
    updated_at = NOW();

-- Insert Package Options (bundle deals with name, description, price, quantity)
INSERT INTO package_option (id, product_id, name, description, price, quantity) VALUES
(1, 1, 'iPhone 15 Pro Complete Bundle', 'iPhone with case and screen protector', 1099.00, 1),
(2, 3, 'MacBook Pro Pro Pack', 'MacBook with accessories', 2699.00, 1),
(3, 5, 'T-Shirt 3-Pack', 'Buy 3 t-shirts and save', 75.00, 3),
(4, 8, 'Living Room Set', 'Complete living room furniture', 1999.00, 1)
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    quantity = EXCLUDED.quantity,
    updated_at = NOW();

-- Reset sequence for package_options
SELECT setval('package_option_id_seq', (SELECT MAX(id) FROM package_option));

-- Final summary
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Product Seed Data Summary:';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Categories: %', (SELECT COUNT(*) FROM category);
    RAISE NOTICE 'Attribute Definitions: %', (SELECT COUNT(*) FROM attribute_definition);
    RAISE NOTICE 'Category Attributes: %', (SELECT COUNT(*) FROM category_attribute);
    RAISE NOTICE 'Products: %', (SELECT COUNT(*) FROM product);
    RAISE NOTICE 'Product Attributes: %', (SELECT COUNT(*) FROM product_attribute);
    RAISE NOTICE 'Product Options: %', (SELECT COUNT(*) FROM product_option);
    RAISE NOTICE 'Product Option Values: %', (SELECT COUNT(*) FROM product_option_value);
    RAISE NOTICE 'Product Variants: %', (SELECT COUNT(*) FROM product_variant);
    RAISE NOTICE 'Variant Option Values: %', (SELECT COUNT(*) FROM variant_option_value);
    RAISE NOTICE 'Package Options: %', (SELECT COUNT(*) FROM package_option);
    RAISE NOTICE '========================================';
    RAISE NOTICE 'All product seed data inserted successfully!';
    RAISE NOTICE '========================================';
END $$;
