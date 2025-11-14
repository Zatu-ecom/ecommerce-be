-- Test Seed Data for GetProductByID Tests
-- This file contains additional test data beyond the base seed data
-- to cover all test scenarios for the GetProductByID API

-- Additional test products for edge cases and specific scenarios

-- Product 101: Product with multiple variants for testing (already covered in base seed)
-- Product 102: Product with empty optional fields
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(102, 'Minimal Product', 4, '', '', '', '', ARRAY[]::TEXT[], 2, NOW(), NOW());

-- Product 103: Product with maximum field lengths
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(103, 
 REPEAT('A', 200), -- Exactly 200 characters
 4,
 REPEAT('B', 100), -- Exactly 100 characters
 'SKU-' || REPEAT('X', 46), -- 50 characters total (SKU- is 4 chars)
 REPEAT('Short description text. ', 20), -- ~500 characters
 REPEAT('Long description text with more details about the product and its features. ', 65), -- ~5000 characters
 ARRAY['tag1', 'tag2', 'tag3', 'tag4', 'tag5', 'tag6', 'tag7', 'tag8', 'tag9', 'tag10', 'tag11', 'tag12', 'tag13', 'tag14', 'tag15', 'tag16', 'tag17', 'tag18', 'tag19', 'tag20'],
 2,
 NOW(),
 NOW());

-- Product 104: Product with Unicode characters
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(104, 
 'Смартфон Продукт 中文产品 😀🎉',
 4,
 'العربية',
 'UNICODE-PROD-001',
 'مرحبا بك في منتجنا',
 'This product has Unicode: 中文, العربية, हिन्दी, 日本語, 한국어, Ελληνικά, Русский and emojis: 😀🎉🌟💻📱',
 ARRAY['unicode', '中文', 'العربية', 'emoji😀'],
 2,
 NOW(),
 NOW());

-- Product 105: Product with zero-price variant
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(105, 'Free Sample Product', 4, 'FreeStuff', 'FREE-001', 'Free product sample', 'This product is offered as a free sample', ARRAY['free', 'sample'], 2, NOW(), NOW());

-- Product 106: Product with special characters in variant options
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(106, 'Special Chars Product', 4, 'SpecialBrand', 'SPECIAL-001', 'Product with special characters', 'Testing special characters in options', ARRAY['special'], 2, NOW(), NOW());

-- Product 107: Product belonging to seller 3 for multi-tenant testing
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(107, 'Seller 3 Product', 7, 'SellerThreeBrand', 'S3-PROD-001', 'Product for seller 3', 'This product belongs to seller 3', ARRAY['seller3'], 3, NOW(), NOW());

-- Product 108: Product with all variants unavailable
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(108, 'Out of Stock Product', 4, 'OutOfStock', 'OOS-001', 'All variants unavailable', 'Product with all variants out of stock', ARRAY['unavailable'], 2, NOW(), NOW());

-- Product 109: Product with many attributes (50 attributes)
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(109, 'Product with Many Attributes', 5, 'TechSpec', 'MANYATTR-001', 'Product with extensive specifications', 'This product has many detailed attributes', ARRAY['detailed'], 2, NOW(), NOW());

-- Product 110: Product with very long SKU
INSERT INTO product (id, name, category_id, brand, base_sku, short_description, long_description, tags, seller_id, created_at, updated_at) VALUES
(110, 'Long SKU Product', 4, 'LongSKU', 'VERYLONGSKU' || REPEAT('-SEGMENT', 8), 'Product with long SKU', 'Testing long SKU handling', ARRAY['longsku'], 2, NOW(), NOW());

-- Reset sequence
SELECT setval('product_id_seq', (SELECT MAX(id) FROM product));

-- Create product options for test products

-- Product 102 (Minimal) - 1 option
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(101, 102, 'Color', 'Color', 1);

-- Product 103 (Max Fields) - 2 options
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(102, 103, 'Color', 'Color', 1),
(103, 103, 'Size', 'Size', 2);

-- Product 104 (Unicode) - 2 options
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(104, 104, 'Color', 'Color 颜色', 1),
(105, 104, 'Storage', 'Storage 储存', 2);

-- Product 105 (Free) - 1 option
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(106, 105, 'Type', 'Type', 1);

-- Product 106 (Special Characters) - 2 options
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(107, 106, 'Color', 'Color', 1),
(108, 106, 'Size', 'Size', 2);

-- Product 107 (Seller 3) - 1 option
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(109, 107, 'Size', 'Size', 1);

-- Product 108 (Out of Stock) - 1 option
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(110, 108, 'Color', 'Color', 1);

-- Product 109 (Many Attributes) - 2 options
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(111, 109, 'Color', 'Color', 1),
(112, 109, 'Config', 'Configuration', 2);

-- Product 110 (Long SKU) - 1 option
INSERT INTO product_option (id, product_id, name, display_name, position) VALUES
(113, 110, 'Color', 'Color', 1);

SELECT setval('product_option_id_seq', (SELECT MAX(id) FROM product_option));

-- Create option values

-- Product 102 option values
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(201, 101, 'Black', 'Black', '#000000', 1);

-- Product 103 option values
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(202, 102, 'Red', 'Red', '#FF0000', 1),
(203, 103, 'M', 'Medium', NULL, 1);

-- Product 104 option values (Unicode)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(204, 104, '红色', 'Red 红色', '#FF0000', 1),
(205, 105, '128GB', '128GB', NULL, 1);

