# TICKET-004: Non-Personalized Strategies

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P2  
> **Estimate**: ~6 days  
> **Dependencies**: [TICKET-001](./TICKET-001-database-foundation-module-setup.md)

## Description

Implement the strategy interface, registry, and all 5 non-personalized recommendation strategies. These sections work without user context and serve as fallbacks for personalized strategies. Architecture in [`architecture.md §8`](../architecture.md:927).

## Scope

### 1. Strategy Interface & Registry

Create [`recommendation/strategy/strategy.go`](../architecture.md:929):

- `RecommendationStrategy` interface: `Name()`, `Generate(ctx, input)`, `CacheKey(input)`, `IsPersonalized()`
- `StrategyInput` struct: SellerID, UserID, Config, Limit, ExcludeIDs, UserContext
- `StrategyOutput` struct: Products, StrategyUsed, FallbackUsed
- `ProductRecommendation` struct: ProductID, VariantID, Score, Reason, PromotionBadge, BannerImageURL

Create [`recommendation/strategy/registry.go`](../architecture.md:1057):

- `RegisterStrategy(s)` — adds to thread-safe registry map
- `GetStrategy(name)` — lookup by name
- `GetAllStrategies()` — returns all registered
- `init()`-based auto-discovery pattern

### 2. New Arrivals Strategy

[`recommendation/strategy/new_arrivals.go`](../architecture.md:1041):

- Source: `product` table, `ORDER BY created_at DESC`
- No personalization, no fallback (always returns products)
- Simplest strategy — good reference implementation

### 3. Best Sellers Strategy

[`recommendation/strategy/best_sellers.go`](../architecture.md:991):

- Source: `product_analytics` table, ordered by weighted score
- Scoring: `total_purchases*0.5 + conversion_rate*0.3 + views_last_7d*0.2`
- Fallback: New Arrivals (when no analytics data exists)

### 4. Trending Strategy

[`recommendation/strategy/trending.go`](../architecture.md:1048):

- Source: `product_analytics` table, ordered by `trending_score DESC`
- Formula: `views_24h*0.4 + purchases_24h*0.4 + wishlist_7d*0.1 + cart_7d*0.1`
- Fallback: New Arrivals

### 5. Hero Products Strategy

[`recommendation/strategy/hero_products.go`](../architecture.md:1025):

- Source: `promoted_product WHERE promotion_type='hero' AND is_active=true`
- Scheduling: `start_at <= NOW() <= end_at` (NULL = no restriction)
- Scoring: `sort_order` (seller-configured)
- Products include banner image URL and call-to-action metadata
- No fallback (returns empty if no hero products configured)

### 6. Seller Promoted Strategy

[`recommendation/strategy/seller_promoted.go`](../architecture.md:1033):

- Source: `promoted_product WHERE promotion_type IN ('featured','deal') AND is_active=true`
- Same scheduling logic as Hero Products
- Scoring: `sort_order`
- Attaches promotion badge when `sale_id` is set

### 7. Integration Tests

Test each strategy independently:

- [ ] New Arrivals: Products in `created_at DESC` order, respects Limit/ExcludeIDs
- [ ] Best Sellers: Weighted scoring, falls back to New Arrivals when empty
- [ ] Trending: Score formula correct, falls back to New Arrivals
- [ ] Hero Products: Active + in-schedule only, expired excluded, empty when none
- [ ] Seller Promoted: Both `featured` and `deal` types, badge on sale link
- [ ] All strategies: Tenant isolation verified
- [ ] All strategies: `IsPersonalized()` returns `false`

## Acceptance Criteria

- [ ] Strategy interface is clean and minimal (4 methods)
- [ ] Registry is thread-safe (`sync.RWMutex`)
- [ ] All strategies auto-register via `init()`
- [ ] Each strategy independently testable
- [ ] Fallback chains verified
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create

```
recommendation/strategy/strategy.go
recommendation/strategy/registry.go
recommendation/strategy/new_arrivals.go
recommendation/strategy/best_sellers.go
recommendation/strategy/trending.go
recommendation/strategy/hero_products.go
recommendation/strategy/seller_promoted.go
test/integration/recommendation/strategies_non_personalized_test.go
```
