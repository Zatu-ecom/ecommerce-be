# Research: Coupon / Discount Codes

**Feature**: 007-coupon-discount-codes  
**Date**: 2026-08-01  
**Purpose**: Resolve technical decisions before Phase 1 design. Informed by codebase audit + [pre-spec.md](./pre-spec.md).

## Research Tasks

### 1. Where does coupon domain live vs cart HTTP?

**Decision**: Discount code CRUD, scopes, validation, calculation, and usage persistence live in `promotion/`. Cart apply/remove HTTP, `cart_applied_coupon` persistence, response assembly, and order snapshots live in `order/`. Order calls `DiscountCodeService` / `PromotionService` interfaces only.

**Rationale**: Matches constitution modular monolith + existing pattern (order already injects `PromotionService` for `ApplyPromotionsToCart`). Coupon definitions are promotion-domain; cart attachment is order-domain.

**Alternatives considered**:
- **All coupon logic in order**: Rejected — duplicates discount domain, breaks extractability of promotion module.
- **HTTP from promotion for apply**: Rejected — cart ownership and auth context already in order; would force promotion to own cart tables.

---

### 2. Entity vs migration drift on `discount_code`

**Decision**: Keep Go entity fields (`description`, `max_discount_amount_cents`, `usage_reset_*`) as source of truth; ALTER table in new migration. Add missing Go fields that already exist in DB (`CurrentUsageCount`). Fix `DiscountCodeProduct.ProductID` to match DB.

**Rationale**: Entity was designed ahead of migration `005`. Constitution forbids editing applied production migrations; new ALTER is correct.

**Alternatives considered**:
- **Remove fields from entity**: Rejected — product requirements need max cap and usage reset.
- **Rewrite migration 005**: Rejected — migration immutability.

---

### 3. Cart stores amounts or references?

**Decision**: Reference-only (`cart_applied_coupon`, `cart_item_promotion`). Recalculate discounts on every cart price build (same as automatic promotions today). Freeze money only on `order_applied_coupon` at checkout.

**Rationale**: Spec FR-009; prevents stale totals when codes deactivate or prices change; matches existing cart philosophy (no prices on `cart_item`).

**Alternatives considered**:
- **Store discount cents on cart**: Rejected — drift, harder self-heal, contradicts current promotion model.

---

### 4. Soft delete for discount codes?

**Decision**: No soft delete. Use `is_active` for lifecycle; hard delete only when zero `discount_code_usage` rows.

**Rationale**: Spec FR-015; `BaseEntity` has no `DeletedAt`; aligns with pre-spec lock.

**Alternatives considered**:
- **GORM soft delete**: Rejected by product decision and existing entity design.

---

### 5. Pricing order: promotions vs coupons

**Decision**: (1) Line prices → (2) `ApplyPromotionsToCart` → (3) `ApplyCouponsToCart` on after-promo subtotal → (4) summary. Respect `canStackWithCoupons` on promotions and `canCombineWithOtherDiscounts` on codes.

**Rationale**: Matches existing CART_API_PRD calculation flow and current cart builder (promos first; coupons were stubbed empty).

**Alternatives considered**:
- **Coupons before promotions**: Rejected — breaks published cart pricing narrative and stacking flags.

---

### 6. Guest carts and coupons

**Decision**: Authenticated customers only. Guest apply returns forbidden/unauthorized. After login/merge, customer applies coupons on user cart.

**Rationale**: Per-customer usage limits and eligibility require user identity; spec FR-007.

**Alternatives considered**:
- **Device-scoped guest coupons**: Rejected for v1 complexity and weak fraud controls.

---

### 7. Calculator design (strategy)

**Decision**: Coupon calculators live only under `promotion/service/discountStrategy/` (one calculator interface per `DiscountType`). Cart orchestration lives only in `promotion/service/coupon_apply_service.go` as `CouponApplyService` (`ApplyCouponsToCart`, `ListAvailableCouponsForCart`). `DiscountCodeService` remains seller CRUD only — do not put cart apply methods on it. Do not create a parallel `coupon_calculator*` package.

**Rationale**: OCP for math; SRP for CRUD vs apply; single file path avoids implementer forks.

**Alternatives considered**:
- **Reuse PromotionStrategy directly**: Rejected — type string mismatch and different config shape (scalar `value` vs `discount_config` JSON).
- **Apply methods on DiscountCodeService**: Rejected — fat interface; CRUD and cart apply change for different reasons.
- **Alternate `coupon_calculator*` package**: Rejected — duplicate home for the same strategies.

---

### 7b. `cart_item_promotion` write behavior (locked)

**Decision**: On every full cart price build, after `ApplyPromotionsToCart` returns applied promotions per item, **sync** `cart_item_promotion`: for each cart line, the stored promotion ID set MUST equal the applied set (delete stale rows, insert missing). References only — never store discount amounts. Self-heal drops rows when a promotion no longer applies on a later build.

**Rationale**: Table is in scope; keeps cart attachments consistent with runtime promo results the same way `cart_applied_coupon` stores coupon references.

**Alternatives considered**:
- **Create table only, never write**: Rejected — leaves dead schema and fails FR-009 sync requirement.
- **Write only when seller pins a promo**: Rejected — auto-promotions are the current product behavior; sync-on-build matches that.
---

### 8. Available coupons UX for thin storefront

**Decision**: Embed `availableCoupons` (applicable + notApplicable) in full `CartResponse` by default (cap ~50). Also expose `GET /api/order/cart/available-coupon` returning the same payload for optional refresh. Apply/Remove return full `CartResponse`.

**Rationale**: Spec SC-002 / FR-008 — one response is enough for storefront; dedicated GET avoids confusion when only coupon list needed.

**Alternatives considered**:
- **Opt-in query flag only**: Rejected as default — clients would forget and under-render.
- **Apply returns coupon-only DTO**: Rejected — forces second Get Cart.

---

### 9. API path base

**Decision**: Seller codes under `/api/promotion/discount-code` (+ `/scope/...`). Customer coupons under `/api/order/cart/coupon` and `/available-coupon` (actual mount today is `APIBaseOrder + "/cart"`, not outdated `/api/cart` in CART_API_PRD).

**Rationale**: Matches live routing constants and sale/promotion modules.

**Alternatives considered**:
- **Introduce `/api/cart`**: Rejected — second base path confuses clients; constitution prefers existing module bases.

---

### 10. Customer segment targeting

**Decision**: Accept `specific_segment` on create; at apply time stub as not eligible until segment engine exists (same as promotions).

**Rationale**: Spec assumption; unblocks CRUD without blocking on deferred segment work.

**Alternatives considered**:
- **Block creating segment codes**: Rejected — sellers would need a later migration of code types.

---

### 11. Concurrent usage at checkout

**Decision**: `IncrementUsageAtomically` with `WHERE current_usage_count < limit`; usage row insert in same DB transaction as order snapshots. Failed increment aborts checkout.

**Rationale**: Same pattern as promotion `IncrementUsageAtomically`; prevents oversell of limited codes.

**Alternatives considered**:
- **Check-then-increment without atomic predicate**: Rejected — race under concurrent checkout.

## Unresolved items

None blocking implementation. Optional polish only: Redis cache of active codes per seller (`T069` adjacent); apply rate-limit remains optional in `T069`.

### BXGY metadata (resolved)

**Decision**: Use `buyQuantity`, `getQuantity`, `getDiscountPercent` on `discount_code.metadata` as documented in `contracts/api-contracts.md`.

### Promotion usage on order (resolved)

**Decision**: Out of scope for 007 — record `discount_code_usage` only. Leave `promotion_usage` for a later promotion-redemption ticket.
