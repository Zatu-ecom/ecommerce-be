# Zatu E-commerce Backend Constitution

## Core Principles

### I. Modular Monolith with Microservices DNA (NON-NEGOTIABLE)

The system is architected as a **modular monolith** where each business domain (User, Product, Order, Payment, Inventory, Promotion, Report, Notification, File) is a fully self-contained module. Each module owns its own entities, repositories, services, handlers, and routes. Modules MUST NOT directly access another module's repositories or internal types. Cross-module communication MUST go through service interfaces or well-defined API contracts.

Every module is designed to be extractable into an independent microservice with zero code changes to its internal structure. The `common/` package provides cross-cutting concerns shared by all modules (auth, caching, logging, middleware, error handling, database, messaging, configuration).

**Rationale**: This architecture enables independent team development, clear ownership boundaries, and a future migration path to microservices without requiring a rewrite. Data isolation and encapsulation prevent cascading failures and tight coupling.

### II. Clean Architecture / Layered Architecture (NON-NEGOTIABLE)

All modules MUST follow a strict layered architecture with unidirectional dependency flow:

```
Handler (Presentation) → Service (Business Logic) → Repository (Data Access) → Database
```

**Layer Rules**:

- **Handlers**: Parse HTTP requests, extract context (userId, sellerId, correlationId), call services, return standardized responses. Handlers MUST NOT contain business logic or direct database access.
- **Services**: Implement business rules, orchestrate repositories, manage transactions, handle caching. Services MUST NOT handle HTTP concerns (request parsing, response formatting). Services return DTOs/models, NOT entities.
- **Repositories**: Execute database queries via GORM. Repositories MUST NOT contain business rules or validation logic. Repositories return entities, NOT DTOs.
- **Entities**: Pure data structures representing database tables. Entities MUST NOT contain business logic.
- **Models**: API request/response DTOs with JSON tags and validation bindings. Models MUST NOT contain database details.

**Layer Bypass is FORBIDDEN**: A handler MUST NOT call a repository directly. A service MUST NOT call another module's repository directly — it MUST go through that module's service interface.

**Rationale**: Strict layer separation ensures testability, maintainability, and the ability to swap implementations (e.g., swap PostgreSQL for another database) without affecting upper layers.

### III. Factory-Singleton Dependency Injection Pattern (NON-NEGOTIABLE)

All modules MUST use the **Factory-Singleton** pattern for dependency injection. Each module with multiple sub-domains uses a three-tier factory structure:

1. **RepositoryFactory**: Creates and caches repository instances (injects `*gorm.DB` from `db.GetDB()`).
2. **ServiceFactory**: Creates and caches service instances (injects repositories and cross-service dependencies from the RepositoryFactory).
3. **HandlerFactory**: Creates and caches handler instances (injects services from the ServiceFactory).
4. **SingletonFactory**: The top-level facade that delegates to all three factories. Uses `sync.Once` for thread-safe lazy initialization.

```go
var (
    instance *SingletonFactory
    once     sync.Once
)

func GetInstance() *SingletonFactory {
    once.Do(func() {
        repoFactory := NewRepositoryFactory()
        serviceFactory := NewServiceFactory(repoFactory)
        handlerFactory := NewHandlerFactory(serviceFactory)
        instance = &SingletonFactory{
            repoFactory:    repoFactory,
            serviceFactory: serviceFactory,
            handlerFactory: handlerFactory,
        }
    })
    return instance
}

func ResetInstance() {
    once = sync.Once{}
    instance = nil
}
```

Each module's `container.go` wires sub-modules via `route.NewXxxModule()`, which internally calls the singleton factory to obtain handlers. The container registers all modules with the Gin router.

**Rationale**: This pattern provides a single source of truth for dependencies, ensures lazy initialization, avoids global mutable state, and enables clean test resets via `ResetInstance()`.

### IV. Test-Driven Development (TDD) & Integration-First Testing (NON-NEGOTIABLE)

All modules/features MUST be built test-first. Write the complete test suite FIRST, then implement production code to make tests pass.

