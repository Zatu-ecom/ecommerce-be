# User Management Integration Tests

## Current Status

✅ **COMPLETE**: Comprehensive integration test suite created for user_management service  
✅ **COMPLETE**: Test infrastructure and helpers set up properly  
✅ **COMPLETE**: Constants and types organized  
✅ **COMPLETE**: Basic test functionality verified  
✅ **COMPLETE**: PostgreSQL test containers implemented and tested  
✅ **COMPLETE**: Container lifecycle management working perfectly  
⚡ **READY**: Integration tests ready to run with real PostgreSQL database

This directory contains comprehensive integration tests for the user_management service based on the API specifications defined in `backend-prd.md`.

## Quick Start

**Prerequisites**: Docker Desktop running

```powershell
# Run all integration tests with PostgreSQL containers
cd user_management/tests
go test -v .

# Run just the container lifecycle test
go test -v -run TestPostgreSQLContainer

# Run basic tests without containers
go test -v simple_test.go test_helpers.go
```

## Overview

These tests cover the complete user management workflow including:

### Authentication APIs
- **User Registration** (`POST /api/auth/register`)
  - Valid registration with all required fields
  - Email validation and duplicate email handling
  - Password confirmation validation
  - Error responses for invalid data

- **User Login** (`POST /api/auth/login`)
  - Valid login with correct credentials
  - Invalid email and password handling
  - Account status validation

- **Token Refresh** (`POST /api/auth/refresh`)
  - Valid token refresh
  - Invalid token handling
  - Missing authentication handling

- **User Logout** (`POST /api/auth/logout`)
  - Token blacklisting functionality

### Core Files

- **`integration_test.go`** - Main integration test file with all API endpoint tests
- **`test_helpers.go`** - Helper functions, constants, and test utilities  
- **`testcontainer_setup.go`** - PostgreSQL test container management
- **`container_test.go`** - Container lifecycle and resilience tests
- **`simple_test.go`** - Basic tests that don't require database
- **`main_test.go`** - Test environment setup and configuration
- **Get User Profile** (`GET /api/users/profile`)
  - Authenticated profile retrieval
  - Profile data structure validation

- **Update User Profile** (`PUT /api/users/profile`)
  - Profile update with valid data
  - Input validation

- **Change Password** (`PATCH /api/users/password`)
  - Valid password change
  - Current password verification
  - Password confirmation validation

### Address Management APIs
- **Get Addresses** (`GET /api/users/addresses`)
  - Retrieve user addresses

- **Add Address** (`POST /api/users/addresses`)
  - Add new address with validation
  - Default address handling

- **Update Address** (`PUT /api/users/addresses/:id`)
  - Update existing address
  - Permission validation

- **Set Default Address** (`PATCH /api/users/addresses/:id/default`)
  - Set address as default

### Authentication Middleware Tests
- Missing authorization header
- Invalid token format
- Invalid JWT token
- Token blacklist validation

## Test Structure

### Files
- `integration_test.go` - Main integration test file with all test cases
- `test_helpers.go` - Helper functions and test utilities
- `main_test.go` - Test setup and teardown configuration

### Key Components
- **IntegrationTestSuite** - Test suite with database and server setup
- **Test Data Factory** - Helper for creating test data
- **API Response Validation** - Structured response validation
- **Test Constants** - Centralized test data constants

## Prerequisites

Before running the tests, ensure you have:

1. **Go** (1.19 or later)
2. **PostgreSQL** or **SQLite** for testing database
3. **Redis** server running (for session management)
4. **Required Go modules**:
   ```bash
   go mod tidy
   ```

## Required Dependencies

Add these to your `go.mod`:

```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/stretchr/testify v1.8.4
    gorm.io/gorm v1.25.4
    gorm.io/driver/sqlite v1.5.4
    gorm.io/driver/postgres v1.5.2
)
```

## Running the Tests

### Run All Tests
```bash
cd user_management/tests
go test -v
```

### Run Specific Test
```bash
go test -v -run TestUserRegistration
```

### Run Tests with Coverage
```bash
go test -v -cover
```

### Run Tests with Race Detection
```bash
go test -v -race
```

## Commands to Control Test Caching

Go automatically caches test results when source code hasn't changed, leading to very fast execution times with `(cached)` output. Here are commands to control this behavior:

### Force Fresh Test Run (Disable Cache)
```bash
# Run all tests without using cache
go test -v . -count=1

# Run specific test without cache
go test -v -run TestUserRegistration -count=1
```

### Clear All Cached Test Results
```bash
# Clear the entire test cache
go clean -testcache

# Then run tests normally
go test -v .
```

### Run Tests with Verbose Caching Info
```bash
# Show detailed build and cache information
go test -v . -x
```

### Understanding Test Cache Behavior

**When Go Uses Cache (Fast Execution):**
- ✅ Source code unchanged
- ✅ Dependencies unchanged  
- ✅ Environment stable
- ⚡ Result: `(cached)` - near-instant execution

