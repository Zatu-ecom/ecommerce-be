# Feature Specification: Coupon / Discount Codes

**Feature Branch**: `007-coupon-discount-codes`  
**Created**: 2026-08-01  
**Status**: Draft  
**Input**: User description: "Complete coupon and discount-code feature from pre-spec: sellers manage discount codes and product/category scopes; customers apply/remove coupons on cart with full priced cart responses computed on the backend; checkout snapshots coupon discounts and records usage; storefront stays thin. Source: specs/007-coupon-discount-codes/pre-spec.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Seller Creates and Manages Discount Codes (Priority: P1)

As a seller, I want to create promotional discount codes (percentage off, fixed amount off, free shipping, or buy-X-get-Y), set rules such as date window, usage limits, minimum purchase, and who can use them, and later activate, deactivate, update, or remove unused codes so I can run coupon campaigns for my store.

**Why this priority**: Without seller-managed codes there is nothing for customers to apply. This is the foundation of the feature and delivers value on its own for campaign setup.

**Independent Test**: A seller can create a code, list it, retrieve it, update rules, deactivate it, and delete an unused code; another seller cannot see or change that code.

**Acceptance Scenarios**:

1. **Given** an authenticated seller, **When** they create a valid discount code with type, value, start date, and optional limits, **Then** the code is saved for their store only and returned with usage count starting at zero.
2. **Given** a seller who already owns code `SAVE20`, **When** they try to create another code with the same text, **Then** creation is rejected as a duplicate for that seller.
3. **Given** two sellers, **When** seller B creates the same code text that seller A already uses, **Then** creation succeeds for seller B (codes are unique per store, not globally).
4. **Given** an existing unused code, **When** the seller deactivates it, **Then** customers can no longer apply it, but the seller can still view and reactivate it.
5. **Given** a code that has never been redeemed on an order, **When** the seller deletes it, **Then** it is permanently removed.
6. **Given** a code that has been redeemed at least once, **When** the seller tries to delete it, **Then** deletion is blocked and they must deactivate it instead.
7. **Given** a seller viewing another seller’s code identifier, **When** they try to get or change it, **Then** they receive a not-found style denial (no cross-store leakage).

---

### User Story 2 - Seller Limits Codes to Specific Catalog Scope (Priority: P1)

As a seller, I want to restrict a discount code to specific products, variants, categories, or collections so coupons only apply to the merchandise I intend.

**Why this priority**: Most campaigns are not store-wide; scoped codes are required for realistic promotions and prevent accidental whole-cart discounts.

**Independent Test**: As a seller, attach products (or categories/collections/variants) to a scoped code, list them, remove some or all, and confirm `all_products` codes reject scope mutations — without requiring cart apply (cart eligibility is covered under User Story 4).

**Acceptance Scenarios**:

1. **Given** a code configured for specific products, **When** the seller adds product IDs to that code, **Then** those products are listed under the code’s product scope.
2. **Given** a code configured for all products, **When** the seller tries to add product scope rows, **Then** the action is rejected.
3. **Given** a scoped code owned by seller A, **When** seller B tries to list or mutate its scope, **Then** access is denied (not found / forbidden per isolation rules).
4. **Given** products on a scoped code, **When** the seller removes specific IDs or clears all scope rows, **Then** the scope list reflects the removal.

---

### User Story 3 - Customer Applies and Removes Coupons on Cart (Priority: P1)

As a logged-in customer, I want to apply a coupon code to my cart, see the updated totals immediately (including automatic promotions plus coupon savings), remove one or all coupons, and see which other coupons I could still use—without computing discounts myself on the storefront.

**Why this priority**: This is the customer-facing core value. Cart must return complete priced results so the client only renders.

**Independent Test**: With a seeded active code and a cart of items, apply the code, verify cart summary coupon fields and line/total savings, remove the code, and confirm totals revert.

**Acceptance Scenarios**:

1. **Given** an authenticated customer with items in cart and a valid active code for that store, **When** they apply the code, **Then** the cart response includes the applied coupon, coupon discount in the summary, and updated totals combining automatic promotions and the coupon.
2. **Given** a cart with an applied coupon, **When** the customer removes that coupon (or removes all coupons), **Then** the cart response no longer lists it and totals are recalculated without that coupon.
3. **Given** an authenticated customer with a cart, **When** they request available coupons (or view cart), **Then** they see applicable coupons with potential savings and non-applicable coupons with clear reasons.
4. **Given** a guest (not logged in) cart, **When** they try to apply a coupon, **Then** the action is denied until they authenticate.
5. **Given** an invalid, expired, not-yet-started, already-applied, over-limit, ineligible, non-combinable, or out-of-scope code, **When** the customer applies it, **Then** they receive a clear business error and the cart is unchanged.

---

### User Story 4 - Cart Always Shows Fresh Discount Totals (Priority: P1)

