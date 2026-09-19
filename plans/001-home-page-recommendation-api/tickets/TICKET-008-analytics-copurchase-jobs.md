# TICKET-008: Analytics & Co-Purchase Jobs

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P4  
> **Estimate**: ~6 days  
> **Dependencies**: [TICKET-001](./TICKET-001-database-foundation-module-setup.md), [TICKET-003](./TICKET-003-user-behavior-tracking.md)

## Description

Implement the background scheduled jobs that power the data-driven strategies. The analytics aggregation job computes product metrics from raw behavior events. The co-purchase job computes "frequently bought together" pairs from order data. Architecture in [`architecture.md §9.4`](../architecture.md:1160) and [`§5.1.5`](../architecture.md:468).

## Scope

### 1. Analytics Aggregation Job

Implement [`recommendation/job/analytics_aggregate_job.go`](../architecture.md:224):

- Registered with existing [`common/scheduler/`](../../common/scheduler/) infrastructure
- Runs every 15 minutes
- **Logic**:
  1. Query `behavior_event` for events since last aggregation timestamp
  2. Group by `product_id`, `seller_id`
  3. Compute metrics:
     - `total_views`, `views_last_7d`, `views_last_24h`
     - `total_purchases`, `purchases_last_7d`, `purchases_last_24h`
     - `total_wishlist_adds`, `wishlist_adds_last_7d`
     - `total_cart_adds`, `cart_adds_last_7d`
  4. Compute scores:
     - `trending_score = views_24h*0.4 + purchases_24h*0.4 + wishlist_7d*0.1 + cart_7d*0.1`
     - `popularity_score` (weighted long-term engagement)
     - `conversion_rate = purchases / views`
  5. UPSERT into `product_analytics` table
  6. Invalidate affected cache keys:
     - `rec:{seller}:section:best_sellers:*`
     - `rec:{seller}:section:trending:*`
     - `rec:{seller}:response:home:*`

### 2. Analytics Service

Implement [`recommendation/service/analytics_service.go`](../architecture.md:198):

- `Aggregate(ctx, sellerID)` — main aggregation logic
- `ComputeTrendingScore(analytics)` — score formula
- `ComputePopularityScore(analytics)` — popularity formula
- `ComputeConversionRate(analytics)` — conversion rate

### 3. Co-Purchase Computation Job

Implement [`recommendation/job/copurchase_computation_job.go`](../architecture.md:468):

- Registered with existing [`common/scheduler/`](../../common/scheduler/) infrastructure
- Runs daily during low-traffic hours
- **Logic**:
  1. Query `order_item` table for products purchased together in same order
  2. For each pair `(product_id, coproduct_id)`:
     - Count co-occurrences
     - Compute `confidence = P(coproduct | product)`
     - Compute `lift = confidence / P(coproduct)`
     - Compute `support = P(product AND coproduct)`
     - Rank pairs per product by co-occurrence count
  3. UPSERT into `product_copurchase` table
  4. Invalidate related cache keys

### 4. Integration Tests

- [ ] Analytics job: Seed behavior events → run job → verify `product_analytics` populated
- [ ] Analytics job: Trending score formula verified
- [ ] Analytics job: Only processes events since last aggregation (idempotent)
- [ ] Analytics job: Cache invalidation triggered
- [ ] Co-purchase job: Seed orders → run job → verify pairs with correct confidence/lift
- [ ] Co-purchase job: Tenant isolation (no cross-seller pairs)
- [ ] Both jobs: Idempotent (re-running produces same result)

## Acceptance Criteria

- [ ] Analytics job runs on schedule (every 15 minutes)
- [ ] All metrics correctly computed from raw events
- [ ] Trending score formula matches spec exactly
- [ ] Co-purchase confidence, lift, and support correctly computed
- [ ] Both jobs scoped per seller (no cross-tenant data leakage)
- [ ] Cache invalidation triggered after each job
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create

```
recommendation/job/analytics_aggregate_job.go
recommendation/job/copurchase_computation_job.go
recommendation/service/analytics_service.go
test/integration/recommendation/jobs_test.go
```
