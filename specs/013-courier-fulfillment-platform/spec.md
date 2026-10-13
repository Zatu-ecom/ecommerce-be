# Feature Specification: Courier Fulfillment Platform

**Feature Branch**: `013-courier-fulfillment-platform` (existing design folder; no new branch — spec consolidates the design docs already there)
**Created**: 2026-10-02
**Status**: Draft
**Input**: User description: "Create the spec.md from the design docs and other required docs in specs/013-courier-fulfillment-platform"
**Design docs**: `data-model.md` (tables), `service-design.md` (adapter + orchestrators), `api-contracts.md` (routes + JSON), `physical-specs.md` (dimensions + units catalog)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Seller connects a courier account (Priority: P1)

A seller opens the shipping settings, picks Shiprocket (or another courier), enters the account credentials, optionally turns on auto-booking, and tests the connection before saving. The platform also keeps one shared account as a fallback so sellers without their own credentials can still ship.

**Why this priority**: Nothing ships without credentials. This is the gate for every other story.

**Independent Test**: Can be fully tested by connecting test credentials, running the connection test, and seeing a masked (never secret-revealing) configuration — delivering a shippable seller with zero shipments yet.

**Acceptance Scenarios**:

1. **Given** a seller with no courier configured, **When** they save valid credentials and test the connection, **Then** the courier shows as configured with masked hints and no secret material is ever displayed or logged.
2. **Given** invalid credentials, **When** they test the connection, **Then** they get a clear "invalid credentials" error and nothing is saved.
3. **Given** a seller with no own credentials, **When** they ship, **Then** the platform's shared account is used transparently.

---

### User Story 2 - Confirmed orders automatically become shippable boxes (Priority: P1)

When an order is confirmed (payment captured, or COD placed), the system automatically splits it into one draft box per warehouse based on where the stock sits, reserving the right units. A single-warehouse order becomes exactly one draft. The seller reviews ready-to-book drafts instead of building shipments by hand.

**Why this priority**: This is the core automation — location-based splitting is the reason the module exists.

**Independent Test**: Can be fully tested by confirming a multi-warehouse order and observing one draft per warehouse with correct items and quantities — delivering planned shipments with no courier contact and no money spent.

**Acceptance Scenarios**:

1. **Given** a confirmed single-warehouse order, **When** planning runs, **Then** exactly one draft exists with all lines at full quantity.
2. **Given** a confirmed order stocked across two warehouses, **When** planning runs, **Then** two drafts exist, each stamped with its warehouse and delivery address, splitting lines by warehouse priority.
3. **Given** the confirmation signal arrives twice, **When** planning re-runs, **Then** no duplicate drafts are created.

---

### User Story 3 - Seller books a box and the courier picks it up (Priority: P1)

The seller reviews a draft (fixing weight/size if needed), books it, and receives a tracking number (AWB). Pickup is scheduled as part of booking or separately. Sellers who opted into auto-booking skip the manual step entirely. A printable shipping label is available on demand.

**Why this priority**: Booking is the moment money is spent with the courier — the module's primary transaction.

**Independent Test**: Can be fully tested by booking a draft and receiving a tracking number plus a downloadable label — delivering a live shipment with pickup arranged.

**Acceptance Scenarios**:

1. **Given** a complete draft, **When** the seller books it, **Then** a tracking number is returned, the box status advances, and a label can be downloaded.
2. **Given** a draft missing weight, **When** booking is attempted, **Then** booking is refused with a clear error and the seller is asked to fix the weight first.
3. **Given** the customer changed address after planning, **When** booking is attempted, **Then** booking is refused until the seller reviews and accepts the new address.
4. **Given** a seller with auto-booking enabled, **When** planning completes unambiguously, **Then** drafts are booked without a click; when the system cannot choose (e.g. multiple couriers, no preference), drafts wait with a clear explanation.

---

### User Story 4 - Tracking updates arrive on their own; buyers can follow along (Priority: P1)

The courier pushes status updates (picked up, in transit, out for delivery, delivered, failed attempt) and the system records them on the right box automatically, keeping a full history. Buyers see live tracking for their own orders without ever waiting on a courier call. A background process heals any missed updates.

