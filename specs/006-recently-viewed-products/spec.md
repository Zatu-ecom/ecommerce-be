# Feature Specification: Recently Viewed Products

**Feature Branch**: `006-recently-viewed-products`  
**Created**: 2026-07-15  
**Status**: Draft  
**Input**: User description: "Implement recently viewed products feature: automatically record user-product pairs when GetProductByID API is called, keep last 10 per user for customer role only, fire-and-forget recording that never breaks product response"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Automatic Recording When Viewing Product (Priority: P1)

As a customer browsing the storefront, whenever I view a product detail page, the system should automatically remember that I viewed this product without any extra action from me. This happens silently in the background and never interferes with the product page loading.

**Why this priority**: This is the core feature — without automatic recording, there are no recently viewed products to display anywhere. It provides the foundation for personalized recommendations and the recently viewed section on the home page.

**Independent Test**: Can be fully tested by having a customer view a product via the product detail endpoint and verifying the record is saved in the database with the correct user ID, product ID, and seller ID.

**Acceptance Scenarios**:

1. **Given** an authenticated customer who has never viewed a product, **When** they fetch product details for product P1, **Then** a new recently viewed record is created linking their user ID to product P1 with the current timestamp.
2. **Given** an authenticated customer who has viewed product P1 before, **When** they fetch product details for product P1 again, **Then** the existing record's viewed timestamp is updated to the current time (no duplicate rows).
3. **Given** an unauthenticated user (no valid JWT token), **When** they fetch product details for any product, **Then** no recently viewed record is created.
4. **Given** an authenticated customer with 10 existing recently viewed records, **When** they view a new product P11, **Then** the system saves the new record and removes the oldest record, keeping exactly 10 recently viewed products.

---

### User Story 2 - Role-Based Recording Restriction (Priority: P1)

As a platform operator, I want recently viewed recording to apply only to customers, not to sellers or admins who browse the storefront for testing, preview, or administrative purposes.

**Why this priority**: Without this restriction, seller and admin browsing activity would pollute the recently viewed data, leading to irrelevant recommendations for actual customers and data integrity issues.

**Independent Test**: Can be fully tested by having a seller fetch a product via the same endpoint and verifying no recently viewed record is created in the database.

**Acceptance Scenarios**:

1. **Given** an authenticated seller (role level 2), **When** they fetch product details for any product, **Then** no recently viewed record is created.
2. **Given** an authenticated admin (role level 1), **When** they fetch product details for any product, **Then** no recently viewed record is created.
3. **Given** an authenticated customer (role level 3), **When** they fetch product details for any product, **Then** a recently viewed record IS created.

---

### User Story 3 - Fire-and-Forget Reliability (Priority: P1)

As a customer browsing the storefront, if the recently viewed recording fails for any reason (database error, timeout, etc.), I should still see my product details without any delay or error message.

**Why this priority**: The product detail page is the primary customer experience. The recently viewed recording is a secondary, non-critical enhancement. Degrading the primary experience for a secondary feature is unacceptable.

**Independent Test**: Can be tested by simulating a database error during recording and verifying the product response is still returned successfully with HTTP 200 and complete product data.

**Acceptance Scenarios**:

1. **Given** any failure occurs while saving the recently viewed record, **When** a customer fetches product details, **Then** the product response is returned successfully with complete data (HTTP 200) and the recording failure is logged but not exposed to the customer.
2. **Given** the database is temporarily unreachable, **When** a customer fetches product details, **Then** the customer receives the product data without any indication of the recording failure.

---

### User Story 4 - Retrieve Recently Viewed Products (Priority: P2)

As a customer, I want to see a list of products I have recently viewed so I can quickly return to items I was interested in without searching again.

**Why this priority**: While automatic recording (P1) provides the data foundation, the retrieval endpoint enables actual customer-facing features like a "Recently Viewed" section on the home page or account page. This is dependent on P1 being completed first.

**Independent Test**: Can be fully tested by having a customer view several products, then calling the retrieval endpoint and verifying the returned list is in reverse chronological order (most recent first).

**Acceptance Scenarios**:

1. **Given** a customer who has viewed 5 products, **When** they request their recently viewed products, **Then** they receive a list of 5 product IDs ordered by most recently viewed first.
2. **Given** a customer who has viewed 15 products (with only 10 retained), **When** they request their recently viewed products, **Then** they receive exactly 10 products (oldest 5 removed) ordered by most recently viewed first.
3. **Given** a customer who has never viewed any product, **When** they request their recently viewed products, **Then** they receive an empty list without errors.

