# Data Model: Storage Config Activation and Listing

## 1) Storage Configuration (existing persisted entity)

Represents storage credentials and metadata persisted in `storage_config`.

- **Identity**:
  - `id` (uint, primary key)
- **Ownership**:
  - `owner_type` (`PLATFORM` | `SELLER`)
  - `owner_id` (nullable uint; required for seller-owned configs, nil for platform)
- **Provider and config metadata**:
  - `provider_id` (uint)
  - `display_name` (string)
  - `bucket_or_container` (string)
  - `region` (string)
  - `endpoint` (string)
  - `base_path` (string)
  - `force_path_style` (bool)
  - `config_json` (json map)
- **Sensitive fields**:
  - `credentials_encrypted` (bytea) - never returned in list response
- **State fields**:
  - `is_active` (bool)
  - `is_default` (bool)
  - `validation_status` (string enum-like values, e.g. `PENDING`)
  - `last_validated_at` (timestamp nullable)
- **Audit fields**:
  - `created_at`, `updated_at`

### Validation rules relevant to feature
- `owner_id` must match token seller context when `owner_type=SELLER` operations are performed.
- For non-seller context operations, activation/listing must target `owner_type=PLATFORM` only.
- Activation input `id` must be valid uint path param.

## 2) Scope Context (derived, non-persisted)

Resolved from auth token and middleware context.

- `role` (string)
- `seller_id` (optional uint)
- `scope_kind` (derived):
  - `seller_scope` when `seller_id` present
  - `platform_scope` when `seller_id` absent

### Role constraints
- Access allowed only for seller and above.
- Scope cannot be overridden by query params.

## 3) List Query Model (new/extended non-persisted model)

`ListStorageConfigQueryParams` (raw binding model) + `ListStorageConfigFilter` (normalized model).

### Multi-value filters (comma-separated input -> []typed)
- `ids` -> `[]uint`
- `providerIds` -> `[]uint`
- `validationStatuses` -> `[]string`

### Single-value filters
- `isActive` -> `*bool`
- `isDefault` -> `*bool`
- `adapterType` -> `*string`
- `search` -> `*string`

### Pagination/sorting (via BaseListParams)
- `page`, `pageSize`, `sortBy`, `sortOrder`

### Prohibited filters
- `sellerId` (must not be accepted)
- `ownerType` (removed for this version)

## 4) List Response Model (new/extended non-persisted model)

`StorageConfigListItem` + paginated envelope.

- Item fields (non-sensitive):
  - `id`, `providerId`, `ownerType`, `displayName`, `bucketOrContainer`
  - `isActive`, `isDefault`, `validationStatus`
  - optional: `region`, `endpoint`, `basePath`, `forcePathStyle`, timestamps
- Pagination block uses shared `common.PaginationResponse`.

## 5) Activation Result Model (new/extended non-persisted model)

`ActivateStorageConfigResponse`:
- `id` (activated config id)
- `isActive` (true)
- `ownerType`
- `ownerId` (optional)
- optional message/status field per common response conventions

## Relationships

- `StorageConfig.provider_id` -> `StorageProvider.id` (many-to-one)
- Scope Context constrains which `StorageConfig` rows are visible/mutable.

## State Transitions

### Activation transition (within resolved scope)
- Precondition: config exists in scope and caller authorized.
- Transition:
  1. Set all scope-matching configs `is_active=false`
  2. Set target config `is_active=true`
- Postcondition: exactly one active config in that scope.

### Idempotent activation
- If target already active, operation returns success and final state remains single-active.

### Concurrency expectation
- Concurrent activation requests in same scope must converge to one active record at completion (transaction/locking strategy handled in implementation).

## Error Model Mapping (feature-relevant)

- Validation errors: invalid path/query values, forbidden filter keys.
- Authorization errors: role below seller, wrong scope ownership.
- Not found: config id absent in resolved scope.
- Internal errors: repository/transaction failures mapped to standardized AppError envelope.