**TDD Workflow (Red-Green-Refactor)**:

1. **DESIGN**: Identify module/feature boundaries and API contracts.
2. **TEST (Red)**: Write complete integration test suite covering behaviors, edge cases, error paths.
3. **IMPLEMENT (Green)**: Write production code to pass all tests — **MUST verify green before proceeding**.
4. **REFACTOR**: Improve code quality while keeping tests green.

**Integration Tests are Primary**:

- Integration tests use **Testcontainers** to spin up real PostgreSQL 16 and Redis 7 containers.
- Tests validate the full request lifecycle: HTTP request → Middleware → Handler → Service → Repository → Database → Response.
- Integration tests are located in `test/integration/<module>/` (e.g., `test/integration/product/`, `test/integration/order/`).
- Test infrastructure lives in `test/integration/setup/` (containers, server, database) and `test/integration/helpers/` (API client, auth, assertions, test data).

**API-First Testing Rule**: Test side-effects through API calls (GET after POST), NOT by querying the database directly. This ensures the full stack is validated.

**Test Data Management**:

- Seed data is applied via SQL migrations in `test/integration/setup/database.go`.
- Each test file MUST clean up its own test data to prevent state leakage between tests.
- Use the shared `APIClient` helper for all HTTP interactions with proper headers (`X-Correlation-ID`, `X-Seller-ID`, `Authorization`).

**Unit Tests**: Optional but recommended for complex algorithms, pure business logic, or utility functions.

**Rationale**: Integration tests catch real-world bugs that unit tests miss (query issues, middleware behavior, transaction boundaries). TDD ensures testable design and prevents over-engineering.

### V. Multi-Tenant Seller Isolation (NON-NEGOTIABLE)

All data operations MUST be scoped by `seller_id`. The system supports multiple sellers (multi-tenant), and data MUST never leak between sellers.

**Implementation**:

- Public APIs require `X-Seller-ID` header for tenant isolation.
- Authenticated APIs extract `sellerId` from JWT claims.
- Every database query involving seller-scoped data MUST include a `WHERE seller_id = ?` clause.
- Repository methods for seller-scoped data MUST accept `sellerID` as a parameter.

**Rationale**: Data isolation is a critical security requirement. A seller MUST only see and manage their own data. Cross-tenant data access would be a severe security breach.

### VI. Correlation ID & Distributed Tracing (NON-NEGOTIABLE)

Every API request MUST include an `X-Correlation-ID` header. Requests without this header are rejected with `400 Bad Request` by the `middleware.CorrelationID()` middleware.

**Propagation Rules**:

- The correlation ID is extracted in middleware and stored in the Gin context.
- All log entries MUST include `correlationId`, `userId`, and `sellerId`.
- The structured logger (`common/log`) automatically injects these fields when the context is passed.
- Correlation IDs MUST be propagated to downstream services and message queues.

**Rationale**: Correlation IDs enable end-to-end request tracing across the entire system, which is essential for debugging, monitoring, and operational excellence.

### VII. Role-Based Access Control (RBAC)

The system enforces three roles with hierarchical permissions:

| Role         | Access Level                                       |
| ------------ | -------------------------------------------------- |
| **Admin**    | Full system access, manage all sellers and users   |
| **Seller**   | Manage own products, inventory, orders, promotions |
| **Customer** | Browse products, manage cart, place orders         |

**Implementation**:

- JWT tokens contain `userId`, `email`, `role`, and `sellerId` claims.
- Route-level middleware enforces roles: `middleware.AdminAuth()`, `middleware.SellerAuth()`, `middleware.CustomerAuth()`.
- `middleware.PublicAPIAuth()` handles unauthenticated browsing with `X-Seller-ID` header.
- `middleware.AuthMiddleware()` validates JWT token presence and validity.

**Middleware Hierarchy**: `CorrelationID()` → `Logger()` → `CORS()` → `AuthMiddleware()` → Role-specific middleware.

