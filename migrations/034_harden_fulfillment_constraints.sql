-- 034: Harden fulfillment constraints (PR #71 review B3/D5).
-- Idempotency keys are seller-scoped: UNIQUE(seller_id, idempotency_key).
-- Plus missing indexes and non-negative money/dimension CHECKs.

-- Drop the global unique (exists in 032) and replace with a scoped partial one.
ALTER TABLE fulfillment_shipment DROP CONSTRAINT IF EXISTS fulfillment_shipment_idempotency_key_key;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_fulfillment_shipment_seller_idem'
    ) THEN
        ALTER TABLE fulfillment_shipment
            ADD CONSTRAINT uq_fulfillment_shipment_seller_idem
            UNIQUE (seller_id, idempotency_key);
    END IF;
END $$;

-- Provider uniques should ignore NULLs explicitly (PG already does, be explicit).
-- No-op if partial variants already exist; kept idempotent.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'uq_fulfillment_ship_prov_order_partial') THEN
        CREATE UNIQUE INDEX uq_fulfillment_ship_prov_order_partial
            ON fulfillment_shipment(provider_code, provider_order_id)
            WHERE provider_order_id IS NOT NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'uq_fulfillment_ship_awb_partial') THEN
        CREATE UNIQUE INDEX uq_fulfillment_ship_awb_partial
            ON fulfillment_shipment(provider_code, awb)
            WHERE awb IS NOT NULL;
    END IF;
END $$;

-- Missing lookup indexes (resolve/freeze + webhook seller list + join).
CREATE INDEX IF NOT EXISTS idx_fulfillment_shipment_provider_config
    ON fulfillment_shipment(provider_config_id);
CREATE INDEX IF NOT EXISTS idx_fulfillment_webhook_log_shipment
    ON fulfillment_webhook_log(shipment_id);
CREATE INDEX IF NOT EXISTS idx_fulfillment_webhook_log_awb
    ON fulfillment_webhook_log(awb);

-- Non-negative money + positive dimensions (weight already checked in 032).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_fulfillment_ship_money_nonneg') THEN
        ALTER TABLE fulfillment_shipment
            ADD CONSTRAINT chk_fulfillment_ship_money_nonneg
            CHECK (cod_cents >= 0 AND (rate_cents IS NULL OR rate_cents >= 0));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_fulfillment_ship_dims_positive') THEN
        ALTER TABLE fulfillment_shipment
            ADD CONSTRAINT chk_fulfillment_ship_dims_positive
            CHECK (
                (length_cm IS NULL OR length_cm > 0) AND
                (breadth_cm IS NULL OR breadth_cm > 0) AND
                (height_cm IS NULL OR height_cm > 0)
            );
    END IF;
END $$;