**Why this priority**: Real-time visibility is the customer-facing promise; missed or stale tracking drives support tickets.

**Independent Test**: Can be fully tested by simulating courier updates for a box and watching its status and history advance, plus a buyer viewing tracking — delivering live visibility with no seller action.

**Acceptance Scenarios**:

1. **Given** a courier status push for a known box, **When** it arrives, **Then** the box status and history update and the right follow-up fires (e.g. delivered completes the box).
2. **Given** a duplicate or out-of-order update, **When** processed, **Then** history stays correct with no duplicate side effects and terminal states never move backward.
3. **Given** updates stopped arriving (missed pushes), **When** the background check runs, **Then** stale boxes are re-synced from the courier automatically.
4. **Given** a buyer views their order, **When** tracking loads, **Then** it returns instantly from our records with no courier call and shows only their own data.

---

### User Story 5 - Seller handles failed deliveries and returns (Priority: P2)

When delivery fails (customer absent, bad address), the seller is notified and chooses: try again or send the box back (RTO). After delivery, the seller can create a return box that travels back and restocks inventory on arrival.

**Why this priority**: Failed-delivery handling is where shipping profit is won or lost (return freight + lost sale); it needs Tamely-defined actions, but only after core booking/tracking works.

**Independent Test**: Can be fully tested by raising a failed-delivery round, acting on it, and completing a return box end to end — delivering recovered revenue or controlled return cost.

**Acceptance Scenarios**:

1. **Given** a failed delivery attempt, **When** the seller chooses re-attempt, **Then** a new delivery round opens and the box resumes its journey on the next update.
2. **Given** a failed delivery the seller gives up on, **When** they choose return-to-origin, **Then** the box is tracked back and marked returned on arrival.
3. **Given** a delivered box, **When** the seller creates a return, **Then** a linked return box is created and inventory is restocked exactly once on its return.

---

### User Story 6 - Products carry shippable weight and size (Priority: P2)

Sellers enter weight and dimensions while creating products, picking familiar units (grams/kilograms, cm/metres) from system-provided lists. The system normalizes everything behind the scenes so booking, rates, and planning always compute in one standard unit.

**Why this priority**: Accurate freight quotes and successful bookings depend on trusted numbers; without this, every booking risks wrong charges or refusal.

**Independent Test**: Can be fully tested by creating a product with kilogram/metre specs and watching planning/booking use correct gram/cm values — delivering trustworthy freight math with no seller-side conversions.

**Acceptance Scenarios**:

1. **Given** the product form, **When** a seller opens shipping specs, **Then** they see the standard measurements with unit choices and enter plain numbers.
2. **Given** specs entered in kilograms/metres, **When** a box is planned and rated, **Then** all math uses the correctly converted base values.
3. **Given** a product with no specs, **When** its order ships, **Then** the system flags it for manual weight entry instead of inventing numbers.

---

### Edge Cases

