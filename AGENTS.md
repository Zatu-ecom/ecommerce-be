# ecommerce-be Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-09-18

## Active Technologies
- Go 1.25+ (per constitution) (003-upload-apis)
- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers, existing File module services (005-product-file-integration)
- PostgreSQL 16 for `product_media` association rows; external blob providers remain owned by File module (005-product-file-integration)
- Go 1.25+ + Gin (HTTP), GORM (ORM), testify/suite (testing), Testcontainers (integration tests) (006-recently-viewed-products)
- PostgreSQL 16 — new `user_recently_viewed` table (006-recently-viewed-products)
- Go 1.25+ + Gin (HTTP), GORM (ORM), validator, testify/suite, Testcontainers (007-coupon-discount-codes)
- PostgreSQL 16 — extend `discount_code` / scope / usage tables; create `cart_applied_coupon`, `cart_item_promotion`; use existing `order_applied_coupon` (007-coupon-discount-codes)
- PostgreSQL 16 — rename `product_variant.price` / `package_option.price` (`DOUBLE PRECISION`) → `price_cents BIGINT NOT NULL`; keep existing `*_cents` columns in `order`/`order_item`/promo; no new money tables (008-money-currency-standardization)
- Go 1.25+ + Gin (HTTP), GORM (ORM), `go-playground/validator/v10` (validation), (009-razorpay-payment-gateway)
- PostgreSQL 16 (new tables + migration `031`; GORM `SingularTable: true`); Redis 7 is (009-razorpay-payment-gateway)
- Go 1.25+ + Gin, GORM, `go-playground/validator/v10`, testify/suite + Testcontainers, stdlib `net/http` + `crypto/hmac`/`sha256` (no Razorpay SDK), `github.com/robfig/cron/v3` via `common/cron` (010-payment-gateway-platform)
- PostgreSQL 16 (`SingularTable: true`); Redis unused by this feature (010-payment-gateway-platform)
- Go 1.25+ (module `ecommerce-be`) + `github.com/redis/go-redis/v9` (new, replaces v8), `golang.org/x/sync` (singleflight — already indirect dep), Gin, GORM, testify/suite, Testcontainers (012-caching-infrastructure)
- PostgreSQL 16 (source of truth, unchanged schema — no migrations in this feature); volatile cache role + durable KV role (Redis 7 now, DragonflyDB at cutover; RESP-compatible) (012-caching-infrastructure)

- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`), AWS SDK v2 (`service/s3`), GCS storage client, Azure Blob SDK (002-blob-adapters)
- PostgreSQL 16 for `storage_config`/`storage_provider` source records; external blob storage providers (S3-compatible, GCS, Azure Blob) for object data (002-blob-adapters)

- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`) (001-activate-storage-config)

## Project Structure

## Commands

# Add commands for Go 1.25+

## Code Style

Go 1.25+: Follow standard conventions

## Recent Changes
- 012-caching-infrastructure: Added Go 1.25+ (module `ecommerce-be`) + `github.com/redis/go-redis/v9` (new, replaces v8), `golang.org/x/sync` (singleflight — already indirect dep), Gin, GORM, testify/suite, Testcontainers
- 010-payment-gateway-platform: Added Go 1.25+ + Gin, GORM, `go-playground/validator/v10`, testify/suite + Testcontainers, stdlib `net/http` + `crypto/hmac`/`sha256` (no Razorpay SDK), `github.com/robfig/cron/v3` via `common/cron`
- 009-razorpay-payment-gateway: Added Go 1.25+ + Gin (HTTP), GORM (ORM), `go-playground/validator/v10` (validation),



<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
