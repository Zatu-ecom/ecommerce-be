# Implementation Plan: Money & Currency Standardization

**Branch**: `008-money-currency-standardization` | **Date**: 2026-08-02 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/008-money-currency-standardization/spec.md`  
**Design draft**: [pre-spec.md](../../plans/009-money-currency-standardization/pre-spec.md)

## Summary

Standardize all money handling across the platform: clients **send** monetary amounts as major units in the **seller's selected (base) currency**; the backend validates fractional precision against that currency's `decimalDigits`, converts to **integer minor units (cents)**, and persists + calculates **only in cents**; every response returns a **shared nested `Money` object** (`amount` + `amountCents` + `formatted`) plus parent `currency` metadata (`code`/`symbol`/`decimalDigits`). Shared contracts (API envelope, pagination, validation error, `Money`, `CurrencyInfo`) move into a new pure `common/model` package with Java-style encapsulation (type + all methods in one file). Catalog `price` columns become `price_cents`.

Technical approach: build `common/model` first (relocate `common/response.go` contracts + add `money.go`/`currency.go`), add a `user` mapper to `CurrencyInfo`, migrate `product_variant.price`/`package_option.price` → `price_cents` with a `ROUND(price*100)` backfill, then convert product → cart/order → promotion/report money I/O module-by-module with cross-module Testcontainers integration tests per the constitution (TDD, integration-first). Coordinated breaking change with frontend.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin (HTTP), GORM (ORM), validator, testify/suite, Testcontainers  
**Storage**: PostgreSQL 16 — rename `product_variant.price` / `package_option.price` (`DOUBLE PRECISION`) → `price_cents BIGINT NOT NULL`; keep existing `*_cents` columns in `order`/`order_item`/promo; no new money tables  
**Testing**: Integration-first — all tests live under `test/` and are organized **by module**: unit tests in `test/<module>/` mirroring source paths (e.g. `test/common/model/money_test.go`, `currency_test.go`, `pagination_test.go`), integration tests in `test/integration/<module>/` (product → `test/integration/product/`, order → `test/integration/order/`, promotion → `test/integration/promotion/`). No mixed cross-module `money/` suite; cross-module flows (product → cart → order) live in the owning module's suite and reuse seeded product/variant data (as `test/integration/order/` already does)  
**Target Platform**: Linux server (Dockerized modular monolith)  
**Project Type**: Web service (REST API) — shared `common/model` package + modules `product`/`order`/`promotion`/`report`/`user`  
**Performance Goals**: Money conversion/formatting is pure CPU; keep cart reprice unchanged in complexity; zero per-request cache misses for currency resolution beyond the existing 1h Redis cache (`GetSellerDefaultCurrency`/`GetPreferredCurrency`)  
**Constraints**: `common/model` MUST NOT import `user`/`product`/`order`/`promotion`/`report`/`payment` (no cycles); no `common/money` package; no `money_convert.go`-style helper sprawl; seller-currency-only writes; strict excess-precision rejection; percentages stay plain numbers; `price_cents` ship with cart `PriceCents` consumption in the same release  
**Scale/Scope**: Multi-tenant sellers; ~5 module areas + shared package; migrations `030_money_currency_standardization.sql` (number may shift); Phases 0–4; large integration matrix (A–E scenarios from pre-spec §14.3)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Evidence |
| --- | --- | --- |
| **I. Modular Monolith** | PASS | Shared money/currency contracts live in `common/model` (pure); each domain keeps its own entities/repos/services; cross-module currency resolution via `user` service interface only |
| **II. Clean Architecture** | PASS | Handler → Service → Repository → DB preserved; money conversion called from service layer; entities are plain data structs |
| **III. Factory-Singleton DI** | PASS | No new module factories; extend existing product/order/promotion singleton factories to inject currency resolution + common/model mappers |
| **IV. TDD & Integration-First** | PASS | Tests written red-first in module-scoped suites: `test/integration/product/` (price write/enforcement), `test/integration/order/` (cart/order round-trip via seeded variants), `test/integration/promotion/` (fixed-amount major input), `test/common/model/` (Money/CurrencyInfo units); assertions via API (GET after POST) not direct DB |
| **V. Multi-Tenant Seller Isolation** | PASS | Currency resolved from token/header `sellerID` → `seller_settings.base_currency_id`; cross-seller price writes impossible |
| **VI. Correlation ID** | PASS | Existing middleware; all routes under same stack; no new middleware |
| **VII. RBAC** | PASS | SellerAuth for price/promo writes; CustomerAuth for cart/order; PublicAPIAuth for guest cart reads |
| **VIII. Backward Compatibility** | PASS with justification | Breaking change (money shape) — documented in spec, pre-spec, contracts + coordinated frontend release; optional short-lived Go type aliases in `common/response.go` for compile migration only (no dual JSON keys) |
| **IX. SOLID** | PASS | Encapsulation per contract file; money methods on `Money`/`CurrencyInfo` types; no helper sprawl; DIP via user service interface for currency resolution |
| **X. Performance** | PASS | Cents math is int64; formatting centralized; existing Redis currency cache reused; no N+1 added |

**Gate Result**: ALL GATES PASS — no Complexity Tracking rows required (breaking-change exception under VIII is explicitly justified and pre-approved by spec).

**Post-design re-check**: Same — contracts (`common/model`) and data model preserve module boundaries; `common/model` remains pure (no domain imports); money math stays inside encapsulated contract methods.

## Project Structure

### Documentation (this feature)

```text
specs/008-money-currency-standardization/
├── plan.md                 # This file
├── research.md             # Phase 0
├── data-model.md           # Phase 1
├── quickstart.md           # Phase 1
├── contracts/
│   ├── money-currency.md   # Phase 1 — Money/CurrencyInfo wire contract
│   └── common-model-layout.md # Phase 1 — common/model file layout + imports
├── checklists/requirements.md
├── spec.md
└── tasks.md                # Phase 2 (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```text
common/
├── model/                                  # NEW — pure contracts package
│   ├── api_response.go                     # MOVE from response.go: Response, ErrorResponse + Success/Error writers
│   ├── pagination.go                       # MOVE from response.go: BaseListParams, PaginationResponse + SetDefaults, NewPaginationResponse
│   ├── validation_error.go                 # MOVE from response.go: ValidationError (+ helpers)
│   ├── currency.go                         # NEW: CurrencyInfo + Factor()/digit helpers (encapsulated)
│   └── money.go                            # NEW: Money + NewMoney/FromCents/ToCents/ValidateMajorAmount/Format + errors (encapsulated)
├── response.go                             # MODIFY: keep only temporary aliases (one PR series), then delete
└── helper/pagination_helper.go             # MODIFY (if used): repoint to common/model