- What happens when stock reserved at checkout is gone by planning time? Planning raises a visible exception with an alert; the order waits for manual shipment creation — stock disagreement is never silently absorbed (no backorder in v1).
- What happens when the courier's update references an unknown tracking number? It is acknowledged and ignored with nothing stored.
- What happens when a forged or badly-signed courier update arrives? It is rejected and nothing is stored or changed.
- What happens when two status updates arrive out of order (e.g. "in transit" after "out for delivery")? The box follows the allowed transition map; terminal states never regress.
- What happens when the customer changes address after the box was planned? Booking is blocked until the seller reviews and accepts the new address.
- What happens when an order is cancelled before/after booking? Pre-booking drafts cancel locally at zero cost; post-booking boxes cancel via the courier where still possible.
- What happens when the seller has no shippable specs on old products? Drafts plan normally with weight empty and wait for manual entry.
- What happens when auto-booking is on but the system can't choose a courier? Drafts wait with an explicit "ambiguous" reason the UI can explain.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST let sellers connect courier accounts (own credentials with a platform shared-account fallback) and test connections without saving.
- **FR-002**: System MUST never display or log secret credential material; dashboards show masked hints only.
- **FR-003**: System MUST auto-create one draft shipment per warehouse when an order is confirmed, with correct items, quantities, and both location references.
- **FR-004**: Planning MUST be idempotent — replays and retries never create duplicate drafts.
- **FR-005**: System MUST refuse booking when weight is missing, the delivery address changed since planning, or the stock hold is dead — each with a specific, actionable error.
- **FR-006**: System MUST support per-seller opt-in auto-booking (plan → book → pickup without clicks) and leave drafts untouched for sellers who opt out.
- **FR-007**: System MUST record every courier status update on the correct box with full history, deduplicate replays, and never move terminal states backward.
- **FR-008**: System MUST re-sync stale in-flight boxes automatically when courier pushes stop arriving.
- **FR-009**: Buyers MUST be able to track only their own orders' boxes, served from our records with no courier call.
- **FR-010**: System MUST support failed-delivery rounds (one open at a time) with re-attempt and return-to-origin actions.
- **FR-011**: System MUST support post-delivery return boxes linked to the original, restocking inventory exactly once on return.
- **FR-012**: System MUST let sellers cancel boxes before pickup (local-only for drafts, via courier once booked where possible).
- **FR-013**: Product creation MUST offer system-defined shippable specs (weight, length, breadth, height) with unit choices; values MUST be numeric and positive.
- **FR-014**: System MUST normalize all physical specs to base units (grams, cm) before planning, rating, or booking.
- **FR-015**: Sellers MUST be able to download/print the shipping label for a booked box on demand.
- **FR-016**: Sellers MUST be able to review every courier update received, including failed ones, scoped to their own shipments.

### Key Entities *(include if feature involves data)*

- **Shipment (box)**: One shippable unit of an order — status lifecycle, tracking number, courier, warehouse + delivery-address references, weight/dims, COD amount. An order has one or more boxes.
- **Shipment line**: Order-item quantity packed in a specific box; supports splitting one line across boxes.
- **Shipment event ledger**: Immutable history of every status change, who/what caused it (seller action, courier push, background sync).
- **Courier provider**: A delivery partner (aggregator like Shiprocket or a direct courier) with capability flags.
- **Seller courier configuration**: Per-seller (or platform-default) credentials plus automation flags (auto-book, rate preference).
- **Failed-delivery (NDR) round**: One open round per box describing a failed attempt and the seller's response.
- **Courier update log**: Verbatim-verified record of each courier push, used for audit and duplicate detection.
- **Physical spec catalog**: System-defined measurable attributes (weight/dims) with allowed units and conversion factors.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sellers can connect a courier account and pass the connection test in under 5 minutes.
- **SC-002**: 100% of confirmed single-warehouse orders show a correct draft within 1 minute of confirmation with no human action.
- **SC-003**: Multi-warehouse orders always produce exactly one draft per stocking warehouse with zero quantity over-allocation.
- **SC-004**: Buyers see tracking updates within 5 minutes of the courier's status change in 99% of cases.
- **SC-005**: Zero duplicate side effects (double bookings, double restocks, double completions) under update replays and concurrent actions.
- **SC-006**: Zero courier bookings ever go out with missing or guessed weight/dimensions.
- **SC-007**: Failed deliveries surface to the seller within 15 minutes with an actionable choice, reducing unacted returns week over week.

## Assumptions

- Checkout guarantees full stock availability, so the planner assumes full coverage (no backorder in v1; arrives with purchase orders).
- New/edited products without shippable specs get a warning badge rather than a hard block; the existing catalog is backfilled via manual entry at booking time.
- Shiprocket is the first courier; the design supports aggregators and direct couriers without redesign.
- The platform shared courier account exists for sellers without their own credentials.
- Existing order (confirm/cancel), inventory (reserve/release per warehouse), product (attributes), and notification modules expose narrow service interfaces for fulfillment to consume.
- Standard web-app expectations apply: authenticated sellers see only their data, buyers only their orders, every request is traceable.
- Out of scope for v1: COD remittance reconciliation (the collect amount travels with the box only), backorder/purchase orders, and stored label archives (labels are re-fetched on demand).
- Failed-delivery and escalation surfacing is in-dashboard (shipment state + webhook logs) for v1; push/email notification channels are deferred.
