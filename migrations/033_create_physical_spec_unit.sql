-- Migration: 033_create_physical_spec_unit.sql
-- Description: Shippable physical-spec catalog (parameter → allowed units + factor).
-- Product module owns it. Keys match attribute_definition.key by convention;
-- attribute_definition_id FK keeps the link hard. Factors are seeded once and
-- immutable by convention (only is_active / display_order change at runtime).
-- Spec: specs/013-courier-fulfillment-platform/physical-specs.md §2.

CREATE TABLE IF NOT EXISTS physical_spec_unit (
    key VARCHAR(50) PRIMARY KEY,  -- weight_g, length_m, ... (matches attribute_definition.key)
    attribute_definition_id BIGINT NOT NULL UNIQUE
        REFERENCES attribute_definition(id) ON DELETE RESTRICT,
    parameter VARCHAR(20) NOT NULL,  -- weight | length | breadth | height
    unit_short VARCHAR(10) NOT NULL,  -- g, kg, cm, m
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

COMMENT ON TABLE physical_spec_unit IS 'Shippable spec catalog: which units each measurement allows + factor to base (g/cm); sellers pick definitions, fulfillment normalizes on read';