**Rationale**: RBAC ensures that users can only perform actions appropriate to their role. Middleware-based enforcement prevents bypass of authorization checks.

### VIII. Backward Compatibility & Regression Prevention (NON-NEGOTIABLE)

All code changes MUST be backward compatible and MUST NOT introduce regressions.

**Requirements**:

- API endpoints MUST maintain existing request/response contracts.
- Database schema changes MUST be backward-compatible during rolling deployments.
- Configuration changes MUST support existing configurations.
- Behavior changes MUST be opt-in via feature flags or new endpoints.
- Test coverage MUST verify backward compatibility scenarios.

**Breaking Change Process**: When breaking changes are necessary, they MUST:

1. Be explicitly documented in the change description.
2. Provide clear migration paths or deprecation timelines.
3. Be approved by technical leads with justification.

**Rationale**: Production systems require stable, predictable behavior. Regressions cause service disruptions, data inconsistencies, and loss of customer trust.

### IX. SOLID Design Principles (Practical Application)

Code MUST follow SOLID principles, but practicality takes precedence over dogmatic adherence.

- **SRP**: Each struct/function has ONE clear reason to change. Split files exceeding 500 lines.
- **OCP**: Prefer composition and strategy patterns (e.g., promotion strategies) over modification of existing code.
- **LSP**: Interface implementations MUST be substitutable for their base types.
- **ISP**: Interfaces MUST be focused and client-specific (avoid "fat interfaces"). Max ~10 methods per interface.
- **DIP**: High-level modules depend on abstractions (interfaces), not concrete implementations. All dependencies injected through constructors.

**Practical Guidelines**:

- Start simple, refactor toward SOLID when complexity emerges.
- Use SOLID to solve real problems (testability, maintainability).
- Document when SOLID is intentionally violated and why.

**Rationale**: SOLID principles create maintainable, testable, and extensible code. The promotion module's strategy pattern is a good example of OCP in practice.

### X. Performance & Scalability by Design

All implementations MUST consider performance from the start.

**Database Performance**:

- Use GORM `Preload()` or `Joins()` to avoid N+1 queries. NEVER loop over records to make individual queries.
- Use proper database indexes on frequently queried columns (`seller_id`, `category_id`, foreign keys).
- Use pagination for all list endpoints via `common.BaseListParams` (default: page=1, pageSize=20, max=100).
- Select only needed fields when full entity loading is unnecessary.
- Use raw SQL via GORM for complex queries when the ORM abstractions are insufficient.

**Caching Strategy**:

- Redis caching with LRU eviction (256MB max memory).
- Cache reads first, fall back to database on miss.
- Invalidate cache on mutations (create, update, delete).
- Use pattern-based cache invalidation for list caches.

**Stateless Services**: All services MUST be stateless to enable horizontal scaling. State lives in PostgreSQL and Redis only.

**Rationale**: E-commerce systems handle high-traffic bursts (flash sales, promotions). Performance cannot be retrofitted; it must be designed in from the beginning.

## Technology Standards

**Language/Runtime**: Go 1.25+
**HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
**Database**: PostgreSQL 16 with GORM (`gorm.io/gorm`, `gorm.io/driver/postgres`)
**Cache**: Redis 7 with go-redis (`github.com/go-redis/redis/v8`)
**Authentication**: JWT via `github.com/golang-jwt/jwt/v5`
**Messaging**: RabbitMQ via `github.com/rabbitmq/amqp091-go`
**Scheduling**: Cron jobs via `github.com/robfig/cron/v3`
**Validation**: `github.com/go-playground/validator/v10` (struct tag-based)
**Logging**: Logrus (`github.com/sirupsen/logrus`) with structured JSON format
**Testing**: Go testing + testify assertions + Testcontainers
**Build Tool**: Go modules (`go.mod`)
**Containerization**: Docker (multi-stage builds), Docker Compose for local development
**Configuration**: Environment variables via `.env` files with `godotenv`

## Module Structure Standards

### Standard Module Layout

Every module MUST follow this directory structure:

