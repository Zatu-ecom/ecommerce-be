# Feature Specification: Money & Currency Standardization

**Feature Branch**: `008-money-currency-standardization`  
**Created**: 2026-08-02  
**Status**: Draft  
**Input**: User description: "Standardize money and currency across the app using plans/009-money-currency-standardization/pre-spec.md — clients send amounts in seller selected currency; backend stores and calculates in cents; responses return a shared Money contract with both major amount and cents plus formatted display; catalog prices rename to price_cents; shared contracts in common/model with encapsulation; strong integration tests required."

**Source pre-spec**: `plans/009-money-currency-standardization/pre-spec.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Seller sets product prices in their currency (Priority: P1)

A seller configures their store’s base currency (for example INR or JPY). When they create or update a product, variant, or package option, they enter the price as a normal currency amount in that selected currency (for example `1299.50`). The system stores the price precisely for later checkout math, and when the seller views the product again they see the same amount in a consistent money shape: numeric amount, exact minor-unit value, and a formatted display string, along with currency metadata.

**Why this priority**: Catalog pricing is the source of all cart and order money. If product prices are wrong or ambiguous, every downstream commerce flow fails.

**Independent Test**: With a seller whose base currency is INR (2 decimal places), create a variant priced `99.99`, reload it, and confirm the returned money shows amount `99.99`, minor units `9999`, and a formatted INR string. Repeat for a JPY seller with price `100` and confirm minor units `100` (not scaled as if it had 2 decimals).

**Acceptance Scenarios**:

1. **Given** a seller with base currency INR (2 decimal places), **When** they create a variant with price `99.99`, **Then** the stored precise value equals 9999 minor units and the read response returns amount `99.99`, minor units `9999`, formatted display, and currency `INR`.
2. **Given** a seller with base currency JPY (0 decimal places), **When** they create a variant with price `100`, **Then** the stored precise value equals 100 minor units and the read response returns amount `100`, minor units `100`, and currency `JPY`.
3. **Given** a seller with base currency JPY, **When** they submit price `100.5`, **Then** the request is rejected as invalid precision for that currency.
4. **Given** a seller with base currency INR, **When** they submit price `99.999`, **Then** the request is rejected as invalid precision for that currency.

---

### User Story 2 - Shopper sees consistent cart and order money (Priority: P1)

A customer (or guest) adds priced products to the cart and later places an order. Every monetary value they see—unit price, line total, discounts, shipping, tax, and grand total—uses the same shared money shape (amount + minor units + formatted text) and the same currency metadata. Cart totals and the resulting order totals match for the same basket.

**Why this priority**: Storefront trust depends on readable, consistent prices. Mismatched cart vs order amounts cause support load and abandoned checkouts.

**Independent Test**: Create an INR-priced variant, add quantity 2 to cart, inspect cart money fields, place an order, and confirm order line and total minor units match the cart snapshot.

**Acceptance Scenarios**:

1. **Given** a cart with an INR variant priced `1299.50` at quantity 2, **When** the cart is retrieved, **Then** unit price minor units are `129950`, line total minor units are `259900`, and both include amount and formatted fields.
2. **Given** that same cart, **When** an order is created, **Then** order item and total money values match the cart’s precise minor units and use the same shared money shape.
3. **Given** a JPY-priced variant of `500`, **When** it is added to cart and ordered, **Then** money values remain `500` minor units (not incorrectly multiplied as if 2-decimal).
4. **Given** a guest cart for a seller, **When** the cart is retrieved, **Then** money uses the seller’s base currency and the same shared money shape as authenticated carts.

---

### User Story 3 - Seller configures discounts in their currency (Priority: P2)

A seller creates promotions or discount codes with fixed money amounts or purchase thresholds (for example “₹100 off” or “minimum purchase ₹1000”). They enter those values as normal currency amounts in their selected currency. The system applies discounts using precise minor-unit math, and when the seller or storefront reviews the offer, money fields use the same shared shape as products and carts. Percentage discounts remain percentages, not money objects.

**Why this priority**: Promotions are high-visibility commercial tools; entering cents by mistake (or seeing inconsistent units) causes wrong discounts and revenue leakage.

**Independent Test**: As an INR seller, create a fixed discount of `50.00` and a minimum purchase of `1000`; confirm stored precise values and that cart eligibility/discount application use those values. As a JPY seller, create fixed `50` and confirm it is not treated as `5000`.

**Acceptance Scenarios**:

1. **Given** an INR seller, **When** they create a fixed-amount promotion of `50.00`, **Then** the system stores 5000 minor units and returns the shared money shape on read.
2. **Given** an INR seller, **When** they set minimum purchase `1000`, **Then** eligibility checks use 100000 minor units.
3. **Given** a JPY seller, **When** they create a fixed-amount discount of `50`, **Then** the system stores 50 minor units (not 5000).
4. **Given** a percentage discount of `15`, **When** it is created or returned, **Then** it remains a plain percentage value and is not wrapped as money.

---

### User Story 4 - Clients always receive one money language (Priority: P2)

Any client (seller admin or storefront) that reads money-bearing resources—products, carts, orders, available coupons, reports—receives the same nested money object and the same currency object. There are no module-specific alternate shapes (for example raw unlabeled integers that look like major amounts but are actually minor units).

**Why this priority**: Cross-app consistency is the point of standardization; partial adoption leaves frontends guessing.

**Independent Test**: Sample product, cart, order, and coupon payloads and assert every money object has exactly amount, minor-unit field, and formatted text; parents expose currency code, symbol, and decimal digits.

**Acceptance Scenarios**:

1. **Given** money-bearing product, cart, and order responses, **When** inspected, **Then** every money value uses the shared nested shape with amount, minor units, and formatted display.
2. **Given** those same responses, **When** inspected, **Then** currency metadata includes code, symbol, and decimal digit count once on the parent resource.
3. **Given** the new contract is active, **When** clients read commerce responses, **Then** legacy ambiguous money fields (unlabeled minor-unit integers or response-only `*Cents` scalars without the shared shape) are not present.

---

### User Story 5 - Platform proves correctness with strong tests (Priority: P1)

Before release, the platform verifies seller-currency input, precise storage, dual response shape, and product→cart→order continuity across at least two currency decimal styles (including a zero-decimal currency). Failures in the core round-trip block release.

**Why this priority**: Money bugs are silent and costly; the pre-spec treats automated proof as mandatory.

**Independent Test**: Run the cross-flow verification suite for INR and JPY sellers covering product write, cart, order, promotion fixed amounts, and response-shape checks.

**Acceptance Scenarios**:

1. **Given** the verification suite, **When** executed in CI, **Then** INR and at least one zero-decimal currency path pass.
2. **Given** a failing product→cart→order precise-value round-trip, **When** CI runs, **Then** the release is blocked.
3. **Given** two sellers with different currencies, **When** they price independently, **Then** each seller’s amounts convert using only their own currency rules.

---

### Edge Cases

- Seller submits more fractional digits than their currency allows → request rejected with a clear validation error.
- Zero-decimal currencies (for example JPY) must not be treated as two-decimal currencies.
- Three-or-more decimal currencies (if configured) must scale using that digit count.
- Percentage fields must never be converted or displayed as money objects.
- Buyer preferred currency differs from seller base currency → until foreign-exchange conversion exists, all presented money remains in the seller’s base currency.
- Existing catalog prices stored as decimal major units must be migrated to precise minor units without silent cross-seller contamination.
- Guest and authenticated carts must both use the shared money contract.
- Fixed-amount vs percentage discount codes must be interpreted by discount type.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST interpret all seller-entered monetary amounts as major units of that seller’s selected/base currency.
- **FR-002**: System MUST validate that submitted monetary amounts do not exceed the fractional precision allowed by the seller’s currency decimal digits, and MUST reject invalid precision.
- **FR-003**: System MUST persist all commercial monetary amounts as integer minor units (cents/paise-style), with storage names that clearly indicate minor units for catalog prices.
- **FR-004**: System MUST perform all discount, cart, tax, shipping, and order total calculations using integer minor units only.
- **FR-005**: System MUST return every monetary field in a shared nested money shape containing: major amount, minor-unit integer, and formatted display string.
- **FR-006**: System MUST return currency metadata (code, symbol, decimal digits) on money-bearing parent resources.
- **FR-007**: System MUST use the same money and currency contracts across product, cart, guest cart, order, promotion, discount code, and report money fields.
- **FR-008**: Clients MUST NOT be required to send minor units on write APIs in this release; writes accept major units only.
- **FR-009**: Percentage-based values MUST remain plain percentages and MUST NOT use the money shape.
- **FR-010**: Until foreign-exchange conversion is delivered, all money presentation MUST use the seller’s base currency even if a buyer has another preferred currency.
- **FR-011**: Catalog prices for variants and package options MUST be stored as explicit minor-unit amounts (not ambiguous decimal “price” values), including migration of existing catalog prices.
- **FR-012**: Shared money and currency contracts MUST be defined once in the platform’s common shared-model layer, with each contract encapsulating its data and related behavior, reusable by all commerce areas without circular dependencies between areas.
- **FR-013**: Existing shared API envelope, pagination, and validation contracts MUST be reorganized into that same common shared-model layer as separate per-contract definitions.
- **FR-014**: System MUST provide automated verification covering seller-currency writes, minor-unit persistence, dual money responses, and product→cart→order continuity for at least one 2-decimal and one 0-decimal currency.
- **FR-015**: Promotion and discount-code fixed amounts and monetary thresholds MUST be entered as major units in seller currency and stored/applied as minor units.
- **FR-016**: Cart and order responses MUST stop exposing ambiguous unlabeled minor-unit integers as if they were major amounts.
- **FR-017**: Money conversion and formatting rules MUST be centralized in the shared money/currency contracts so individual commerce areas do not assume a fixed two-decimal currency.

### Key Entities

- **Seller currency**: The store’s selected/base currency (code, symbol, decimal digit count) that defines how seller-entered amounts are interpreted and how money is presented until FX exists.
- **Money value**: A commercial amount expressed for clients as major amount + minor units + formatted display, always tied to a currency context.
- **Catalog price**: The sellable price of a variant or package option, persisted as minor units and shown via the shared money value.
- **Cart totals**: Runtime pricing summary (subtotal, discounts, shipping, tax, total) derived from catalog minor units and promotions.
- **Order money snapshot**: Persisted commercial totals and line amounts captured at checkout from the cart’s precise minor units.
- **Promotion / discount monetary rule**: Fixed amounts or thresholds entered in seller currency and applied using minor-unit math; distinct from percentage rules.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In verification runs, 100% of sampled product, cart, and order money fields use the shared money shape (amount + minor units + formatted) with parent currency metadata.
- **SC-002**: For a standard INR basket, product→cart→order precise minor-unit values match end-to-end with zero drift in automated checks.
- **SC-003**: For a standard JPY basket, product→cart→order precise minor-unit values match end-to-end with zero incorrect 100× scaling in automated checks.
- **SC-004**: 100% of invalid excess-precision price submissions in the verification suite are rejected before persistence.
- **SC-005**: Sellers can create a fixed discount in their currency and see the same major amount reflected on read without manually converting to minor units.
- **SC-006**: After rollout, support/QA can no longer find module-specific conflicting money meanings in cart vs order vs product for the covered flows (spot-check checklist passes).
- **SC-007**: Core money round-trip and dual-currency verification suites are required gates; a failure blocks release.

## Assumptions

- Source of truth for intent is `plans/009-money-currency-standardization/pre-spec.md`; this spec states the user/business outcomes, while planning may retain technical design detail.
- Seller selected currency means `seller_settings` base currency already present in the platform.
- Inbound excess fractional digits are **rejected** (not silently rounded).
- Nested shared money object is the client contract (not flat parallel fields).
- This release is a **coordinated breaking change** for money-bearing client payloads (no long-term dual legacy fields unless planning later decides otherwise).
- Foreign-exchange / buyer-currency price conversion remains out of scope; display stays on seller base currency.
- Payment gateway HTTP surface rewrite is out of scope beyond aligning with the same money meaning when touched.
- Promotion/coupon business rules (stacking, eligibility types) do not change—only money representation and I/O.
- Historical catalog decimals are assumed two-decimal for backfill unless an audit finds non-2dp sellers that need special handling.
- Discount-code percentage vs fixed amount continues to share a value field interpreted by discount type unless planning chooses a column split.
- Reports expose money using the same shared shape for revenue/discount metrics in seller currency.

## Out of Scope

- Live FX conversion and multi-currency price books per seller.
- Changing promotion stacking, eligibility, or coupon apply business rules.
- Full payment gateway adapter redesign.
- Accepting client-sent minor units on write APIs in v1.
