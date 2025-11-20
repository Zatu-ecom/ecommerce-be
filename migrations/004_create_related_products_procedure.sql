-- Migration: 004_create_related_products_procedure.sql
-- Description: Create stored procedure for intelligent related products retrieval
-- Created: 2025-11-10

DROP FUNCTION IF EXISTS get_related_products_scored(BIGINT, BIGINT, INT, INT, TEXT);
DROP FUNCTION IF EXISTS get_related_products_count(BIGINT, BIGINT, TEXT);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_product_category_seller ON product(category_id, seller_id);
CREATE INDEX IF NOT EXISTS idx_product_brand_seller ON product(brand, seller_id);
CREATE INDEX IF NOT EXISTS idx_product_tags ON product USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_product_seller_created ON product(seller_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_category_parent ON category(parent_id);
CREATE INDEX IF NOT EXISTS idx_variant_product_price ON product_variant(product_id, price);
-- TODO: Add stock index when inventory service is integrated
CREATE INDEX IF NOT EXISTS idx_product_category_brand ON product(category_id, brand, seller_id);
CREATE INDEX IF NOT EXISTS idx_product_created_at ON product(created_at DESC);

-- Main function
CREATE OR REPLACE FUNCTION get_related_products_scored(
    p_product_id BIGINT,
    p_seller_id BIGINT DEFAULT NULL,
    p_limit INT DEFAULT 10,
    p_offset INT DEFAULT 0,
    p_strategies TEXT DEFAULT 'all'
)
RETURNS TABLE (
    product_id BIGINT,
    product_name VARCHAR,
    category_id BIGINT,
    category_name VARCHAR,
    parent_category_id BIGINT,
    parent_category_name VARCHAR,
    brand VARCHAR,
    sku VARCHAR,
    short_description TEXT,
    long_description TEXT,
    tags TEXT[],
    seller_id BIGINT,
    has_variants BOOLEAN,
    min_price DOUBLE PRECISION,
    max_price DOUBLE PRECISION,
    allow_purchase BOOLEAN,
    total_variants BIGINT,
    in_stock_variants BIGINT,
    created_at VARCHAR,
    updated_at VARCHAR,
    final_score INTEGER,
    relation_reason TEXT,
    strategy_used TEXT
) 
LANGUAGE plpgsql
AS $$
DECLARE
    v_source_category_id BIGINT;
    v_source_parent_category_id BIGINT;
    v_source_brand VARCHAR;
    v_source_tags TEXT[];
    v_source_min_price NUMERIC(10,2);
    v_source_max_price NUMERIC(10,2);
    v_source_seller_id BIGINT;
    v_enable_same_category BOOLEAN := TRUE;
    v_enable_same_brand BOOLEAN := TRUE;
    v_enable_sibling_category BOOLEAN := TRUE;
    v_enable_parent_category BOOLEAN := TRUE;
    v_enable_child_category BOOLEAN := TRUE;
    v_enable_tag_matching BOOLEAN := TRUE;
    v_enable_price_range BOOLEAN := TRUE;
    v_enable_seller_popular BOOLEAN := TRUE;
BEGIN
    SELECT 
        p.category_id,
        c.parent_id,
        p.brand,
        p.tags,
        p.seller_id,
        COALESCE(MIN(v.price), 0),
        COALESCE(MAX(v.price), 0)
    INTO 
        v_source_category_id,
        v_source_parent_category_id,
        v_source_brand,
        v_source_tags,
        v_source_seller_id,
        v_source_min_price,
        v_source_max_price
    FROM product p
    LEFT JOIN category c ON p.category_id = c.id
    LEFT JOIN product_variant v ON p.id = v.product_id
    WHERE p.id = p_product_id
    GROUP BY p.id, p.category_id, c.parent_id, p.brand, p.tags, p.seller_id;

    IF v_source_category_id IS NULL THEN
        RAISE EXCEPTION 'Product not found: %', p_product_id;
    END IF;

    IF p_seller_id IS NOT NULL AND v_source_seller_id != p_seller_id THEN
        RAISE EXCEPTION 'Product not found: %', p_product_id;
    END IF;

    IF p_strategies != 'all' THEN
        v_enable_same_category := p_strategies LIKE '%same_category%';
        v_enable_same_brand := p_strategies LIKE '%same_brand%';
        v_enable_sibling_category := p_strategies LIKE '%sibling_category%';
        v_enable_parent_category := p_strategies LIKE '%parent_category%';
        v_enable_child_category := p_strategies LIKE '%child_category%';
        v_enable_tag_matching := p_strategies LIKE '%tag_matching%';
        v_enable_price_range := p_strategies LIKE '%price_range%';
        v_enable_seller_popular := p_strategies LIKE '%seller_popular%';
    END IF;

    RETURN QUERY
    WITH 
    same_category AS (
        SELECT p.id, 100 as base_score, 'same_category' as strategy, 'Same category' as relation_reason
        FROM product p
        WHERE v_enable_same_category AND p.category_id = v_source_category_id AND p.id != p_product_id 
          AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    same_brand AS (
        SELECT p.id, 80 as base_score, 'same_brand' as strategy, 'Same brand: ' || p.brand as relation_reason
        FROM product p
        WHERE v_enable_same_brand AND p.brand = v_source_brand AND p.brand != '' AND p.brand IS NOT NULL
          AND p.category_id != v_source_category_id AND p.id != p_product_id 
          AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    sibling_category AS (
        SELECT p.id, 70 as base_score, 'sibling_category' as strategy, 'Related category: ' || c.name as relation_reason
        FROM product p INNER JOIN category c ON p.category_id = c.id
        WHERE v_enable_sibling_category AND v_source_parent_category_id IS NOT NULL 
          AND c.parent_id = v_source_parent_category_id AND p.category_id != v_source_category_id 
          AND p.id != p_product_id AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    parent_category AS (
        SELECT p.id, 60 as base_score, 'parent_category' as strategy, 'Broader category: ' || c.name as relation_reason
        FROM product p INNER JOIN category c ON p.category_id = c.id
        WHERE v_enable_parent_category AND v_source_parent_category_id IS NOT NULL 
          AND p.category_id = v_source_parent_category_id AND p.id != p_product_id 
          AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    child_category AS (
        SELECT p.id, 55 as base_score, 'child_category' as strategy, 'Sub-category: ' || c.name as relation_reason
        FROM product p INNER JOIN category c ON p.category_id = c.id
        WHERE v_enable_child_category AND c.parent_id = v_source_category_id AND p.id != p_product_id 
          AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    tag_matching AS (
        SELECT p.id,
            CASE 
                WHEN cardinality(ARRAY(SELECT UNNEST(p.tags) INTERSECT SELECT UNNEST(v_source_tags))) >= 5 THEN 50
                WHEN cardinality(ARRAY(SELECT UNNEST(p.tags) INTERSECT SELECT UNNEST(v_source_tags))) >= 3 THEN 40
                WHEN cardinality(ARRAY(SELECT UNNEST(p.tags) INTERSECT SELECT UNNEST(v_source_tags))) = 2 THEN 30
                ELSE 20
            END as base_score,
            'tag_matching' as strategy, 'Similar tags' as relation_reason
        FROM product p
        WHERE v_enable_tag_matching AND v_source_tags IS NOT NULL AND cardinality(v_source_tags) > 0 
          AND p.tags && v_source_tags AND p.id != p_product_id 
          AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    price_range AS (
        SELECT p.id, 25 as base_score, 'price_range' as strategy, 'Similar price range' as relation_reason
        FROM product p
        INNER JOIN (SELECT pv.product_id, MIN(pv.price) as min_price, MAX(pv.price) as max_price
                    FROM product_variant pv GROUP BY pv.product_id) pv ON p.id = pv.product_id
        WHERE v_enable_price_range AND v_source_min_price > 0 
          AND pv.min_price BETWEEN v_source_min_price * 0.7 AND v_source_max_price * 1.3 
          AND p.id != p_product_id AND (p_seller_id IS NULL OR p.seller_id = p_seller_id)
    ),
    seller_popular AS (
        SELECT p.id, 15 as base_score, 'seller_popular' as strategy, 'More from this seller' as relation_reason
        FROM product p
        WHERE v_enable_seller_popular AND p.seller_id = v_source_seller_id AND p.id != p_product_id
        ORDER BY p.created_at DESC LIMIT 50
    ),
    all_strategies AS (
        SELECT * FROM same_category UNION ALL SELECT * FROM same_brand UNION ALL
        SELECT * FROM sibling_category UNION ALL SELECT * FROM parent_category UNION ALL
        SELECT * FROM child_category UNION ALL SELECT * FROM tag_matching UNION ALL
        SELECT * FROM price_range UNION ALL SELECT * FROM seller_popular
    ),
    scored_products AS (
        SELECT s.id, s.strategy, s.relation_reason, s.base_score,
            CASE WHEN p.brand = v_source_brand AND p.brand != '' AND p.brand IS NOT NULL AND p.category_id = v_source_category_id THEN 50 ELSE 0 END as brand_category_bonus,
            CASE WHEN p.brand = v_source_brand AND p.brand != '' AND p.brand IS NOT NULL AND c.parent_id = v_source_parent_category_id AND p.category_id != v_source_category_id THEN 30 ELSE 0 END as brand_sibling_bonus,
            CASE WHEN v_source_tags IS NOT NULL AND cardinality(v_source_tags) > 0 THEN cardinality(ARRAY(SELECT UNNEST(p.tags) INTERSECT SELECT UNNEST(v_source_tags))) * 5 ELSE 0 END as tag_bonus,
            CASE WHEN pv.min_price BETWEEN v_source_min_price * 0.9 AND v_source_max_price * 1.1 THEN 15 ELSE 0 END as price_similarity_bonus,
            CASE WHEN p.created_at > NOW() - INTERVAL '30 days' THEN 10 ELSE 0 END as recency_bonus,
            0 as stock_bonus, -- TODO: Add when inventory service is integrated
            0 as stock_penalty, -- TODO: Add when inventory service is integrated
            CASE WHEN v_source_min_price > 0 AND (pv.max_price > v_source_max_price * 2 OR pv.max_price < v_source_min_price * 0.5) THEN -20 ELSE 0 END as price_diff_penalty
        FROM all_strategies s
        INNER JOIN product p ON s.id = p.id
        LEFT JOIN category c ON p.category_id = c.id
        LEFT JOIN (SELECT pv.product_id, MIN(pv.price) as min_price, MAX(pv.price) as max_price, BOOL_OR(pv.allow_purchase) as allow_purchase, COUNT(*) as total_variants, COUNT(*) as in_stock_variants FROM product_variant pv GROUP BY pv.product_id) pv ON p.id = pv.product_id
    ),
    deduplicated_scored AS (
        SELECT sp.id,
            MAX(sp.base_score + sp.brand_category_bonus + sp.brand_sibling_bonus + sp.tag_bonus + sp.price_similarity_bonus + sp.recency_bonus + sp.stock_bonus + sp.stock_penalty + sp.price_diff_penalty) as final_score,
            (ARRAY_AGG(sp.relation_reason ORDER BY (sp.base_score + sp.brand_category_bonus + sp.brand_sibling_bonus + sp.tag_bonus) DESC))[1] as relation_reason,
            (ARRAY_AGG(sp.strategy ORDER BY (sp.base_score + sp.brand_category_bonus + sp.brand_sibling_bonus + sp.tag_bonus) DESC))[1] as strategy_used
        FROM scored_products sp
        GROUP BY sp.id
        HAVING MAX(sp.base_score + sp.brand_category_bonus + sp.brand_sibling_bonus + sp.tag_bonus + sp.price_similarity_bonus + sp.recency_bonus + sp.stock_bonus + sp.stock_penalty + sp.price_diff_penalty) >= 10
    ),
    ranked_products AS (
        SELECT ds.id, ds.final_score, ds.relation_reason, ds.strategy_used,
               ROW_NUMBER() OVER (ORDER BY ds.final_score DESC, ds.id DESC) as rn
        FROM deduplicated_scored ds
    ),
    paginated_ids AS (
        SELECT rp.id, rp.final_score, rp.relation_reason, rp.strategy_used
        FROM ranked_products rp
        WHERE rp.rn > p_offset
        ORDER BY rp.final_score DESC, rp.id DESC
        LIMIT p_limit
    )
    SELECT p.id, p.name, p.category_id, c.name, c.parent_id, pc.name, p.brand, p.base_sku AS sku, 
           p.short_description, p.long_description, p.tags, p.seller_id,
           COALESCE(CASE WHEN pv.total_variants > 0 THEN TRUE ELSE FALSE END, FALSE), 
           COALESCE(pv.min_price, 0.0), 
           COALESCE(pv.max_price, 0.0),
           COALESCE(pv.allow_purchase, FALSE),
           COALESCE(pv.total_variants, 0::BIGINT), 
           COALESCE(pv.in_stock_variants, 0::BIGINT),
           p.created_at::VARCHAR, 
           p.updated_at::VARCHAR, 
           pi.final_score, 
           pi.relation_reason, 
           pi.strategy_used
    FROM paginated_ids pi
    INNER JOIN product p ON pi.id = p.id
    LEFT JOIN category c ON p.category_id = c.id
    LEFT JOIN category pc ON c.parent_id = pc.id
    LEFT JOIN (SELECT v.product_id, MIN(v.price) as min_price, MAX(v.price) as max_price, BOOL_OR(v.allow_purchase) as allow_purchase, COUNT(*) as total_variants, COUNT(*) as in_stock_variants FROM product_variant v GROUP BY v.product_id) pv ON p.id = pv.product_id
    ORDER BY pi.final_score DESC, p.created_at DESC;
END;
$$;

CREATE OR REPLACE FUNCTION get_related_products_count(p_product_id BIGINT, p_seller_id BIGINT DEFAULT NULL, p_strategies TEXT DEFAULT 'all')
RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE v_count BIGINT;
BEGIN
    SELECT COUNT(*) INTO v_count FROM get_related_products_scored(p_product_id, p_seller_id, 999999, 0, p_strategies);
    RETURN v_count;
END;
$$;

ANALYZE product;
ANALYZE category;
ANALYZE product_variant;

COMMENT ON FUNCTION get_related_products_scored IS 'Retrieves related products using 8 strategies. NOTE: Stock management TODO when inventory service is integrated.';
COMMENT ON FUNCTION get_related_products_count IS 'Returns count of related products for pagination.';
