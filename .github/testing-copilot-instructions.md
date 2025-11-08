# GitHub Copilot Instructions - Go E-commerce Backend

## Testing Philosophy & Quality Standards

### Core Testing Principles

**Quality Over Pass Rate**: Write tests to validate code behavior and identify potential issues, not just to make everything pass.

**Test Failures Are Valid Outcomes**: If there's a logic issue in the implementation, it's completely fine — and expected — for the test to fail. That's the purpose of testing.

**Real Issues Should Fail Tests**: Tests should catch real implementation bugs. A failing test indicates a genuine problem that must be fixed in the production code, not the test.

And we are not asserting the response message we should the assert the https code always. because the message can be changed later but the code should remain same for the particular scenario. and if nessory then we can assert the code because the messgae and code is one to one mapping so code will never change for the particular scenario

### Test Design Guidelines

#### 1. **Write Meaningful Assertions**

Examples:

```go
// ✅ GOOD: Clear, specific assertion with descriptive message
assert.Equal(t, http.StatusOK, w.Code, "Response status should be 200 OK")
assert.NotNil(t, variant["id"], "Variant ID should not be nil")
assert.Len(t, selectedOptions, 2, "Should have 2 selected options")

// ❌ BAD: Generic assertion without context
assert.True(t, w.Code == 200)
```

#### 2. **Test Real Behavior, Not Mocked Behavior**

Examples:

```go
// ✅ GOOD: Test actual database interaction with real containers
func TestCreateCategory(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    containers.RunAllMigrations(t)
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

    // Test with real database, real Redis, real HTTP handlers
    client := helpers.NewAPIClient(server)
    response := client.Post(t, "/api/categories", requestBody)

    // Validate actual response from real service
    assert.Equal(t, http.StatusCreated, response.Code)
}

// ❌ BAD: Over-mocking that doesn't test real behavior
func TestCreateCategory_WithMock(t *testing.T) {
    mockService := &MockCategoryService{}
    mockService.On("Create").Return(mockData, nil)
    // This only validates the mock setup, not real code
}
```

#### 3. **Use Integration Tests with TestContainers**

- Integration tests MUST use real PostgreSQL database via TestContainers
- Integration tests MUST use real Redis cache via TestContainers
- Integration tests MUST use real service layers and repositories
- Only mock external dependencies (payment gateways, third-party APIs)
- Example: All tests in `test/integration/` use real database and Redis

#### 4. **Test Both Happy Path and Edge Cases**

Examples:

```go
// Test happy path
t.Run("Success - Basic variant creation with minimal fields", func(t *testing.T) {
    // Test implementation
})

// Test edge cases
t.Run("Error - Missing required SKU field", func(t *testing.T) {
    // Test implementation
})

t.Run("Error - Duplicate SKU returns conflict", func(t *testing.T) {
    // Test implementation
})

t.Run("Error - Non-existent product returns not found", func(t *testing.T) {
    // Test implementation
})
```

#### 5. **Use Descriptive Test Names**

Format: `Test[FunctionName]` for top-level, then `t.Run("[Status] - [Scenario]")` for subtests

Examples:

```go
func TestCreateVariant(t *testing.T) {
    t.Run("Success - Basic variant creation with minimal fields", func(t *testing.T) {
        // Test implementation
    })

    t.Run("Success - Variant creation with all fields populated", func(t *testing.T) {
        // Test implementation
    })

    t.Run("Error - Missing required SKU field", func(t *testing.T) {
        // Test implementation
    })
}
```

**Naming Conventions:**

- `Success - [scenario description]` for successful operations
- `Error - [error condition]` for error cases
- `Admin [action]` / `Seller [action]` / `Customer [action]` for role-based tests

#### 6. **Document Test Intent with Comments**

Examples:

```go
// TestCreateVariant validates variant creation functionality
// including validation, option selection, and database persistence
func TestCreateVariant(t *testing.T) {
    // Setup test containers
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    t.Run("Success - Variant with size and color options", func(t *testing.T) {
        // Login as seller (check seed data for available seller accounts)
        sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)

        // Use product from seed data that has options configured
        // Check migrations/seeds/002_seed_product_data.sql for available products
        var productID uint
        containers.DB.Model(&entity.Product{}).
            Where("seller_id = ?", sellerID).
            Select("id").
            First(&productID)

        // Test implementation...
    })
}
```

#### 7. **Validate Complete Object State**

Examples:

