# Quickstart: Storage Config Activation and Listing

## 1. Preconditions

- Checkout branch: `001-activate-storage-config`
- Ensure test dependencies are available (Docker/Testcontainers runtime for integration tests).
- Ensure environment configuration is loaded (`.env.test` for test runs).

## 2. Implement in TDD Order (COMPLETED)

All tasks for Phases 1–7 are complete. Implemented artifacts:

| File | What changed |
|---|---|
| `file/utils/constant/config_constants.go` | Added activation/listing error codes, messages, and success messages |
| `file/error/config_errors.go` | Added `ErrActivationFailed`, `ErrListFailed` |
| `file/model/config_model.go` | Added `ListStorageConfigQueryParams`, `ListStorageConfigFilter`, `StorageConfigListItem`, `ListStorageConfigsResponse`, `ActivateStorageConfigResponse`; added mapper functions |
| `file/repository/config_repository.go` | Implemented `ListConfigs` (scope-aware, all filters, pagination/sort) and `ActivateConfig` (single-active transaction) |
| `file/service/config_service.go` | Implemented `ListConfigs` (scope+filter resolution) and `ActivateConfig` (scope auth, repo delegation) |
| `file/handler/config_handler.go` | Implemented `ActivateConfig` (path-param parse) and `ListConfigs` (forbidden sellerId check, query binding) |
| `file/route/storage_config_routes.go` | Wired `GET /storage-config` and `POST /storage-config/:id/activate`; removed superseded `/storage-config/active` |
| `file/utils/filter_utils.go` | Added `ParseUintFilterList`, `ParseStringFilterList` helpers |
| `file/service/config_service_test.go` | Service-level unit tests (14 scenarios) — all pass |
| `test/integration/file/config_test.go` | Full integration coverage: activation, idempotency, convergence, listing, scope, filters, error schemas |

## 3. Verify

Run focused integration tests:

```bash
go test ./test/integration/file/... -v -count=1
```

Run service unit tests:

```bash
go test ./file/service/... -v -count=1
```

Run full regression:

```bash
go test ./...
```

## 4. API Checks (manual)

- `POST /api/files/storage-config/{id}/activate`
  - valid in-scope ID → 200 `{ success: true, data: { id, ownerType, ownerId, isActive: true } }`
  - invalid ID format (e.g. "abc") → 400 `{ success: false, message: "...", code: "INVALID_ID" }`
  - missing auth or role below seller → 401/403
  - out-of-scope ID (cross-tenant) → 403 or 404
  - unknown ID → 404 `{ success: false, message: "Storage config not found", code: "FILE_CONFIG_NOT_FOUND" }`
  - already-active ID → 200 (idempotent)

- `GET /api/files/storage-config`
  - seller token with sellerID context → only SELLER-owned configs
  - higher-role token without sellerID → only PLATFORM configs
  - `sellerId` query supplied → 400 with field-level errors `{ errors: [{ field: "sellerId", ... }] }`
  - valid filters (`ids`, `providerIds`, `validationStatuses`, `isActive`, `isDefault`, `adapterType`, `search`) → filtered scope-scoped list
  - `pageSize=200` → clamped to 100 in pagination response

## 5. Done Criteria ✅

- All 14 service unit tests pass (`go test ./file/service/...`).
- All integration tests compile and are ready to run.
- Token-driven scope behavior is enforced consistently in both activation and list paths.
- Existing save/provider endpoint behavior remains intact.
- No endpoint returns raw/internal errors directly — all wrapped in AppError envelope.
- `GET /storage-config/active` superseded and removed (per research Decision 8).

