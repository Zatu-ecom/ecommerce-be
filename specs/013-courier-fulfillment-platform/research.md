# Research: Courier Fulfillment Platform (013)

All unknowns below were resolved during the design reviews in `data-model.md`, `service-design.md`, `api-contracts.md`, and `physical-specs.md`. No NEEDS CLARIFICATION remains. Each entry records Decision / Rationale / Alternatives considered.

## R1. Provider abstraction shape

- **Decision**: Single `CourierPartner` Go interface (core ops + credential lifecycle), plus optional capability interfaces (`PickupScheduler`, `NDRHandler`, `ReturnHandler`, `RTORequester`) discovered by type-assertion. Orchestrators never branch on provider codes; provider specifics confined to one folder per courier.
- **Rationale**: Mirrors the proven `010-payment-gateway-platform` (`PaymentGateway` + `RefundStatusFetcher`). New courier = new folder + seed rows + one factory line; zero orchestrator, migration, or contract changes.
- **Alternatives considered**: Per-provider service switches (rejected: orchestrator grows with every courier); fat single interface with `Unsupported` returns (rejected: forces meaningless implementations; optional interfaces express capability honestly).

## R2. First adapter

- **Decision**: Shiprocket (aggregator) first; Delhivery/direct couriers later through the same contract.
- **Rationale**: One integration covers many couriers; Shiprocket API surface (create/assign-AWB/pickup/track/NDR/return/label/webhook) maps 1:1 onto the contract. Empty `fulfillment/shiprocket/` stub already reserved the home.
- **Alternatives considered**: Direct-courier first (rejected: one integration covers one network; aggregator defers the choice to runtime rate shopping).

## R3. Booking granularity

- **Decision**: One `BookShipment` method (create + AWB + optional pickup inside the adapter), not separate create/assign/pickup calls on the contract.
- **Rationale**: Couriers differ (single-call AWB vs multi-step); exposing steps leaks the provider's shape. Optional `PickupScheduler` covers couriers where pickup is genuinely separate.
- **Alternatives considered**: Step-by-step contract (rejected: forces single-API couriers to invent empty steps; more distributed-transaction surface).

## R4. Credential ownership

- **Decision**: Hybrid — platform-default config (`seller_id IS NULL`) + per-seller override, resolved once at book time and frozen on the shipment (`provider_config_id`).
- **Rationale**: New sellers ship on day one; serious sellers bring their own rates/keys. Freezing the config id makes webhooks/tracking immune to later credential rotation.
- **Alternatives considered**: Platform-only (rejected: blocks sellers with negotiated rates); seller-only (rejected: onboarding friction, no day-one shipping).

## R5. Status vocabulary

- **Decision**: One vocabulary — shipment `status` = `ShipmentAction` (+`ignore`) = ledger `event_type`; transitions governed by an explicit allow-list, not a rank ladder; terminal states never regress.
- **Rationale**: Couriers skip scans and revisit states (OFD→in-transit); a rank-based check would reject legal moves. One list removes an entire class of mapping drift.
- **Alternatives considered**: Three parallel enums (rejected: drift between status/action/event); rank-precedence guard (rejected: false rejections on real courier behavior).

## R6. Webhook + reconciler duality

- **Decision**: Courier push is primary (real-time NDR/tracking), cron reconciler is the safety net; both funnel into one shared `Apply`. Unknown AWB → 200/persist-nothing (couriers retry 4xx until disabled); bad signature → 401/persist-nothing.
- **Rationale**: 20–50 events per shipment over days make pure polling rate-limit-expensive and NDR-late; shared apply path makes the two channels unable to diverge. Direct copy of the payment webhook pattern.
- **Alternatives considered**: Polling-only (rejected: volume × latency × NDR action window); webhook-only (rejected: missed pushes during downtime would strand boxes).

## R7. Shipment location modeling

- **Decision**: Shipments carry `pickup_location_id` + `delivery_address_id` as plain ids (no cross-module FKs, resolved via service hooks); no pincode columns; pickup nickname derived at runtime as `S{seller}L{location}` (no stored alias, no mapping table).
- **Rationale**: Module-boundary safe (microservice-extractable), single source of truth in owning modules, zero per-shipment manual addressing. Pincodes resolve from ids at call time.
- **Alternatives considered**: Stored pincodes (rejected: duplicate source of truth); stored alias + mapping table (rejected then removed: manual overhead; derivation is total); cross-module FKs (rejected: extraction-breaking).

## R8. Planning trigger and automation depth

- **Decision**: Planner runs on the committed `pending → confirmed` transition (never at order placement); auto-creates one draft per warehouse (greedy priority allocation + same-pass reservation); booking stays seller-confirmed unless the tenant `auto_book` flag is on.
- **Rationale**: Pending orders may fail payment — planning earlier wastes reservations. Separating free planning from money-spending booking gives a free cancel window and keeps auto-book opt-in.
- **Alternatives considered**: Plan at placement (rejected: dead-order drafts); full-auto for everyone (rejected: books freight without review; needs proven planner first).

## R9. Stock disagreement policy

- **Decision**: No backorder in v1 (arrives with purchase orders). Planner assumes checkout-guaranteed full coverage; shortfall raises `STOCK_MISMATCH` (exception + alert); missing weight degrades to manual PATCH; corrupt data is never guessed.
- **Rationale**: Checkout already reserves; building placeholder backorder states now would force a painful migration later. Loud exceptions centralize tomorrow's backorder entry points.
- **Alternatives considered**: Partial-plan + remainder state now (rejected: premature backorder model without PO semantics).

## R10. Physical specs sourcing

- **Decision**: Backend-owned unit-variant definitions (`weight_g`/`weight_kg`…), seller picks definition (= unit choice, no new columns), one API serves the grouped catalog, fulfillment normalizes to g/cm on read, at most one key per family per product.
- **Rationale**: Free-text specs are unparseable; fixed single units punish sellers; per-product unit columns add schema for what definition choice already expresses.
- **Alternatives considered**: Free-text attributes (rejected: unmachineable); fixed canonical units (rejected: seller friction); unit column on `product_attribute` (rejected: unnecessary once choice = definition).
