# EPIC-001: Home Page Recommendation API

> **Status**: Backlog  
> **Priority**: P0  
> **Owner**: TBD  
> **Architecture**: [`plans/001-home-page-recommendation-api/architecture.md`](../architecture.md)  
> **Created**: 2026-07-13  
> **Target**: MVP by Ticket 7 (Home Page API live)

## Description

Build a **Home Page Recommendation API** for a multi-tenant SaaS e-commerce platform. Returns personalized product sections (Recently Viewed, Best Sellers, Trending, etc.) per seller with Redis caching, async event tracking, and background analytics.

## Deliverables

- [ ] Seller-configurable recommendation sections (8 section types)
- [ ] User behavior tracking (views, clicks, purchases) via RabbitMQ
- [ ] Cached home page API with <10ms P50 latency
- [ ] Background analytics aggregation (15-min intervals)
- [ ] Multi-tenant isolation (every query scoped by `seller_id`)

---

## Child Tickets

### 📦 Foundation

- [ ] [TICKET-001: Database Foundation & Module Setup](./TICKET-001-database-foundation-module-setup.md)
- [ ] [TICKET-002: Seller Configuration API](./TICKET-002-seller-configuration-api.md)

### 📊 Behavior Tracking

- [ ] [TICKET-003: User Behavior Tracking](./TICKET-003-user-behavior-tracking.md)

### 🧩 Strategy Engine

- [ ] [TICKET-004: Non-Personalized Strategies](./TICKET-004-non-personalized-strategies.md)
- [ ] [TICKET-005: Personalized Strategies](./TICKET-005-personalized-strategies.md)

### 🚀 Home Page Orchestration

- [ ] [TICKET-006: Caching Layer](./TICKET-006-caching-layer.md)
- [ ] [TICKET-007: Home Page API & Orchestrator](./TICKET-007-home-page-api-orchestrator.md)

### 📈 Background Jobs

- [ ] [TICKET-008: Analytics & Co-Purchase Jobs](./TICKET-008-analytics-copurchase-jobs.md)

### 🔧 Production Hardening

- [ ] [TICKET-009: Production Hardening](./TICKET-009-production-hardening.md)
- [ ] [TICKET-010: Documentation & Load Testing](./TICKET-010-documentation-load-testing.md)

---

## Dependencies

- Product module [`product/`](../../product/) — product query, variant query, wishlist services
- Order module [`order/`](../../order/) — order repository for best-sellers analytics
- Promotion tables [`migrations/022`](../../migrations/022_create_sale_table_and_promotion_sale_id.sql) — sale/promotion references
- Existing Redis cache [`common/cache/redis.go`](../../common/cache/redis.go)
- RabbitMQ messaging [`common/messaging/rabbitmq/`](../../common/messaging/rabbitmq/)
- Scheduler [`common/scheduler/`](../../common/scheduler/)
- Related products stored procedure [`product/query/related_products_queries.go`](../../product/query/related_products_queries.go)

## Notes

- Follow existing modular-monolith patterns from [`ARCHITECTURE.md`](../../ARCHITECTURE.md)
- All queries scoped by `seller_id`
- Cache key pattern: `rec:{seller}:{scope}:{identifier}`
- Behavior events partitioned by month for scale
