# Implementation Plan: Coupon / Discount Codes

**Branch**: `007-coupon-discount-codes` | **Date**: 2026-08-01 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/007-coupon-discount-codes/spec.md`  
**Design draft**: [pre-spec.md](./pre-spec.md)

## Summary

Deliver end-to-end discount codes (coupons): sellers manage codes and catalog scopes in the **promotion** module; customers apply/remove coupons on cart via the **order** module; cart responses return full runtime-priced totals (promotions then coupons); checkout writes immutable coupon snapshots and records usage atomically. Storefront stays thin—all eligibility, stacking, and money math live on the backend.

Technical approach: align DB with existing Go entities (ALTER `discount_code`, create `cart_applied_coupon` + `cart_item_promotion`), implement DiscountCode CRUD/scopes/calculator in promotion, wire cart + order lifecycle in order via service interfaces only, TDD with Testcontainers integration tests per constitution.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Gin (HTTP), GORM (ORM), validator, testify/suite, Testcontainers  
**Storage**: PostgreSQL 16 — extend `discount_code` / scope / usage tables; create `cart_applied_coupon`, `cart_item_promotion`; use existing `order_applied_coupon`  
**Testing**: Integration-first (`test/integration/promotion/`, `test/integration/order/`) + optional unit tests for coupon calculators  
**Target Platform**: Linux server (Dockerized modular monolith)  
**Project Type**: Web service (REST API) — modules `promotion/` + `order/`  
**Performance Goals**: Cart reprice (promotions + coupons) p95 under 300ms for carts with at most 50 line items under Testcontainers/local integration load; atomic usage increment safe under concurrent checkout  
**Constraints**: Seller isolation on every code query; no soft-delete; cart stores references only (runtime calc); guest coupons forbidden; order must not access promotion repositories  
**Scale/Scope**: Multi-tenant sellers; ~20+ HTTP endpoints (CRUD + scopes + cart coupons); Phases 0–6; large integration matrix (T-S*, T-SC*, T-C*, T-E*)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Evidence |
| --- | --- | --- |
| **I. Modular Monolith** | PASS | Discount domain in `promotion/`; cart/order attach in `order/`; cross-module via `DiscountCodeService` / `PromotionService` interfaces only |
| **II. Clean Architecture** | PASS | Handler → Service → Repository → DB; calculators in service layer; entities have no business logic |
| **III. Factory-Singleton DI** | PASS | Extend promotion + order singleton factories; `ResetInstance()` for tests |
| **IV. TDD & Integration-First** | PASS | Phase tests written red-first; APIClient + Testcontainers; assert via Get Cart / Get Order |
| **V. Multi-Tenant Seller Isolation** | PASS | All code queries scoped by `seller_id`; cross-seller get → not found |
| **VI. Correlation ID** | PASS | Existing middleware; all routes under same stack |
| **VII. RBAC** | PASS | SellerAuth for code CRUD/scopes; CustomerAuth for cart coupons |
| **VIII. Backward Compatibility** | PASS | Existing cart fields already include coupon placeholders; enrich without breaking promo response shape |
| **IX. SOLID** | PASS | Strategy calculators by discount type; focused interfaces; DIP for order→promotion |
| **X. Performance** | PASS | Indexes on seller/active, usage (code,user), cart junctions; atomic increment; optional later cache |

**Gate Result**: ALL GATES PASS — no Complexity Tracking rows required.

**Post-design re-check**: Same — contracts and data model preserve module boundaries and runtime cart pricing.

## Project Structure

### Documentation (this feature)

```text
specs/007-coupon-discount-codes/
├── plan.md                 # This file
├── research.md             # Phase 0
├── data-model.md           # Phase 1
├── quickstart.md           # Phase 1
├── contracts/
│   └── api-contracts.md    # Phase 1
├── pre-spec.md             # Detailed design draft (source for HOW)
├── checklists/requirements.md
├── spec.md
└── tasks.md                # Phase 2 (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```text
migrations/
└── 028_align_discount_code_and_cart_coupon.sql   # NEW

promotion/
├── entity/discount_code.go                       # MODIFY: +CurrentUsageCount
├── entity/discount_code_scope.go                 # MODIFY: +ProductID
├── entity/usage.go                               # EXISTS
├── model/discount_code_*.go                      # NEW contracts
├── model/coupon_cart_model.go                    # NEW internal cart contracts
├── error/discount_code_error.go                  # NEW
├── utils/constant/discount_code_constants.go     # NEW
├── repository/discount_code*.go                  # NEW
├── service/discount_code_service.go              # NEW seller CRUD only
├── service/coupon_apply_service.go               # NEW ApplyCouponsToCart + ListAvailableCouponsForCart
├── service/discountStrategy/*.go                 # NEW coupon calculators (percentage/fixed/shipping/bxgy)
├── handler/discount_code*.go                     # NEW
├── route/discount_code*.go                       # NEW
├── factory/discount_code_mapper.go               # NEW
├── factory/singleton/*                           # MODIFY wire (CRUD + CouponApplyService)
└── container.go                                  # MODIFY register modules

order/
├── entity/cart.go                                # EXISTS CartAppliedCoupon, CartItemPromotion
├── entity/order_applied_coupon.go                # EXISTS
├── repository/cart_repository.go                 # MODIFY coupon attach + cart_item_promotion sync
├── repository/order_repository.go                # MODIFY CreateOrderAppliedCoupons
├── service/cart_service.go / cart_operations.go  # MODIFY reprice + coupons + sync cart_item_promotion after promos
├── service/order_lifecycle.go                    # MODIFY snapshot + usage
├── handler/cart_handler.go                       # MODIFY Apply/Remove/Available
├── route/cart_route.go                           # MODIFY register coupon routes
├── factory/cart_response_builder.go              # MODIFY fill coupons
├── model/cart_model.go                           # MODIFY AvailableCoupons
└── factory/singleton/service_factory.go          # MODIFY inject DiscountCodeService + CouponApplyService

test/integration/
├── promotion/discount_code_*_test.go             # NEW
└── order/cart_coupon_*_test.go + checkout        # NEW
```

**Structure Decision**: Existing modular monolith. No new top-level module. Promotion owns definitions + engine; order owns cart attach HTTP + checkout persistence.

## Complexity Tracking

> No constitution violations — intentionally empty.

## Implementation Phases (preview for tasks)

| Phase | Focus |
| --- | --- |
| 0 | Migration + entity fixes + errors/constants/models + test scaffolds |
| 1 | Seller DiscountCode CRUD + tests |
| 2 | Scope APIs + tests |
| 3 | `CouponApplyService` + `discountStrategy/` calculators |
| 4 | Cart integration + `cart_item_promotion` sync-on-build + full CartResponse |
| 5 | Order snapshot + usage recording |
| 6 | Harden (races, rate-limit, docs) |

**Phase mapping (plan preview → tasks.md)**: Plan “0” ≈ tasks Phase 1–2; plan “1–5” ≈ tasks US1–US5 phases; plan “6” ≈ tasks Polish.