```
module_name/
├── container.go                 # Module registration & route wiring
├── entity/                      # Database models (GORM structs)
│   └── *.go                     # Entities inherit from common.BaseEntity
├── model/                       # API DTOs (request/response structs)
│   └── *.go                     # JSON tags + validation bindings
├── repository/                  # Data access layer
│   ├── *_repository.go          # Interface definition
│   └── *_repository_impl.go     # GORM implementation
├── service/                     # Business logic layer
│   ├── *_service.go             # Interface definition
│   └── *_service_impl.go        # Implementation
├── handler/                     # HTTP handlers (controllers)
│   └── *_handler.go             # Parse requests, call services, return responses
├── route/                       # Route definitions
│   └── *_routes.go              # URL mapping + middleware application
├── factory/                     # Dependency injection (if module has sub-domains)
│   └── singleton/               # Singleton factory pattern
├── mapper/                      # Object mappers (complex query result mapping)
├── query/                       # Complex query builders, search, filters or you can create the constant in the repo
├── error/                       # Module-specific error definitions
└── utils/                       # Module-specific utility functions
```

### Active Modules

| Module           | Description                         | Sub-domains                                                |
| ---------------- | ----------------------------------- | ---------------------------------------------------------- |
| **user**         | Authentication, profiles, addresses | Auth, Profile, Address                                     |
| **product**      | Catalog management                  | Category, Attribute, Product, Variant, Option, Wishlist    |
| **order**        | Cart & order lifecycle              | Cart, Order (with status machine)                          |
| **inventory**    | Stock management & reservations     | Inventory, Reservation                                     |
| **promotion**    | Discount & promotion engine         | Promotion rules, Strategy pattern (Bundle, BuyXGetY, etc.) |
| **payment**      | Payment processing                  | Payment methods, transactions                              |
| **report**       | Business analytics & reporting      | Sales summary, sales trends                                |
| **notification** | User notifications                  | Email, push notification (future)                          |
| **file**         | File storage provider configuration | Provider config, file upload/download                      |
| **fulfillment**  | Order fulfillment (future)          | Shipping, tracking                                         |
| **subscription** | Subscription management (future)    | Plans, billing cycles                                      |

## API Response Standards

### Standardized Response Format

All API responses MUST use the standardized response helpers from `common/response.go`:

**Success Response**:

```json
{
    "success": true,
    "message": "Product created successfully",
    "data": { ... }
}
```

**Error Response**:

```json
{
  "success": false,
  "message": "Product not found",
  "code": "PRODUCT_NOT_FOUND"
}
```

**Validation Error Response**:

```json
{
  "success": false,
  "message": "Validation failed",
  "code": "VALIDATION_ERROR",
  "errors": [{ "field": "name", "message": "Name is required" }]
}
```

**Paginated Response** (list endpoints):

```json
{
    "success": true,
    "message": "Products fetched",
    "data": {
        "items": [ ... ],
        "pagination": {
            "currentPage": 1,
            "totalPages": 5,
            "totalItems": 100,
            "itemsPerPage": 20,
            "hasNext": true,
            "hasPrev": false
        }
    }
}
```

### Response Helpers

- `common.SuccessResponse(c, statusCode, message, data)` — Standardized success
- `common.ErrorResp(c, statusCode, message)` — Generic error
- `common.ErrorWithCode(c, statusCode, message, code)` — Error with code
- `common.ErrorWithValidation(c, statusCode, message, errors, code)` — Validation errors

## Error Handling Standards

### Structured Application Errors

All modules MUST define service-specific errors using `common/error.AppError`:

```go
var ProductNotFound = error.NewAppError("PRODUCT_NOT_FOUND", "Product not found", 404)
var InvalidSKU = error.NewAppError("INVALID_SKU", "SKU must be unique and alphanumeric", 400)
```

**Error Flow**:

1. **Repository**: Wrap database errors with context using `fmt.Errorf("...: %w", err)`.
2. **Service**: Return `*AppError` for known business errors. Wrap unexpected errors.
3. **Handler**: Check `error.IsAppError(err)` or `error.AsAppError(err)` to map to HTTP responses. Unknown errors → 500.

