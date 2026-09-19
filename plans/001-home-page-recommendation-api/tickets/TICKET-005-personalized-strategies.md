# TICKET-005: Personalized Strategies

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P2  
> **Estimate**: ~6 days  
> **Dependencies**: [TICKET-004](./TICKET-004-non-personalized-strategies.md), [TICKET-003](./TICKET-003-user-behavior-tracking.md)

## Description

Implement the 3 personalized recommendation strategies that vary per user. These require user context (JWT auth) and have fallback chains to non-personalized strategies. Architecture in [`architecture.md §8.2`](../architecture.md:980).

## Scope

### 1. Recently Viewed Strategy

[`recommendation/strategy/recently_viewed.go`](../architecture.md:982):

- Primary source: Redis ZSET `rv:{seller_id}:{user_id}` → `ZREVRANGE` for most recent
- Fallback: PostgreSQL `user_recently_viewed` table (Redis miss)
- Scoring: Recency (more recent = higher rank)
- Cold start: Returns empty section (correct — user hasn't viewed anything)
- Personalized: `true`
- Cache key includes `user_id`

### 2. Wishlist Strategy

[`recommendation/strategy/wishlist.go`](../architecture.md:1003):

- Source: Existing [`wishlist_service.go`](../../product/service/wishlist_service.go) via dependency injection
- Scoring: Most recently added first
- Personalized: `true`
- Returns empty for users with empty wishlist or anonymous users

### 3. Related Products Strategy

[`recommendation/strategy/related_products.go`](../architecture.md:1011):

- Primary source: Reuses existing [`get_related_products_scored`](../../product/query/related_products_queries.go) stored procedure
- Input: Recent purchases + recent views of the user
- Sub-strategies applied in priority order:
  1. **Category-matching**: Products in same category as user's recent views
  2. **Brand-matching**: Products of same brand as user's purchases
  3. **Bought-together**: If `product_copurchase` data exists for user's products
- Cold start: Falls back to Best Sellers Strategy
- Personalized: `true`

### 4. Integration Tests

Test each strategy independently:

- [ ] Recently Viewed: Redis ZSET path, PostgreSQL fallback, empty state
- [ ] Wishlist: Populated wishlist, empty wishlist, anonymous user
- [ ] Related Products: Category matching, brand matching, bought-together sub-strategies
- [ ] Related Products: Falls back to Best Sellers for new users
- [ ] All strategies: `IsPersonalized()` returns `true`
- [ ] All strategies: Cache key includes `user_id`
- [ ] All strategies: Tenant isolation verified

## Acceptance Criteria

- [ ] Recently Viewed reads from Redis ZSET with correct key pattern
- [ ] Wishlist reuses existing service (no duplicate queries)
- [ ] Related Products reuses existing stored procedure
- [ ] All strategies have correct fallback chains
- [ ] Empty states handled gracefully (empty section, not error)
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create

```
recommendation/strategy/recently_viewed.go
recommendation/strategy/wishlist.go
recommendation/strategy/related_products.go
test/integration/recommendation/strategies_personalized_test.go
```
