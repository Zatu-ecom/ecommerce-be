# TICKET-010: Documentation & Load Testing

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P5  
> **Estimate**: ~3 days  
> **Dependencies**: [TICKET-007](./TICKET-007-home-page-api-orchestrator.md)

## Description

Document all recommendation API endpoints and verify performance targets through load testing. Architecture performance budget in [`architecture.md §14.1`](../architecture.md:1521).

## Scope

### 1. API Documentation (Postman)

Update existing [`postman/Ecommerce API Collection.postman_collection.json`](../../postman/Ecommerce%20API%20Collection.postman_collection.json) with all recommendation endpoints:

**Home Page**:

- `GET /api/recommendation/home` — with all query param combinations

**Behavior Tracking**:

- `POST /api/recommendation/track/event` — with all event type examples

**Configuration**:

- `GET /api/recommendation/config`
- `PUT /api/recommendation/config`
- `GET /api/recommendation/config/sections`
- `PUT /api/recommendation/config/sections/:key`
- `DELETE /api/recommendation/config/sections/:key`
- `PATCH /api/recommendation/config/sections/reorder`
- `DELETE /api/recommendation/config/cache`

**Promoted Products**:

- `GET /api/recommendation/promoted`
- `POST /api/recommendation/promoted`
- `PUT /api/recommendation/promoted/:id`
- `DELETE /api/recommendation/promoted/:id`

Each endpoint documented with:

- Request/response examples
- Error responses
- Authentication requirements
- Required headers

### 2. Seller Admin Guide

Create documentation covering:

- How recommendation sections work (8 section types explained)
- How to configure sections (enable/disable, reorder, change limits)
- How to promote products to home page (hero, featured, deal)
- How to set up hero banners with scheduling
- Caching behavior explained (when changes take effect)
- Troubleshooting common issues

### 3. Load Testing

Write k6/vegeta load test scripts verifying performance targets:

| Metric                   | Target |
| ------------------------ | ------ |
| P50 latency (cached)     | <10ms  |
| P50 latency (cache miss) | <100ms |
| P99 latency              | <250ms |
| Cache hit rate target    | >90%   |

**Test Scenarios**:

- [ ] **Cached warm**: 1000 concurrent users, all caches warm → verify <10ms P50
- [ ] **Cache miss**: 100 concurrent users, caches cold → verify <100ms P50
- [ ] **Mixed load**: 500 concurrent users, 80% cache hit rate
- [ ] **Config change under load**: Update config while serving traffic
- [ ] **Redis failure simulation**: Verify degraded performance is acceptable
- [ ] **Steady state**: 10-minute sustained load at 500 req/s

### 4. Test Results Documentation

Document load test results:

- Performance against targets
- Bottlenecks identified
- Recommendations for scaling

## Acceptance Criteria

- [ ] All endpoints documented in Postman collection with examples
- [ ] Postman collection imports without errors
- [ ] Seller admin guide covers all configuration tasks
- [ ] Load test scripts committed to repo
- [ ] Performance targets verified or gaps documented
- [ ] Bottlenecks identified with remediation recommendations

## Files to Create/Modify

```
postman/Ecommerce API Collection.postman_collection.json (update)
docs/recommendation/seller-admin-guide.md
test/load/recommendation/home_page_load_test.js (k6 script)
test/load/recommendation/README.md (results documentation)
```
