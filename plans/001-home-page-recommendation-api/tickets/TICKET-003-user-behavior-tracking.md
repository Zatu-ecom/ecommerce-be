# TICKET-003: User Behavior Tracking

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P1  
> **Estimate**: ~5 days  
> **Dependencies**: [TICKET-001](./TICKET-001-database-foundation-module-setup.md)

## Description

Build the complete user behavior tracking pipeline — API endpoint, RabbitMQ publishing, consumer processing, and Redis recently-viewed management. Architecture in [`architecture.md §9`](../architecture.md:1085).

## Scope

### 1. Track Event API Endpoint

Implement `POST /api/recommendation/track/event` in [`recommendation/handler/recommendation_handler.go`](../architecture.md:213). API contract in [`architecture.md §6.3`](../architecture.md:803):

- Accepts up to 20 events per request (batched)
- Validates each event: `eventType` required (valid enum), `productId` required
- Publishes events to RabbitMQ exchange `ecommerce.events` with routing key `behavior.<event_type>`
- Returns `202 Accepted` immediately — never blocks on DB writes
- Invalid events rejected with `400` and specific error message
- Anonymous users (no JWT) still accepted

**Event Types** (from [`architecture.md §9.2`](../architecture.md:1115)):

| Event Type        | Trigger                        | Data Captured                                       |
| ----------------- | ------------------------------ | --------------------------------------------------- |
| `product_view`    | Product detail page viewed     | `duration_ms`, `scroll_depth_pct`, `source_section` |
| `product_click`   | Product clicked from a section | `source_section`, `position_in_section`             |
| `search`          | Search performed               | `query`, `results_count`, `clicked_product_id`      |
| `add_to_cart`     | Product added to cart          | `quantity`, `variant_id`                            |
| `purchase`        | Order placed                   | `order_id`, `quantity`, `unit_price`                |
| `wishlist_add`    | Product added to wishlist      | `wishlist_id`                                       |
| `wishlist_remove` | Product removed from wishlist  | `wishlist_id`                                       |

### 2. Behavior Event Consumer

Implement [`recommendation/consumer/behavior_event_consumer.go`](../architecture.md:220):

- Consumes from queues: `recommendation.behavior.product_view`, `recommendation.behavior.product_click`, `recommendation.behavior.add_to_cart`, `recommendation.behavior.purchase`, `recommendation.behavior.wishlist_add`
- For `product_view` events:
  - `ZADD rv:{seller}:{user} {product_id} {timestamp}` in Redis
  - `ZREMRANGEBYRANK rv:{seller}:{user} 0 -51` (keep last 50)
- For `purchase` events:
  - `INCR purchase_count:{seller}:{product_id}:{date}` in Redis
- All events: batch insert into `behavior_event` table
- ACK only after both Redis and DB operations succeed
- NACK on transient failures (requeue)
- Dead-letter queue for persistent failures

### 3. Client-Side Tracking Snippet

Provide JavaScript tracking snippet (from [`architecture.md §9.3`](../architecture.md:1127)):

- Batches events client-side (up to 20)
- Auto-flushes every 5 seconds
- Flushes on page unload (`beforeunload`)

### 4. Integration Tests

End-to-end tests covering:

- [ ] Send single event via API → verify it appears in RabbitMQ
- [ ] Send batch of 20 events → verify all processed
- [ ] `product_view` → verify Redis ZSET contains product with correct timestamp
- [ ] Multiple views of same product → ZSET updated (not duplicated)
- [ ] More than 50 views → ZSET trimmed to 50
- [ ] `purchase` → verify Redis counter incremented
- [ ] All event types → verify rows in `behavior_event` table
- [ ] Anonymous events (no user_id) → accepted and stored
- [ ] Invalid event type → 400 error
- [ ] Missing product_id → 400 error
- [ ] Consumer handles malformed message (move to DLQ)
- [ ] Tenant isolation: Seller A's events not visible to Seller B

## Acceptance Criteria

- [ ] Track endpoint returns 202, never blocks on DB writes
- [ ] Events flow end-to-end: HTTP → RabbitMQ → Consumer → Redis + PostgreSQL
- [ ] Redis ZSET correctly maintained (add, trim to 50)
- [ ] Consumer handles all 7 event types
- [ ] Failed messages NACK'd and requeued
- [ ] Integration tests use `testify/suite` + test containers (PostgreSQL + Redis + RabbitMQ)

## Files to Create

```
recommendation/handler/recommendation_handler.go (track endpoint)
recommendation/consumer/behavior_event_consumer.go
test/integration/recommendation/tracking_test.go
```
