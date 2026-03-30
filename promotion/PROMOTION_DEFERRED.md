# Promotion Engine — Deferred Items

Items intentionally deferred from the current phase. They should be picked up in subsequent phases.

---

## Phase: Order Module Integration

### Usage Recording on Order Completion
- **Entity**: `PromotionUsage` (exists in `entity/usage.go`)
- **What's needed**: After order placement, create a `PromotionUsage` record for each applied promotion and call `IncrementUsageAtomically` in the same DB transaction.
- **Repo method ready**: `IncrementUsageAtomically(ctx, promotionID, usageLimit)` — already implemented.
- **Why deferred**: Requires order module changes; usage recording belongs in the order flow.

---

## Phase: Customer Segments

### Customer Segment Matching
- **Entity**: `CustomerSegment` (exists in `entity/customer_segment.go`)
- **What's needed**: Implement a `CustomerSegmentService` that evaluates segment rules against customer data. Wire it into `isCustomerEligible` in `promotion_service_apply.go`.
- **Current state**: Stub returns `false` — promotions targeting `specific_segment` will always be skipped.

---

## Phase: Discount Code / Coupon

### Discount Code CRUD
- **Entities**: `DiscountCode` (`entity/discount_code.go`), `DiscountCodeScope` (`entity/discount_code_scope.go`), `DiscountCodeUsage` (`entity/usage.go`)
- **What's needed**: Full handler, service, repository, route, and factory code.
- **Tracking**: Will be documented in a separate `DISCOUNT_CODE_TODO.md`.

---

## Phase: Scheduler

### Auto-Start / Auto-End Promotions
- **Entity fields**: `AutoStart`, `AutoEnd` in `promotion.go`
- **What's needed**: A background scheduler/cron job that:
  - Transitions `scheduled → active` when `StartsAt` arrives and `AutoStart == true`
  - Transitions `active → ended` when `EndsAt` passes and `AutoEnd == true`
- **Current state**: These transitions must be done manually via the `UpdateStatus` API.
