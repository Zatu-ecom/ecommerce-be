# Phase 0 Research: Storage Config Activation and Listing

## Decision 1: Scope resolution is token-driven, never query-driven for seller identity
- **Decision**: Determine visibility and mutation scope from authenticated token context. If token includes seller ID, operate in seller scope; if not, operate in platform scope.
- **Rationale**: Prevents tenant spoofing and aligns with module clarifications and constitution seller-isolation requirements.
- **Alternatives considered**:
  - Accept `sellerId` as query filter: rejected due to cross-tenant exposure risk.
  - Infer scope from request path/query only: rejected because it weakens auth boundary.

## Decision 2: Remove `ownerType` from public list filters for this version
- **Decision**: `ownerType` is derived from token scope behavior and not accepted as list filter input.
- **Rationale**: Simplifies contract, prevents conflicting scope hints, and follows clarified product direction.
- **Alternatives considered**:
  - Allow `ownerType` but validate conflicts: rejected as unnecessary complexity for current requirements.
  - Allow `ownerType` and silently override: rejected due to hidden behavior and weak API transparency.

## Decision 3: Activation authorization mirrors listing scope exactly
- **Decision**: Activation uses same scope rules as listing: seller-context can activate only seller-owned configs; non-seller-context can activate only platform configs.
- **Rationale**: Consistent policy reduces bugs and makes tests deterministic.
- **Alternatives considered**:
  - Let higher roles activate any config ID: rejected due to inconsistent isolation model.
  - Separate policy between list and activate: rejected because it increases regression risk.

## Decision 4: Activation semantics include idempotency and scope-level single-active convergence
- **Decision**: Re-activating already-active config succeeds without conflicting state changes; concurrent requests converge to one active config in scope.
- **Rationale**: Matches clarified acceptance criteria and prevents inconsistent runtime behavior.
- **Alternatives considered**:
  - Return conflict on repeated activation: rejected as unnecessary friction for clients.
  - Best-effort activation without convergence guarantees: rejected due to data consistency risk.

## Decision 5: Filter shape follows existing inventory/promotion patterns
- **Decision**: Use `common.BaseListParams` for pagination/sort; multi-value filters parsed from comma-separated strings (`ids`, `providerIds`, `validationStatuses`); single-value filters for booleans/adapter type/search.
- **Rationale**: Preserves project consistency and lowers implementation risk.
- **Alternatives considered**:
  - Introduce JSON-based filter object in query: rejected as inconsistent with current API conventions.
  - Use repeated query keys only (`ids=1&ids=2`): rejected for mismatch with existing parsing helpers.

## Decision 6: Standardized error mapping remains AppError-based
- **Decision**: Reuse `file/error` AppError patterns for validation, not-found, forbidden, and internal failures; add/adjust constants where needed.
- **Rationale**: Aligns with constitution error handling standards and existing handler behavior.
- **Alternatives considered**:
  - Return raw DB/Go errors from service: rejected due to unstable contract.
  - Introduce ad-hoc endpoint-specific error envelope: rejected for inconsistency.

## Decision 7: Integration-first test strategy using existing file config suite
- **Decision**: Extend `test/integration/file/config_test.go` (and split only if needed) to cover all new endpoint behavior.
- **Rationale**: Existing suite already owns file-config flows and test infra setup.
- **Alternatives considered**:
  - Unit-only coverage for service/repository: rejected because full middleware/auth/scope behavior must be verified end-to-end.
  - New standalone suite immediately: deferred unless file grows beyond maintainability threshold.

## Decision 8: Route migration strategy
- **Decision**: Replace `GET /storage-config/active` route usage with `GET /storage-config` list endpoint behavior while keeping other routes unchanged.
- **Rationale**: Matches current feature scope and avoids unrelated endpoint churn.
- **Alternatives considered**:
  - Keep both active and list endpoints: rejected as out-of-scope and potentially conflicting.