user/
├── service/user_service.go                 # EXISTS: GetSellerDefaultCurrency / GetPreferredCurrency (keep, already cached)
├── factory/currency_mapper.go              # NEW: user.CurrencyResponse → commonModel.CurrencyInfo
└── (no DB access moves into common)

product/
├── entity/product_variant.go               # MODIFY: Price float64 → PriceCents int64 (gorm column price_cents)
├── entity/package_option.go                # MODIFY: Price float64 → PriceCents int64
├── model/variant_model.go                  # MODIFY: request/response money fields → major-in / Money-out
├── model/product_model.go                  # MODIFY: price fields + minPrice/maxPrice filters
├── mapper/variant_mapper.go                # MODIFY: DefaultPrice/MinPrice/MaxPrice float64 → cents
├── mapper/product_mapper.go                # MODIFY: min_price/max_price → cents-aware
├── repository/variant_repository.go        # MODIFY: price filters/queries → price_cents
├── utils/variant_classification.go         # MODIFY: defaultPrice float64 → cents
├── factory/variant_factory.go / product_factory.go  # MODIFY: ToCents on write
└── handler/variant_handler.go / product_handler.go / package_option_handler.go  # MODIFY: resolve seller currency → validate → convert

order/
├── factory/cart_response_builder.go        # MODIFY: replace formatCurrencyWithSymbol + raw cent ints with commonModel.Money + currency
├── service/cart_operations.go              # MODIFY: remove variant.Price * 100 → read variant.PriceCents
├── service/cart_service.go                 # MODIFY: same
├── service/cart_coupon_ops.go              # MODIFY: coupon Money mapping
├── service/guest_cart_service.go           # MODIFY: same Money contract (seller default currency)
└── model/cart_model.go + order models      # MODIFY: response DTOs → nested Money + parent currency

