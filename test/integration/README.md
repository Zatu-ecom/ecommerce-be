# Integration Tests

This directory contains integration tests for all services in the ecommerce backend.

## Structure

```
test/integration/
├── setup/              # Test infrastructure setup (reusable across all tests)
│   ├── containers.go   # TestContainers (Postgres, Redis)
│   ├── database.go     # Database migrations & seeding
│   └── server.go       # HTTP server setup
│
├── helpers/            # Shared test utilities (reusable across all tests)
│   ├── api_client.go   # HTTP request wrapper (GET, POST, PUT, DELETE)
│   └── auth_helper.go  # Authentication helpers (login, token management)
│
├── user/               # USER SERVICE tests
│   └── auth_test.go    # Authentication tests (login, register, logout)
│
└── product/            # PRODUCT SERVICE tests (coming soon)
    ├── category_test.go
    ├── product_test.go
    └── variant_test.go
```

## Running Tests

### Run all integration tests

```bash
go test -v ./test/integration/...
```

### Run specific service tests

```bash
# User service tests
go test -v ./test/integration/user

# Product service tests
go test -v ./test/integration/product
```

### Run a specific test

```bash
go test -v ./test/integration/user -run TestAuth
```

## Writing New Tests

### 1. Create a test file in the appropriate service folder

Example: `test/integration/user/profile_test.go`

```go
package user

import (
    "testing"
    "ecommerce-be/test/integration/helpers"
    "ecommerce-be/test/integration/setup"
)

func TestUserProfile(t *testing.T) {
    // Setup containers
    containers := setup.SetupTestContainers(t)
    defer containers.Cleanup(t)

    // Run migrations
    containers.RunMigrations(t, "migrations/001_create_user_tables.sql")
    containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

    // Setup server
    server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

    // Create API client
    client := helpers.NewAPIClient(server)

    // Login to get token
    token := helpers.Login(t, client, "user@example.com", "password123")
    client.SetToken(token)

    // Write your tests here
    t.Run("get user profile", func(t *testing.T) {
        // Test implementation
    })
}
```

### 2. Use the API client for HTTP requests

```go
// POST request
response := client.Post(t, "/api/endpoint", requestBody)

// GET request
response := client.Get(t, "/api/endpoint")

// PUT request
response := client.Put(t, "/api/endpoint", requestBody)

// DELETE request
response := client.Delete(t, "/api/endpoint")

// Parse response
data := helpers.ParseResponse(t, response.Body)
```

### 3. Use authentication helper when needed

```go
// Login and get token
token := helpers.Login(t, client, "email@example.com", "password")
client.SetToken(token)

// Now all subsequent requests will include the token
response := client.Get(t, "/api/protected-endpoint")
```

## Test Data

- Migrations are in: `migrations/`
- Seed data is in: `migrations/seeds/`

Each test should:

1. Set up clean containers
2. Run necessary migrations
3. Seed required test data
4. Clean up after completion

## Notes

- Each test gets fresh Docker containers (Postgres + Redis)
- Tests are isolated and can run in parallel
- Use meaningful test names: `TestServiceName_FeatureName`
- Group related tests using `t.Run()` subtests
