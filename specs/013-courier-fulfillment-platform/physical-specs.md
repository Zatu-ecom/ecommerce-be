# 013 — Shippable Product Specs (dimensions + units) · Plan + Setup Script

> Why this exists: courier booking needs real numbers (weight in grams, dims in cm), but products carry only free-form attributes. Nothing before booking ever did math on physical specs, so the gap never hurt — fulfillment surfaces it. This doc defines the one backend-owned source of truth, the API that serves it, the dashboard rule, the fulfillment read path, and the missing-data fallbacks.
>
> Decisions locked in chat: selectable units per parameter (laptop in cm, furniture in m); unit variants as definitions (no new columns on attribute tables); API shape `[{parameter, options: [{key, unit: {short, full}}]}]`; missing data degrades to human action, corrupt data to loud errors.

## 1. Design summary

- **One new table** `physical_spec_unit` (product module): parameter → allowed units + factor to base unit. Base units: grams (weight), cm (dims).
- **Eight definition rows** in `attribute_definition` (unit variants: `weight_g`, `weight_kg`, `length_cm`, `length_m`, …), linked 1:1 from the new table. Sellers attach these to products through the existing definition↔product relation — the "unit choice" *is* the definition choice, so `product_attribute` needs no new column.
- **One API** serving the grouped spec list to the dashboard.
- **One reader** in fulfillment (`FulfillmentProductHooks.GetPhysicalSpecs`): join by key, × factor, hand the planner base units.

## 2. DDL — migration `033_create_physical_spec_unit.sql` (lands after `032`)

```sql
-- Migration: 033_create_physical_spec_unit.sql
-- Description: Shippable physical-spec catalog (parameter → allowed units + factor).
-- Product module owns it. Keys match attribute_definition.key by convention;
-- attribute_definition_id FK keeps the link hard. Factors are seeded once and
-- immutable by convention (only is_active / display_order change at runtime).

CREATE TABLE IF NOT EXISTS physical_spec_unit (
    key VARCHAR(50) PRIMARY KEY,  -- weight_g, length_m, ... (matches attribute_definition.key)
    attribute_definition_id BIGINT NOT NULL UNIQUE
        REFERENCES attribute_definition(id) ON DELETE RESTRICT,
    parameter VARCHAR(20) NOT NULL,  -- weight | length | breadth | height
    unit_short VARCHAR(10) NOT NULL, -- g, kg, cm, m
    unit_full VARCHAR(50) NOT NULL,   -- gram, kilogram, centimetre, metre
    factor_to_base NUMERIC NOT NULL CHECK (factor_to_base > 0),
    is_base BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Exactly one base unit per family (g for weight, cm for dims).
CREATE UNIQUE INDEX IF NOT EXISTS uq_physical_spec_unit_base
    ON physical_spec_unit (parameter) WHERE is_base;

CREATE INDEX IF NOT EXISTS idx_physical_spec_unit_parameter
    ON physical_spec_unit (parameter);
```

## 3. Core setup script — `migrations/seeds/core/005_seed_physical_specs.sql`

Core (ALL environments), fixed ids 100–107 (mock seeds occupy 1–5+), idempotent upserts, sequence reset. Run order: after `032`/`033` migrations and after attribute tables exist.

```sql
-- Seed: 005_seed_physical_specs.sql
-- Description: Shippable physical-spec definitions + unit catalog for fulfillment.
-- Environment: ALL (core data) — planner, rates, and booking read these keys.
-- Notes: fixed ids (mock seeds use 1-5+); idempotent; factor values are physics.

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
ON CONFLICT (id) DO UPDATE SET
    key = EXCLUDED.key,
    name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    updated_at = NOW();

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
ON CONFLICT (key) DO UPDATE SET
    attribute_definition_id = EXCLUDED.attribute_definition_id,
    parameter = EXCLUDED.parameter,
    unit_short = EXCLUDED.unit_short,
    unit_full = EXCLUDED.unit_full,
    factor_to_base = EXCLUDED.factor_to_base,
    is_base = EXCLUDED.is_base,
    display_order = EXCLUDED.display_order,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

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
```

## 4. The one API (product module)

`GET /api/products/attribute-definitions?scope=fulfillment` — active rows grouped by `parameter`, ordered by `display_order`:

```json
[
  { "parameter": "weight", "name": "Weight",
    "options": [ { "key": "weight_g", "unit": { "short": "g", "full": "gram" } },
                 { "key": "weight_kg", "unit": { "short": "kg", "full": "kilogram" } } ] },
  { "parameter": "length", "name": "Length",
    "options": [ { "key": "length_cm", "unit": { "short": "cm", "full": "centimetre" } },
                 { "key": "length_m", "unit": { "short": "m", "full": "metre" } } ] }
]
```

No hardcoded unit lists in the frontend: adding `mm`/`lb` later is one definition row + one catalog row.

## 5. Dashboard + validation rule

- Product form calls the API, renders a fixed "Shipping specs" block: numeric input + unit dropdown per parameter. `product_attribute` stores `{attribute_definition_id, value}` — number only, unit implied by the chosen definition.
- Product write path enforces: known shippable key, numeric value > 0, **at most one key per family per product** (a product with both `length_cm` and `length_m` is rejected). `category_attribute.is_required` can later hard-require specs per category (see §8).
- Existing catalog without specs gets a "shipping specs missing" badge, not a blocker.

## 6. Fulfillment read path

`FulfillmentProductHooks.GetPhysicalSpecs(variantIDs)` (service-design §6): join `product_attribute` → `physical_spec_unit` by key → return grams/cm after × factor. Planner sums weights, applies the dims heuristic (max L, max B, stacked H), stamps the draft; rates/booking reuse the same reader. In-memory catalog load at boot (same cachekit version pattern as other catalogs); boot self-check mirrors the seed guard (every family keeps one active base row).

## 7. Missing/bad data — fallback matrix

| Situation | Stage | Behavior |
|---|---|---|
| New/edited product, specs absent | Product creation | Warn + badge, don't block |
| Existing product without specs, order arrives | Planner | Plans draft, weight NULL, flagged for manual PATCH |
| Draft without weight at book | Book | 400 `WEIGHT_REQUIRED` → config `default_weight_grams` → else visibly skipped |
| Value unparseable / unknown key | Any read | Loud error + alert; never guessed conversions or silent zeros |
| Two keys of one family on a product | Product write | Rejected |
| Family without active base row | Boot / seed | Refuse planning traffic (seed raises; service self-checks) |

Rule of thumb: **missing data degrades to human action, corrupt data to loud errors** — the system never invents numbers.

## 8. Execution + tests

1. Land `032` (fulfillment tables) first, then `033` + core seed `005` above; run all migrations from scratch and confirm the NOTICE output (4 families, 8 units).
2. Product: specs endpoint, write validation, dashboard block + badge; integration tests (response shape, per-family rejection, non-numeric rejected).
3. Fulfillment docs: `FulfillmentProductHooks` + reader + matrix into service-design §5.2/§6 (follow-up edit).
4. Seed idempotency: re-run `005` twice, confirm counts unchanged.

Open calls carried over: (a) warn-vs-block for new products in shippable categories (`category_attribute.is_required` is the lever); (b) backfill beyond badge + PATCH flow (bulk seller reminder?).
