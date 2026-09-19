# TICKET-002: Seller Configuration API

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P0  
> **Estimate**: ~5 days  
> **Dependencies**: [TICKET-001](./TICKET-001-database-foundation-module-setup.md)

## Description

Build the seller-facing configuration management API. Sellers can independently control which recommendation sections appear on their home page, their order, strategy parameters, and product limits. API contract in [`architecture.md §6.2`](../architecture.md:792) and [`§6.4`](../architecture.md:832).

## Scope

### 1. Configuration Service

Implement [`recommendation/service/config_service.go`](../architecture.md:197):

- `GetConfig(ctx, sellerID)` — Get seller-level config
- `UpdateConfig(ctx, sellerID, req)` — Update global settings (enable/disable, cache TTL, feature flags)
- `GetSections(ctx, sellerID)` — List all sections sorted by `sort_order`
- `UpdateSection(ctx, sellerID, sectionKey, req)` — Update a section's config
- `DeleteSection(ctx, sellerID, sectionKey)` — Remove a section
- `ReorderSections(ctx, sellerID, sectionKeys)` — Reorder sections (single transaction)
- `InvalidateCache(ctx, sellerID)` — Clear all recommendation caches for seller

**Validation Rules** (from [`architecture.md §12.2`](../architecture.md:1365)):

- At least one section must be enabled
- `sort_order` values must be unique per seller
- `max_products` must be between 1 and 50
- `strategy_key` must be a registered strategy
- `fallback_strategy_key` must not create circular dependencies
- When disabling a section, re-compute sort orders of remaining sections

### 2. Promoted Products Management

Implement [`recommendation/service/promoted_product_service.go`](../architecture.md:498):

- `ListPromoted(ctx, sellerID)` — List all promoted products
- `CreatePromoted(ctx, sellerID, req)` — Add product to promotion
- `UpdatePromoted(ctx, id, req)` — Update promotion details (banner, scheduling, sort order)
- `DeletePromoted(ctx, id)` — Remove promotion
- Scheduling validation: `start_at` must be before `end_at`

### 3. HTTP Handlers

Implement [`recommendation/handler/config_handler.go`](../architecture.md:215):

**Config Endpoints**:

- `GET /api/recommendation/config` — Get seller config
- `PUT /api/recommendation/config` — Update seller config
- `GET /api/recommendation/config/sections` — List sections
- `PUT /api/recommendation/config/sections/:key` — Update section
- `DELETE /api/recommendation/config/sections/:key` — Delete section
- `PATCH /api/recommendation/config/sections/reorder` — Reorder sections
- `DELETE /api/recommendation/config/cache` — Invalidate cache

**Promoted Product Endpoints**:

- `GET /api/recommendation/promoted` — List promoted products
- `POST /api/recommendation/promoted` — Add promoted product
- `PUT /api/recommendation/promoted/:id` — Update promoted product
- `DELETE /api/recommendation/promoted/:id` — Remove promoted product

### 4. Routes

Register all config routes in [`recommendation/route/recommendation_route.go`](../architecture.md:217) with appropriate middleware (seller auth for config, admin auth for sensitive operations).

### 5. Integration Tests

Write integration tests covering:

- [ ] Create default config on seller onboarding
- [ ] Get/update seller-level config
- [ ] CRUD for sections (create, update, delete, reorder)
- [ ] Validation errors: missing fields, invalid strategy, circular fallback, duplicate sort_order
- [ ] Tenant isolation: Seller A cannot access Seller B's config
- [ ] Cache invalidation on config change
- [ ] Promoted product CRUD with scheduling validation
- [ ] Promoted product with sale link

## Acceptance Criteria

- [ ] All endpoints return structured error responses on validation failure
- [ ] Config changes publish `config.updated` event to RabbitMQ
- [ ] Cache invalidation clears all `rec:{seller}:*` keys
- [ ] Promoted product scheduling respects `start_at`/`end_at`
- [ ] All handlers follow `BaseHandler` pattern from [`common/handler/base_handler.go`](../../common/handler/base_handler.go)
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create

```
recommendation/service/config_service.go
recommendation/service/promoted_product_service.go
recommendation/handler/config_handler.go
recommendation/route/recommendation_route.go
test/integration/recommendation/config_test.go
```
