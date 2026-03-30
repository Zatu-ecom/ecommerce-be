-- Align promotion scope tables with db.BaseEntity, which expects both created_at and updated_at.

ALTER TABLE promotion_product
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE promotion_product_variant
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE promotion_category
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE promotion_collection
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_promotion_product_updated_at') THEN
        CREATE TRIGGER update_promotion_product_updated_at
        BEFORE UPDATE ON promotion_product
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_promotion_product_variant_updated_at') THEN
        CREATE TRIGGER update_promotion_product_variant_updated_at
        BEFORE UPDATE ON promotion_product_variant
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_promotion_category_updated_at') THEN
        CREATE TRIGGER update_promotion_category_updated_at
        BEFORE UPDATE ON promotion_category
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_promotion_collection_updated_at') THEN
        CREATE TRIGGER update_promotion_collection_updated_at
        BEFORE UPDATE ON promotion_collection
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
