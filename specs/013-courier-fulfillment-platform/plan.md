# Implementation Plan: Courier Fulfillment Platform

**Branch**: `013-courier-fulfillment-platform` | **Date**: 2026-10-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/013-courier-fulfillment-platform/spec.md`

## Summary

Ship orders through courier partners via a provider-agnostic gateway (Shiprocket first): sellers connect accounts (or use the platform default), confirmed orders auto-split into one draft box per warehouse, sellers book boxes to receive tracking numbers, courier pushes advance tracking automatically with a cron safety net, and failed deliveries/returns follow defined actions. Approach mirrors `010-payment-gateway-platform` (single provider contract, locate→verify→apply webhooks, shared reconciler path) — see `research.md` R1–R10.

## Technical Context

**Language/Version**: Go 1.25+ (module `ecommerce-be`)
**Primary Dependencies**: Gin (HTTP), GORM (ORM), `go-playground/validator/v10`, testify/suite, Testcontainers, stdlib `net/http` + `crypto/hmac`/`sha256` (no courier SDK), `robfig/cron/v3` via `common/cron`, `redis/go-redis/v9` via `common/cachekit`
**Storage**: PostgreSQL 16 (new tables, migration `032`; `033` for the physical-spec catalog; GORM `SingularTable: true`); Redis 7 volatile (rates/track/catalog, fail-open) + durable KV (idempotency, version counters, rate limiter, Shiprocket JWT)
**Testing**: `go test ./test/integration/fulfillment/... -v` (Testcontainers PG+Redis, Wiremock Shiprocket); full endpoint matrix per constitution (happy/authN/authZ/validation/edge/correlation/seller-isolation)
**Target Platform**: Linux server (Docker multi-stage, docker-compose)
**Project Type**: Web service module (modular monolith, microservice-extractable)
**Performance Goals**: Track/rate reads served from volatile cache (45–90s jittered TTL, singleflight); reconciler bulk-tracks 100 AWBs/batch with `SKIP LOCKED`; no N+1 (`Preload(Items)`); paginated lists (pageSize ≤ 100)
**Constraints**: Courier HTTP never inside DB transactions; POSTs never retried (provider-deduped); no `KEYS`/prefix deletes on request path; webhook route exempt from mandatory-correlation-ID middleware (generates + echoes)
**Scale/Scope**: Per-seller multi-warehouse sellers; 20–50 tracking events per box over days; reconciler sweeps only stale rows

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence |
|---|---|---|
| I. Modular monolith — no cross-repo access | ✅ PASS | `fulfillment/` owns entity/repo/service/handler/route/factory; order/inventory/product reached only via `FulfillmentOrderHooks` / `FulfillmentInventoryHooks` / `FulfillmentProductHooks` (`contracts/courier-partner.md`, service-design §6) |
| II. Layered architecture | ✅ PASS | Handler→Service→Repository→DB; services return DTOs, repos return entities; shared `NormalizedApplier` for webhook+cron (§5.6) |
| III. Factory-singleton DI | ✅ PASS | `CourierPartnerFactory` registry + three-tier singleton factory per module standard; constructor injection throughout |
| IV. TDD, integration-first | ✅ PASS | Test matrix in service-design §9 + quickstart §5; suite pattern, API-first assertions, per-test cleanup per constitution |
| V. Seller isolation | ✅ PASS | `seller_id` on every seller-scoped row/query; wrong-owner → 404 (api-contracts §1); cross-seller probe tests required |
| VI. Correlation ID | ✅ PASS | Mandatory on all endpoints (400 `CORRELATION_ID_REQUIRED`); webhook generates+echoes; propagated to logs/courier calls |
| VII. RBAC | ✅ PASS | `SellerAuth` (dashboard/config/book), `CustomerAuth` + ownership check (track), public webhook (signature-verified) |
| VIII. Backward compat | ✅ PASS | New tables/endpoints only; no ALTERs of existing tables; no contract changes to other modules (hooks are additive) |
| IX. SOLID | ✅ PASS | ISP via optional capability interfaces (`NDRHandler`, `ReturnHandler`, …); OCP via adapter-per-folder; ≤10-method interfaces |
| X. Performance by design | ✅ PASS | cachekit volatile/durable roles respected; jittered TTLs; singleflight; exact-key dels + version counters; indexes per query path |

Post-design re-check: no new violations introduced by Phase 1 artifacts. No Complexity Tracking entries needed.

## Project Structure

### Documentation (this feature)

```text
specs/013-courier-fulfillment-platform/
├── plan.md              # This file
├── spec.md              # Stakeholder WHAT/WHY
├── research.md          # Decisions R1–R10
├── data-model.md        # Tables T1–T8 + flows
├── service-design.md    # Adapter, orchestrators, hooks, edge cases
├── api-contracts.md     # Routes + JSON envelopes
├── physical-specs.md    # Spec catalog DDL + core seed script
├── quickstart.md        # Bring-up + first flows
├── contracts/           # Code-facing Go interfaces
│   └── courier-partner.md
├── checklists/
│   └── requirements.md  # Spec quality gate (all pass)
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

New `fulfillment/` module + cross-module touch points only (no existing files modified except `main.go` wiring):

```text
fulfillment/
├── container.go
├── entity/            # shipment, items, events, provider, config, webhook_log, ndr, pickup mapping
├── model/             # DTOs (request/response, validation tags, pointer updates, dive)
├── repository/        # *_repository.go + *_repository_impl.go (GORM, seller-scoped)
├── service/
│   ├── courier/contract.go + crypto.go
│   ├── courier/shiprocket/   # adapter, auth, credentials, http, rates, shipment, track, ndr, webhook
│   ├── shipment_service.go, shipment_planner.go, rate_service.go, tracking_service.go
│   ├── webhook_service.go, apply_normalized.go, reconcile_service.go, provider_config_service.go
├── factory/courier_partner_factory.go + singleton/
├── handler/  route/  error/  cache/  validator/  utils/
migrations/
├── 032_create_fulfillment_tables.sql
└── 033_create_physical_spec_unit.sql
migrations/seeds/core/005_seed_physical_specs.sql
test/integration/fulfillment/   # suite-per-area, full endpoint matrix
main.go                         # +fulfillment.NewContainer(router) (only existing file touched)
```

**Structure Decision**: Standard module layout per constitution (mirrors `payment/` + `product/`), chosen because the feature is a new business domain with sub-domains (courier adapters) requiring the factory/singleton pattern.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified

None — all gates pass without exceptions.

## Rollback

New tables only (`032`, `033`), no ALTERs of existing tables: rollback is `DROP TABLE` in reverse-dependency order (webhook log → events → items/NDR → shipments → configs/fields → providers; `physical_spec_unit` anytime). No data migration to reverse. Seeds are idempotent upserts — safe to re-run after rollback + re-migrate.