As a customer, whenever I view or change my cart, I want promotions and coupons recalculated from current prices and current campaign rules so I never see stale savings if a code was turned off or prices changed.

**Why this priority**: Trust and correct checkout amounts depend on runtime recalculation; storing frozen discount money on the cart would drift.

**Independent Test**: Apply a coupon, then have the seller deactivate the code; on next cart view the coupon is gone and totals update. Also verify scoped codes only discount eligible cart items, and stacking flags are honored.

**Acceptance Scenarios**:

1. **Given** a cart with an applied coupon that later becomes invalid (deactivated, expired, or no longer meeting minimums), **When** the customer fetches the cart, **Then** the system drops or ignores the invalid coupon and returns corrected totals.
2. **Given** cart contents and any applied coupons, **When** the customer fetches the cart, **Then** discount amounts are freshly computed (not read as pre-stored money on the cart).
3. **Given** automatic promotions and coupons that may stack, **When** the cart is priced, **Then** promotions are applied first and coupons are evaluated on the after-promotion amounts, respecting “cannot combine” and “promotions may block coupons” rules.
4. **Given** a scoped code and a customer cart that contains none of the scoped items, **When** the customer applies the code, **Then** application fails because the code is not applicable.
5. **Given** a scoped code and a cart with at least one eligible item, **When** the customer applies the code, **Then** the discount is calculated only against eligible items (or shipping, for free-shipping type) according to the code rules.
6. **Given** promotions applied to cart lines, **When** the cart is priced, **Then** `cart_item_promotion` rows are synced to match currently applied promotion IDs per line (references only; amounts still computed at runtime).

---

### User Story 5 - Checkout Records Coupon Redemption (Priority: P2)

As a customer completing checkout, I want my order to permanently record which coupons were used and how much was saved, and as a seller I want usage counts and per-customer limits to update so codes cannot be over-redeemed.

**Why this priority**: Campaign integrity and order history require redemption tracking; depends on cart apply (P1) working first.

**Independent Test**: Place an order with an applied coupon; verify the order shows the coupon snapshot, usage is recorded for that customer, and a second attempt beyond the per-customer limit is rejected.

**Acceptance Scenarios**:

1. **Given** a cart with a valid applied coupon, **When** the customer places an order, **Then** the order retains an immutable coupon snapshot (code, type, discount saved) independent of later code edits.
2. **Given** a successful order with a coupon, **When** redemption completes, **Then** the code’s total usage increases and that customer’s usage toward the per-customer (and reset-window) limit is recorded.
3. **Given** a code at its global or per-customer usage limit, **When** another customer (or the same customer) tries to apply or checkout with it, **Then** the action fails with a clear limit error.
4. **Given** two checkouts racing for the last remaining global use, **When** both attempt to complete, **Then** at most one succeeds in consuming that use; the other fails safely without corrupting counts.

---

### Edge Cases

- Duplicate apply of the same code on one cart is rejected.
- Minimum purchase / minimum quantity not met → apply fails with a clear reason.
- Free-shipping coupon when shipping is already zero → little or no additional shipping savings; still valid if rules otherwise pass.
- Percentage coupon with a maximum cap → savings never exceed the cap.
- Code unique per seller; casing/spacing normalized so `save20` and `SAVE20` are the same code.
- Guest carts cannot apply coupons; after login/merge, customer must apply coupons while authenticated.
- Customer segment–targeted codes: until segment matching exists, treat as not eligible (documented assumption).
- Hard delete blocked after any redemption; deactivate remains available.
- Converted (checked-out) carts do not keep reusable coupon applications for a new active cart.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Sellers MUST be able to create, list, retrieve, update, activate/deactivate, and delete (when unused) discount codes for their own store only.
- **FR-002**: Discount codes MUST support at least: percentage off, fixed amount off, free shipping, and buy-X-get-Y styles, with seller-defined value and optional maximum discount cap where applicable.
- **FR-003**: Sellers MUST be able to set date windows, total usage limits, per-customer usage limits (including optional reset periods), minimum purchase, minimum quantity, eligibility (everyone / new customers / specific segment), and whether the code can combine with other discounts.
- **FR-004**: Sellers MUST be able to scope a code to all products or to specific products, variants, categories, or collections, and manage those scope memberships.
- **FR-005**: Discount code text MUST be unique within a seller’s store and treated case-insensitively after normalization.
- **FR-006**: Authenticated customers MUST be able to apply a coupon to their cart, remove a specific coupon, remove all coupons, and see available vs not-applicable coupons with reasons.
- **FR-007**: Guest users MUST NOT be able to apply coupons.
- **FR-008**: Cart view and coupon mutate actions MUST return a complete cart picture: items, automatic promotions, applied coupons, available coupon hints, and summary totals so the storefront does not compute discounts.
- **FR-009**: Cart MUST store only which coupons and cart-item promotions are attached—not the monetary discount amounts; amounts MUST be recalculated whenever the cart is priced. On each cart price build, after automatic promotions are determined, the system MUST sync `cart_item_promotion` so each cart line’s stored promotion IDs match the promotions just applied (insert missing, delete stale).
- **FR-010**: Pricing MUST apply automatic promotions before coupons and honor stacking / “cannot combine” / “promotions block coupons” rules.
- **FR-011**: Invalid applied coupons MUST be cleaned up or ignored on cart refresh so customers see correct totals.
- **FR-012**: On successful order placement with coupons, the system MUST snapshot coupon details onto the order and record usage against global and per-customer limits atomically with checkout.
- **FR-013**: Cross-seller access to another seller’s codes or applying another store’s code in the wrong store context MUST be prevented.
- **FR-014**: Clear, stable error outcomes MUST exist for invalid, expired, not started, usage exhausted, already used, min purchase/qty, not eligible, cannot combine, already applied, and not applicable cases.
- **FR-015**: Soft-delete of discount codes MUST NOT be used; lifecycle is active/inactive plus hard delete only when never redeemed.

