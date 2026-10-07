-- Seed: 005_seed_physical_specs.sql
-- Description: Shippable physical-spec definitions + unit catalog for fulfillment.
-- Environment: ALL (core data) — planner, rates, and booking read these keys.
-- Notes: fixed ids (mock seeds use 1-5+); idempotent; factor values are physics.
-- Spec: specs/013-courier-fulfillment-platform/physical-specs.md §3.

-- ============================================================================
-- 1. ATTRIBUTE DEFINITIONS - unit variants (the seller's "unit choice" is
--    which definition row they attach; product_attribute needs no new column)
-- ============================================================================
INSERT INTO attribute_definition (id, key, name, unit, allowed_values, created_at, updated_at) VALUES
(100, 'weight_g',  'Weight (g)',  'g',  NULL, NOW(), NOW()),
(101, 'weight_kg', 'Weight (kg)', 'kg', NULL, NOW(), NOW()),
(102, 'length_cm', 'Length (cm)', 'cm', NULL, NOW(), NOW()),
(103, 'length_m',  'Length (m)',  'm',  NULL, NOW(), NOW()),
(104, 'breadth_cm','Breadth (cm)','cm', NULL, NOW(), NOW()),
(105, 'breadth_m', 'Breadth (m)', 'm',  NULL, NOW(), NOW()),
(106, 'height_cm', 'Height (cm)', 'cm', NULL, NOW(), NOW()),
(107, 'height_m',  'Height (m)',  'm',  NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

SELECT setval('attribute_definition_id_seq', (SELECT MAX(id) FROM attribute_definition));

-- ============================================================================
-- 2. PHYSICAL SPEC UNITS - parameter families + factors to base (g / cm)
-- ============================================================================
INSERT INTO physical_spec_unit (
    key, attribute_definition_id, parameter,
    unit_short, unit_full, factor_to_base, is_base,
    display_order, is_active, created_at, updated_at
) VALUES
('weight_g',   100, 'weight',  'g',  'gram',       1,    TRUE,  1, TRUE, NOW(), NOW()),
('weight_kg',  101, 'weight',  'kg', 'kilogram',   1000, FALSE, 2, TRUE, NOW(), NOW()),
('length_cm',  102, 'length',  'cm', 'centimetre', 1,    TRUE,  1, TRUE, NOW(), NOW()),
('length_m',   103, 'length',  'm',  'metre',      100,  FALSE, 2, TRUE, NOW(), NOW()),
('breadth_cm', 104, 'breadth', 'cm', 'centimetre', 1,    TRUE,  1, TRUE, NOW(), NOW()),
('breadth_m',  105, 'breadth', 'm',  'metre',      100,  FALSE, 2, TRUE, NOW(), NOW()),
('height_cm',  106, 'height',  'cm', 'centimetre', 1,    TRUE,  1, TRUE, NOW(), NOW()),
('height_m',   107, 'height',  'm',  'metre',      100,  FALSE, 2, TRUE, NOW(), NOW())
ON CONFLICT (key) DO NOTHING;

-- ------------------------------
-- Summary + self-check
-- ------------------------------
DO $$
DECLARE
    families INT;
    without_base INT;
BEGIN
    SELECT COUNT(DISTINCT parameter) INTO families FROM physical_spec_unit WHERE is_active;
    SELECT COUNT(*) INTO without_base FROM (
        SELECT parameter FROM physical_spec_unit WHERE is_active
        GROUP BY parameter HAVING BOOL_OR(is_base) = FALSE
    ) s;
    IF without_base > 0 THEN
        RAISE EXCEPTION 'physical_spec_unit: % familie(s) without an active base unit', without_base;
    END IF;
    RAISE NOTICE 'Physical spec catalog seeded: % families, % units.',
        families, (SELECT COUNT(*) FROM physical_spec_unit WHERE is_active);
END $$;
