# 🏗️ Zatu E-commerce Backend - Architecture Documentation

> **Last Updated**: November 20, 2025  
> **Target Audience**: New developers joining the project

---

## 📋 Table of Contents

1. [Project Overview](#project-overview)
2. [High-Level Architecture](#high-level-architecture)
3. [Directory Structure](#directory-structure)
4. [Architectural Patterns](#architectural-patterns)
5. [Standard Service Structure](#standard-service-structure)
6. [Common Module](#common-module-shared-code)
7. [Request Flow](#request-flow)
8. [Authentication & Authorization](#authentication--authorization)
9. [Testing Strategy](#testing-strategy)
10. [Development Guidelines](#development-guidelines)
11. [Roadmap & Future Architecture](#roadmap--future-architecture)

---

## 🎯 Project Overview

**Zatu E-commerce Backend** is a modular, scalable REST API built with Go (Golang) and Gin framework. The project is designed for easy transition to microservices architecture in the future.

### Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL
- **Cache**: Redis
- **ORM**: GORM
- **Authentication**: JWT (JSON Web Tokens)
- **Testing**: Go testing framework + testcontainers

### Key Features

- Multi-tenant support (Seller isolation via `X-Seller-ID`)
- Distributed tracing (via `X-Correlation-ID`)
- Role-based access control (Admin, Seller, Customer)
- Product variant system with dynamic options
- Category hierarchy with attribute inheritance
- Comprehensive test coverage

---

## 🏛️ High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                          │
│  (Mobile App, Web App, Admin Dashboard, Postman)            │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP/REST
┌────────────────────▼────────────────────────────────────────┐
│                   API GATEWAY (Future)                       │
│              (Load Balancer, Rate Limiting)                  │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                     MAIN APPLICATION                         │
│                        (main.go)                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           MIDDLEWARE LAYER                           │   │
│  │  • CorrelationID (Mandatory)                        │   │
│  │  • Logger                                           │   │
│  │  • CORS                                             │   │
│  │  • Authentication (JWT)                             │   │
│  │  • Authorization (Role-based)                       │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              MODULE CONTAINERS                       │   │
│  │  ┌──────┐ ┌─────────┐ ┌───────┐ ┌─────────┐       │   │
│  │  │ User │ │ Product │ │ Order │ │ Payment │ ...   │   │
│  │  └──────┘ └─────────┘ └───────┘ └─────────┘       │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    │                │                │
┌───▼────┐     ┌────▼─────┐    ┌────▼──────┐
│ Redis  │     │PostgreSQL│    │  Storage  │
│ Cache  │     │ Database │    │  (Future) │
└────────┘     └──────────┘    └───────────┘
```

---

## 📁 Directory Structure

```
ecommerce-be/
│
├── main.go                          # 🚀 Application Entry Point
│                                    # Initializes: Logger, DB, Redis, Middleware, Modules
│
├── common/                          # 🔧 Cross-Module Shared Code
│   └── helper/                      # 🛠️ Pure Utility Functions (Future)
├── migrations/                      # 🗄️ Database Schema & Seed Data
├── test/                            # 🧪 Integration Tests
│
├── user/                            # 👤 User Service (Microservice-ready)
├── product/                         # 📦 Product Service (Microservice-ready)
├── order/                           # 🛒 Order Service (Future)
├── payment/                         # 💳 Payment Service (Future)
├── notification/                    # 📧 Notification Service (Future)
│
├── .env                            # Environment variables (gitignored)
├── go.mod                          # Go module dependencies
└── [Documentation files]
```

### 🎯 Key Principles

**Monolithic Structure with Microservices DNA:**

- Each service folder (`user/`, `product/`, `order/`) is **self-contained**
- Services communicate only through well-defined interfaces
- No direct cross-service dependencies (use events/APIs when needed)
- Can extract any service into a separate microservice without code changes

**Current Architecture**: Modular Monolith  
**Future Migration Path**: Extract services → Deploy independently → Service mesh

---

## 🎨 Architectural Patterns

### 1. **Clean Architecture / Layered Architecture**

```
┌─────────────────────────────────────────┐
│         PRESENTATION LAYER              │  (Handlers)
│  - HTTP handlers (controllers)          │  • Parse requests
│  - Request/Response mapping             │  • Call services
│  - Input validation                     │  • Return responses
└──────────────┬──────────────────────────┘
               │ DTOs
┌──────────────▼──────────────────────────┐
│          BUSINESS LOGIC LAYER           │  (Services)
│  - Domain rules                         │  • Business rules
│  - Orchestration                        │  • Transaction management
│  - Complex operations                   │  • Orchestrate repos
└──────────────┬──────────────────────────┘
               │ Domain models
┌──────────────▼──────────────────────────┐
│         DATA ACCESS LAYER               │  (Repositories)
│  - Database queries (GORM)              │  • CRUD operations
│  - Data persistence                     │  • Query builders
│  - Raw SQL (complex queries)            │  • Data mapping
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│            DATABASE LAYER               │
│  PostgreSQL + Redis                     │
└─────────────────────────────────────────┘
```

**Benefits:**

- ✅ **Separation of Concerns**: Each layer has a single responsibility
- ✅ **Testability**: Easy to mock dependencies
- ✅ **Maintainability**: Changes in one layer don't affect others
- ✅ **Scalability**: Easy to extract modules into microservices

### 2. **Repository Pattern**

**Purpose**: Abstract data access logic from business logic

```go
// Interface (contract)
type ProductRepository interface {
    Create(product *entity.Product) error
    FindByID(id uint) (*entity.Product, error)
    Update(product *entity.Product) error
    Delete(id uint) error
}

// Implementation
type ProductRepositoryImpl struct {
    db *gorm.DB
}

func (r *ProductRepositoryImpl) Create(product *entity.Product) error {
    return r.db.Create(product).Error
}
```

**Benefits:**

- ✅ Database agnostic (can switch from PostgreSQL to MongoDB)
- ✅ Easy to test (mock repositories in tests)
- ✅ Centralized data access logic

### 3. **Service Layer Pattern**

**Purpose**: Encapsulate business logic and orchestrate operations

```go
type ProductService interface {
    CreateProduct(req model.CreateProductRequest) (*model.ProductResponse, error)
    GetProductByID(id uint) (*model.ProductResponse, error)
}

type ProductServiceImpl struct {
    productRepo repositories.ProductRepository
    variantRepo repositories.VariantRepository
    cache       cache.Cache
}

func (s *ProductServiceImpl) CreateProduct(req model.CreateProductRequest) (*model.ProductResponse, error) {
    // 1. Validate business rules
    // 2. Create product entity
    // 3. Save to database via repository
    // 4. Invalidate cache
    // 5. Return DTO
}
```

**Benefits:**

- ✅ Business logic is reusable across different handlers
- ✅ Transactions are managed here
- ✅ Complex operations are orchestrated

### 4. **Factory Pattern (Dependency Injection)**

**Purpose**: Create and manage dependencies

```go
// Singleton Factory manages all dependencies
type SingletonFactory struct {
    repoFactory    *RepositoryFactory
    serviceFactory *ServiceFactory
    handlerFactory *HandlerFactory
}

func GetInstance() *SingletonFactory {
    once.Do(func() {
        instance = &SingletonFactory{...}
    })
    return instance
}
```

**Benefits:**

- ✅ Single source of truth for dependencies
- ✅ Lazy initialization
- ✅ Easy to reset for testing

### 5. **Module Pattern**

**Purpose**: Organize code by business domain

Each module (User, Product, Order) is self-contained:

```go
// Module interface
type Module interface {
    RegisterRoutes(router *gin.Engine)
}

// Each module implements this interface
func NewProductModule() *ProductModule {
    factory := singleton.GetInstance()
    return &ProductModule{
        productHandler: factory.GetProductHandler(),
    }
}
```

**Benefits:**

- ✅ Easy to extract into microservices
- ✅ Clear boundaries between domains
- ✅ Parallel development by teams

---

---

## 🏢 Standard Service Structure

Every service (user, product, order, payment, notification) follows the **same pattern**:

```
service_name/                    # e.g., product/, user/, order/
│
├── container.go                 # 📦 Service Registration & Module Wiring
│   └─ Registers all sub-modules (e.g., product has: category, attribute, variant)
│   └─ Initializes factory pattern
│   └─ Mounts routes to Gin router
│
├── entity/                      # 🗃️ Database Models (GORM)
│   └─ Pure data structures representing DB tables
│   └─ Inherit from common.BaseEntity (ID, CreatedAt, UpdatedAt)
│   └─ Define relationships, indexes, constraints
│
├── model/                       # 📋 API Data Transfer Objects (DTOs)
│   └─ Request models (JSON → Go struct)
│   └─ Response models (Go struct → JSON)
│   └─ Validation tags (binding:"required", validate:"email")
│
├── repositories/                # 💾 Data Access Layer
│   ├── *_repository.go         # Interface definition
│   └── *_repository_impl.go    # GORM implementation
│   └─ CRUD operations, complex queries, transactions
│   └─ Returns entities, NOT DTOs
│
├── service/                     # 🧠 Business Logic Layer
│   ├── *_service.go            # Interface definition
│   └── *_service_impl.go       # Business logic implementation
│   └─ Orchestrates repositories
│   └─ Implements business rules & validation
│   └─ Manages transactions
│   └─ Handles caching
│   └─ Returns DTOs, NOT entities
│
├── handlers/                    # 🎯 HTTP Request Handlers (Controllers)
│   └── *_handler.go
│   └─ Parse HTTP requests (JSON, query params, path params)
│   └─ Call service layer
│   └─ Return standardized responses (common.Success/Error)
│   └─ Extract context data (userId, sellerId, correlationId)
│
├── routes/                      # 🛤️ Route Definitions
│   └── *_routes.go
│   └─ Define URL paths
│   └─ Apply middleware (auth, validation)
│   └─ Map routes to handlers
│
├── factory/                     # 🏭 Dependency Injection
│   └── singleton/              # Singleton pattern for complex services
│   └─ Creates & caches dependencies (repos, services, handlers)
│   └─ Used when service has many sub-modules
│
├── mapper/                      # 🔄 Object for Complex Queries query to remove N+1 issues
│
│
├── query/                       # 🔍 Complex Queries
│   └─ Search builders, filters, pagination
│   └─ Raw SQL for performance-critical queries
│
├── errors/                      # ❌ Service-Specific Errors
│   └─ Custom error types
│   └─ Error codes & messages
│
├── utils/                       # 🛠️ Service Utilities
│   └─ Helper functions specific to this service
│
└── validator/                   # ✅ Business Validation
    └─ Complex validation logic beyond struct tags
```

### 📐 Layer Responsibilities

| Layer          | Responsibility  | What Goes Here                                  | What Doesn't               |
| -------------- | --------------- | ----------------------------------------------- | -------------------------- |
| **Entity**     | Database schema | Table structure, relationships, indexes         | Business logic, validation |
| **Model**      | API contract    | Request/response structure, JSON tags           | Database details           |
| **Repository** | Data access     | CRUD, queries, raw SQL                          | Business rules, validation |
| **Service**    | Business logic  | Rules, orchestration, transactions              | HTTP handling, DB queries  |
| **Handler**    | HTTP handling   | Parse requests, call services, return responses | Business logic, DB access  |
| **Routes**     | Routing         | URL mapping, middleware                         | Business logic             |

---

## 🔧 Common Module (Shared Code)

The `common/` folder contains **cross-cutting concerns** used by ALL services:

```
common/
│
├── auth/                        # 🔐 Authentication & Authorization
│   ├── jwt.go                  # JWT token generation & parsing
│   ├── auth_helpers.go         # Extract user/role from JWT context
│   ├── auth_middleware.go      # JWT validation middleware
│   └── seller_validation.go    # Seller-specific validation logic
│
├── cache/                       # 🚀 Redis Caching
│   ├── redis.go                # Redis client initialization
│   └── cache_invalidation.go   # Cache invalidation patterns
│
├── constants/                   # 📌 Application Constants
│   ├── role_constants.go       # User roles (Admin, Seller, Customer)
│   ├── auth_constants.go       # JWT expiry, secrets
│   ├── cache_constants.go      # Cache keys & TTL
│   ├── error_constant.go       # Standard error codes
│   └── redis_constants.go      # Redis configuration
│
├── db/                          # 🗄️ Database Utilities
│   ├── db.go                   # PostgreSQL connection (GORM)
│   ├── base_entity.go          # Common entity fields (ID, timestamps)
│   └── string_array.go         # Custom GORM types
│
├── error/                       # ❌ Centralized Error Handling
│   ├── app_error.go            # Custom error types (AppError)
│   └── common_errors.go        # Reusable errors (NotFound, Unauthorized)
│
├── handler/                     # 🎯 Base Handler Utilities
│   └── base_handler.go         # Common handler methods
│
├── log/                         # 📝 Centralized Logging
│   └── logger.go               # Logrus-based structured logging
│
├── middleware/                  # 🛡️ HTTP Middleware
│   ├── middleware.go           # CorrelationID (mandatory), Logger, CORS
│   ├── public_api_middleware.go # Public API + Seller isolation
│   ├── admin_middleware.go     # Admin role check
│   ├── seller_middleware.go    # Seller role check
│   └── customer_middleware.go  # Customer role check
│
├── validator/                   # ✅ Request Validation
│   ├── request_validator.go    # Validation helper functions
│   └── doc.go                  # Documentation
│
├── container.go                 # 📦 Container interface for modules
└── response.go                  # 📤 Standardized API responses (Success/Error)
```

### Key Functions in Common

**Authentication (`auth/`)**:

- `GenerateToken(user)` - Create JWT token
- `ParseToken(tokenString)` - Validate & extract claims
- `GetUserIDFromContext(c)` - Extract authenticated user ID
- `GetSellerIDFromContext(c)` - Extract seller ID for multi-tenancy

**Caching (`cache/`)**:

- `Get(key)` - Retrieve from Redis
- `Set(key, value, ttl)` - Store in Redis
- `Delete(key)` - Invalidate cache
- `InvalidatePattern(pattern)` - Bulk invalidation

**Middleware (`middleware/`)**:

- `CorrelationID()` - MANDATORY for all requests
- `Logger()` - Log all requests/responses
- `CORS()` - Handle cross-origin requests
- `AuthMiddleware()` - Validate JWT
- `SellerAuth()` - Check seller role
- `PublicAPIAuth()` - Extract seller ID from header OR JWT

**Response (`response.go`)**:

- `Success(c, statusCode, data, message)` - Standardized success response
- `Error(c, statusCode, message)` - Standardized error response

### Helper Submodule (Future)

The `common/helper/` folder will contain **pure utility functions** with no external dependencies:

```
common/helper/
│
├── string_helper.go             # String manipulation (slugify, sanitize)
├── time_helper.go               # Date/time utilities
├── pagination_helper.go         # Pagination calculation
├── file_helper.go               # File upload/download utilities
├── encryption_helper.go         # Encryption/decryption (non-auth)
└── formatter_helper.go          # Format currency, numbers, etc.
```

**Difference between `common/` modules and `common/helper/`:**

- `common/auth`, `common/cache`, etc. → Have external dependencies (DB, Redis, Logger)
- `common/helper/` → Pure functions, no dependencies, easily testable, stateless

---

## 🔄 Request Flow

### Typical API Request Lifecycle

```
1. CLIENT
   HTTP Request (POST/GET/PUT/DELETE)
   Headers: Authorization, X-Correlation-ID, X-Seller-ID (if public)
   Body: JSON request payload
   │
   ▼
2. MIDDLEWARE CHAIN
   ├─ CorrelationID() ──► Extract/validate correlation ID (MANDATORY)
   ├─ Logger() ──────────► Log incoming request
   ├─ CORS() ────────────► Handle cross-origin requests
   └─ Auth/Role Check() ─► Validate JWT & check permissions
   │
   ▼
3. ROUTER
   Match route pattern → Invoke corresponding handler
   │
   ▼
4. HANDLER (Presentation Layer)
   ├─ Parse & validate request (JSON → DTO)
   ├─ Extract context data (userId, sellerId, etc.)
   ├─ Call service layer with DTO
   │
   ▼
5. SERVICE (Business Logic Layer)
   ├─ Validate business rules
   ├─ Orchestrate multiple repositories
   ├─ Manage transactions
   ├─ Update cache
   │
   ▼
6. REPOSITORY (Data Access Layer)
   ├─ Execute database queries (GORM)
   ├─ Map entities
   ├─ Handle DB errors
   │
   ▼
7. DATABASE
   ├─ Execute SQL queries
   ├─ Return results
   │
   ▼
8. RESPONSE FLOW (Reverse)
   Repository → Service → Handler
   │
   ▼
9. HANDLER
   ├─ Map entity → DTO
   ├─ Build standardized response
   └─ Return JSON: { success: bool, data: {...}, message: "..." }
   │
   ▼
10. CLIENT
    Receives HTTP response with status code & JSON body
```

**Key Points:**

- Every request MUST have `X-Correlation-ID` header
- Public APIs require `X-Seller-ID` for multi-tenancy
- Handlers never contain business logic
- Services never handle HTTP concerns
- Repositories only interact with database

---

## 🔐 Authentication & Authorization

### 1. **Correlation ID** (Mandatory for ALL requests)

```http
X-Correlation-ID: 550e8400-e29b-41d4-a716-446655440000
```

- **Purpose**: Distributed tracing and debugging
- **Middleware**: `middleware.CorrelationID()`
- **Behavior**: If not provided, request is rejected with `400 Bad Request`

### 2. **Seller ID** (Mandatory for public APIs)

```http
X-Seller-ID: 2
```

- **Purpose**: Multi-tenant data isolation
- **Middleware**: `middleware.PublicAPIAuth()`
- **Behavior**:
  - Public routes (GET /products, GET /categories) REQUIRE seller ID
  - If JWT token is present, seller ID is extracted from token

### 3. **JWT Authentication**

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

- **Token Structure**:

```json
{
  "userId": 123,
  "email": "seller@example.com",
  "role": "seller",
  "sellerId": 2,
  "exp": 1700000000
}
```

- **Middleware Hierarchy**:
  - `middleware.AuthMiddleware()` → Validate JWT
  - `middleware.CustomerAuth()` → Check role = "customer"
  - `middleware.SellerAuth()` → Check role = "seller"
  - `middleware.AdminAuth()` → Check role = "admin"

### 4. **Role-Based Access Control**

**Available Roles:**

- `admin` - Full system access
- `seller` - Manage own products/inventory
- `customer` - Browse and purchase

**Access Patterns:**

- Public endpoints (browsing) → Require `X-Seller-ID` for multi-tenancy
- Authenticated endpoints → Require valid JWT token
- Role-specific endpoints → Additional role check in middleware
- Admin endpoints → Restricted to admin role only

Each service defines its own authorization rules based on business requirements.

---

## 🧪 Testing Strategy

### Current Approach: **Test-Driven Development (TDD) with Integration Tests**

We prioritize **integration tests** over unit tests. Every API endpoint must have full test coverage.

#### Why Integration Tests?

✅ **Pros**:

- Tests real behavior (DB, Redis, middleware, full request flow)
- Catches integration issues early
- Validates API contracts
- No mocking complexity
- Confidence in deployments

⚠️ **Unit Tests**: Optional (developer's choice)  
Use unit tests for complex algorithms or business logic, but not mandatory.

---

### Test Structure

```
test/
└── integration/
    ├── setup/                   # 🏗️ Test Infrastructure
    │   ├── containers.go       # Testcontainers (PostgreSQL, Redis)
    │   ├── server.go           # Test server setup
    │   └── cleanup.go          # Resource cleanup
    │
    ├── helpers/                 # 🛠️ Test Utilities
    │   ├── api_client.go       # HTTP client wrapper
    │   ├── auth_helper.go      # Login, JWT generation
    │   ├── assertion.go        # Custom assertions
    │   └── data_builder.go     # Test data builders
    │
    ├── user/                    # 👤 User Service Tests
    │   ├── auth_test.go        # Login, register, logout
    │   ├── profile_test.go     # Get/update profile
    │   └── address_test.go     # CRUD addresses
    │
    ├── product/                 # 📦 Product Service Tests
    │   ├── category/
    │   │   ├── create_category_test.go
    │   │   ├── get_all_categories_test.go
    │   │   └── delete_category_test.go
    │   ├── product/
    │   │   ├── create_product_test.go
    │   │   ├── get_product_test.go
    │   │   └── search_products_test.go
    │   ├── variant/
    │   └── product_option/
    │
    └── order/                   # 🛒 Order Service Tests (Future)
```

---

### Test Lifecycle

```
┌─────────────────────────────────────────┐
│  1. Start Testcontainers                │
│     - PostgreSQL container              │
│     - Redis container                   │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  2. Run Migrations                      │
│     - Create tables                     │
│     - Apply schema                      │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  3. Seed Test Data                      │
│     - Admin user                        │
│     - Seller user                       │
│     - Sample categories                 │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  4. Initialize Test Server              │
│     - Create Gin router                 │
│     - Apply middleware                  │
│     - Register routes                   │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  5. Run Tests                           │
│     - Execute test cases                │
│     - Validate responses                │
│     - Check database state              │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  6. Cleanup                             │
│     - Stop containers                   │
│     - Clear test data                   │
└─────────────────────────────────────────┘
```

---

### Example Integration Test

```go
func TestCreateProduct(t *testing.T) {
    // 1. Setup containers & server
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    // 2. Authenticate as seller
    token := helpers.LoginAsSeller(t, client)
    client.SetToken(token)

    // 3. Prepare request
    req := map[string]any{
        "name": "iPhone 15 Pro",
        "categoryId": 1,
        "basePrice": 99999,
        "description": "Latest iPhone",
    }

    // 4. Make API call
    w := client.Post(t, "/api/products", req,
        map[string]string{
            "X-Correlation-ID": "test-123",
            "X-Seller-ID": "2",
        })

    // 5. Assert response
    assert.Equal(t, http.StatusCreated, w.Code)

    var response map[string]any
    json.Unmarshal(w.Body.Bytes(), &response)

    assert.True(t, response["success"].(bool))
    assert.NotNil(t, response["data"])

    // 6. Verify database state
    var product entity.Product
    containers.DB.First(&product, response["data"].(map[string]any)["id"])
    assert.Equal(t, "iPhone 15 Pro", product.Name)
    assert.Equal(t, uint(2), product.SellerID)
}
```

---

### Test Coverage Requirements

| Endpoint Type                           | Coverage Required                          |
| --------------------------------------- | ------------------------------------------ |
| **Public APIs** (GET /products)         | ✅ Happy path + seller isolation           |
| **Authenticated APIs** (POST /products) | ✅ Happy path + auth failures + validation |
| **Admin APIs** (DELETE /users/:id)      | ✅ Happy path + role checks                |
| **Edge Cases**                          | ✅ Invalid IDs, missing fields, duplicates |

**Mandatory Test Scenarios:**

1. ✅ Happy path (valid request → success response)
2. ✅ Authentication (missing token, invalid token, expired token)
3. ✅ Authorization (wrong role, wrong seller)
4. ✅ Validation (missing required fields, invalid formats)
5. ✅ Edge cases (not found, duplicates, constraints)
6. ✅ Correlation ID (missing, invalid)

---

### Test Helpers

**Predefined Test Users** (from `seeds/001_seed_user_data.sql`):

```go
var TestUsers = struct {
    Admin struct {
        Email    string
        Password string
        ID       uint
    }
    Seller struct {
        Email    string
        Password string
        ID       uint
        SellerID uint
    }
    Customer struct {
        Email    string
        Password string
        ID       uint
    }
}{
    Admin:    {Email: "admin@example.com", Password: "admin123", ID: 1},
    Seller:   {Email: "jane.merchant@example.com", Password: "seller123", ID: 2, SellerID: 2},
    Customer: {Email: "alice.j@example.com", Password: "customer123", ID: 3},
}
```

**Helper Functions**:

```go
// Login helpers
helpers.LoginAsAdmin(t, client) string
helpers.LoginAsSeller(t, client) string
helpers.LoginAsCustomer(t, client) string

// Request builders
helpers.CreateCategory(t, client, name string) uint
helpers.CreateProduct(t, client, data map[string]any) uint

// Assertions
helpers.AssertSuccess(t, w, expectedCode int)
helpers.AssertError(t, w, expectedCode int, expectedMessage string)
helpers.AssertSellerIsolation(t, db, productID uint, sellerID uint)
```

---

### Running Tests

```bash
# Run all integration tests
go test ./test/integration/... -v

# Run specific service tests
go test ./test/integration/product/... -v

# Run specific test
go test ./test/integration/product/product -run TestCreateProduct -v

# Run with coverage
go test ./test/integration/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## 📚 Development Guidelines

### 1. **Adding a New Service Module**

When adding a new business domain (e.g., `review`, `wishlist`, `notification`):

```bash
# Create standard service structure
mkdir -p new_service/{entity,model,repositories,service,handlers,routes}

# Create container
touch new_service/container.go

# Register in main.go
# Add: _ = new_service.NewContainer(router)
```

**container.go** template:

```go
package new_service

import "github.com/gin-gonic/gin"

func NewContainer(router *gin.Engine) *common.Container {
    // Initialize factory (if needed)
    // Initialize handlers
    // Register routes

    return &common.Container{
        Modules: []common.Module{NewModule()},
    }
}
```

---

### 2. **Adding a New Endpoint (Step-by-Step)**

Follow this order: **Entity → Repository → Service → Handler → Route → Test**

#### Step 1: Define Entity (`entity/`)

```go
type YourEntity struct {
    db.BaseEntity
    Name        string `gorm:"size:255;not null"`
    Description string `gorm:"type:text"`
    // Add your fields with GORM tags
}
```

#### Step 2: Create Repository (`repositories/`)

```go
// Interface
type YourRepository interface {
    Create(entity *entity.YourEntity) error
    FindByID(id uint) (*entity.YourEntity, error)
    // Add CRUD methods
}

// Implementation
type YourRepositoryImpl struct {
    db *gorm.DB
}

func (r *YourRepositoryImpl) Create(entity *entity.YourEntity) error {
    return r.db.Create(entity).Error
}
```

#### Step 3: Create Service (`service/`)

```go
// Interface
type YourService interface {
    Create(req model.CreateRequest) (*model.Response, error)
}

// Implementation
type YourServiceImpl struct {
    repo repositories.YourRepository
    // Inject dependencies
}

func (s *YourServiceImpl) Create(req model.CreateRequest) (*model.Response, error) {
    // 1. Validate business rules
    // 2. Create entity
    // 3. Save to DB via repository
    // 4. Invalidate cache (if needed)
    // 5. Return DTO
}
```

#### Step 4: Create Handler (`handlers/`)

```go
func (h *YourHandler) Create(c *gin.Context) {
    var req model.CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Extract context data if needed
    userID := auth.GetUserIDFromContext(c)

    response, err := h.service.Create(req)
    if err != nil {
        common.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    common.Success(c, http.StatusCreated, response, "Created successfully")
}
```

#### Step 5: Register Route (`routes/`)

```go
func (m *YourModule) RegisterRoutes(router *gin.Engine) {
    routes := router.Group("/api/your-resource")
    {
        routes.POST("", middleware.AuthMiddleware(), m.handler.Create)
        routes.GET("/:id", middleware.PublicAPIAuth(), m.handler.GetByID)
        // Add more routes
    }
}
```

#### Step 6: Write Integration Test (`test/integration/`)

````go
func TestCreateYourEntity(t *testing.T) {
    // Setup
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)
    token := helpers.LoginAs<Role>(t, client) // Admin/Seller/Customer
    client.SetToken(token)

    // Prepare request
    req := map[string]any{
        "name": "Test Name",
        // Add test data
    }

    // Execute
    w := client.Post(t, "/api/your-resource", req, helpers.CorrelationHeaders())

    // Assert
    helpers.AssertSuccess(t, w, http.StatusCreated)
}
```---

### 3. **Code Conventions**

#### Naming Conventions

```go
// Interfaces: Noun
type EntityService interface {}
type EntityRepository interface {}

// Implementations: Noun + Impl
type EntityServiceImpl struct {}
type EntityRepositoryImpl struct {}

// Methods: Verb + Noun
func CreateEntity() {}
func GetEntityByID() {}
func UpdateEntity() {}
func DeleteEntity() {}

// Files: snake_case
entity_service.go
entity_service_impl.go
create_entity_test.go
````

#### Error Handling

```go
// Always check errors
if err != nil {
    logger.Error("Operation failed", err)
    return nil, err
}

// Use custom errors for business logic
if entity == nil {
    return nil, errors.EntityNotFound
}

// Wrap errors for context
if err := r.db.Create(&product).Error; err != nil {
    return fmt.Errorf("failed to save product: %w", err)
}
```

#### Logging

````go
// Use structured logging with context
logger.Info("Operation successful",
    "entityId", entity.ID,
    "context", contextInfo)

logger.Error("Operation failed",
    "error", err,
    "operation", "create")

// Correlation ID is automatically added by middleware
```---

### 4. **Mandatory Headers**

Every API request MUST include:

```go
headers := map[string]string{
    "X-Correlation-ID": "uuid-v4-here",        // Mandatory for ALL
    "X-Seller-ID": "2",                        // For public APIs
    "Authorization": "Bearer jwt-token-here",  // For authenticated APIs
    "Content-Type": "application/json",
}
````

---

### 5. **Database Migrations**

Add migration when changing schema:

```bash
# Create new migration file
touch migrations/00X_add_your_table.sql
```

```sql
-- migrations/00X_add_your_table.sql
CREATE TABLE your_table (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    -- Add your columns
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Add indexes for foreign keys and frequently queried columns
CREATE INDEX idx_your_table_name ON your_table(name);
```

Run migration:

```bash
cd migrations && ./run_migrations.sh
```

---

### 6. **Git Workflow**

```bash
# 1. Create feature branch from main
git checkout main
git pull origin main
git checkout -b feature/add-reviews

# 2. Make changes (follow TDD)
# - Write integration test first
# - Implement endpoint
# - Run tests

# 3. Commit with conventional commits
git add .
git commit -m "feat(module): add new endpoint

- Add entity and migration
- Implement service and repository
- Add handler
- Add integration tests with full coverage

Closes #XXX"

# 4. Push and create PR
git push origin feature/your-feature

# 5. Ensure PR checklist:
# ✅ All tests pass
# ✅ Integration tests cover all scenarios
# ✅ Code follows conventions
# ✅ Migration added (if schema change)
# ✅ Postman collection updated
```

**Commit Message Format:**

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

---

### 7. **Common Pitfalls & Best Practices**

| ❌ Don't                        | ✅ Do                            |
| ------------------------------- | -------------------------------- |
| Put business logic in handlers  | Keep handlers thin, use services |
| Query DB directly from handlers | Use repository layer             |
| Return entities to client       | Convert to DTOs                  |
| Use global variables            | Inject dependencies              |
| Forget correlation ID           | Always include in headers        |
| Skip integration tests          | Test every endpoint              |
| Hardcode configuration          | Use environment variables        |
| Ignore seller isolation         | Always filter by seller_id       |
| Return raw errors               | Use standardized error responses |

**Best Practices:**

1. ✅ One endpoint = One integration test minimum
2. ✅ Always validate seller ownership (multi-tenancy)
3. ✅ Use transactions for multi-step operations
4. ✅ Invalidate cache after mutations
5. ✅ Log important operations with context
6. ✅ Handle edge cases (not found, duplicates)
7. ✅ Use pagination for list endpoints
8. ✅ Add indexes for foreign keys

---

## 🚀 Roadmap & Future Architecture

### Phase 1: Monolith (Current) ✅

- All services in one codebase
- Single deployment
- Shared database
- Easy development & debugging

### Phase 2: Microservices Transition (Planned)

When traffic/team grows, extract services:

```
┌─────────────────────────────────────────────┐
│           API Gateway                        │
│  (Load Balancer, Rate Limiting, Auth)       │
└────┬──────────┬──────────┬─────────┬────────┘
     │          │          │         │
┌────▼────┐ ┌──▼──────┐ ┌─▼──────┐ ┌▼────────┐
│Service A│ │Service B│ │Service C│ │Service D│
└────┬────┘ └────┬────┘ └────┬───┘ └────┬────┘
     │           │           │          │
┌────▼────┐ ┌───▼─────┐ ┌───▼────┐ ┌──▼──────┐
│   DB A  │ │  DB B   │ │  DB C  │ │  DB D   │
└─────────┘ └─────────┘ └────────┘ └─────────┘
```

**Migration Strategy:**

1. Extract service into separate repo
2. Create API endpoints for cross-service communication
3. Migrate database schema
4. Deploy independently
5. Use service mesh (Istio/Linkerd) for communication

---

## 🎓 Learning Path for New Developers

### Week 1: Setup & Basics

- [ ] Clone repo & setup local environment
- [ ] Run migrations & seed data
- [ ] Run integration tests
- [ ] Import Postman collection & test APIs
- [ ] Read this ARCHITECTURE.md

### Week 2: Code Walkthrough

- [ ] Understand `main.go` (entry point)
- [ ] Study `common/` module (auth, middleware, DB)
- [ ] Trace one API request end-to-end (User Login)
- [ ] Understand factory pattern in `product/factory/`

### Week 3: First Contribution

- [ ] Pick a simple task (add new field to existing entity)
- [ ] Write integration test first (TDD)
- [ ] Implement the feature
- [ ] Create PR with proper commit message

### Week 4: Complex Feature

- [ ] Add a new endpoint with full flow
- [ ] Entity → Repository → Service → Handler → Route → Test
- [ ] Handle edge cases & validation
- [ ] Update Postman collection

---

## 📞 Getting Help

### When Stuck:

1. **Check Tests**: `test/integration/` has examples for every pattern
2. **Review Similar Code**: Find a similar endpoint and follow its structure
3. **Read PRDs**: `backend-prd.md`, `PRODUCT_SERVICE_PRD.md`
4. **Check Logs**: Correlation ID helps trace requests
5. **Ask Team**: Use correlation ID when reporting issues

### Debugging Tips:

```bash
# Check logs for correlation ID
grep "correlation-id-here" logs/app.log

# Run specific test
go test -v ./test/integration/product/product -run TestCreateProduct

# Check database state
psql -U postgres -d ecommerce -c "SELECT * FROM product WHERE id = 123;"

# Check Redis cache
redis-cli GET "product:123"
```

---

## 📚 Additional Resources

- **Go Best Practices**: https://golang.org/doc/effective_go
- **Gin Framework**: https://gin-gonic.com/docs/
- **GORM Documentation**: https://gorm.io/docs/
- **Testcontainers**: https://golang.testcontainers.org/
- **Clean Architecture**: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html

---

## 📝 Quick Reference

### Common Commands

```bash
# Run app
go run main.go

# Run all tests
go test ./test/integration/... -v

# Format code
gofumpt -l -w .
golines -w .

# Run migrations
cd migrations && ./run_migrations.sh

# Build
go build -o bin/app main.go
```

### Environment Variables

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=ecommerce
REDIS_HOST=localhost
REDIS_PORT=6379
JWT_SECRET=your-secret-key
PORT=8080
```

---

**Happy Coding! 🎉**  
_Remember: Write tests first, keep services independent, and always add correlation ID!_