-- Product 105 option values (Free)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(206, 106, 'Trial', 'Trial Sample', NULL, 1);

-- Product 106 option values (Special Characters)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(207, 107, 'Red & Blue', 'Red & Blue', NULL, 1),
(208, 108, 'Size: M/L', 'Size: M/L', NULL, 1);

-- Product 107 option values (Seller 3)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(209, 109, 'L', 'Large', NULL, 1);

-- Product 108 option values (Out of Stock)
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(210, 110, 'Gray', 'Gray', '#808080', 1);

-- Product 109 option values
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(211, 111, 'Silver', 'Silver', '#C0C0C0', 1),
(212, 112, 'Standard', 'Standard Config', NULL, 1);

-- Product 110 option values
INSERT INTO product_option_value (id, option_id, value, display_name, color_code, position) VALUES
(213, 113, 'Blue', 'Blue', '#0000FF', 1);

SELECT setval('product_option_value_id_seq', (SELECT MAX(id) FROM product_option_value));

-- Create variants

-- Product 102 variants (Minimal - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(201, 102, 'MINIMAL-BLK', 9.99, ARRAY['https://example.com/minimal.jpg'], true, false, true);

-- Product 103 variants (Max Fields - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(202, 103, 'MAXFIELD-RED-M', 99.99, ARRAY['https://example.com/maxfield.jpg'], true, false, true);

-- Product 104 variants (Unicode - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(203, 104, 'UNICODE-红色-128GB', 899.99, ARRAY['https://example.com/unicode-phone.jpg'], true, false, true);

-- Product 105 variants (Zero Price - 1 free variant, 1 paid variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(204, 105, 'FREE-TRIAL-V1', 0.00, ARRAY['https://example.com/free-sample1.jpg'], true, false, true),
(205, 105, 'PAID-VERSION', 19.99, ARRAY['https://example.com/paid-sample.jpg'], true, false, false);

-- Product 106 variants (Special Characters - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(206, 106, 'SPECIAL-R&B-M/L', 79.99, ARRAY['https://example.com/special-chars.jpg'], true, false, true);

-- Product 107 variants (Seller 3 - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(207, 107, 'S3-PROD-L', 149.99, ARRAY['https://example.com/seller3-product.jpg'], true, false, true);

-- Product 108 variants (Out of Stock - 1 variant, unavailable)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(208, 108, 'OOS-GRAY', 59.99, ARRAY['https://example.com/outofstock.jpg'], false, false, true);

-- Product 109 variants (Many Attributes - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(209, 109, 'MANYATTR-SLV-STD', 1299.99, ARRAY['https://example.com/manyattr.jpg'], true, false, true);

-- Product 110 variants (Long SKU - 1 variant)
INSERT INTO product_variant (id, product_id, sku, price, images, allow_purchase, is_popular, is_default) VALUES
(210, 110, 'VERYLONGSKU-SEGMENT-SEGMENT-SEGMENT-BLUE', 99.99, ARRAY['https://example.com/longsku.jpg'], true, false, true);

SELECT setval('product_variant_id_seq', (SELECT MAX(id) FROM product_variant));

-- Link variants to option values

-- Product 102
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(201, 101, 201);

-- Product 103
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(202, 102, 202),
(202, 103, 203);

-- Product 104
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(203, 104, 204),
(203, 105, 205);

-- Product 105 (Free - variant 1 with trial option)
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(204, 106, 206);

-- Product 105 (Free - variant 2 with trial option) - Note: both use same option value
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(205, 106, 206);

-- Product 106
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(206, 107, 207),
(206, 108, 208);

-- Product 107
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(207, 109, 209);

-- Product 108
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(208, 110, 210);

-- Product 109
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(209, 111, 211),
(209, 112, 212);

-- Product 110
INSERT INTO variant_option_value (variant_id, option_id, option_value_id) VALUES
(210, 113, 213);

-- Create many attributes for Product 109 (50 attributes)
DO $$
DECLARE
    i INTEGER;
    attr_id BIGINT;
    attr_key TEXT;
    attr_name TEXT;
BEGIN
    FOR i IN 1..50 LOOP
        attr_key := 'spec_' || i;
        attr_name := 'Specification ' || i;
        
        -- Check if attribute definition exists
        SELECT id INTO attr_id FROM attribute_definition WHERE key = attr_key;
        
        -- Insert attribute definition if not exists
        IF attr_id IS NULL THEN
            INSERT INTO attribute_definition (key, name, unit, allowed_values, created_at, updated_at)
            VALUES (attr_key, attr_name, 'unit', NULL, NOW(), NOW())
            RETURNING id INTO attr_id;
        END IF;
        
        -- Insert product attribute
        INSERT INTO product_attribute (product_id, attribute_definition_id, value, sort_order, created_at, updated_at)
        VALUES (109, attr_id, 'Value ' || i, i, NOW(), NOW());
    END LOOP;
END $$;

-- Summary
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'GetProductByID Test Seed Data Summary:';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Additional Test Products: %', (SELECT COUNT(*) FROM product WHERE id >= 102);
    RAISE NOTICE 'Additional Product Options: %', (SELECT COUNT(*) FROM product_option WHERE id >= 101);
    RAISE NOTICE 'Additional Option Values: %', (SELECT COUNT(*) FROM product_option_value WHERE id >= 201);
    RAISE NOTICE 'Additional Variants: %', (SELECT COUNT(*) FROM product_variant WHERE id >= 201);
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Test seed data inserted successfully!';
    RAISE NOTICE '========================================';
END $$;