---

### Edge Cases

- **Concurrent views**: What happens when a customer opens two product pages simultaneously? Each upsert is atomic; however, trimming may briefly result in 11 entries before both requests complete. The next view trims back to 10.
- **Deleted product**: What happens when a product is deleted from the catalog? The recently viewed record is automatically removed due to the cascade delete relationship.
- **Deleted user**: The `user_id` column references the user table. If a user is deleted, their recently viewed records should be cleaned up (handled by existing user deletion logic or cascade).
- **User with zero recently viewed**: When a user who has never viewed any product requests their list, an empty list is returned without error.
- **Same product, different sellers**: The unique constraint is on `(user_id, product_id)`, so a customer cannot have the same product from different sellers. Viewing the same product always updates the existing row regardless of seller.
- **Product ID not found**: If a customer attempts to view a non-existent product, the product API returns a 404 error before recording logic executes, so no record is created.
- **Race condition during trim**: When two views push a user from 10 to 12 entries, the trim logic may briefly have more than 10 entries but stabilizes to exactly 10 on the next view.
- **Malformed or expired token**: Invalid tokens are rejected by authentication middleware before the handler executes, so no recording attempt is made.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST automatically create or update a recently viewed record when an authenticated customer successfully fetches product details.
- **FR-002**: System MUST associate each recently viewed record with the user ID, product ID, seller ID, and the timestamp of the view.
- **FR-003**: System MUST update the viewed timestamp (not create a duplicate) when a customer re-views a product they have already viewed.
- **FR-004**: System MUST maintain a maximum of 10 recently viewed records per user, automatically removing the oldest entry when the limit is exceeded.
- **FR-005**: System MUST restrict recently viewed recording to customers only (role level 3), skipping recording for sellers (role level 2) and admins (role level 1).
- **FR-006**: System MUST NOT return any error to the customer if the recently viewed recording fails; the product response MUST be delivered successfully regardless of recording outcome.
- **FR-007**: System MUST log recording failures for operational visibility without exposing them to end users.
- **FR-008**: System MUST skip recording entirely for unauthenticated users (no valid JWT token).
- **FR-009**: System MUST remove recently viewed records for a product when that product is deleted from the catalog.
- **FR-010**: Users MUST be able to retrieve their recently viewed products in reverse chronological order (most recent first), with a maximum of 10 entries.

### Key Entities

- **Recently Viewed Record**: Represents a customer's view of a specific product. Contains the customer's user ID, the product ID, the seller ID of the product at the time of viewing, and the timestamp when the customer viewed it. Each customer-product pair is unique (re-viewing updates the timestamp rather than creating a new record).
- **Product**: The existing product catalog entity. Recently viewed records reference products, and when a product is deleted, its associated recently viewed records are automatically removed.
- **User (Customer)**: The existing user entity. Only users with the customer role generate recently viewed records. Records are tied to a specific user ID and represent that user's browsing history.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Product detail pages load with no perceptible delay from recently viewed recording — the recording adds less than 5 milliseconds to the total response time.
- **SC-002**: 100% of product detail fetches by authenticated customers successfully return product data regardless of whether the recording succeeds or fails.
- **SC-003**: No customer with a valid token ever sees an error message caused by recently viewed recording failures.
- **SC-004**: Each customer's recently viewed list stabilizes at exactly 10 products. Under concurrent viewing scenarios, the list may briefly have 11 entries before the next view trims back to 10 — this is self-correcting and acceptable for a fire-and-forget feature.
- **SC-005**: Re-viewing a previously viewed product always reflects the most recent view timestamp, with zero duplicate entries per product.
- **SC-006**: Zero recently viewed records are created for seller or admin user roles browsing the storefront.

## Assumptions

- The existing authentication system (JWT-based) reliably identifies authenticated users and their roles. The recently viewed feature depends on this existing infrastructure.
- The product detail API (`GET /api/product/:productId`) is the primary trigger for recording views. Other product discovery methods (search, listings, recommendations) are out of scope for the initial recording trigger.
- The 10-product retention limit is a reasonable default for the initial release. Future iterations may make this configurable per seller or per user preference.
- User accounts are not frequently deleted; if user deletion is implemented in the future, cleanup of associated recently viewed records should be handled as part of that feature.
- The retrieval endpoint (P2 user story) is secondary and can be delivered after the recording foundation is complete, aligning with the recommendation engine's home page integration.
- Existing database infrastructure (PostgreSQL) and migration tooling are used to create and manage the recently viewed table.