**When Go Runs Fresh Tests (Full Execution):**
- 🐳 Real container startup (~2-3 seconds per test)
- 🔄 Database operations and migrations
- 🚀 Network calls (Redis, HTTP requests)
- ⏱️ Result: Full execution time (~40+ seconds for complete suite)

### Recommended Development Workflow
```bash
# During active development (use cache for speed)
go test -v .

# Before committing changes (force fresh run)
go test -v . -count=1

# After major changes (clear cache and test)
go clean -testcache && go test -v .
```

## Test Configuration

### Environment Variables
Set these environment variables for testing:
```bash
export GIN_MODE=test
export JWT_SECRET=test-jwt-secret-key
export REDIS_URL=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=1
```

### Database Setup
The tests use SQLite in-memory database by default. To use PostgreSQL:

```go
// In SetupTestSuite function, replace:
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

// With:
dsn := "host=localhost user=test_user password=test_pass dbname=test_db port=5432 sslmode=disable"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

## Integration with Actual Application

To integrate these tests with your actual application:

1. **Update `setupRoutes` function** in `integration_test.go`:
   ```go
   func setupRoutes(router *gin.Engine, db *gorm.DB) {
       // Replace placeholder with actual route setup
       userContainer := user_management.NewContainer(db)
       userHandlers := userContainer.GetHandlers()
       
       authGroup := router.Group("/api/auth")
       {
           authGroup.POST("/register", userHandlers.Register)
           authGroup.POST("/login", userHandlers.Login)
           // ... other routes
       }
   }
   ```

2. **Import your actual handlers and dependencies**

3. **Update entity imports** to match your project structure

## Test Data

### Test Constants
All test data is centralized in constants:
- `TestPassword = "Password123!"`
- `TestEmail = "john.doe@example.com"`
- `TestPhone = "+1234567890"`
- And more...

### Sample Test User
Default test user created for authenticated tests:
```json
{
  "firstName": "Test",
  "lastName": "User",
  "email": "test.user.{unique}@example.com",
  "password": "Password123!",
  "phone": "+1234567890",
  "dateOfBirth": "1990-01-01",
  "gender": "male"
}
```

## API Response Validation

Tests validate responses match the API specification:

### Success Response Structure
```json
{
  "success": true,
  "message": "Operation successful",
  "data": {
    // Response data
  }
}
```

### Error Response Structure
```json
{
  "success": false,
  "message": "Error description",
  "code": "ERROR_CODE",
  "errors": [
    {
      "field": "fieldName",
      "message": "Specific error message"
    }
  ]
}
```

## Test Coverage

The tests aim to cover:
- ✅ **Happy Path Scenarios** - All valid operations
- ✅ **Error Handling** - Invalid inputs and edge cases
- ✅ **Authentication** - Token validation and security
- ✅ **Authorization** - Permission checks
- ✅ **Data Validation** - Input validation and sanitization
- ✅ **Response Format** - API specification compliance

## Continuous Integration

For CI/CD pipelines, run tests with:
```bash
# Install dependencies
go mod download

# Run tests with coverage
go test -v -cover -race ./user_management/tests/

# Generate coverage report
go test -coverprofile=coverage.out ./user_management/tests/
go tool cover -html=coverage.out -o coverage.html
```

## Known Issues

### ✅ Database Driver Issue - RESOLVED
~~The integration tests previously used SQLite which required CGO (C compiler).~~

**Current Solution**: 
- ✅ **PostgreSQL Test Containers implemented** - No CGO required
- ✅ **Production-like testing** - Uses real PostgreSQL like production  
- ✅ **Automatic cleanup** - Containers start/stop automatically
- ✅ **Full test coverage** - All API endpoints can be tested

### Requirements for Full Integration Tests

**Docker Required**:
- Docker Desktop must be installed and running
- Internet connection for first run (downloads PostgreSQL image ~80MB)
- No additional setup needed - containers are managed automatically

**Test Execution**:
```powershell
# Full integration test suite (requires Docker)
go test -v .

# Container-specific tests
go test -v -run TestPostgreSQLContainer

# Basic tests only (no Docker required)  
go test -v simple_test.go test_helpers.go
```

### Common Issues

1. **Database Connection Errors**
   - Ensure PostgreSQL/SQLite is accessible
   - Check database permissions
   - Verify connection string

2. **Redis Connection Errors**
   - Ensure Redis server is running
   - Check Redis configuration
   - Verify network connectivity

3. **Import Errors**
   - Update import paths to match your project structure
   - Ensure all dependencies are installed
   - Run `go mod tidy`

4. **Test Failures**
   - Check if actual handlers are properly integrated
   - Verify API endpoints match the specification
   - Ensure database migrations are applied

### Debug Mode
To enable debug logging in tests:
```go
// In test setup
gin.SetMode(gin.DebugMode)
```

## Contributing

When adding new tests:
1. Follow the existing test structure
2. Use test constants for data
3. Validate both success and error scenarios
4. Include response structure validation
5. Add documentation for new test cases

## Related Documentation
- `backend-prd.md` - Complete API specification
- Project README - Overall project setup
- API Documentation - Endpoint details