```go
// ✅ GOOD: Validate all important fields
variant := helpers.GetResponseData(t, response, "variant")
assert.NotNil(t, variant["id"], "Variant should have ID")
assert.Equal(t, float64(productID), variant["productId"], "Product ID should match")
assert.Equal(t, expectedSKU, variant["sku"], "SKU should match")
assert.Equal(t, expectedPrice, variant["price"], "Price should match")
assert.NotNil(t, variant["createdAt"], "CreatedAt should be set")
assert.NotNil(t, variant["updatedAt"], "UpdatedAt should be set")

// Check nested objects
selectedOptions, ok := variant["selectedOptions"].([]interface{})
assert.True(t, ok, "selectedOptions should be an array")
assert.Len(t, selectedOptions, 2, "Should have 2 selected options")

// ❌ BAD: Only validate one field
assert.Equal(t, "NIKE-TSHIRT-NAVY-XL", variant["sku"])
```

#### 8. **Use Test Setup Properly**

Examples:

```go
func TestCreateCategory(t *testing.T) {
    // Setup test containers (PostgreSQL + Redis)
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    // Run migrations to set up schema
    containers.RunAllMigrations(t)

    // Seed data if needed
    containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
    containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")

    // Setup test server with real database and Redis
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

    // Create API client
    client := helpers.NewAPIClient(server)

    t.Run("Test scenario", func(t *testing.T) {
        // Test implementation
    })
}
```

### Test Coverage Goals

#### Required Test Scenarios

1. **Unit Tests** - Test individual components in isolation

   - Service layer logic (business rules)
   - Repository methods (data access)
   - Validators and mappers
   - Utility functions
   - Error handling

2. **Integration Tests** - Test component interactions

   - REST API endpoints with real database
   - Service layer with real repository
   - Database migrations
   - Authentication and authorization
   - Cache invalidation

3. **Validation Tests** - Test business rules
   - Input validation (required fields, formats)
   - Business logic constraints
   - Data integrity rules
   - Role-based access control

### Common Testing Patterns in This Project

#### Pattern 1: REST API Integration Tests with TestContainers

```go
func TestCreateProduct(t *testing.T) {
    // Setup: Start PostgreSQL and Redis containers
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    // Setup: Run migrations and seeds
    containers.RunAllMigrations(t)
    containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

    // Setup: Initialize server with real dependencies
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    t.Run("Success - Create product with valid data", func(t *testing.T) {
        // Given: Authenticate as seller
        sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
        client.SetToken(sellerToken)

        // When: Create product
        requestBody := map[string]interface{}{
            "name":        "Premium Laptop",
            "description": "High-performance laptop",
            "price":       1299.99,
        }
        w := client.Post(t, "/api/products", requestBody)

        // Then: Validate response
        response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
        product := helpers.GetResponseData(t, response, "product")

        // Then: Validate all important fields
        assert.NotNil(t, product["id"], "Product should have ID")
        assert.Equal(t, "Premium Laptop", product["name"], "Name should match")
        assert.Equal(t, 1299.99, product["price"], "Price should match")
    })
}
```

#### Pattern 2: Database Migration Tests

