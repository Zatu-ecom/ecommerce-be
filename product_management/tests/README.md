# Product Management Integration Tests

## Current Status

✅ **COMPLETE**: Comprehensive integration test suite created for product_management service  
✅ **COMPLETE**: Test infrastructure and helpers set up properly  
✅ **COMPLETE**: Constants and types organized  
✅ **COMPLETE**: Basic test functionality verified  
✅ **COMPLETE**: PostgreSQL test containers implemented and tested  
✅ **COMPLETE**: Container lifecycle management working perfectly  
⚡ **READY**: Integration tests ready to run with real PostgreSQL database

This directory contains comprehensive integration tests for the product_management service based on the API specifications defined in `PRODUCT_SERVICE_PRD.md`.

## Quick Start

**Prerequisites**: Docker Desktop running

```bash
# Run all integration tests with PostgreSQL containers
cd product_management/tests
go test -v .

# Run specific test categories
go test -v -run TestCategory
go test -v -run TestAttribute
go test -v -run TestProduct

# Run just the container lifecycle test
go test -v -run TestPostgreSQLContainer

# Run basic tests without containers
go test -v simple_test.go test_helpers.go
```

## Test Structure

### Core Test Files
- **`main_test.go`** - Test environment setup and configuration
- **`test_helpers.go`** - Helper functions, constants, and test utilities
- **`testcontainer_setup.go`** - PostgreSQL test container management
- **`container_test.go`** - Container lifecycle and resilience tests
- **`simple_test.go`** - Basic tests that don't require database

### Category Management Tests
- **`category_test.go`** - Complete category CRUD operations and hierarchy tests
  - Create, read, update, delete categories
  - Parent-child relationships and hierarchies
  - Category attribute configuration
  - Validation and error handling

### Attribute Definition Tests
- **`attribute_test.go`** - Attribute definition and configuration tests
  - Create, read, update, delete attributes
  - Data type validation
  - Allowed values handling
  - Category attribute associations

### Product Management Tests
- **`product_test.go`** - Product CRUD and management tests
  - Create, read, update, delete products
  - Product search and filtering
  - Stock management
  - Related products
  - Package options

### Search and Filter Tests
- **`search_filter_test.go`** - Advanced search and filtering tests
  - Text search functionality
  - Category-based filtering
  - Price range filtering
  - Attribute-based filtering
  - Pagination

### Cache and Performance Tests
- **`cache_test.go`** - Caching functionality tests
  - Cache key generation
  - TTL validation
  - Cache invalidation
  - Performance benchmarks

### Integration Workflow Tests
- **`workflow_test.go`** - End-to-end workflow tests
  - Complete product creation workflow
  - Category-attribute-product relationships
  - Bulk operations
  - Data consistency

## Test Coverage

### Category Management APIs
- **Create Category** (`POST /api/categories`)
  - Valid category creation
  - Parent-child relationships
  - Validation and error handling

- **Get Categories** (`GET /api/categories`)
  - List all categories
  - Hierarchical structure
  - Pagination

- **Get Category by ID** (`GET /api/categories/:categoryId`)
  - Single category retrieval
  - Related data loading

- **Update Category** (`PUT /api/categories/:categoryId`)
  - Category modification
  - Relationship updates

- **Delete Category** (`DELETE /api/categories/:categoryId`)
  - Category removal
  - Constraint validation

### Attribute Management APIs
- **Create Attribute** (`POST /api/attributes`)
  - Attribute definition creation
  - Data type validation
  - Allowed values handling

- **Get Attributes** (`GET /api/attributes`)
  - List all attributes
  - Filtering and pagination

- **Update Attribute** (`PUT /api/attributes/:attributeId`)
  - Attribute modification
  - Validation updates

- **Delete Attribute** (`DELETE /api/attributes/:attributeId`)
  - Attribute removal
  - Dependency checking

### Product Management APIs
- **Create Product** (`POST /api/products`)
  - Product creation with attributes
  - Category association
  - Validation and error handling

- **Get Products** (`GET /api/products`)
  - Product listing
  - Filtering and pagination
  - Related data loading

- **Search Products** (`GET /api/products/search`)
  - Text search functionality
  - Advanced filtering
  - Result relevance

- **Update Product** (`PUT /api/products/:productId`)
  - Product modification
  - Attribute updates
  - Stock management

- **Delete Product** (`DELETE /api/products/:productId`)
  - Product removal
  - Dependency validation

### Advanced Features
- **Product Filters** (`GET /api/products/filters`)
  - Available filter options
  - Dynamic filter generation

- **Related Products** (`GET /api/products/:productId/related`)
  - Similar product suggestions
  - Category-based recommendations

- **Stock Management** (`PATCH /api/products/:productId/stock`)
  - Stock updates
  - Availability tracking

## Test Data Management

### Test Fixtures
- Predefined test categories with hierarchies
- Sample attribute definitions
- Test products with various configurations
- Mock user accounts for authentication

### Database Cleanup
- Automatic cleanup between test runs
- Transaction rollback for test isolation
- Data consistency validation

### Performance Testing
- Load testing for search operations
- Cache performance benchmarks
- Database query optimization tests

## Running Tests

### Environment Setup
```bash
# Set test environment variables
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_NAME=testdb
export TEST_DB_USER=testuser
export TEST_DB_PASSWORD=testpass
```

### Test Execution
```bash
# Run all tests
go test -v .

# Run specific test file
go test -v category_test.go

# Run tests with coverage
go test -v -coverprofile=coverage.out .
go tool cover -html=coverage.out

# Run tests in parallel
go test -v -parallel 4 .
```

### Debugging Tests
```bash
# Run single test function
go test -v -run TestCreateCategory

# Run tests with verbose output
go test -v -test.v .

# Run tests with timeout
go test -v -timeout 5m .
```

## Test Results

### Expected Output
```
✅ PostgreSQL test container started successfully
📊 Connection string: host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable
🧪 Running Category Management Tests...
✅ TestCreateCategory: PASS
✅ TestGetCategories: PASS
✅ TestUpdateCategory: PASS
✅ TestDeleteCategory: PASS
🧪 Running Attribute Tests...
✅ TestCreateAttribute: PASS
✅ TestGetAttributes: PASS
🧪 Running Product Tests...
✅ TestCreateProduct: PASS
✅ TestSearchProducts: PASS
🛑 PostgreSQL test container terminated successfully
PASS
ok      ecommerce-be/product_management/tests   15.234s
```

### Test Categories
- **Unit Tests**: Individual function testing
- **Integration Tests**: API endpoint testing with database
- **End-to-End Tests**: Complete workflow testing
- **Performance Tests**: Load and stress testing
- **Security Tests**: Authentication and authorization

## Troubleshooting

### Common Issues
1. **Docker not running**: Ensure Docker Desktop is started
2. **Port conflicts**: Check if ports 5432, 6379 are available
3. **Database connection**: Verify test environment variables
4. **Test timeout**: Increase timeout for slow operations

### Debug Mode
```bash
# Enable debug logging
export TEST_DEBUG=true
go test -v -test.v .
```

## Contributing

When adding new tests:
1. Follow the existing test structure
2. Use descriptive test names
3. Include proper cleanup and teardown
4. Add test data fixtures
5. Document test scenarios in README
