# Tasks: Money & Currency Standardization

**Input**: Design documents from `/specs/008-money-currency-standardization/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, pre-spec.md
**Tests**: **MANDATORY** (constitution TDD + integration-first). Write failing tests before production code in each story phase. All tests live under `test/` organized **by module** — unit in `test/<module>/`, integration in `test/integration/<module>/`. No cross-module `money/` suite; cross-module flows live in the owning module's suite using seeded data.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable (different files, no incomplete dependency)
- **[Story]**: `[US1]`…`[US5]` on story phases only
- Every task includes concrete file paths

## Path Conventions

- Shared contracts: `common/model/`
- User currency resolution: `user/service/user_service.go`, `user/factory/currency_mapper.go`
- Product module: `product/`
- Order/cart module: `order/`
- Promotion module: `promotion/`
- Report module: `report/`
- Migration: `migrations/`
- Tests: `test/common/model/` (unit), `test/integration/<module>/` (integration)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the pure `common/model` package (move shared contracts + add Money/CurrencyInfo), add the user mapper, and migrate catalog price columns to `price_cents`. All stories depend on these.

- [x] T001 Create `common/model/` package with `api_response.go` (move `Response`/`ErrorResponse` + `SuccessResponse`/`ErrorWithValidation`/`ErrorWithCode`/`ErrorResp` from `common/response.go`), `pagination.go` (move `BaseListParams`/`PaginationResponse` + `SetDefaults`/`NewPaginationResponse`), `validation_error.go` (move `ValidationError`); keep `common/response.go` as thin re-export aliases for one PR series
- [x] T002 [P] Add `common/model/currency.go` with `CurrencyInfo{Code,Symbol,DecimalDigits}` + `Factor()` and digit validation (encapsulated — all currency behavior in this file only)
- [x] T003 [P] Add `common/model/money.go` with `Money{Amount,AmountCents,Formatted}` + `NewMoney`, `FromCents`, `ToCents` (half-up, strict precision), `ValidateMajorAmount`, `Format`, money-specific errors (encapsulated — all money behavior in this file only)
- [x] T004 [P] Add `user/factory/currency_mapper.go` mapping `user/model.CurrencyResponse` → `common/model.CurrencyInfo`
- [x] T005 [P] Write unit tests in `test/common/model/money_test.go` (factor 0–4, half-up rounding, excess-precision reject, format with symbols, zero/large values)
- [x] T006 [P] Write unit tests in `test/common/model/currency_test.go` (Factor, digit validation) and `test/common/model/pagination_test.go` (SetDefaults, NewPaginationResponse)
- [x] T007 Create migration `migrations/030_money_currency_standardization.sql`: add `product_variant.price_cents BIGINT` + `package_option.price_cents BIGINT` (nullable), backfill `ROUND(price * 100)::BIGINT`, set NOT NULL, drop `price`, recreate indexes (`idx_variant_product_price` → on `(product_id, price_cents)`), update `migrations/004_create_related_products_procedure.sql` price references
- [x] T008 [P] Update seed SQL to `price_cents`: `migrations/seeds/mock/002_seed_products.sql` (`product_variant` + `package_option` inserts), `migrations/seeds/mock/003_seed_related_products.sql`
- [x] T009 Verify Testcontainers migration runner picks up `030_*.sql` in `test/integration/setup/database.go` (lexical ordering only)

**Checkpoint**: `common/model` pure package compiles; `make migrate` / Testcontainers apply `price_cents`; unit tests green.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Repoint imports to `common/model`, wire user currency resolution into module factories, and prove no import cycles. **No user story HTTP until this completes.**

**⚠️ CRITICAL**: Blocks US1–US5

- [x] T010 [P] Migrate all `common.Response`/`common.PaginationResponse`/`common.BaseListParams`/`common.SuccessResponse`/`common.ErrorWithCode`/`common.ErrorWithValidation`/`common.ErrorResp` call sites to `common/model` imports across `user/`, `product/`, `order/`, `promotion/`, `report/`, `payment/`, `notification/`, `inventory/`, `fulfillment/`, `subscription/` (or rely on temporary aliases then remove) — `payment/` **is touch-only** (shared-contract import alignment); payment gateway adapter redesign is out of scope per spec
- [x] T011 [P] Add `userFactory` + `GetUserService()` wiring precedent already in `order/factory/singleton/service_factory.go`; add currency resolution accessor (`GetSellerDefaultCurrency`/`GetPreferredCurrency`) to `product/factory/singleton/service_factory.go` for price writes
- [x] T012 [P] Add arch/import check (CI or Makefile target): `go list -f '{{.ImportPath}} {{.Imports}}' ./common/model` must NOT import `user|product|order|promotion|report|payment`

**Checkpoint**: Foundation ready — stories can start (US1 first for MVP).

---



## Phase 3: User Story 1 — Seller Sets Product Prices in Their Currency (P1) 🎯 MVP

**Goal**: Product/variant/package price writes interpret major units in the seller's base currency, validate `decimalDigits`, store `price_cents`; reads return nested `Money` + parent `currency`.

**Independent Test**: INR seller creates variant `99.99` → DB `9999` cents → GET returns Money `99.99`/`9999`/formatted + currency INR; JPY seller `100` → `100` cents; `100.5` (JPY) and `99.999` (INR) rejected.

### Tests for US1 (write first — must FAIL)

- [x] T013 [P] [US1] Add price-cents suite helpers/setup in `test/integration/product/variant/price_cents_setup_test.go` (seed currency/seller settings, constants)
- [x] T014 [P] [US1] Write seller-currency write + DB-cents + Money-response tests (INR 99.99→9999, JPY 100→100, package option) in `test/integration/product/variant/price_cents_test.go` and `test/integration/product/package_option/price_cents_test.go`
- [x] T015 [P] [US1] Write excess-precision rejection tests (JPY `100.5`, INR `99.999`) and min/max price filter tests (major-in → cents query) in `test/integration/product/variant/price_cents_validation_test.go` — assert rejection returns `400` with the standard validation envelope (`ErrorWithValidation`: `success:false`, `code:"VALIDATION_ERROR"`, `errors:[{field:"price",...}]`) per constitution API Response Standards



### Implementation for US1

- [x] T016 [US1] Change `product/entity/product_variant.go` and `product/entity/package_option.go`: `Price float64` → `PriceCents int64` (`gorm:"column:price_cents"`)
- [x] T017 [P] [US1] Update `product/model/variant_model.go` + `product/model/product_model.go`: request `price` stays major (float64), response `price` becomes `Money` type; add parent `currency` (`CurrencyInfo`) to **variant detail/list and product detail/list** responses (FR-006); `ListVariantsRequest.MinPrice/MaxPrice` stay major
- [x] T018 [US1] Update `product/mapper/variant_mapper.go` + `product/mapper/product_mapper.go`: `DefaultPrice/MinPrice/MaxPrice` float64 → cents-aware
- [x] T019 [US1] Update `product/repository/variant_repository.go`: `product_variant.price` → `product_variant.price_cents` in filters (`applyPriceAndStatusFilters`), aggregation (`priceAgg` Min/Max), select list (`price as price`)
- [x] T020 [US1] Update `product/factory/variant_factory.go`: `CreateVariantFromRequest`/`UpdateVariantEntity`/`BulkUpdateVariantEntity`/`BuildVariantDetailResponse`/`BuildVariantResponse` — major-in requests via `commonModel.CurrencyInfo.ToCents` (resolved from seller), Money-out via `commonModel.NewMoney`
- [x] T021 [US1] Update `product/utils/variant_classification.go` defaultPrice float64 → cents
- [x] T022 [US1] Wire seller-currency resolution into `product/service/variant_service.go` + `product/service/package_option_service.go` + `product/service/variant_bulk_service.go`: resolve currency from token `sellerID` via user service `GetSellerDefaultCurrency` (seller's base currency — always, never buyer-preferred) → `ToCents` on write → persist `PriceCents`
- [x] T023 [US1] Update `product/handler/variant_handler.go` + `product/handler/package_option_handler.go` + `product/handler/product_handler.go` to pass sellerID/currency context (no business logic in handlers)
- [x] T024 [US1] Update related-products price range: `migrations/004_create_related_products_procedure.sql` min/max on `price_cents` + `product/repository/variant_repository.go` related-products select
- [x] T025 [US1] Run US1 integration tests until green for `test/integration/product/variant/price_cents_*.go` + `test/integration/product/package_option/price_cents_test.go` (`go test ./test/integration/product/... -run PriceCents -v`)

**Checkpoint**: MVP — seller currency prices are stored precisely and returned in the shared Money shape.

---

## Phase 4: User Story 2 — Shopper Sees Consistent Cart and Order Money (P1)

**Goal**: Cart + order responses use nested `Money` everywhere with parent `currency`; cart math reads `PriceCents` (no `* 100`); guest cart uses seller base currency; cart→order cents match.

**Independent Test**: Seed INR variant 1299.50 → add qty 2 → cart `unitPrice.amountCents=129950`, `lineTotal.amountCents=259900` → create order → order item/total cents match cart; JPY variant `500` never becomes `50000`; guest cart same shape.

### Tests for US2 (write first — must FAIL)

- [x] T026 [P] [US2] Write cart Money round-trip tests (INR seeded variant → cart → order cents continuity, coupon discount coherent) in `test/integration/order/cart/money_roundtrip_test.go`
- [x] T027 [P] [US2] Write JPY zero-decimal no-`*100` tests + guest cart same Money contract in `test/integration/order/cart/jpy_zero_decimal_test.go`
- [x] T028 [P] [US2] Write Money shape + currency metadata schema assertions (every money field `amount`/`amountCents`/`formatted`, parent `currency` code/symbol/decimalDigits, no bare `unitPrice` int) in `test/integration/order/money_contract_shape_test.go` — put the shape-check helper in `test/integration/helpers/` (e.g. `assertMoney(t, node)` / `assertCurrency(t, node)`) and reuse it across the US2/US4/US5 suites to avoid duplicated assertion logic



### Implementation for US2

- [x] T029 [US2] Update `order/model/cart_model.go` + order models: response DTOs → `commonModel.Money` fields + parent `currency` (remove raw cent int fields / `*Formatted` twins); available-coupon DTO fixed `value`/thresholds/potential discount → `Money` (FR-005)
- [x] T030 [US2] Replace `formatCurrencyWithSymbol` in `order/factory/cart_response_builder.go` with `commonModel.NewMoney`/`Format` (all money JSON + formatted strings)
- [x] T031 [P] [US2] Remove `variant.Price * 100` bridges: `order/service/cart_operations.go` (line ~262) and `order/service/cart_service.go` (line ~469) → read `variant.PriceCents` directly
- [x] T032 [US2] Update `order/service/guest_cart_service.go` to use seller default currency (existing `GetSellerDefaultCurrency`) for the same Money contract — read-side presentation stays seller base currency even if a buyer has a preferred currency (FR-010; no FX)
- [x] T033 [US2] Update `order/service/cart_coupon_ops.go` coupon discount Money mapping
- [x] T034 [US2] Run US2 integration tests until green for `test/integration/order/cart/money_roundtrip_test.go`, `jpy_zero_decimal_test.go`, `money_contract_shape_test.go` (`go test ./test/integration/order/... -run Money -v`)

**Checkpoint**: Storefront sees one consistent money shape across cart and order.

---

## Phase 5: User Story 3 — Seller Configures Discounts in Their Currency (P2)

**Goal**: Promotion/discount-code fixed amounts + thresholds entered as major units in seller currency, validated + stored as cents, returned as Money; percentages stay plain numbers.

**Independent Test**: INR seller creates fixed promo `50.00` → stored 5000 → GET Money `50.00`/`5000`; `minPurchaseAmount: 1000` → eligibility uses 100000; JPY seller fixed `50` → stored 50 (not 5000); percentage `15` stays plain.

### Tests for US3 (write first — must FAIL)

- [x] T035 [P] [US3] Write fixed-amount + threshold major-in → cents + Money response tests (INR 50.00→5000, JPY 50→50, percentage stays plain) in `test/integration/promotion/discount_code_money_currency_test.go`
- [x] T036 [P] [US3] Write promotion-sale money tests (create fixed_amount promo with `discountConfig.amount` major 50.00 INR → stored 5000 + Money response; `minPurchaseAmount`/`maxDiscountAmount` major; JPY seller `amount` 50 → 50 not 5000) in `test/integration/promotion/promotion_money_currency_test.go`



### Implementation for US3

- [x] T037 [US3] Update `promotion/model/discount_code_request_model.go`: `Value` (fixed) major-in, `MinPurchaseAmountCents`/`MaxDiscountAmountCents` → `MinPurchaseAmount`/`MaxDiscountAmount` major (with `discountType`-aware interpretation; percentage keeps plain `value`)
- [x] T038 [P] [US3] Update `promotion/model/discount_code_response_model.go`: monetary fields → `commonModel.Money` (fixed value, thresholds); percentage `value` stays plain number
- [x] T039 [US3] Update `promotion/factory/discount_code_mapper.go` + `promotion/service/discount_code_service.go`: resolve seller currency → `ToCents` on write → map Money on read; strategies unchanged (cents in/out)
- [x] T040 [US3] Update `promotion/model/promotion_request_model.go`: `MinPurchaseAmountCents`/`MaxDiscountAmountCents` → `MinPurchaseAmount`/`MaxDiscountAmount` major (seller currency, validated)
- [x] T041 [P] [US3] Update `promotion/model/discount_config_model.go`: config DTO money keys major (`amount`, `minOrder`, `maxDiscount`, `bundlePrice`, `maxShippingDiscount`) — `ToCents` at boundary; `promotion/entity/promotion.go` `discount_config` JSONB keeps internal cents keys
- [x] T042 [US3] Update `promotion/model/promotion_response_model.go`: monetary fields → `commonModel.Money` (thresholds); `discount_config` values in responses map through Money for monetary keys
- [x] T043 [US3] Update `promotion/factory/promotion_mapper.go` + `promotion/service/promotion_service_command.go`: resolve seller currency → convert config money major→cents on write → Money on read; strategies unchanged (cents in/out)
- [x] T044 [US3] Run US3 integration tests until green for `test/integration/promotion/discount_code_money_currency_test.go` + `promotion_money_currency_test.go` (`go test ./test/integration/promotion/ -run MoneyCurrency -v`)

**Checkpoint**: Promotions use the same seller-currency money rules as products.

---

## Phase 6: User Story 4 — Clients Always Receive One Money Language (P2)

**Goal**: Every money-bearing response across product, cart, order, coupons, and reports uses the shared nested shape + parent currency; no module-specific alternate shapes; reports stop raw `/100.0`.

**Independent Test**: Sample product/cart/order/coupon/report payloads; assert every money object has exactly `amount`/`amountCents`/`formatted` and parents expose currency code/symbol/decimalDigits; no bare `*Cents` scalars or unlabeled ints.

### Tests for US4 (write first — must FAIL)

- [x] T045 [P] [US4] Write report Money-shape tests (revenue/discount metrics as Money, no float `/100` twins) in `test/integration/report/money_metrics_test.go`



### Implementation for US4

- [x] T046 [US4] Update `report/repository/report_repository.go`: return aggregate cents (e.g. `SUM(total_cents)`) instead of `/100.0` float columns
- [x] T047 [US4] Update `report/factory/summary_response_builder.go` + `report/model/report_model.go`: revenue/discount metrics → `commonModel.NewMoney` (seller currency digits); remove raw `/100.0` conversions
- [x] T048 [US4] Update `order/model/cart_model.go` available-coupon DTOs + `promotion/model/discount_code_response_model.go` + `promotion/model/promotion_response_model.go` — ensure no bare `*Cents` scalars remain on the wire
- [x] T049 [US4] Remove temporary aliases in `common/response.go`; verify all imports point to `common/model`
- [x] T050 [US4] Run US4 integration tests until green for `test/integration/report/money_metrics_test.go` + full module suites

**Checkpoint**: One money language app-wide; no legacy ambiguous fields.

---

## Phase 7: User Story 5 — Platform Proves Correctness with Strong Tests (P1)

**Goal**: Automated verification covering seller-currency writes, minor-unit persistence, dual money responses, and product→cart→order continuity for at least one 2-decimal and one 0-decimal currency. Round-trip failures block release.

**Independent Test**: Full A–E scenario suite green across INR + JPY sellers; a broken product→cart→order round-trip blocks merge.

### Tests for US5 (write first — must FAIL)

- [x] T051 [P] [US5] Write cross-seller isolation tests (Seller A INR vs Seller B JPY price independence, auth token seller used for currency) in `test/integration/product/variant/cross_seller_currency_test.go`
- [x] T052 [P] [US5] Write full product→cart→order continuity test (seed INR variant → write → cart → order → cents match end-to-end, no drift) in `test/integration/order/order_money_e2e_test.go`
- [x] T053 [P] [US5] Write JPY zero-decimal end-to-end continuity (product→cart→order `500` stays `500`, no `*100` scaling) in `test/integration/order/cart/jpy_e2e_test.go`



### Implementation for US5

- [x] T054 [US5] Add a **CI gate target** for money verification: run `test/common/model/`, `test/integration/product/...`, `test/integration/order/...`, `test/integration/promotion/...`, `test/integration/report/...` and **fail the build** on any round-trip regression (SC-007 — core money round-trip + dual-currency suites are required gates, not advisory make targets)
- [x] T055 [US5] Ensure all seed data + fixtures consistent with `price_cents` and Money contract (audit `migrations/seeds/mock/*.sql`, test helpers)
- [x] T056 [US5] Run all five money-related integration suites + unit tests until green (`go test ./test/common/model/ ./test/integration/product/... ./test/integration/order/... ./test/integration/promotion/... ./test/integration/report/... -count=1`)

**Checkpoint**: Release-blocking money verification suite green.

---

## Phase 8: Polish & Cross-Cutting

**Purpose**: Hardening, docs, consistency, regression.

- [x] T057 [P] Verify quickstart smoke path documented in `specs/008-money-currency-standardization/quickstart.md` against running APIs (unit + module-scoped integration commands)
- [x] T058 [P] Update `AGENTS.md` Active Technologies / Recent Changes via `.specify/scripts/bash/update-agent-context.sh codex` (if not already done)
- [x] T059 [P] Add a unit test for `common/model` import-cycle guard as a CI step (`go list` check in Makefile or CI config)
- [x] T060 Run full related suites (`go test ./test/integration/product/... ./test/integration/order/... ./test/integration/promotion/... ./test/integration/report/... -count=1`) and fix regressions
- [x] T061 Confirm no `* 100` / `/ 100` remain on money paths outside `common/model/money.go`/`currency.go` (grep audit of `product/`, `order/`, `promotion/`, `report/`)

---

## Dependencies & Execution Order

### Plan ↔ tasks phase map


| Plan preview     | tasks.md                                                     |
| ---------------- | ------------------------------------------------------------ |
| 0 Foundation     | Phase 1 Setup (T001–T009) + Phase 2 Foundational (T010–T012) |
| 1 Product        | Phase 3 US1                                                  |
| 2 Cart/Order     | Phase 4 US2                                                  |
| 3 Promo          | Phase 5 US3                                                  |
| 4 Reports + gate | Phase 6 US4 + Phase 7 US5                                    |
| Harden           | Phase 8 Polish                                               |




### Phase Dependencies

- **Phase 1 Setup** → no deps
- **Phase 2 Foundational** → after Setup; **blocks all stories**
- **US1 (Phase 3)** → after Foundational — **MVP**
- **US2 (Phase 4)** → after US1 (needs `PriceCents` + Money response from product)
- **US3 (Phase 5)** → after Foundational (independent of US2 but shares Money contract)
- **US4 (Phase 6)** → after US2 + US3 (needs all modules on Money shape)
- **US5 (Phase 7)** → after US1 + US2 (needs full product→cart→order continuity)
- **Polish (Phase 8)** → after desired stories

### User Story Dependencies

```text
US1 (Product prices in seller currency)
  ├── US2 (Cart/order money consistency)   [needs PriceCents + Money]
  └── US3 (Promo/discount money)           [needs Money contract only]
US2 ──> US4 (One money language app-wide)  [needs cart + order + promo + report]
US1 + US2 ──> US5 (Verification suite)     [needs full round-trip]
```

### Within Each Story

1. Tests written and failing
2. Models/entities/repos
3. Services/factories/handlers
4. Tests green

### Parallel Opportunities

- T002–T006 (common/model files + unit tests); T008 (seed SQL); T010–T012 (import migration)
- T013–T015 (US1 tests); T017 (US1 model); T026–T028 (US2 tests); T031 (cart bridge); T035–T036 (US3 tests); T038 [P] + T041 [P] (US3 models); T045 (US4 test); T051–T053 (US5 tests); T057–T059 (Polish)
- After Foundational: US1 must lead; then US2 and US3 can partially parallelize (different modules: order vs promotion) if staffed

---

## Parallel Example: User Story 1

```bash
# Tests in parallel (after T013 setup):
# T014, T015 → test/integration/product/variant/price_cents_*.go + package_option

# Then implement sequentially:
# T016 entity → T017 model → T018 mapper → T019 repo → T020 factory → T021 util → T022 service → T023 handler → T024 related-products → T025 green
```

## Parallel Example: User Story 2

```bash
# Tests: T026, T027, T028 in parallel
# Impl: T029 model [P] with T030 response builder [P] and T031 cart bridge [P]
# Then T032 guest cart → T033 coupon ops → T034 green
```

## Parallel Example: User Story 3

```bash
# Tests: T035 (discount code) + T036 (promotion sale) in parallel
# Impl: T037 discount-code model → T038 discount-code response [P] with T040 promotion request [P] + T041 config DTO [P]
# Then T039 discount-code mapper/service → T042 promotion response → T043 promotion mapper/service → T044 green
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Phase 1 + 2
2. Phase 3 US1 → demo seller can price in their currency, store cents, read Money
3. Stop and validate

### Incremental Delivery

1. US1 → seller-currency catalog pricing (foundation)
2. US2 → storefront cart/order money consistency
3. US3 → seller promo/discount money
4. US4 → one money language app-wide (incl. reports)
5. US5 → release-blocking verification suite
6. Polish

### Suggested MVP Scope

**US1 only** (seller product prices in their currency). Minimum storefront slice is **US1 + US2** (prices + cart/order). Production-complete = **US1–US5** (verification gate before release).

---

## Notes

- **Breaking change**: Money-bearing payloads change shape (nested `Money` + parent `currency`). Coordinate admin + storefront frontend release. Document in contracts + quickstart.
- **No** `common/money` **package** — money is a shared contract in `common/model` (`money.go`/`currency.go`), encapsulated (type + methods in same file).
- **No cross-module** `test/integration/money/` **suite** — tests are module-scoped: `test/common/model/` unit, `test/integration/<module>/` integration. Cross-module flows live in the owning module's suite using seed data.
- **Seller-currency-only writes**: resolve currency from token `sellerID` → `seller_settings.base_currency_id`; never accept client cents on write (v1).
- **Write vs read currency**: writes always resolve the seller's **base currency** (`GetSellerDefaultCurrency` — never buyer-preferred, per FR-001). Read-side presentation is also seller base currency until FX (FR-010); buyer `currency_id`/`display_prices_in_buyer_currency` does not change money output in this feature.
- **Strict precision**: reject excess fractional digits (no silent rounding).
- **Percentages** stay plain numbers; only fixed monetary values use `Money`.
- `price_cents` **ship with cart** `PriceCents` **consumption in the same release** — no partial deploy of the column rename.
- Currency resolution stays in `user` module; `common/model` never imports domain packages.
- Detailed contracts/logic: `pre-spec.md` (§5 Money shape, §14 test matrix A–E) and `contracts/money-currency.md` + `contracts/common-model-layout.md`.
