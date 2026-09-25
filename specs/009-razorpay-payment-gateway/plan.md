# Implementation Plan: Razorpay Payment Gateway Integration

**Branch**: `009-razorpay-payment-gateway` | **Date**: 2026-08-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-razorpay-payment-gateway/spec.md`

## Summary

Integrate Razorpay as the first real payment provider in the payment module, behind a
provider-agnostic abstraction so future providers (Stripe, Cashfree, PayU) can be added without
touching payment or notification business logic. A customer initiates payment for a pending order;
the backend creates a payment session with the provider and, on the provider's webhook, marks the
payment captured/failed and auto-confirms/fails the order through the order service. Sellers manage
their own per-provider credentials (encrypted), view a provider catalog dashboard, and can issue full
or partial refunds. Concurrency, webhook signature verification, idempotency, and an append-only
payment event ledger are first-class concerns.

The detailed architecture, database design, and file-level changes live in `pre-spec.md` (the
approved plan). This document records the technical context, constitution gates, and the Phase 0/1
design artifacts.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: Gin (HTTP), GORM (ORM), `go-playground/validator/v10` (validation),
`testify/suite` + Testcontainers (integration testing), stdlib `net/http` + `crypto/hmac`/`crypto/sha256`
(Razorpay client, no third-party SDK)
**Storage**: PostgreSQL 16 (new tables + migration `031`; GORM `SingularTable: true`); Redis 7 is
used by other modules but not directly required by this feature
**Testing**: Go integration tests under `test/integration/payment/` using Testcontainers; a local
`httptest.Server` fakes the Razorpay API and webhook signing
**Target Platform**: Linux server (backend service)
**Project Type**: Modular monolith web service (payment module)
**Performance Goals**: Payment/refund initiation and webhook processing are correct and idempotent
under concurrent duplicate webhooks; no artificial latency on the success path
**Constraints**: Webhook signature verification must use the raw body; conditional status transitions
(`WHERE status = expected`) prevent regression; seller isolation is enforced on all reads/writes
**Scale/Scope**: Multi-seller; one provider (Razorpay) now, N providers later; payments are
polymorphically referenced (order now, subscription later)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Modular Monolith / Microservices DNA | PASS | Payment module owns its entities/repos/services/handlers/routes; cross-module order updates go through the order **service** interface only. |
| II. Clean Architecture / Layered Architecture | PASS | Handler → Service → Repository → DB; gateways are adapters behind `PaymentGateway`; no layer bypass. |
| III. Factory-Singleton DI | PASS | New `payment/factory/singleton/` mirrors `product/factory/singleton/`; added to test singleton reset. |
| IV. TDD / Integration-First | PASS | Full `test/integration/payment/` suite against a fake Razorpay server; happy + auth + validation + edge + isolation. |
| V. Multi-Tenant Seller Isolation | PASS | Payments, refunds, and gateway configs are always scoped by `seller_id`. |
| VI. Correlation ID / Tracing | PASS | All handlers use the shared middleware; context propagated to services/repos. |
| VII. RBAC | PASS | Customer/seller routes use `CustomerAuth`/`SellerAuth`; webhook route uses signature, not role auth. |
| VIII. Backward Compatibility | PASS | New migration is additive (except dropping columns already established as redundant in `009`, which are unused); `009` is never modified. |
| IX. SOLID (SRP/OCP/LSP/ISP/DIP) | PASS | Strategy + factory; `PaymentGateway` is gateway-agnostic; OCP satisfied (new provider = new adapter + registry entry). |
| X. Performance & Scalability | PASS | Indexed lookups, conditional updates, idempotent webhook dedupe; no N+1 on the critical path. |

**Result**: PASS — no violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/009-razorpay-payment-gateway/
├── pre-spec.md          # Approved detailed technical plan (source of truth)
├── spec.md              # Business specification
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
payment/
├── container.go                  # Registers PaymentModule, GatewayModule, WebhookModule
├── entity/                       # GORM models
├── model/                        # Request/response/gateway DTOs
├── repository/                   # Data access interfaces + impls
├── service/                      # PaymentService, GatewayService, WebhookService
│   └── payment_gateway/          # PaymentGateway interface + Razorpay adapter + credentials
├── factory/
│   ├── payment_gateway_factory.go
│   └── singleton/                # repository/service/handler/singleton factories
├── handler/                      # HTTP handlers
├── route/                        # Route modules
├── error/                        # AppError sentinels
└── utils/constant/               # Error codes + gateway constants

order/
├── service/order_service.go      # Add order-confirm/fail hooks (no payment import)
└── repository/order_repository.go

migrations/
├── 031_create_razorpay_payment_integration.sql
└── seeds/core/004_seed_payment_gateways.sql

test/
├── integration/payment/          # Razorpay integration suite
└── integration/setup/singletons.go
```

**Structure Decision**: Follow the existing modular-monolith layout (`payment/` mirrors
`product/`/`order/`). The order module exposes payment-outcome hooks through its service interface so
the payment module never reaches into order internals (no import cycle).

## Complexity Tracking

> No constitution violations. This section is intentionally empty.
