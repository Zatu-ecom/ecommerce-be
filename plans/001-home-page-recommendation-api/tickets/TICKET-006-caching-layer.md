# TICKET-006: Caching Layer

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P3  
> **Estimate**: ~4 days  
> **Dependencies**: [TICKET-001](./TICKET-001-database-foundation-module-setup.md)

## Description

Implement the multi-level Redis caching layer for the recommendation module. This is the performance backbone — the home page API must serve <10ms P50 for cached requests. Architecture in [`architecture.md §10`](../architecture.md:1172).

## Scope

### 1. Cache Service

Implement [`recommendation/service/cache_service.go`](../architecture.md:199):

**Cache Operations**:

- `GetSection(ctx, sellerID, sectionKey, limit, userID)` — Get cached section products
- `SetSection(ctx, sellerID, sectionKey, limit, userID, output)` — Cache section products
- `GetFullResponse(ctx, sellerID, userID)` — Get cached full home page response
- `SetFullResponse(ctx, sellerID, userID, response)` — Cache full response
- `InvalidateSellerCache(ctx, sellerID)` — Delete all `rec:{seller}:*` keys
- `InvalidateSectionCache(ctx, sellerID, sectionKey)` — Delete section-specific keys
- `InvalidateUserCache(ctx, sellerID, userID)` — Delete user-specific keys

**Cache Key Pattern** (from [`architecture.md §10.1`](../architecture.md:1174)):

```
rec:{seller}:{scope}:{identifier}[:{variant}]

Examples:
rec:2:section:best_sellers:10
rec:2:section:new_arrivals:10
rec:2:user:789:recently_viewed:10
rec:2:response:home:anon
rec:2:config:sections
```

**TTL Strategy** (from [`architecture.md §10.2`](../architecture.md:1190)):

| Cache Level                      | TTL            | Rationale                           |
| -------------------------------- | -------------- | ----------------------------------- |
| Full home page (anonymous)       | 60s            | Fast-changing; acceptable staleness |
| Full home page (authenticated)   | 30s            | Personalization means shorter TTL   |
| Section-level (non-personalized) | 120s           | Sections change slowly              |
| Product analytics                | 300s           | Analytics change slowly             |
| Configuration                    | 600s           | Config rarely changes               |
| User recently viewed (ZSET)      | 7 days sliding | Keep last 7 days of views           |
| Co-purchase                      | 3600s          | Relationships change slowly         |

### 2. Cache Invalidation Consumer

Implement [`recommendation/consumer/cache_invalidation_consumer.go`](../architecture.md:1325):

Listen to events from other modules and invalidate affected caches:

| Trigger Event              | Cache Keys to Invalidate                                                          |
| -------------------------- | --------------------------------------------------------------------------------- |
| `product.created`          | `rec:{seller}:section:new_arrivals`                                               |
| `product.updated`          | `rec:{seller}:section:new_arrivals`, `rec:{seller}:section:related_products`      |
| `product.deleted`          | All section caches for seller                                                     |
| `order.placed`             | `rec:{seller}:section:best_sellers`, `rec:{seller}:section:trending`, user caches |
| `config.updated`           | `rec:{seller}:config:*`, `rec:{seller}:response:*`                                |
| `promoted_product.changed` | `rec:{seller}:section:hero_products`, `rec:{seller}:section:seller_promoted`      |

### 3. Cache Warming Job

Implement [`recommendation/job/cache_warm_job.go`](../architecture.md:225):

- Runs at startup and on schedule (e.g., every hour)
- Identifies active sellers (recent orders/views)
- Pre-computes anonymous home page response
- Pre-computes non-personalized section caches
- Processes sellers sequentially to avoid DB overload

### 4. Integration Tests

- [ ] Cache hit: Section cache returns correctly
- [ ] Cache miss: Service falls through to strategy execution
- [ ] Full response cache: Anonymous and authenticated paths
- [ ] Cache invalidation: Publish event → verify cache cleared
- [ ] Cache warming: Job populates caches for active sellers
- [ ] TTL: Cached data expires correctly
- [ ] Redis failure: Graceful degradation (log, don't crash)

## Acceptance Criteria

- [ ] All operations use existing [`common/cache/redis.go`](../../common/cache/redis.go) client
- [ ] Cache keys follow pattern `rec:{seller}:{scope}:{identifier}`
- [ ] TTL values match strategy table
- [ ] Errors from Redis are logged, not propagated (graceful degradation)
- [ ] Cache invalidation consumer handles all 6 trigger events
- [ ] Cache warming job does not overload database
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create

```
recommendation/service/cache_service.go
recommendation/consumer/cache_invalidation_consumer.go
recommendation/job/cache_warm_job.go
test/integration/recommendation/cache_test.go
```