promotion/
├── model/discount_code_request_model.go    # MODIFY: request fields major units (minPurchaseAmount, maxDiscountAmount, value)
├── model/discount_code_response_model.go   # MODIFY: monetary fields → Money
├── factory/discount_code_mapper.go         # MODIFY: major → cents at boundary
├── service/discount_code_service.go        # MODIFY: resolve seller currency → ToCents
├── repository/discount_code_repository.go  # MODIFY: (columns already cents; filter mapping only)
└── (strategies unchanged — cents in, cents out)

report/
└── (MODIFY: revenue/discount metrics → Money via commonModel.NewMoney; drop raw /100.0)

migrations/
└── 030_money_currency_standardization.sql  # NEW: price → price_cents rename + backfill + index rework (number may shift)

test/                                        # ALL tests live here, organized by module
├── common/model/                            # NEW — unit tests (no DB) for shared contracts
│   ├── money_test.go                        # factor 0–4, half-up, excess-precision reject, format, zero/large
│   ├── currency_test.go                     # Factor(), digit validation
│   └── pagination_test.go                   # SetDefaults, NewPaginationResponse (moved behavior)
└── integration/
    ├── product/                             # MODIFY + NEW — product module suite
    │   ├── variant/price_cents_test.go      # NEW: seller-currency write (INR 99.99→9999, JPY 100→100), excess-precision reject
    │   ├── package_option/price_cents_test.go  # NEW: same rules for package options
    │   └── (existing price assertions updated to Money + price_cents DB checks)
    ├── order/                               # MODIFY + NEW — order module suite (owns cart/order round-trip)
    │   ├── cart/money_roundtrip_test.go     # NEW: seeded INR variant → cart → order cents continuity (B1/B2/B5)
    │   ├── cart/jpy_zero_decimal_test.go    # NEW: JPY no *100 bug (B3) + guest cart same Money contract (B4)
    │   └── (existing cart/order payloads updated to Money shape)
    ├── promotion/                           # MODIFY + NEW — promotion module suite
    │   ├── discount_code_money_currency_test.go  # NEW: fixed amount major-in → cents, JPY 50≠5000, thresholds (C)
    │   └── (existing fixed-amount configs updated to major input)
    └── report/                              # MODIFY — revenue/discount metrics as Money
```

**Structure Decision**: Existing modular monolith + new pure shared package. `common/model` is the single home for money/currency + relocated API contracts; each commerce module keeps its domain logic and imports `common/model` + `user` (for currency resolution) only. No new top-level module, and **no cross-module `test/integration/money/` suite** — tests follow the existing convention: `test/<module>/` for unit, `test/integration/<module>/` for integration, with cross-module flows covered inside the owning module's suite using seeded data.

## Complexity Tracking

> No constitution violations requiring Complexity Tracking. The single exception — Constitution VIII (backward compatibility) — is an explicitly approved, documented breaking change (spec §Assumptions + pre-spec §13 Compatibility), not a hidden violation.

## Implementation Phases (preview for tasks)

| Phase | Focus |
| --- | --- |
| 0 | `common/model` (move response contracts + add money/currency) + user mapper + import migration + `test/common/model/` unit tests |
| 1 | Product migration (`price_cents`) + entities/models/repos + seller-currency write enforcement + `test/integration/product/` price tests |
| 2 | Cart/order: remove `* 100` bridges, dual `Money` responses, guest cart + `test/integration/order/` round-trip tests |
| 3 | Promotion/discount-code: major-unit requests, Money responses + `test/integration/promotion/` currency tests |
| 4 | Reports + remove aliases + arch/import gate + full A–E CI coverage across module suites |

**Phase mapping (plan preview → tasks.md)**: Plan “0” ≈ tasks Foundation; plan “1” ≈ tasks Product; plan “2” ≈ tasks Cart/Order; plan “3” ≈ tasks Promo; plan “4” ≈ tasks Reports + Harden.
