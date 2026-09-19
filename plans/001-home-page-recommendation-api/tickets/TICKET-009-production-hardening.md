# TICKET-009: Production Hardening

> **Epic**: [EPIC-001](./EPIC-001-home-page-recommendation-api.md)  
> **Priority**: P5  
> **Estimate**: ~4 days  
> **Dependencies**: [TICKET-007](./TICKET-007-home-page-api-orchestrator.md)

## Description

Harden the recommendation module for production: rate limiting, Redis circuit breaker, and Prometheus monitoring. Architecture in [`architecture.md §14`](../architecture.md:1519) and [`§15`](../architecture.md:1578).

## Scope

### 1. Rate Limiting

Add per-seller rate limiting on `GET /api/recommendation/home`:

- Default: 100 requests/second per seller
- Uses Redis-based sliding window rate limiter
- Returns `429 Too Many Requests` with `Retry-After` header when exceeded
- Configurable limit per seller via `seller_recommendation_config`

### 2. Redis Circuit Breaker

Implement circuit breaker for Redis dependency:

- Monitor Redis connection health
- Circuit opens after N consecutive failures (configurable, default 5)
- When open: skip all cache reads, execute strategies directly from database
- Circuit half-opens after cooldown period (configurable, default 30s)
- Circuit closes after successful health check
- Degraded mode and recovery both logged with severity

### 3. Prometheus Monitoring

Add metrics endpoint exposing:

| Metric                                 | Type      | Labels                                    |
| -------------------------------------- | --------- | ----------------------------------------- |
| `recommendation_requests_total`        | Counter   | `seller_id`, `personalized`, `cached`     |
| `recommendation_request_duration_ms`   | Histogram | `seller_id`                               |
| `recommendation_cache_hit_total`       | Counter   | `seller_id`, `cache_level`, `section_key` |
| `recommendation_cache_miss_total`      | Counter   | `seller_id`, `cache_level`, `section_key` |
| `recommendation_strategy_latency_ms`   | Histogram | `strategy_name`, `fallback_used`          |
| `recommendation_section_product_count` | Gauge     | `seller_id`, `section_key`                |
| `recommendation_errors_total`          | Counter   | `error_type`, `seller_id`                 |

### 4. Alert Rules

Define alert thresholds:

- Cache hit rate below 80% for 5 minutes → Warning
- P99 latency above 250ms for 5 minutes → Warning
- Error rate above 1% for 5 minutes → Critical
- Redis circuit breaker open → Critical

### 5. Integration Tests

- [ ] Rate limiting: Exceed limit → verify 429 with Retry-After
- [ ] Rate limiting: Different sellers have independent limits
- [ ] Circuit breaker: Stop Redis → verify DB-only fallback works
- [ ] Circuit breaker: Restart Redis → verify recovery
- [ ] Metrics: Verify counters increment correctly under load

## Acceptance Criteria

- [ ] Rate limiter uses Redis (shared across instances)
- [ ] Circuit breaker prevents cascading failures
- [ ] All metrics exposed via `/metrics` endpoint
- [ ] Alert rules documented
- [ ] Integration tests use `testify/suite` + test containers

## Files to Create/Modify

```
recommendation/middleware/rate_limiter.go
recommendation/service/circuit_breaker.go
recommendation/metrics/metrics.go
test/integration/recommendation/hardening_test.go
```
