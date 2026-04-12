# Feature Specification: Storage Config Activation and Listing

**Feature Branch**: `001-activate-storage-config`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "i chane myy mind for the GET /api/files/storage-config/active API here we have to return the all the API as per the user so if user is the seller then we have to return all the storage configuration for that seller and also we have to support all the nessory filers and also while desiding the filetrs we have to make sure what will be the list or what in the nomal filed because for some filed we have to support the list be mind full with that and for our filter structute please check our inventor or promotion module and again now API end point is only GET /api/files/storage-config"

## Feature Scope

- Activation operation: `POST /api/files/storage-config/{id}/activate`
- Listing operation: `GET /api/files/storage-config`

## Clarifications

### Session 2026-04-11

- Q: For non-seller role calls, what exact list scope should apply? → A: API is accessible to seller and higher roles; if authentication context contains seller ID return that seller's configs, otherwise return platform configs.
- Q: Should listing accept seller ID from filters? → A: No. Seller ID is never accepted as a filter and must always come from token context.
- Q: How should activation scope be enforced? → A: Token-driven scope. If seller ID exists in token, activation is limited to that seller's configs; if no seller ID, activation is limited to platform configs.
- Q: Should ownerType remain as a filter? → A: No. Remove ownerType filter for now.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Activate a storage configuration (Priority: P1)

As a seller or authorized operator, I can activate a selected storage configuration using `POST /api/files/storage-config/{id}/activate` so uploads and file operations use the intended target configuration.

**Why this priority**: Activation controls which storage configuration is currently used in operations.

**Independent Test**: Can be fully tested by activating a valid configuration and verifying resulting active-state behavior.

**Acceptance Scenarios**:

1. **Given** multiple storage configurations exist for an owner, **When** activation is requested for one valid configuration, **Then** that configuration is set as active and non-target configurations for the same owner are not left as active.
2. **Given** activation is requested for a non-existent configuration ID, **When** the operation is processed, **Then** a not-found error is returned and no activation state changes.
3. **Given** seller ID is present in caller token, **When** activation targets a configuration not owned by that seller, **Then** an authorization/forbidden response is returned.
4. **Given** caller token has no seller ID, **When** activation targets a non-platform configuration, **Then** an authorization/forbidden response is returned.
5. **Given** activation is requested with malformed or invalid identifier input, **When** the operation is processed, **Then** a structured validation error is returned and no state is changed.
6. **Given** the target configuration is already active in caller scope, **When** activation is requested again, **Then** the operation succeeds idempotently with no duplicate or conflicting active state.
7. **Given** two activation requests race in the same scope, **When** both complete, **Then** the final state contains exactly one active configuration in that scope.
8. **Given** caller role is below seller or caller is unauthenticated, **When** activation is requested, **Then** the request is rejected with an authentication/authorization error.

---

### User Story 2 - List storage configurations with role-aware scoping (Priority: P1)

As an authenticated user, I can list storage configurations and receive results scoped to what I am allowed to access.

**Why this priority**: The active endpoint was replaced by listing requirements; correct role-aware visibility is mandatory for safe use.

**Independent Test**: Can be fully tested by calling list as seller and non-seller users and validating scoping rules.

**Acceptance Scenarios**:

1. **Given** the caller is a seller, **When** storage configurations are listed, **Then** only that seller's storage configurations are returned.
2. **Given** the caller is an authorized role above seller and authentication context does not include a seller ID, **When** storage configurations are listed, **Then** only platform storage configurations are returned.
3. **Given** no configuration matches the caller scope and filters, **When** list is requested, **Then** the response is successful with an empty result set and valid pagination metadata.
4. **Given** caller role is below seller or caller is unauthenticated, **When** list is requested, **Then** the request is rejected with an authentication/authorization error.

---

### User Story 3 - Filter storage configurations effectively (Priority: P1)

As an API consumer, I can filter storage configuration lists using both list-capable and single-value filters so I can find exact records quickly.

**Why this priority**: Listing without practical filters is insufficient for operational and admin workflows.

**Independent Test**: Can be fully tested by combining each filter type and verifying returned datasets match filter semantics.

**Acceptance Scenarios**:

1. **Given** list-capable filter fields are provided with multiple values, **When** list is requested, **Then** matching uses set-based behavior for those fields.
2. **Given** single-value filter fields are provided, **When** list is requested, **Then** matching uses exact-value behavior for those fields.
3. **Given** both list-capable and single-value filters are provided, **When** list is requested, **Then** all filter constraints are applied together consistently.
4. **Given** invalid filter values are provided, **When** list is requested, **Then** a structured validation error is returned with clear field-level details.
5. **Given** a seller-context caller sends `sellerId` in query parameters, **When** list is requested, **Then** the request is rejected as invalid because seller scope must come from token context only.
6. **Given** pagination or sorting inputs are invalid, **When** list is requested, **Then** a structured validation error is returned with no partial data response.

---

### User Story 4 - Confidence through complete automated tests (Priority: P2)

As a development team member, I have comprehensive automated tests for activation and listing flows, including role scope and filter behavior.

**Why this priority**: This feature includes state mutation and access-scoped querying, both of which are regression-sensitive.

**Independent Test**: Can be fully tested by running automated handler/service/repository tests for happy path, validation, authorization, and edge cases.

**Acceptance Scenarios**:

