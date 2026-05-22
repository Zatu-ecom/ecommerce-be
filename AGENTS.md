# ecommerce-be Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-05-20

## Active Technologies
- Go 1.25+ (per constitution) (003-upload-apis)
- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers, existing File module services (005-product-file-integration)
- PostgreSQL 16 for `product_media` association rows; external blob providers remain owned by File module (005-product-file-integration)

- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`), AWS SDK v2 (`service/s3`), GCS storage client, Azure Blob SDK (002-blob-adapters)
- PostgreSQL 16 for `storage_config`/`storage_provider` source records; external blob storage providers (S3-compatible, GCS, Azure Blob) for object data (002-blob-adapters)

- Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers infrastructure (`test/integration/setup`) (001-activate-storage-config)

## Project Structure

## Commands

# Add commands for Go 1.25+

## Code Style

Go 1.25+: Follow standard conventions

## Recent Changes
- 005-product-file-integration: Added Go 1.25+ + Gin, GORM, validator, testify/suite, Testcontainers, existing File module services
- 004-file-read-delete-apis: Added Go 1.25+
- 004-file-read-delete-apis: Added [if applicable, e.g., PostgreSQL, CoreData, files or N/A]



<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
