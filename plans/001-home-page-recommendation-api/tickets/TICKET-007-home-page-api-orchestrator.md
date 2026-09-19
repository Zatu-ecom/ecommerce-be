# TICKET-007: Home Page API & Orchestrator

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P3  
> **Estimate**: ~8 days  
> **Dependencies**: [TICKET-004](./TICKET-004-non-personalized-strategies.md), [TICKET-005](./TICKET-005-personalized-strategies.md), [TICKET-006](./TICKET-006-caching-layer.md)

## Description

Build the main home page API endpoint and the orchestrator service that assembles the full response. This is the core deliverable — the endpoint that frontend calls to render the home page. Architecture in [`architecture.md §6.1`](../architecture.md:644) and [`§13.1`](../architecture.md:1395).

## Scope

### 1. Recommendation Orchestrator Service

Implement [`recommendation/service/recommendation_service.go`](../architecture.md:196):

**Flow**:

1. Check full response cache → return if hit
2. Load section config (cached) → get enabled sections sorted by `sort_order`
3. Run all strategies in **parallel goroutines** with `sync.WaitGroup` and configurable timeout (default 3s)
4. **Deduplicate products across sections**:
   - Higher-priority sections (lower `sort_order`) keep products
   - Exceptions: "Recently Viewed" and "Wishlist" are never deduplicated
5. Assemble response with section ordering
6. Populate full response cache
7. Return assembled response

**Graceful Degradation**:

- If one strategy fails, remaining sections still returned
- If Redis is down, all strategies execute from database
- If a strategy times out, skip that section

### 2. Home Page API Handler

Implement `GET /api/recommendation/home` in [`recommendation/handler/recommendation_handler.go`](../architecture.md:213):

**Headers**:

- `X-Correlation-ID` (required)
- `X-Seller-ID` (required — tenant isolation)
- `Authorization: Bearer <jwt>` (optional — enables personalized sections)

**Query Parameters**:

| Parameter         | Type   | Default       | Description                             |
| ----------------- | ------ | ------------- | --------------------------------------- |
| `sections`        | string | (all enabled) | Comma-separated section keys to include |
| `max_per_section` | int    | (from config) | Override max products per section       |
| `skip_cache`      | bool   | false         | Bypass cache for fresh results          |
| `device_type`     | string | "web"         | "web", "mobile", "tablet"               |

**Response Structure** (from [`architecture.md §6.1`](../architecture.md:665)):

```json
{
  "success": true,
  "data": {
    "sections": [
      {
        "sectionKey": "hero_products",
        "sectionName": "Featured Deals",
        "sectionType": "hero",
        "sortOrder": 1,
        "displayHint": "carousel",
        "products": [
          /* ProductResponse objects */
        ],
        "totalProducts": 5,
        "meta": {
          "strategyUsed": "hero_products",
          "personalized": false,
          "cachedAt": "2026-07-13T11:20:00Z"
        }
      }
      /* ... more sections ... */
    ],
    "totalSections": 8,
    "totalProducts": 71,
    "meta": {
      "sellerId": 2,
      "personalized": true,
      "cached": false,
      "responseTimeMs": 45
    }
  }
}
```

**Section Types & Display Hints**:

- `hero` → `carousel` (Hero Products)
- `grid` → `grid` (Best Sellers, New Arrivals, Trending, Related Products)
- `grid` → `horizontal_scroll` (Recently Viewed)
- `grid` → `grid` (Wishlist, Seller Promoted)

### 3. Route Registration

Register `GET /api/recommendation/home` in [`recommendation/route/recommendation_route.go`](../architecture.md:217):

- Middleware chain: CorrelationID → Logger → Seller Isolation → (optional) Customer Auth
- Wire module in [`common/container.go`](../../common/container.go)

### 4. Integration Tests

Full integration test suite:

- [ ] **Anonymous user**: Returns non-personalized sections only
- [ ] **Authenticated user**: Returns personalized sections (recently viewed, wishlist, related)
- [ ] **Cache hit**: Full response cache returns correctly
- [ ] **Cache miss + section cache hit**: Individual section caches used
- [ ] **Full cache miss**: All strategies execute from DB
- [ ] **Deduplication**: Product in 2 sections excluded from lower-priority one
- [ ] **Dedup exceptions**: Recently Viewed and Wishlist never deduplicated
- [ ] **Section filter**: `sections=hero_products,best_sellers` returns only those 2
- [ ] **Max per section**: `max_per_section=3` limits each section
- [ ] **Skip cache**: `skip_cache=true` returns fresh data
- [ ] **Config change**: Disable a section → it disappears from response
- [ ] **Empty data**: New seller with no products → empty sections gracefully
- [ ] **Tenant isolation**: Seller A sees only their products
- [ ] **Performance**: P50 <100ms uncached, P50 <10ms cached
- [ ] **Error cases**: Missing seller_id, disabled recommendations

## Acceptance Criteria

- [ ] All strategies execute in parallel via goroutines
- [ ] Deduplication works correctly across sections
- [ ] Response matches exact JSON structure from architecture doc
- [ ] Works for both anonymous and authenticated users
- [ ] All query parameters functional
- [ ] Response meta includes `responseTimeMs`, `cached`, `personalized`
- [ ] Performance targets met (P50 <10ms cached, <100ms uncached)
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create/Modify

```
recommendation/service/recommendation_service.go
recommendation/handler/recommendation_handler.go (home page endpoint)
recommendation/route/recommendation_route.go (home page route)
common/container.go (register recommendation module)
test/integration/recommendation/home_page_test.go
```