```go
func TestMigrations(t *testing.T) {
    // Given: Fresh database container
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    // When: Run all migrations
    containers.RunAllMigrations(t)

    // Then: Verify schema exists
    var tableCount int
    err := containers.DB.Raw(`
        SELECT COUNT(*) FROM information_schema.tables
        WHERE table_schema = 'public'
    `).Scan(&tableCount).Error

    assert.NoError(t, err, "Query should succeed")
    assert.Greater(t, tableCount, 0, "Should have tables created")

    // Then: Verify specific tables exist
    var exists bool
    err = containers.DB.Raw(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_name = 'products'
        )
    `).Scan(&exists).Error

    assert.NoError(t, err, "Query should succeed")
    assert.True(t, exists, "Products table should exist")
}
```

#### Pattern 3: Authentication and Authorization Tests

```go
func TestAuthenticationFlow(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    containers.RunAllMigrations(t)
    containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    t.Run("Error - Unauthorized access without token", func(t *testing.T) {
        // When: Access protected endpoint without token
        w := client.Get(t, "/api/products")

        // Then: Should return unauthorized
        helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
    })

    t.Run("Error - Seller cannot access admin endpoints", func(t *testing.T) {
        // Given: Login as seller
        sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
        client.SetToken(sellerToken)

        // When: Try to access admin endpoint
        w := client.Get(t, "/api/admin/users")

        // Then: Should return forbidden
        helpers.AssertErrorResponse(t, w, http.StatusForbidden)
    })
}
```

#### Pattern 4: Error Handling Tests

```go
func TestErrorScenarios(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    containers.RunAllMigrations(t)
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    t.Run("Error - Invalid request body returns bad request", func(t *testing.T) {
        adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
        client.SetToken(adminToken)

        // When: Send invalid data
        requestBody := map[string]interface{}{
            "name": "",  // Invalid: empty name
        }
        w := client.Post(t, "/api/categories", requestBody)

        // Then: Validate error response
        response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
        assert.Contains(t, response["error"], "name", "Error should mention name field")
    })

    t.Run("Error - Duplicate resource returns conflict", func(t *testing.T) {
        // First creation succeeds
        w1 := client.Post(t, "/api/categories", requestBody)
        helpers.AssertSuccessResponse(t, w1, http.StatusCreated)

        // Second creation with same data fails
        w2 := client.Post(t, "/api/categories", requestBody)
        helpers.AssertErrorResponse(t, w2, http.StatusConflict)
    })
}
```

#### Pattern 5: Cache Invalidation Tests

```go
func TestCacheInvalidation(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    containers.RunAllMigrations(t)
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    t.Run("Cache invalidation on category update", func(t *testing.T) {
        adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
        client.SetToken(adminToken)

        // Create category
        createBody := map[string]interface{}{"name": "Electronics"}
        w1 := client.Post(t, "/api/categories", createBody)
        response1 := helpers.AssertSuccessResponse(t, w1, http.StatusCreated)
        category := helpers.GetResponseData(t, response1, "category")
        categoryID := int(category["id"].(float64))

        // Get category (caches result)
        url := fmt.Sprintf("/api/categories/%d", categoryID)
        w2 := client.Get(t, url)
        helpers.AssertSuccessResponse(t, w2, http.StatusOK)

        // Update category (should invalidate cache)
        updateBody := map[string]interface{}{"name": "Electronics Updated"}
        w3 := client.Put(t, url, updateBody)
        helpers.AssertSuccessResponse(t, w3, http.StatusOK)

        // Get category again (should fetch fresh data)
        w4 := client.Get(t, url)
        response4 := helpers.AssertSuccessResponse(t, w4, http.StatusOK)
        updatedCategory := helpers.GetResponseData(t, response4, "category")

        assert.Equal(t, "Electronics Updated", updatedCategory["name"],
            "Should return updated name from database, not cache")
    })
}
```

### Test Execution Standards

1. **Always run tests after code changes** to validate behavior
2. **Check test output carefully** - look for actual failures, not just exit codes
3. **Read failure messages** - they contain valuable debugging information
4. **Fix implementation bugs**, not tests (unless test logic is genuinely wrong)
5. **Run full test suite** before committing code

**Running Tests:**

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test file
go test -v ./test/integration/product/variant/create_variant_test.go

# Run specific test function
go test -v -run TestCreateVariant ./test/integration/product/variant

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Test Quality Checklist

Before considering a test complete, verify:

- [ ] Test name clearly describes what is being tested
- [ ] Test has proper documentation comments
- [ ] Test validates actual behavior, not just mocks
- [ ] Assertions have descriptive failure messages (third argument in `assert.*`)
- [ ] Test covers both happy path and edge cases
- [ ] Test setup is clean and reusable
- [ ] Test is independent (doesn't rely on execution order)
- [ ] Test uses appropriate test data (seeded users, products)
- [ ] Test validates all important fields in response objects
- [ ] Test will fail if implementation has a bug
- [ ] Test properly cleans up containers with `defer containers.Cleanup(t)`

## Project-Specific Testing Context

> **Note**: This section provides general patterns and principles. Specific service names, test data, and directory structures will evolve as the project grows. Always refer to the actual codebase structure and seed files rather than relying on hardcoded examples in this document.

### Application Architecture

This is a modular monolithic Go application designed for future microservices:

- **Modular Structure**: Each business domain (e.g., user, product, order) has its own module
- **Common Layer**: Shared utilities, middleware, database, cache, authentication
- **Testing Strategy**: Integration tests with TestContainers (PostgreSQL + Redis)

### Test Data Management

**IMPORTANT: Always use seeded data from migration scripts**

- Test data is defined in `migrations/seeds/*.sql` files
- Reference seed files when writing tests (e.g., `containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")`)
- Check seed scripts to understand available test users, products, and their relationships
- Seed data may change over time - tests should be flexible and query data dynamically when possible

**Discovering Test Data:**

1. **Before writing tests**, always check `migrations/seeds/*.sql` files to understand:

   - What users are available (admin, seller, customer, etc.)
   - What products/categories exist
   - Relationships between entities (which seller owns which products)
   - Available test credentials

2. **Query data dynamically** in tests rather than hardcoding IDs:

   ```go
   // Find entities by their attributes, not by hardcoded IDs
   var seller entity.User
   containers.DB.Where("role = ?", "seller").First(&seller)
   ```

3. **Document seed dependencies** at the top of test files:

   ```go
   // TestCreateVariant requires:
   // - migrations/seeds/001_seed_user_data.sql (for seller authentication)
   // - migrations/seeds/002_seed_product_data.sql (for test products with options)
   func TestCreateVariant(t *testing.T) { ... }
   ```

4. **Keep seed data realistic** and representative of production scenarios
5. **Create helper functions** to find common test entities (see Testing Best Practices section)

### Test Directory Structure

```
test/
├── integration/          # Integration tests with real DB and Redis
│   ├── <module>/        # Tests organized by module (product, user, order, etc.)
│   ├── helpers/         # Test utilities (APIClient, assertions, auth helpers)
│   └── setup/           # TestContainers setup (containers, database, server)
└── unit/                # Unit tests (if separated from module code)
```

**Test Organization Principles:**

- Tests should be organized by module/feature, not by type
- Each module directory can contain subdirectories for different entities
- Helper functions should be reusable across all tests
- Setup utilities should handle container lifecycle management

### External Dependencies Guidelines

**ALWAYS Mock External Services:**

- Payment gateways (Stripe, PayPal, etc.) - prevent real charges
- Email services (SendGrid, SES, etc.) - prevent sending real emails
- SMS services (Twilio, etc.) - prevent sending real SMS
- Third-party APIs (shipping, tax calculation, etc.) - prevent external calls
- OAuth providers (Google, Facebook, etc.) - prevent real authentication flows

**NEVER Mock Internal Components:**

- Database (use TestContainers with PostgreSQL)
- Redis Cache (use TestContainers with Redis)
- Service layers within the application
- Repository layers
- HTTP handlers
- Business logic

**Why?** Mocking internal components creates tests that validate mocks, not actual behavior. Integration tests must exercise the full internal stack.

## Code Quality Standards

### Go Code Style

- Use **tabs** for indentation (Go standard)
- Follow `gofmt` formatting
- Follow `golint` recommendations
- Use Go 1.25 features
- Use `goimports` to organize imports

### Documentation Requirements

- All exported functions must have doc comments
- Doc comments should start with the function name
- Explain WHY, not just WHAT

```go
// CreateProduct creates a new product for the authenticated seller.
// It validates the product data, checks seller permissions, and stores
// the product in the database with cache invalidation.
func CreateProduct(ctx context.Context, req *model.CreateProductRequest) (*entity.Product, error) {
    // Implementation
}
```

### Assertions Library

- Primary: **testify/assert** (`assert.Equal()`, `assert.NotNil()`)
- Use **testify/require** for critical assertions that should stop test execution (`require.NoError()`)

```go
// Use assert for non-critical checks
assert.Equal(t, expectedValue, actualValue, "Should match")

// Use require for critical checks (stops test if fails)
require.NoError(t, err, "Should not have error")
require.NotNil(t, response, "Response should not be nil")
```

### Error Handling

- Always check errors explicitly
- Use custom error types in `common/error/`
- Return meaningful error messages
- Log errors with context

```go
// ✅ GOOD: Explicit error handling
product, err := service.CreateProduct(ctx, req)
if err != nil {
    return nil, fmt.Errorf("failed to create product: %w", err)
}

// ❌ BAD: Ignoring errors
product, _ := service.CreateProduct(ctx, req)
```

### Logging

- Use `logrus` for structured logging
- Include context in log messages
- Use appropriate log levels (Debug, Info, Warn, Error)

```go
log.WithFields(logrus.Fields{
    "product_id": productID,
    "seller_id":  sellerID,
}).Info("Product created successfully")

log.WithError(err).Error("Failed to create product")
```

## Testing Best Practices

### 1. Use Table-Driven Tests for Multiple Scenarios

```go
func TestValidateSKU(t *testing.T) {
    tests := []struct {
        name    string
        sku     string
        wantErr bool
    }{
        {"Valid SKU", "NIKE-TSHIRT-001", false},
        {"Empty SKU", "", true},
        {"Too long", strings.Repeat("A", 256), true},
        {"Invalid characters", "SKU#@!", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateSKU(tt.sku)
            if tt.wantErr {
                assert.Error(t, err, "Should return error")
            } else {
                assert.NoError(t, err, "Should not return error")
            }
        })
    }
}
```

### 2. Use Test Helpers for Common Operations

```go
// helpers/auth.go
func Login(t *testing.T, client *APIClient, email, password string) string {
    requestBody := map[string]interface{}{
        "email":    email,
        "password": password,
    }

    w := client.Post(t, "/api/auth/login", requestBody)
    response := AssertSuccessResponse(t, w, http.StatusOK)

    token, ok := response["token"].(string)
    require.True(t, ok, "Token should be a string")
    require.NotEmpty(t, token, "Token should not be empty")

    return token
}

// helpers/test_data.go
// Helper to find test entities from seeded data
func FindAdminUser(t *testing.T, db *gorm.DB) *entity.User {
    var user entity.User
    err := db.Where("role = ?", "admin").First(&user).Error
    require.NoError(t, err, "Should find admin user from seed data")
    return &user
}

func FindSellerUser(t *testing.T, db *gorm.DB) *entity.User {
    var user entity.User
    err := db.Where("role = ?", "seller").First(&user).Error
    require.NoError(t, err, "Should find seller user from seed data")
    return &user
}

func FindProductBySeller(t *testing.T, db *gorm.DB, sellerID uint) *entity.Product {
    var product entity.Product
    err := db.Where("seller_id = ?", sellerID).First(&product).Error
    require.NoError(t, err, "Should find product for seller from seed data")
    return &product
}
```

### 3. Isolate Tests with Subtests

```go
func TestProductAPI(t *testing.T) {
    // Shared setup
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    containers.RunAllMigrations(t)
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)
    client := helpers.NewAPIClient(server)

    // Each subtest is isolated
    t.Run("Create product", func(t *testing.T) { /* test */ })
    t.Run("Get product", func(t *testing.T) { /* test */ })
    t.Run("Update product", func(t *testing.T) { /* test */ })
    t.Run("Delete product", func(t *testing.T) { /* test */ })
}
```

### 4. Test Concurrent Operations

```go
func TestConcurrentProductCreation(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    const goroutines = 10
    errChan := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        go func(id int) {
            // Create product concurrently
            sku := fmt.Sprintf("SKU-%d", id)
            _, err := service.CreateProduct(ctx, &model.CreateProductRequest{
                SKU: sku,
            })
            errChan <- err
        }(i)
    }

    // Wait for all goroutines
    for i := 0; i < goroutines; i++ {
        err := <-errChan
        assert.NoError(t, err, "Concurrent creation should succeed")
    }
}
```

### 5. Validate Database State Directly

```go
func TestCategoryDeletion(t *testing.T) {
    // ... setup and delete category via API ...

    // Verify deletion in database
    var count int64
    err := containers.DB.Model(&entity.Category{}).
        Where("id = ?", categoryID).
        Count(&count).Error

    require.NoError(t, err, "Database query should succeed")
    assert.Equal(t, int64(0), count, "Category should be deleted from database")
}
```

## Migration Testing

### Test Migration Files

```go
func TestMigration002_CreateProductTables(t *testing.T) {
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    // Run only migrations up to 002
    containers.RunSpecificMigrations(t, []string{
        "migrations/001_create_user_tables.sql",
        "migrations/002_create_product_tables.sql",
    })

    // Verify tables exist
    tables := []string{"products", "product_variants", "product_options"}
    for _, table := range tables {
        var exists bool
        err := containers.DB.Raw(`
            SELECT EXISTS (
                SELECT FROM information_schema.tables
                WHERE table_name = ?
            )
        `, table).Scan(&exists).Error

        require.NoError(t, err, "Query should succeed")
        assert.True(t, exists, fmt.Sprintf("Table %s should exist", table))
    }
}
```

---

**Remember**: The goal of testing is to build confidence in the code, catch bugs early, and document expected behavior. Write tests that provide value, not just coverage numbers.

**Critical Note**: While writing tests, **NEVER modify production code to make tests pass**. If there is a bug in the production code, the test should fail and report why. The developer should then fix the production code to make the test pass. Tests should reveal bugs, not hide them.
