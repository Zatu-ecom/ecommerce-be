# Implementation Plan: Payment Gateway Platform

**Branch**: `010-payment-gateway-platform` | **Date**: 2026-09-06 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/010-payment-gateway-platform/spec.md`  
**Base branch**: `009-razorpay-payment-gateway` (this feature **refactors** that work; it does not start empty)

## Binding documents (do not invent a second design)

Read in this order. Later docs win only where this table says they win.

| Priority | Document | Role |
|----------|----------|------|
| 1 | [spec.md](./spec.md) | Product: who, what, success, out of scope |
| 2 | [pre-spec.md](./pre-spec.md) | Architecture: one interface, locate-then-verify, schema, algorithms |
| 3 | This `plan.md` + [research.md](./research.md) + [data-model.md](./data-model.md) + [contracts/](./contracts/) + [quickstart.md](./quickstart.md) | How to implement; must match 1 and 2 |
| 4 | [phased-todos.md](./phased-todos.md) (copy of research `plan.md`) | Phase checklist — **superseded** where it conflicts with spec/pre-spec (see research.md R-01) |
| 5 | `plans/010-payment-seller-dashboard-gaps/` originals | Same content as copies in this folder |

**Hard rule for `/speckit.implement` and flash models:** if a file in `payment/service/*.go` (except factory singleton) contains Razorpay event names, `ParseRazorpay`, `DecryptSensitive`, or `FindByCode(GATEWAY_CODE_RAZORPAY)`, the implementation is **wrong**.

## Summary

Turn the Razorpay-only payment slice into a **provider platform**: sellers keep **sandbox and production** credentials, customers check out with a **generic** `checkout.gatewayCode` + `checkout.fields` payload, webhooks are verified with **that payment’s seller** secret, stuck pending payments are reconciled, and adding Stripe later is a **new folder + seed + one factory line**.

Payment is **not in production**. Schema changes are **in-place edits** to `009` / `031` / `008` / seed `004` — **no** migration `032`, **no** API aliases (`keyId`, `/webhooks/razorpay` as a separate handler).

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin, GORM, `go-playground/validator/v10`, testify/suite + Testcontainers, stdlib `net/http` + `crypto/hmac`/`sha256` (no Razorpay SDK), `github.com/robfig/cron/v3` via `common/cron`  
**Storage**: PostgreSQL 16 (`SingularTable: true`); Redis unused by this feature  
**Testing**: `test/integration/payment/` + unit tests under `test/payment/` and `test/common/` (never beside production packages)  
**Target Platform**: Linux backend service  
**Project Type**: Modular monolith web service (`payment/` module + `user` seller_settings field)  
**Performance Goals**: Webhook locate by indexed session/payment/txn id (no N-seller HMAC); cron batch 100 + `SKIP LOCKED`; list `pageSize` max 100; events only on GET-by-id  
**Constraints**: Raw-body HMAC; tenant isolation; AES-256-GCM secrets; fail closed if `ENCRYPTION_KEY` missing; POST initiate/refund no retry; GET status/test 2 retries on 429/5xx  
**Scale/Scope**: Multi-seller; one adapter (Razorpay) now; N adapters later without editing orchestrators

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Modular Monolith | PASS | Payment owns payment tables/services. Order confirm/fail/attach via `order/service` only. Seller payments mode via `user` settings service. File logos via `FileDisplayGateway`. |
| II. Clean Architecture | PASS | Handler → Service → Repository. Adapters behind `PaymentGateway`. Handlers do not call repos. |
| III. Factory-Singleton DI | PASS | Existing `payment/factory/singleton`. New adapter registered only in `service_factory.go`. New repos (join tables, reconcile) added to repository factory. |
| IV. TDD / Integration-First | PASS | Extend `test/integration/payment/`; unit tests for HMAC/mask/merge/`applyNormalized` under `test/`. Fake `httptest.Server` — never real Razorpay in CI. |
| V. Seller Isolation | PASS | Config, list, refunds, webhook logs scoped by `seller_id`. Webhook verify uses **txn.seller_id** config, not `configs[0]`. |
| VI. Correlation ID | PASS (existing webhook exception) | Authenticated payment APIs still require `X-Correlation-ID`. `POST /api/payment/webhooks/` already skipped via `UnlessSkipped`. |
| VII. RBAC | PASS | Customer initiate + GET own txn; seller dashboard/refunds/logs; webhook public + signature. GET txn by id must accept **customer or seller**. |
| VIII. Backward Compatibility | **JUSTIFIED EXCEPTION** | Constitution forbids editing applied prod migrations. Payment **is not in production**. Spec FR-032 requires rewriting `009`/`031`/`008`/seed in place and **no** aliases. Documented in Complexity Tracking. |
| IX. SOLID | PASS | OCP: one `PaymentGateway` interface; frozen `WebhookAction` switch in base webhook service. |
| X. Performance | PASS | Geo `EXISTS` joins; skip locked cron; no events on list; no unverified body persistence. |

**Result**: PASS with one documented exception (VIII / in-place migrations).

## Project Structure

### Documentation (this feature)

```text
specs/010-payment-gateway-platform/
├── spec.md
├── pre-spec.md              # Binding architecture (copy of research)
├── phased-todos.md          # Phase checklist (superseded on conflicts — see R-01)
├── plan.md                  # This file
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── payment-api.md
│   └── gateway-adapter.md
├── checklists/requirements.md
└── tasks.md                 # Created by /speckit.tasks (T001–T066)
```

### Source Code (repository root)

```text
payment/
├── container.go
├── entity/                    # Drop payment_method.go; add join entities; txn.environment
├── model/                     # Drop keyId; drop gateway_response.go; CheckoutPayload
├── repository/                # Dual-env config; geo joins; status filter; SKIP LOCKED pending
├── service/
│   ├── payment_service.go     # Gateway-agnostic initiate/refund/select
│   ├── webhook_service.go     # Locate → verify → applyNormalized only
│   ├── payment_gateway_service.go  # Dual env, mask, test, compose webhook URL
│   ├── reconcile_service.go   # NEW: cron worker
│   └── payment_gateway/
│       ├── contract.go        # PaymentGateway + WebhookAction + DTOs (move from payment_gateway.go)
│       ├── crypto.go          # ResolveEncryptionKey only
│       └── razorpay/
│           ├── adapter.go
│           ├── credentials.go
│           ├── http.go
│           └── webhook.go
├── factory/payment_gateway_factory.go
├── factory/singleton/         # Register razorpay.New; wire reconcile cron
├── handler/ + route/
├── error/ + utils/constant/   # No RAZORPAY_EVENT_* outside razorpay/

user/entity/seller_settings.go + model + service  # payments_environment
order/service/                   # Unchanged public hooks

migrations/
├── 008_create_geo_tables.sql                    # ADD payments_environment
├── 009_create_payment_tables.sql                # REWRITE CREATE (no 032)
├── 031_create_razorpay_payment_integration.sql  # Slim: event table + unique event_id
└── seeds/core/004_seed_payment_gateways.sql     # IN/INR joins, no webhook_url

common/config/app.go           # PUBLIC_API_BASE_URL, PAYMENT_PENDING_TTL_MINUTES, REFUND_STUCK_TTL_MINUTES
common/cron/                   # Register payment.reconcile_pending
common/middleware/             # Skip prefix already /api/payment/webhooks/

test/integration/payment/
test/payment/payment_gateway/razorpay/
test/payment/apply_normalized_test.go
```

**Structure Decision**: Stay inside the existing `payment/` modular-monolith layout. Provider code moves from flat `payment/service/payment_gateway/razorpay_*.go` into `payment/service/payment_gateway/razorpay/`. User module only gains `payments_environment`. Order module is not redesigned.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| VIII: edit `009`/`031`/`008` instead of additive `032` | Payment never shipped to prod; leftover columns/arrays/`payment_method`/`webhook_url` would rot forever | Additive `032` + API aliases contradict spec FR-032 and pre-spec §8; dual schema forever |

## Implementation notes for flash models

1. Follow [quickstart.md](./quickstart.md) phase order. Tests first where the suite already exists; add failing tests then code.
2. Copy interface signatures from [contracts/gateway-adapter.md](./contracts/gateway-adapter.md) exactly. Do not keep `VerifyWebhook`/`ParseWebhook`/`RefundType`/`FetchPayment` on the interface.
3. Copy HTTP JSON from [contracts/payment-api.md](./contracts/payment-api.md). Do not leave `keyId` on initiate.
4. SQL columns from [data-model.md](./data-model.md). Recreate local DB after editing migrations.
5. Forbidden grep (must be **zero** matches except factory singleton + `razorpay/` folder):

```text
rg "RAZORPAY_EVENT|ParseRazorpay|DecryptSensitive|GATEWAY_CODE_RAZORPAY|HandleRazorpay|keyId" \
  payment/service/*.go payment/handler payment/route payment/model
```

(`checkout.fields.keyId` inside Razorpay adapter JSON map is allowed. Top-level response field `KeyID` is not.)

## Constitution Check (post–Phase 1 design)

Re-evaluated after `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`. Same results as the table above: **PASS** with the documented VIII exception (in-place migrations because payment is not in production). Cross-module access remains service-interface only. `PaymentGateway` stays ≤ methods on one focused interface (~11 methods: Code, 5 credential, Initiate, Refund, Test, Peek, Normalize, FetchRemote — at the constitution ~10 guideline). If a review wants ISP split, **do not** split: pre-spec explicitly rejected a second codec interface.

## Phase 0 / Phase 1 outputs

| Artifact | Path |
|----------|------|
| Research | [research.md](./research.md) |
| Data model | [data-model.md](./data-model.md) |
| HTTP contracts | [contracts/payment-api.md](./contracts/payment-api.md) |
| Adapter contract | [contracts/gateway-adapter.md](./contracts/gateway-adapter.md) |
| Implementer quickstart | [quickstart.md](./quickstart.md) |

**Stop:** `/speckit.plan` does not create `tasks.md`. Next: `/speckit.tasks` then `/speckit.implement`.