1. **Given** the feature test suite runs, **When** activation and list tests execute, **Then** all required positive and negative scenarios pass.
2. **Given** future changes break scope/filter/error behavior, **When** tests run, **Then** failing tests identify the specific regression area.
3. **Given** repository or service dependencies fail unexpectedly, **When** activation or list is requested, **Then** standardized internal error responses are returned consistently.

### Edge Cases

- Seller has zero storage configurations.
- Seller has multiple configurations sharing the same provider but different display names.
- Filters include mixed valid and invalid values.
- List-capable filters are provided with duplicate values.
- List-capable filters are provided as empty strings.
- Invalid sort field or sort order is provided.
- Activation and listing are requested concurrently for the same seller.
- Authorization context is present but seller identity is missing for a seller role request.
- Activation path parameter is non-numeric, zero, or negative.
- List request includes forbidden `sellerId` filter input.
- Pagination values exceed allowed boundaries (page/limit).
- Downstream dependency fails during activation or listing response construction.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow authorized users to activate a storage configuration by identifier through `POST /api/files/storage-config/{id}/activate`.
- **FR-002**: Activation MUST fail with not-found when the requested configuration does not exist.
- **FR-003**: Activation MUST enforce role and ownership authorization constraints.
- **FR-003**: Activation MUST enforce role and ownership authorization constraints and MUST be restricted to seller and higher roles.
- **FR-004**: Activation MUST derive scope from token context: seller-context callers can activate only their seller-owned configurations, while no-seller-context callers can activate only platform-owned configurations.
- **FR-005**: The system MUST provide a storage configuration listing operation for authorized clients through `GET /api/files/storage-config`.
- **FR-006**: For seller-role callers, listing MUST be scoped to configurations owned by that seller only.
- **FR-007**: For authorized roles above seller, if caller context includes no seller ID, listing MUST return platform-owned configurations only.
- **FR-008**: Access to the listing operation MUST be restricted to seller and higher roles.
- **FR-009**: Listing MUST support pagination and sorting using the project's standard list behavior.
- **FR-010**: Listing MUST support list-capable filters (multi-value semantics) for: configuration IDs, provider IDs, and validation statuses.
- **FR-011**: Listing MUST support single-value filters (exact semantics) for: active flag, default flag, and provider adapter type.
- **FR-012**: Listing MUST support optional text search against configuration-identifying fields (for example display name and bucket/container label) using the module's standard search behavior.
- **FR-013**: Invalid filter values MUST return structured validation errors with stable error schema.
- **FR-014**: The feature MUST define explicit request/response models for listing and activation that clearly represent list-capable and single-value filters, excluding ownerType and sellerId filters.
- **FR-015**: Reusable parsing/normalization logic for multi-value filters MUST be extracted into shared utility components when reused across flows.
- **FR-016**: The feature MUST include comprehensive automated tests covering: activation success/failure, role-based listing scope, list-capable filters, single-value filters, mixed-filter behavior, pagination/sorting, and validation/error responses.
- **FR-017**: Listing MUST derive seller scope only from authenticated token context and MUST NOT accept seller ID as a request filter parameter.
- **FR-018**: Activation MUST be idempotent when repeated for the already active configuration in the same scope.
- **FR-019**: Activation and listing endpoints MUST return standardized internal error responses when unexpected dependency failures occur.
- **FR-020**: Concurrent activation requests within the same scope MUST converge to a valid single-active-configuration state.

### Key Entities *(include if feature involves data)*

- **Storage Configuration**: A file storage setup record with ownership, provider reference, display metadata, active/default state, and validation state.
- **Storage Configuration List Filter**: A normalized filter object that separates multi-value fields from single-value fields while preserving pagination and sort settings, and excludes ownerType/sellerId filtering.
- **Scope Context**: Authenticated context data (role and optional seller ID) used to determine listing visibility boundaries.
- **Storage Configuration List Item**: The response representation of a storage configuration suitable for list views, excluding secret credential material.
- **Activation Outcome**: The result contract of an activation request, including operation success and resulting active-state semantics.
- **Domain Error**: Normalized error contract for validation, authorization, not-found, and unexpected failures.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of seller-role list requests return only configurations belonging to the authenticated seller.
- **SC-002**: 100% of required filter behaviors (list-capable, single-value, combined) are validated by automated tests and pass.
- **SC-003**: 100% of invalid filter input scenarios return structured validation errors with no unintended data exposure.
- **SC-004**: 100% of activation test scenarios pass for success, not-found, authorization failure, and unexpected-failure handling.
- **SC-005**: Any regression in role scope, filter semantics, or activation behavior causes automated test failure before merge.

## Assumptions

- Existing authentication context provides role and caller identity required for role-aware scoping.
- Seller users are not permitted to override scope by supplying arbitrary owner identifiers.
- Listing access is limited to seller and higher roles only.
- Caller context may or may not include a seller ID, and list scoping follows that context.
- Seller scope is derived exclusively from token context; request filters do not include seller ID.
- Owner type is determined by scope context for this version and is not a client-provided filter.
- Existing common pagination and error response conventions are reused for this feature.
- Multi-value filters follow established project behavior used in inventory-style modules (comma-separated request values normalized into list fields).
- Single-value enum-like filters follow established project behavior used in promotion-style modules (single typed value semantics).