**Rules**:

- NEVER return raw `errors.New("...")` for business logic failures.
- ALWAYS wrap errors with context at each layer boundary.
- Error codes MUST be unique, uppercase, and use underscores (e.g., `INSUFFICIENT_STOCK`).

## Code Quality Standards

### Magic Values Policy

Do NOT use magic strings, numbers, or literal values in production code.

1. **String Literals**: Use constants or enums.
   - ❌ `if status == "active"` → ✅ `if status == constants.StatusActive`
2. **Numeric Literals**: Use named constants.
   - ❌ `if retries > 5` → ✅ `if retries > maxRetries`
3. **Configuration Values**: Extract to `.env` / config structs.
4. **HTTP Status**: Use `http.StatusOK`, `http.StatusCreated`, etc.
5. **Error Codes**: Define in module's `error/` package.

### File Size Limits

| Item          | Max Lines  | Action if Exceeded                     |
| ------------- | ---------- | -------------------------------------- |
| **Method**    | 50 lines   | Extract into smaller helper methods    |
| **File**      | 500 lines  | Split into multiple files by concern   |
| **Interface** | 10 methods | Split into smaller, focused interfaces |

### Pointer Usage for Optional Fields

Use pointers for optional/nullable fields in update request models to distinguish between "not provided" (nil) and "set to zero value":

```go
type UpdateProductRequest struct {
    Name  *string  `json:"name" binding:"omitempty,min=3,max=200"`
    Price *float64 `json:"price" binding:"omitempty,gt=0"`
    Stock *int     `json:"stock" binding:"omitempty,gte=0"`
}
```

### Code Documentation & Commenting (CRITICAL)

- **Mandatory Comments**: All normal methods, complex logic blocks, and exported types MUST include clear, descriptive comments.
- **Explain the "Why" and "What"**: Comments must explain _what_ the block of code does and _why_ it is necessary, not just how it does it.
- **Accessible to All Contexts**: Comments should be written in plain English so that non-technical stakeholders or AI pair-programmers can easily understand the business rules and domain logic being implemented without having to reverse-engineer complex Go syntax.

### Code Pattern Consistency

- **ALWAYS** check existing patterns before writing new code. Follow established conventions unless there is a documented reason to deviate.
- Service interfaces + implementation separation is the default pattern.
- Constructor injection via `NewXxx(deps...) *XxxImpl` functions.
- Use helper functions in `common/helper/` for truly reusable utilities.
- Log at appropriate levels (DEBUG for trace, INFO for events, WARN for recoverable issues, ERROR for failures).

## Database Standards

### Schema Migrations

- All migrations live in `migrations/` as numbered SQL files (e.g., `001_create_user_tables.sql`).
- Migrations are run via `migrations/run_migrations.sh` or `make migrate`.
- Seed data lives in `migrations/seeds/`.
- Migrations MUST be backward-compatible during rolling deployments.
- Migration files MUST NEVER be modified after being applied to production. Create a new migration instead.

### GORM Conventions

- Use `schema.NamingStrategy{SingularTable: true}` — table names are singular (e.g., `product`, not `products`).
- All entities inherit common base fields: `ID`, `CreatedAt`, `UpdatedAt`.
- Use `gorm:"index"` for foreign key columns and frequently filtered columns.
- Use `gorm:"uniqueIndex"` for naturally unique fields (SKU, email).
- Use `gorm.DeletedAt` for soft deletes when required.
- Use GORM transactions (`db.Transaction(func(tx *gorm.DB) error { ... })`) for multi-step operations.

## Testing Standards

### Test Infrastructure

```
test/integration/
├── setup/
│   ├── container.go    # Testcontainers for PostgreSQL & Redis
│   ├── database.go     # Migration runner & seed data
│   └── server.go       # Gin router setup with all modules registered
├── helpers/
│   ├── api_client.go   # HTTP client wrapper with header support
│   ├── auth_helper.go  # Login helpers, JWT token generation
│   ├── assertions.go   # Custom assertion helpers
│   └── test_data.go    # Shared test data constants
└── <module>/           # Module-specific integration tests
```

