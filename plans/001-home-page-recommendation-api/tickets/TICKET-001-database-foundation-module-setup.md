# TICKET-001: Database Foundation & Module Setup

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P0  
> **Estimate**: ~10 days  
> **Dependencies**: None

## Description

Create the complete database foundation and module scaffolding for the recommendation engine. This includes all 7 tables, GORM entities, repositories, factory wiring, constants, errors, and the module container — everything needed before any feature work begins.

## Scope

### 1. Database Migration

Create [`migrations/023_create_recommendation_tables.sql`](../architecture.md:278) with all 7 tables:

| Table                          | Key Purpose                                                                |
| ------------------------------ | -------------------------------------------------------------------------- |
| `seller_recommendation_config` | Per-seller global config (enable/disable, cache TTL, feature flags)        |
| `section_config`               | Per-section config per seller (8 sections, sort order, strategy, fallback) |
| `behavior_event`               | **Partitioned by month** — append-only user behavior tracking              |
| `product_analytics`            | Pre-aggregated product metrics (views, purchases, trending score)          |
| `product_copurchase`           | Frequently bought together pairs with confidence/lift/support              |
| `promoted_product`             | Seller-curated promoted products (hero, featured, deal, banner)            |
| `user_recently_viewed`         | Durability backup for Redis recently-viewed data                           |

- `behavior_event` must use `PARTITION BY RANGE (event_time)` with monthly partitions
- All foreign keys reference existing tables (`product`, `sale`)
- All indexes created per schema
- Rollback script included

### 2. Seed Migration

Create seed migration that inserts default `seller_recommendation_config` + 8 `section_config` rows for **every existing seller**. Default sections per [`architecture.md §7.2`](../architecture.md:858):

| Order | Section Key        | Display Name      | Max Products | Personalized |
| ----- | ------------------ | ----------------- | ------------ | ------------ |
| 1     | `hero_products`    | Featured Deals    | 5            | No           |
| 2     | `recently_viewed`  | Recently Viewed   | 10           | Yes          |
| 3     | `seller_promoted`  | Featured Products | 8            | No           |
| 4     | `best_sellers`     | Best Sellers      | 10           | No           |
| 5     | `new_arrivals`     | New Arrivals      | 10           | No           |
| 6     | `trending`         | Trending Now      | 10           | No           |
| 7     | `wishlist`         | Your Wishlist     | 10           | Yes          |
| 8     | `related_products` | Related Products  | 10           | Yes          |

### 3. GORM Entities

Create [`recommendation/entity/`](../architecture.md:174) with 6 files:

- `recommendation_config.go` — `SellerRecommendationConfig`
- `section_config.go` — `SectionConfig`
- `behavior_event.go` — `BehaviorEvent`
- `product_analytics.go` — `ProductAnalytics`
- `product_copurchase.go` — `ProductCopurchase`
- `promoted_product.go` — `PromotedProduct`

All entities use `BaseEntity` from [`common/db/base_entity.go`](../../common/db/base_entity.go). JSON tags follow snake_case convention.

### 4. Constants & Errors

Create [`recommendation/constants/`](../architecture.md:234):

- `cache_keys.go` — Cache key patterns and TTL constants
- `strategy_names.go` — Strategy name constants
- `event_types.go` — Behavior event type constants
- `config_defaults.go` — Default section configuration

Create [`recommendation/error/recommendation_error.go`](../architecture.md:240):

- `ErrSectionNotFound`, `ErrInvalidStrategy`, `ErrCircularFallback`, `ErrInvalidSectionKey`, `ErrConfigNotFound`, `ErrPromotedProductNotFound`

### 5. Repositories

Create [`recommendation/repository/`](../architecture.md:188):

- `recommendation_config_repository.go` — CRUD for seller config + sections (all scoped by `seller_id`)
- `promoted_product_repository.go` — CRUD for promoted products
- `behavior_repository.go` — Batch insert into partitioned `behavior_event` table
- `analytics_repository.go` — Query/UPSERT `product_analytics`
- `copurchase_repository.go` — Query/UPSERT `product_copurchase`

### 6. Factory & Container

Create [`recommendation/factory/singleton/`](../architecture.md:227) following exact patterns from [`product/factory/singleton/`](../../product/factory/singleton/):

- `singleton_factory.go`, `repository_factory.go`, `service_factory.go`, `handler_factory.go`

Create [`recommendation/container.go`](../architecture.md:172) — module registration & wiring.

### 7. Model DTOs

Create [`recommendation/model/`](../architecture.md:182):

- `recommendation_model.go` — Home page request/response DTOs
- `section_response_model.go` — Section-level response structures
- `config_model.go` — Config CRUD DTOs
- `strategy_model.go` — Strategy input/output models

## Acceptance Criteria

- [ ] Migration runs successfully on clean database
- [ ] Seed inserts config for all existing sellers (idempotent)
- [ ] All entities compile without errors
- [ ] All repositories follow tenant isolation (`WHERE seller_id = ?`)
- [ ] Factory/container follows existing module pattern exactly
- [ ] Module registers in `common/container.go` without errors
- [ ] Integration test: module container initializes correctly with test containers

## Files to Create

```
migrations/023_create_recommendation_tables.sql
migrations/024_rollback_recommendation_tables.sql
migrations/seeds/recommendation/001_seed_default_recommendation_config.sql
recommendation/entity/recommendation_config.go
recommendation/entity/section_config.go
recommendation/entity/behavior_event.go
recommendation/entity/product_analytics.go
recommendation/entity/product_copurchase.go
recommendation/entity/promoted_product.go
recommendation/constants/cache_keys.go
recommendation/constants/strategy_names.go
recommendation/constants/event_types.go
recommendation/constants/config_defaults.go
recommendation/error/recommendation_error.go
recommendation/model/recommendation_model.go
recommendation/model/section_response_model.go
recommendation/model/config_model.go
recommendation/model/strategy_model.go
recommendation/repository/recommendation_config_repository.go
recommendation/repository/promoted_product_repository.go
recommendation/repository/behavior_repository.go
recommendation/repository/analytics_repository.go
recommendation/repository/copurchase_repository.go
recommendation/factory/singleton/singleton_factory.go
recommendation/factory/singleton/repository_factory.go
recommendation/factory/singleton/service_factory.go
recommendation/factory/singleton/handler_factory.go
recommendation/container.go
```
