-- Align collection_product with db.BaseEntity, which expects both created_at and updated_at.

ALTER TABLE collection_product
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_collection_product_updated_at') THEN
        CREATE TRIGGER update_collection_product_updated_at
        BEFORE UPDATE ON collection_product
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
