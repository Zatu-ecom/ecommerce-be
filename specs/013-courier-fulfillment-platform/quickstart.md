# Quickstart: Courier Fulfillment Platform (013)

## 1. Migrate + seed (from scratch, per CODING_STANDARDS)

```bash
dropdb ecommerce && createdb ecommerce
cd migrations && ./run_migrations.sh   # includes 032_create_fulfillment_tables.sql + 033_create_physical_spec_unit.sql
```

Verify:

```sql
\d fulfillment_shipment
SELECT code, kind FROM courier_provider;                    -- expect shiprocket row
SELECT parameter, COUNT(*) FROM physical_spec_unit
  WHERE is_active GROUP BY parameter;                       -- 4 families, 8 units
```

Seeds: `005_seed_physical_specs.sql` (core, ALL envs) + NULL-seller `courier_provider_config` placeholder from the `032` seed step.

## 2. Wire the module (one line + container)

```go
// main.go registerContainer(), alongside order/payment/...
fulfillment.NewContainer(router)
```

Container registers: seller courier routes (`SellerAuth`), customer track (`CustomerAuth`), webhook (`POST /api/fulfillment/webhooks/:code`, public, exempt from the mandatory-correlation-ID middleware — it generates + echoes one).

## 3. First manual flow (no courier account needed for drafts)

1. Seller: `PUT /couriers/shiprocket/configure` (test creds or empty → platform default applies).
2. Confirm an order (payment capture or COD) → planner creates drafts (one per warehouse).
3. Seller: `PATCH /shipments/:id` (weight/dims) → `POST /shipments/:id/book` → AWB returned.
4. `GET /shipments/:id/label` streams the PDF; `POST /shipments/:id/refresh` pulls live status.

## 4. Webhook + cron

- Register `{PUBLIC_API_BASE_URL}/api/fulfillment/webhooks/shiprocket` (shown by `GET /couriers`) in the Shiprocket dashboard; matrix: unknown AWB → 200/store-nothing, bad signature → 401/store-nothing, verified → 200 always (failures heal via cron).
- Crons via `common/cron`: `recover_drafts` 2m, `reconcile_pending` 5m, `ndr_sweep` 15m.

## 5. Tests

```bash
go test ./test/integration/fulfillment/... -v   # Testcontainers PG+Redis; Wiremock Shiprocket
```

Cover per constitution: happy path, authN/authZ failures, validation, edge cases, correlation-ID enforcement, seller isolation. Every endpoint needs the full matrix (see service-design §9 + api-contracts §7).

## 6. Full doc map

| Doc | Owns |
|---|---|
| `spec.md` | WHAT/WHY (stakeholder spec) |
| `plan.md` | HOW (this implementation plan) |
| `research.md` | Decisions R1–R10 with rationale |
| `data-model.md` | Tables T1–T8 + `physical_spec_unit` flows |
| `service-design.md` | Adapter, orchestrators, hooks, edge cases |
| `api-contracts.md` | Routes + JSON envelopes |
| `physical-specs.md` | Spec catalog DDL + core seed script |
| `contracts/courier-partner.md` | Code-facing Go interfaces |