### Test Suite Pattern & Naming Convention

All integration tests MUST use the `suite.Suite` pattern from `github.com/stretchr/testify/suite` to share state and setup/teardown logic.
Constants MUST be used for all API endpoints to ensure easy refactoring in the future.

**Setup Suite Example (`setup_suite_test.go`)**:

```go
const (
    OrderAPIEndpoint     = "/api/order"
    OrderByIDAPIEndpoint = "/api/order/%d"
)

type OrderSuite struct {
    suite.Suite
    container *setup.TestContainer
    server    http.Handler
    client    *helpers.APIClient
}

func (s *OrderSuite) SetupSuite() {
    // Initialize containers, DB, server, and clients
}

func TestOrderSuite(t *testing.T) {
    suite.Run(t, new(OrderSuite))
}
```

**Test Naming Convention**:
Test files: `<feature>_test.go` (e.g., `order_create_test.go`)
Test methods: `Test<ActualTestScenario>` in PascalCase attached to the suite.

**Test Documentation Requirement**:
Every test method MUST include comments detailing the scenario and the specific behaviors or edge cases being tested. This provides essential context for future developers and AI assistants.

```go
// ─── Create Order: Returns Correct Status ──────────────────────────────────
// Scenario: A valid customer adds an item to their cart and submits an order.
// Validates:
// 1. Order is created successfully (201 Created).
// 2. The initial status of the new order is 'pending'.
func (s *OrderSuite) TestCreateOrderReturnsCorrectStatus() {
    w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
    helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
}
```

### Mandatory Test Scenarios

Every API endpoint MUST test:

1. ✅ Happy path (valid request → success response)
2. ✅ Authentication failures (missing/invalid/expired token)
3. ✅ Authorization failures (wrong role, wrong seller)
4. ✅ Validation errors (missing required fields, invalid formats)
5. ✅ Edge cases (not found, duplicates, constraint violations)
6. ✅ Correlation ID enforcement (missing → 400)
7. ✅ Seller isolation (seller A cannot see seller B's data)

### Running Tests

```bash
make test          # Run all integration tests
make test-pretty   # Run with formatted summary (uses gotestsum)
make test-failed   # Re-run only failed tests from last run
make test-json     # JSON output for CI/CD
```

## Deployment & Infrastructure

### Docker Setup

- **Dockerfile**: Multi-stage build (`builder` → `runtime` stages) for minimal image size.
- **docker-compose.yml**: Orchestrates PostgreSQL 16, Redis 7, and application containers.
- **Environment**: Configuration via `.env` files (`.env` for dev, `.env.test` for tests).

### Graceful Shutdown

The application handles OS signals (`SIGINT`, `SIGTERM`) for graceful shutdown:

1. Stop accepting new HTTP requests.
2. Wait for ongoing requests to complete (30s timeout).
3. Close database connections.
4. Close Redis connections.
5. Stop cron scheduler.

### Background Workers

- **Redis Worker Pool**: Background task processing via `common/scheduler`.
- **Cron Jobs**: Scheduled tasks via `common/cron` using `robfig/cron`.
- Workers are started before the HTTP server begins accepting requests.

## Governance

**Amendment Process**: Constitution changes require documentation of rationale and validation against existing codebase patterns.

**Compliance Review**: All code changes MUST verify compliance with this constitution. Deviations MUST be documented with justification.

**TDD Enforcement**: Code reviews MUST verify test-first approach. Production code without adequate test coverage MUST be rejected.

**Pattern Consistency**: Before introducing a new pattern, check if a similar pattern exists in the codebase. Follow existing patterns unless there is a documented, justified reason to deviate.

**Complexity Justification**: Any design that violates these principles MUST document the violation and explain why it is necessary.

**Version**: 1.0.0 | **Ratified**: 2026-04-09 | **Last Amended**: 2026-04-09