### Key Entities *(include if feature involves data)*

- **Discount Code**: A seller-owned coupon definition (code text, discount style/value, rules, dates, active flag, usage counters/limits, optional segment targeting).
- **Discount Code Scope**: Links a code to eligible products, variants, categories, or collections.
- **Cart Applied Coupon**: Association between a customer cart and a discount code (reference only).
- **Cart Item Promotion**: Association between a cart line and a promotion (reference only; supports cart discount integration alongside coupons).
- **Discount Code Usage**: Record that a customer redeemed a code on a specific order, with amounts saved.
- **Order Applied Coupon**: Immutable copy of coupon application details stored with the order for history and support.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A seller can create a usable discount code and attach catalog scope in under 5 minutes without engineering help.
- **SC-002**: An authenticated customer can apply a valid coupon and see updated cart totals (promotions + coupon) in a single response, with no second “recalculate” step required on the storefront.
- **SC-003**: 100% of cart pricing responses for carts with coupons show totals that match backend recalculation rules (no reliance on client-side math).
- **SC-004**: After a successful checkout with a coupon, order history shows the coupon snapshot and the code cannot be redeemed beyond its configured limits in subsequent attempts.
- **SC-005**: In seller-isolation checks, 0 cross-tenant reads or updates of another seller’s discount codes succeed.
- **SC-006**: At least 95% of first-time apply attempts with a correctly entered valid code succeed without support intervention (excluding intentional rule failures like min purchase).
- **SC-007**: When a previously applied code becomes invalid, the next cart view corrects totals without requiring the customer to manually clear the coupon first.

## Terminology

| Term | Meaning |
| --- | --- |
| **Discount code** | Seller-managed campaign definition (CRUD under `/api/promotion/discount-code`) |
| **Coupon** | Customer action of applying/removing a discount code on cart (`/api/order/cart/coupon`) |
| **Promotion** | Automatic (non-code) campaign already applied via `ApplyPromotionsToCart` |

Do not use “promo code” in new documentation; prefer the terms above.

## Assumptions

- Source of detailed design intent is `pre-spec.md` in this feature folder; this spec states the user-facing WHAT/WHY while planning/tasks will cover HOW.
- Existing automatic promotion application on cart already exists and remains the first pricing stage; this feature adds coupons on top.
- Storefront is built by a separate client team; backend owns all eligibility, stacking, and money math.
- Authenticated customer carts only for coupons; guest coupon apply is out of scope for v1.
- Customer-segment rule evaluation is stubbed: codes targeting a specific segment are treated as not eligible until a later segment feature.
- Available coupons are included with the full cart response by default (capped list), with an optional dedicated “available coupons” view for the same data.
- Buy-X-get-Y discount codes use metadata: `buyQuantity` (int ≥ 1), `getQuantity` (int ≥ 1), `getDiscountPercent` (0–100; 100 = free items). Invalid metadata fails create/update validation.
- Currency formatting in cart responses follows the store’s existing currency handling.
- Soft delete is intentionally not used for discount codes or cart coupon attachments.
- Recording automatic **promotion** usage (`promotion_usage`) on order is **out of scope** for this feature; only **discount code / coupon** usage (`discount_code_usage`) is required.

## Scope Boundaries

### In Scope

- Seller discount-code lifecycle and catalog scopes
- Customer cart apply/remove/available coupons and full priced cart responses
- Runtime recalculation model (reference-only cart attachments)
- Checkout coupon snapshot and usage enforcement
- Multi-tenant seller isolation and guest-coupon denial

### Out of Scope

- Storefront UI implementation
- Client-side discount calculation
- Full customer-segment rule engine (segment-targeted codes are rejected at apply as not eligible until a later feature)
- Guest-cart coupon apply
- Redesigning the existing automatic promotion catalog beyond stacking interaction with coupons
- Recording `promotion_usage` rows for automatic promotions on checkout (coupon/`discount_code_usage` only)
