# 📐 Coding Standards & Best Practices

> **Last Updated**: November 20, 2025  
> **Purpose**: Ensure code quality, maintainability, and consistency across all services

---

## 📋 Table of Contents

1. [Core Principles](#core-principles)
2. [Architecture Rules](#architecture-rules)
3. [Code Organization](#code-organization)
4. [Performance Guidelines](#performance-guidelines)
5. [Error Handling](#error-handling)
6. [Logging Standards](#logging-standards)
7. [Go-Specific Best Practices](#go-specific-best-practices)
8. [Design Patterns](#design-patterns)
9. [Security Guidelines](#security-guidelines)
10. [Database Migration Guidelines](#database-migration-guidelines)
11. [Code Review Checklist](#code-review-checklist)

---

## 🎯 Core Principles

### SOLID Principles (Mandatory)

Every code change must adhere to SOLID:

1. **S**ingle Responsibility Principle

   - Each function/struct has ONE clear purpose
   - If you can't describe it in one sentence, it's doing too much

2. **O**pen/Closed Principle

   - Open for extension, closed for modification
   - Use interfaces and dependency injection

3. **L**iskov Substitution Principle

   - Implementations must be interchangeable
   - Repository implementations can swap without breaking services

4. **I**nterface Segregation Principle

   - Keep interfaces focused and minimal
   - Don't force implementations to depend on methods they don't use

5. **D**ependency Inversion Principle
   - Depend on abstractions (interfaces), not concrete implementations
   - All dependencies injected through constructors

### DRY (Don't Repeat Yourself)

❌ **Bad:**

```go
// In product service
func (s *ProductService) ValidateCategory(id uint) error {
    if id == 0 {
        return errors.New("category ID required")
    }
    // validation logic
}

// In variant service (DUPLICATE)
func (s *VariantService) ValidateCategory(id uint) error {
    if id == 0 {
        return errors.New("category ID required")
    }
    // same validation logic
}
```

✅ **Good:**

```go
// In common/validator/category_validator.go
func ValidateCategory(id uint) error {
    if id == 0 {
        return errors.New("category ID required")
    }
    // validation logic
}

// Both services use the common validator
```

### Composition Over Inheritance

✅ **Good - Reuse through composition:**

```go
// Base request model
type CreateVariantRequest struct {
    SKU         string  `json:"sku" binding:"required"`
    Price       float64 `json:"price" binding:"required"`
    Stock       int     `json:"stock"`
}

// Product creation reuses variant model
type CreateProductRequest struct {
    Name        string                 `json:"name" binding:"required"`
    CategoryID  uint                   `json:"categoryId" binding:"required"`
    Variants    []CreateVariantRequest `json:"variants" binding:"required"`
}
```

---

## 🏗️ Architecture Rules

### Rule #1: Never Bypass Service Layers

**Critical Rule**: Services must NEVER call other services' repositories directly.

❌ **WRONG:**

```go
// In ProductService - NEVER DO THIS
func (s *ProductServiceImpl) CreateProduct(req model.CreateProductRequest) error {
    // ❌ Directly accessing AttributeRepository
    attrs, err := s.attributeRepo.FindByCategoryID(req.CategoryID)
    if err != nil {
        return err
    }
}
```

✅ **CORRECT:**

```go
// In ProductService - Always call the service
func (s *ProductServiceImpl) CreateProduct(req model.CreateProductRequest) error {
    // ✅ Call AttributeService which handles validation & business logic
    attrs, err := s.attributeService.GetByCategoryID(req.CategoryID)
    if err != nil {
        return err
    }
}
```

**Why?**

- Service layer contains business rules, validation, caching
- Bypassing it skips critical checks
- Breaks encapsulation and maintainability

### Rule #2: Follow Standard Structure

Every service MUST follow this structure (refer to `product/` as the standard):

```
service_name/
├── container.go          # Dependency wiring
├── entity/              # Database models
├── model/               # DTOs (request/response)
├── repositories/        # Data access
│   ├── *_repository.go      # Interface
│   └── *_repository_impl.go # Implementation
├── service/             # Business logic
│   ├── *_service.go         # Interface
│   └── *_service_impl.go    # Implementation
├── handlers/            # HTTP handlers
├── routes/              # Route definitions
├── factory/             # Dependency injection (optional)
├── errors/              # Service-specific errors
├── validator/           # Business validation
└── utils/               # Service utilities
```

### Rule #3: Dependency Injection via Constructors

All dependencies MUST be injected as constructor parameters.

❌ **Bad:**

```go
// Global variable
var db *gorm.DB

type ProductRepository struct {}

func (r *ProductRepository) Create(p *entity.Product) error {
    return db.Create(p).Error // Using global
}
```

✅ **Good:**

```go
type ProductRepository struct {
    db *gorm.DB // Injected dependency
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
    return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(p *entity.Product) error {
    return r.db.Create(p).Error
}
```

### Rule #4: Use Singletons for Factories

All factories should follow singleton pattern:

```go
var (
    instance *SingletonFactory
    once     sync.Once
)

func GetInstance() *SingletonFactory {
    once.Do(func() {
        instance = &SingletonFactory{
            db:    db.GetDB(),
            redis: cache.GetRedis(),
        }
    })
    return instance
}

// For tests
func ResetInstance() {
    instance = nil
}
```

---

## 📦 Code Organization

### File Size Limits

| Item          | Max Lines  | Reason                                             |
| ------------- | ---------- | -------------------------------------------------- |
| **Method**    | 50 lines   | Beyond this indicates SRP violation                |
| **File**      | 500 lines  | Should be split into multiple files                |
| **Interface** | 10 methods | Too many methods = interface segregation violation |

### When to Split Files

❌ **Too Large (600+ lines):**

```
product_service_impl.go (650 lines)
├── CreateProduct() - 80 lines
├── UpdateProduct() - 70 lines
├── DeleteProduct() - 40 lines
├── GetProduct() - 30 lines
├── SearchProducts() - 120 lines
└── ... more methods
```

✅ **Split Properly:**

```
service/
├── product_service.go              # Interface (10 lines)
├── product_service_impl.go         # Main CRUD (200 lines)
├── product_service_search.go       # Search logic (150 lines)
└── product_service_validation.go   # Validation logic (100 lines)
```

### Method Length Guidelines

If a method exceeds 50 lines, refactor it:

❌ **Too Long (80 lines):**

```go
func (s *ProductServiceImpl) CreateProduct(req model.CreateProductRequest) error {
    // Validation - 15 lines
    if req.Name == "" {
        return errors.New("name required")
    }
    // ... more validation

    // Category check - 10 lines
    category, err := s.categoryRepo.FindByID(req.CategoryID)
    // ... error handling

    // Transaction - 20 lines
    tx := s.db.Begin()
    // ... transaction logic

    // Create product - 15 lines
    product := &entity.Product{...}
    // ... creation logic

    // Create variants - 20 lines
    for _, v := range req.Variants {
        // ... variant logic
    }
}
```

✅ **Refactored (well-structured):**

```go
func (s *ProductServiceImpl) CreateProduct(req model.CreateProductRequest) error {
    if err := s.validateCreateRequest(req); err != nil {
        return err
    }

    category, err := s.getCategoryOrFail(req.CategoryID)
    if err != nil {
        return err
    }

    return s.executeProductCreation(req, category)
}

func (s *ProductServiceImpl) validateCreateRequest(req model.CreateProductRequest) error {
    // Validation logic (10-15 lines)
}

func (s *ProductServiceImpl) getCategoryOrFail(id uint) (*entity.Category, error) {
    // Category fetching (5-10 lines)
}

func (s *ProductServiceImpl) executeProductCreation(req model.CreateProductRequest, cat *entity.Category) error {
    // Transaction and creation (30-40 lines)
}
```

---

## ⚡ Performance Guidelines

### Rule #1: Avoid N+1 Queries

❌ **Bad - N+1 Query:**

```go
func (s *ProductService) GetProductsWithCategories() ([]model.ProductResponse, error) {
    products, _ := s.productRepo.FindAll()

    var responses []model.ProductResponse
    for _, p := range products {
        // ❌ N queries for N products
        category, _ := s.categoryRepo.FindByID(p.CategoryID)
        responses = append(responses, model.ProductResponse{
            Product:  p,
            Category: category,
        })
    }
    return responses, nil
}
```

✅ **Good - Use Eager Loading:**

```go
func (s *ProductService) GetProductsWithCategories() ([]model.ProductResponse, error) {
    // ✅ Single query with JOIN
    products, _ := s.productRepo.FindAllWithCategories()
    return mapper.ToProductResponses(products), nil
}

// In repository
func (r *ProductRepository) FindAllWithCategories() ([]*entity.Product, error) {
    var products []*entity.Product
    err := r.db.Preload("Category").Find(&products).Error
    return products, err
}
```

### Rule #2: Avoid Redundant Loops

❌ **Bad - Multiple Passes:**

```go
func ProcessProducts(products []entity.Product) []model.ProductResponse {
    // First loop - filter
    var active []entity.Product
    for _, p := range products {
        if p.IsActive {
            active = append(active, p)
        }
    }

    // Second loop - transform
    var responses []model.ProductResponse
    for _, p := range active {
        responses = append(responses, mapper.ToResponse(p))
    }

    return responses
}
```

✅ **Good - Single Pass:**

```go
func ProcessProducts(products []entity.Product) []model.ProductResponse {
    responses := make([]model.ProductResponse, 0, len(products))

    for _, p := range products {
        if p.IsActive {
            responses = append(responses, mapper.ToResponse(p))
        }
    }

    return responses
}
```

### Rule #3: Use Caching Wisely

```go
func (s *ProductService) GetProduct(id uint) (*model.ProductResponse, error) {
    // 1. Check cache first
    cacheKey := fmt.Sprintf("product:%d", id)
    if cached, err := s.cache.Get(cacheKey); err == nil {
        return cached, nil
    }

    // 2. Fetch from DB
    product, err := s.productRepo.FindByID(id)
    if err != nil {
        return nil, err
    }

    // 3. Cache for future requests
    response := mapper.ToResponse(product)
    s.cache.Set(cacheKey, response, 10*time.Minute)

    return response, nil
}
```

**Cache Invalidation:**

```go
func (s *ProductService) UpdateProduct(id uint, req model.UpdateRequest) error {
    if err := s.productRepo.Update(id, req); err != nil {
        return err
    }

    // ✅ Invalidate cache after mutation
    s.cache.Delete(fmt.Sprintf("product:%d", id))
    s.cache.DeletePattern("products:*") // Invalidate list caches

    return nil
}
```

### Rule #4: Optimize Database Queries

✅ **Use Indexes:**

```go
type Product struct {
    ID         uint   `gorm:"primaryKey"`
    SellerID   uint   `gorm:"index"` // ✅ Index for filtering
    CategoryID uint   `gorm:"index"` // ✅ Index for joins
    SKU        string `gorm:"uniqueIndex"` // ✅ Unique constraint + index
}
```

✅ **Use Pagination:**

```go
func (r *ProductRepository) FindAll(page, pageSize int) ([]*entity.Product, error) {
    var products []*entity.Product
    offset := (page - 1) * pageSize

    err := r.db.
        Limit(pageSize).
        Offset(offset).
        Find(&products).Error

    return products, err
}
```

✅ **Select Only Needed Fields:**

```go
// ❌ Bad - Fetches all fields
var products []entity.Product
db.Find(&products)

// ✅ Good - Only needed fields
var products []struct {
    ID   uint
    Name string
}
db.Model(&entity.Product{}).Select("id, name").Find(&products)
```

---

## ❌ Error Handling

### Rule #1: Use Application-Level Errors

Define meaningful errors in service's `errors/` package:

```go
// product/errors/product_errors.go
package errors

import "ecommerce-be/common/error"

var (
    ProductNotFound = error.NewAppError(
        "PRODUCT_NOT_FOUND",
        "Product not found",
        404,
    )

    InvalidSKU = error.NewAppError(
        "INVALID_SKU",
        "SKU must be unique and alphanumeric",
        400,
    )

    InsufficientStock = error.NewAppError(
        "INSUFFICIENT_STOCK",
        "Not enough stock available",
        400,
    )
)
```

### Rule #2: Return Structured Errors

❌ **Bad - Generic Errors:**

```go
func (s *ProductService) GetProduct(id uint) (*entity.Product, error) {
    product, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.New("product not found") // ❌ Generic
    }
    return product, nil
}
```

✅ **Good - Structured Errors:**

```go
func (s *ProductService) GetProduct(id uint) (*entity.Product, error) {
    product, err := s.repo.FindByID(id)
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, errors.ProductNotFound // ✅ Application error
        }
        return nil, fmt.Errorf("failed to fetch product: %w", err)
    }
    return product, nil
}
```

### Rule #3: Wrap Errors with Context

```go
func (r *ProductRepository) Create(product *entity.Product) error {
    if err := r.db.Create(product).Error; err != nil {
        // ✅ Wrap with context
        return fmt.Errorf("failed to create product (sku=%s): %w", product.SKU, err)
    }
    return nil
}
```

### Rule #4: Handle Errors at the Right Level

```go
// Handler - Convert errors to HTTP responses
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id := c.Param("id")

    product, err := h.service.GetProduct(id)
    if err != nil {
        // Check if it's an AppError
        if appErr, ok := err.(*error.AppError); ok {
            common.Error(c, appErr.StatusCode, appErr.Message)
            return
        }
        // Unknown error
        common.Error(c, 500, "Internal server error")
        return
    }

    common.Success(c, 200, product, "Product fetched")
}
```

---

## 📝 Logging Standards

### Rule #1: Use Structured Logging

❌ **Bad - String Concatenation:**

```go
logger.Info("User created product with ID " + strconv.Itoa(productID))
```

✅ **Good - Structured Logs:**

```go
logger.Info("Product created",
    "productId", product.ID,
    "userId", userID,
    "sellerId", sellerID,
    "sku", product.SKU,
)
```

### Rule #2: Log at Appropriate Levels

```go
// DEBUG - Detailed trace for development
logger.Debug("Entering CreateProduct method", "requestId", reqID)

// INFO - Significant events
logger.Info("Product created successfully", "productId", product.ID)

// WARN - Recoverable issues
logger.Warn("Cache miss for product", "productId", id)

// ERROR - Errors requiring attention
logger.Error("Failed to create product", "error", err, "productId", id)
```

### Rule #3: Always Log Correlation ID and other requried fileds

Middleware automatically adds it, but ensure it's propagated:
for each requrest we should have 3 thing with message

- sellerId
- userId
- correlationId

In use our log it has all requred method, you have to pass the context and messge it will handle above things

```go
func (s *ProductService) CreateProduct(ctx context.Context, req model.CreateProductRequest) error {
    correlationID := ctx.Value("correlationID").(string)

    logger.Info("Creating product",
        "correlationId", correlationID, // ✅ Always include
        "productName", req.Name,
    )
}
```

### Rule #4: Log Request/Response for APIs

This is also handle by the middleware in our application

```go
func (h *ProductHandler) CreateProduct(c *gin.Context) {
    var req model.CreateProductRequest
    c.ShouldBindJSON(&req)

    // Log incoming request
    logger.Info("Incoming create product request",
        "correlationId", c.GetHeader("X-Correlation-ID"),
        "userId", auth.GetUserIDFromContext(c),
        "requestBody", req,
    )

    product, err := h.service.CreateProduct(req)

    if err != nil {
        logger.Error("Create product failed",
            "correlationId", c.GetHeader("X-Correlation-ID"),
            "error", err,
        )
        common.Error(c, 500, err.Error())
        return
    }

    // Log successful response
    logger.Info("Create product successful",
        "correlationId", c.GetHeader("X-Correlation-ID"),
        "productId", product.ID,
    )

    common.Success(c, 201, product, "Product created")
}
```

---

## 🔷 Go-Specific Best Practices

### Rule #1: Follow Idiomatic Go

**Naming Conventions:**

```go
// ✅ Good
type ProductService interface {}
type productServiceImpl struct {} // Unexported implementation

// Methods - MixedCaps
func (s *productServiceImpl) CreateProduct() {}
func (s *productServiceImpl) getInternalData() {} // Unexported

// Variables - camelCase or snake_case
var productCache map[string]*Product
const maxRetryCount = 3

// Files - snake_case
product_service_impl.go
product_repository_test.go
```

### Rule #2: Avoid Global Variables

❌ **Bad:**

```go
var (
    db    *gorm.DB
    cache *redis.Client
)

func InitDB() {
    db = gorm.Open(...)
}
```

✅ **Good:**

```go
type ProductService struct {
    db    *gorm.DB
    cache *redis.Client
}

func NewProductService(db *gorm.DB, cache *redis.Client) *ProductService {
    return &ProductService{
        db:    db,
        cache: cache,
    }
}
```

### Rule #3: Use Pointers for Optional/Nullable Fields

**Critical Rule**: In update models and anywhere you need to distinguish between:

- **Not provided** (null/omitted) → Don't update the field
- **Empty value** ("", 0, false) → Update to empty/zero value

Always use **pointers** for optional fields.

❌ **Bad - Can't distinguish null from empty:**

```go
type UpdateRequest struct {
    Name  string `json:"name"`  // ❌ Can't tell if omitted or empty string
    Brand string `json:"brand"` // ❌ Same problem
    Stock int    `json:"stock"` // ❌ Can't tell if omitted or zero
}

// If user sends {}, all fields will be zero values
// You'll accidentally clear fields that shouldn't be updated!
```

✅ **Good - Use pointers for optional fields:**

```go
type UpdateRequest struct {
    Name  *string `json:"name"  binding:"omitempty,min=3,max=200"` // ✅ Pointer
    Brand *string `json:"brand" binding:"omitempty,max=100"`        // ✅ Pointer
    Stock *int    `json:"stock" binding:"omitempty,gte=0"`         // ✅ Pointer
}

// Check if field was provided
if req.Name != nil {
    product.Name = *req.Name // Update only if provided
}
if req.Brand != nil {
    product.Brand = *req.Brand // Can set to empty string if *req.Brand == ""
}
if req.Stock != nil {
    product.Stock = *req.Stock // Can set to 0 if *req.Stock == 0
}
```

**When to use pointers:**

- ✅ Update/PATCH request models (need null support)
- ✅ Optional configuration fields
- ✅ Fields that can be explicitly cleared
- ✅ Nullable database columns

**When NOT to use pointers:**

- ❌ Create request models (all required fields)
- ❌ Response models (always return actual values)
- ❌ Required fields with `binding:"required"`

**Example from Product Service:**

```go
// Create - No pointers (all fields are set)
type ProductCreateRequest struct {
    Name       string   `json:"name"     binding:"required"`
    CategoryID uint     `json:"categoryId" binding:"required"`
    Tags       []string `json:"tags"     binding:"max=20"`
}

// Update - Pointers for optional updates
type ProductUpdateRequest struct {
    Name       *string   `json:"name"       binding:"omitempty,min=3,max=200"`
    CategoryID *uint     `json:"categoryId" binding:"omitempty"`
    Tags       *[]string `json:"tags"       binding:"omitempty,max=20"`
}
```

### Rule #4: Validate Nested Structs with `dive`

When you have arrays of objects or nested structs in request models, use `dive` to apply validation to each element.

❌ **Bad - Only validates array, not elements:**

```go
type ProductRequest struct {
    Variants []VariantRequest `json:"variants" binding:"required"` // ❌ Only checks array exists
}

type VariantRequest struct {
    SKU   string  `json:"sku"   binding:"required"`
    Price float64 `json:"price" binding:"required,gt=0"`
}
// Validation won't check individual variant fields!
```

✅ **Good - Use `dive` for nested validation:**

```go
type ProductRequest struct {
    Variants []VariantRequest `json:"variants" binding:"required,dive"` // ✅ Validates each variant
}

type VariantRequest struct {
    SKU   string  `json:"sku"   binding:"required"`
    Price float64 `json:"price" binding:"required,gt=0"`
}
// Now each variant's SKU and Price are validated!
```

**`dive` Examples:**

```go
// Array of structs
type ProductCreateRequest struct {
    Options    []OptionRequest    `json:"options"    binding:"dive"` // ✅ Validates each option
    Variants   []VariantRequest   `json:"variants"   binding:"required,min=1,dive"` // ✅ Required + nested validation
    Attributes []AttributeRequest `json:"attributes" binding:"omitempty,dive"` // ✅ Optional + nested validation
}

// Nested object
type OrderRequest struct {
    ShippingAddress AddressRequest `json:"shippingAddress" binding:"required,dive"` // ✅ Validates address fields
}

// Map with struct values
type ConfigRequest struct {
    Settings map[string]SettingValue `json:"settings" binding:"dive"` // ✅ Validates each map value
}
```

### Rule #5: Add Basic Validation in Models

**Important**: Keep business logic in service layer, but add **basic structural validation** in models.

✅ **In Models (Structural Validation):**

```go
type CreateProductRequest struct {
    Name       string  `json:"name"     binding:"required,min=3,max=200"`   // ✅ Length checks
    Price      float64 `json:"price"    binding:"required,gt=0"`            // ✅ Range validation
    Email      string  `json:"email"    binding:"required,email"`           // ✅ Format validation
    CategoryID uint    `json:"categoryId" binding:"required"`               // ✅ Required check
    Tags       []string `json:"tags"    binding:"max=20,dive,min=1,max=50"` // ✅ Array + element validation
}
```

✅ **In Service Layer (Business Validation):**

```go
func (s *ProductService) CreateProduct(req model.CreateProductRequest) error {
    // ✅ Business logic checks

    // Check if category exists and is active
    category, err := s.categoryService.GetByID(req.CategoryID)
    if err != nil || !category.IsActive {
        return errors.InvalidCategory
    }

    // Check SKU uniqueness (database constraint)
    exists, _ := s.productRepo.ExistsBySKU(req.SKU)
    if exists {
        return errors.DuplicateSKU
    }

    // Validate seller permissions
    if !s.canSellerCreateInCategory(req.SellerID, req.CategoryID) {
        return errors.Unauthorized
    }
}
```

**Validation Guidelines:**

| Type                   | Where   | Example                              |
| ---------------------- | ------- | ------------------------------------ |
| Length/Size            | Model   | `binding:"min=3,max=200"`            |
| Format                 | Model   | `binding:"email"`, `binding:"url"`   |
| Range                  | Model   | `binding:"gt=0"`, `binding:"gte=18"` |
| Required               | Model   | `binding:"required"`                 |
| Nested                 | Model   | `binding:"dive"`                     |
| Database constraints   | Service | SKU uniqueness, FK existence         |
| Business rules         | Service | Category active, seller permissions  |
| Cross-field validation | Service | StartDate < EndDate                  |
| External dependencies  | Service | Payment gateway validation           |

**Common Validation Tags:**

```go
// Required
binding:"required"

// String length
binding:"min=3,max=200"

// Numbers
binding:"gt=0"    // greater than
binding:"gte=0"   // greater than or equal
binding:"lt=100"  // less than
binding:"lte=100" // less than or equal

// Format
binding:"email"
binding:"url"
binding:"uuid"
binding:"alphanum"

// Arrays
binding:"max=20"        // max array length
binding:"min=1"         // min array length
binding:"dive"          // validate each element
binding:"unique"        // ensure unique elements

// Custom combinations
binding:"required,email,max=100"
binding:"omitempty,gt=0,lt=1000"
binding:"required,min=1,dive"  // required array with element validation
```

### Rule #6: Proper Context Management

```go
// ✅ Pass context as first parameter
func (s *ProductService) CreateProduct(ctx context.Context, req model.CreateProductRequest) error {
    // Check if context is cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Pass context to downstream calls
    if err := s.repo.Create(ctx, product); err != nil {
        return err
    }
}

// Repository also accepts context
func (r *ProductRepository) Create(ctx context.Context, product *entity.Product) error {
    return r.db.WithContext(ctx).Create(product).Error
}
```

### Rule #7: Return Errors, Don't Panic❌ **Bad:**

```go
func (s *ProductService) GetProduct(id uint) *entity.Product {
    product, err := s.repo.FindByID(id)
    if err != nil {
        panic(err) // ❌ Never panic in application code
    }
    return product
}
```

✅ **Good:**

```go
func (s *ProductService) GetProduct(id uint) (*entity.Product, error) {
    product, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err // ✅ Return error
    }
    return product, nil
}
```

### Rule #8: Use Pointers Wisely

```go
// For large structs or when you need to modify
func (s *ProductService) Update(product *entity.Product) error {
    return s.repo.Update(product) // Pass pointer
}

// For small data or read-only
func (s *ProductService) IsActive(status string) bool {
    return status == "active" // Pass by value
}

// Receiver types - be consistent
type ProductService struct {
    db *gorm.DB
}

// Use pointer receiver for consistency
func (s *ProductService) GetProduct(id uint) (*entity.Product, error) {}
func (s *ProductService) CreateProduct(req model.CreateRequest) error {}
```

### Rule #9: Properly Handle Goroutines

```go
// ✅ Good - Wait for goroutines
func (s *ProductService) BulkProcess(products []entity.Product) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(products))

    for _, p := range products {
        wg.Add(1)
        go func(product entity.Product) {
            defer wg.Done()
            if err := s.ProcessProduct(product); err != nil {
                errChan <- err
            }
        }(p) // ✅ Pass as parameter, don't capture loop variable
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}
```

---

## 🎨 Design Patterns

Below design pattern is just exmaple but we have to always think which pattern we can use to improve scalability, performance, and less maintenance

### Strategy Pattern (Payment Example)

Use when behavior changes based on type:

```go
// Define interface
type PaymentStrategy interface {
    ProcessPayment(amount float64) error
    ValidatePayment() error
}

// Implementations
type WalletPayment struct {
    walletID string
}

func (w *WalletPayment) ProcessPayment(amount float64) error {
    // Wallet-specific logic
}

type CreditCardPayment struct {
    cardNumber string
    cvv        string
}

func (c *CreditCardPayment) ProcessPayment(amount float64) error {
    // Credit card-specific logic
}

// Payment service uses strategy
type PaymentService struct {}

func (s *PaymentService) Pay(strategy PaymentStrategy, amount float64) error {
    if err := strategy.ValidatePayment(); err != nil {
        return err
    }
    return strategy.ProcessPayment(amount)
}

// Usage
func ProcessOrder(paymentType string, amount float64) error {
    var strategy PaymentStrategy

    switch paymentType {
    case "wallet":
        strategy = &WalletPayment{...}
    case "credit_card":
        strategy = &CreditCardPayment{...}
    case "debit_card":
        strategy = &DebitCardPayment{...}
    default:
        return errors.New("unsupported payment type")
    }

    return paymentService.Pay(strategy, amount)
}
```

**Benefits:**

- Add new payment types without modifying existing code (Open/Closed Principle)
- Each payment type is independent and testable

### Factory Pattern (Already Used)

```go
// Used in product/factory/singleton/
type SingletonFactory struct {
    productRepo    repositories.ProductRepository
    productService service.ProductService
}

func GetInstance() *SingletonFactory {
    once.Do(func() {
        instance = &SingletonFactory{}
        instance.initRepositories()
        instance.initServices()
    })
    return instance
}
```

### Repository Pattern (Already Used)

Abstracts data access:

```go
type ProductRepository interface {
    Create(product *entity.Product) error
    FindByID(id uint) (*entity.Product, error)
}

type ProductRepositoryImpl struct {
    db *gorm.DB
}
```

---

## 🔒 Security Guidelines

### Rule #1: Never Log Sensitive Data

❌ **Bad:**

```go
logger.Info("User login attempt",
    "email", user.Email,
    "password", user.Password, // ❌ NEVER log passwords
    "creditCard", user.CreditCard, // ❌ NEVER log PII
)
```

✅ **Good:**

```go
logger.Info("User login attempt",
    "userId", user.ID,
    "email", maskEmail(user.Email), // ✅ Mask or hash
)
```

### Rule #2: Validate All Inputs

```go
func (h *ProductHandler) CreateProduct(c *gin.Context) {
    var req model.CreateProductRequest

    // ✅ Validate request structure
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Error(c, 400, "Invalid request format")
        return
    }

    // ✅ Validate business rules
    if err := validator.ValidateCreateProduct(req); err != nil {
        common.Error(c, 400, err.Error())
        return
    }
}
```

### Rule #3: Sanitize Outputs

```go
// Sanitize before returning to client
type UserResponse struct {
    ID    uint   `json:"id"`
    Email string `json:"email"`
    // ❌ Don't include: Password, PasswordHash, CreditCard, etc.
}
```

### Rule #4: Use Parameterized Queries

```go
// ✅ GORM automatically uses parameterized queries
db.Where("email = ?", email).First(&user)

// ❌ NEVER use string concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
```

---

## 🗄️ Database Migration Guidelines

To ensure database consistency across environments and avoid conflicts, follow these strict rules for all migration and seed scripts.

### Rule #1: Always Create New Migration Scripts

**Critical Rule**: NEVER modify existing migration files that have been committed.

❌ **WRONG - Modifying existing migration:**

```sql
-- migrations/002_create_product_tables.sql (ALREADY COMMITTED)
-- ❌ Don't add new columns here
ALTER TABLE product ADD COLUMN new_field VARCHAR(255);
```

✅ **CORRECT - Create new migration:**

```sql
-- migrations/005_add_product_new_field.sql (NEW FILE)
ALTER TABLE product ADD COLUMN new_field VARCHAR(255);
```

**Why?**

- Other developers may have already run the old migration
- Production/staging databases are already migrated
- Modifying old migrations causes version conflicts
- CI/CD pipelines track migration checksums

**Migration Naming Convention:**

```
migrations/
├── 001_create_user_tables.sql
├── 002_create_product_tables.sql
├── 003_alter_timestamps_to_timestamptz.sql
├── 004_create_related_products_procedure.sql
└── 005_add_your_new_change.sql  ← Next sequential number
```

### Rule #2: Maintain High-Quality Seed Data

Seed data serves TWO critical purposes:

1. **Demo data** for local development
2. **Test data** for integration tests

**Seed Data Requirements:**

✅ **Good Seed Data - Realistic & Consistent:**

```sql
-- seeds/002_seed_product_data.sql

-- Every product MUST have at least one variant (API rule)
INSERT INTO product (id, name, category_id, seller_id, base_sku, created_at, updated_at)
VALUES (1, 'iPhone 15 Pro', 1, 2, 'IPH15PRO', NOW(), NOW());

-- ✅ Variant for the product (required)
INSERT INTO product_variant (product_id, sku, price, stock, is_default, created_at, updated_at)
VALUES (1, 'IPH15PRO-BLK-256', 99999, 50, true, NOW(), NOW());

-- ✅ Realistic relationships
INSERT INTO product_attribute (product_id, key, name, value, unit)
VALUES (1, 'storage', 'Storage', '256', 'GB');
```

❌ **Bad Seed Data - Random/Inconsistent:**

```sql
-- ❌ Product without variant (violates API rules)
INSERT INTO product (id, name, category_id, seller_id)
VALUES (999, 'Test Product', 999, 999);  -- No variant!

-- ❌ Meaningless data
INSERT INTO product (id, name, category_id, seller_id)
VALUES (1, 'asdfasdf', 123, 456);  -- Random garbage

-- ❌ Broken relationships
INSERT INTO product_variant (product_id, sku, price)
VALUES (9999, 'SKU', 100);  -- product_id=9999 doesn't exist!
```

**Seed Data Quality Checklist:**

- [ ] All foreign keys reference existing records
- [ ] Data is realistic and meaningful (not "test123", "asdf")
- [ ] Maintains referential integrity
- [ ] Follows same validation rules as API
- [ ] Covers common use cases for testing
- [ ] Includes edge cases (empty optional fields, max lengths)

### Rule #3: Never Break Existing Tests

Seed data is tightly coupled with integration tests.

❌ **WRONG - Removing/modifying seed data:**

```sql
-- seeds/001_seed_user_data.sql
-- ❌ Don't remove this - tests depend on it!
DELETE FROM "user" WHERE email = 'admin@example.com';

-- ❌ Don't change IDs - tests reference them!
UPDATE "user" SET id = 999 WHERE email = 'admin@example.com';
```

✅ **CORRECT - Adding new seed data:**

```sql
-- seeds/001_seed_user_data.sql
-- ✅ Add new test user without affecting existing ones
INSERT INTO "user" (id, email, password_hash, role, seller_id)
VALUES (10, 'newuser@example.com', '$2a$10$...', 'customer', NULL);
```

**When You MUST Modify Seed Data:**

If changes are absolutely necessary (e.g., bug fix, schema change):

1. **Update seed data**

```sql
-- seeds/002_seed_product_data.sql
UPDATE product SET category_id = 5 WHERE id = 1;
```

2. **Update affected integration tests**

```go
// test/integration/product/get_product_test.go
func TestGetProduct(t *testing.T) {
    // Update expected category_id
    assert.Equal(t, uint(5), product.CategoryID) // Changed from 3 to 5
}
```

3. **Document the change in PR**

```
## Breaking Changes
- Updated product ID=1 category from 3 to 5
- Affected tests: TestGetProduct, TestSearchProducts
- Tests have been updated accordingly
```

### Rule #4: Seed Data for New Features

When adding new tables or columns:

✅ **Add corresponding seed data:**

```sql
-- migrations/006_add_reviews_table.sql
CREATE TABLE review (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT REFERENCES product(id),
    user_id BIGINT REFERENCES "user"(id),
    rating INTEGER NOT NULL,
    comment TEXT
);

-- seeds/004_seed_review_data.sql (NEW SEED FILE)
-- ✅ Add demo reviews for existing products/users
INSERT INTO review (product_id, user_id, rating, comment)
VALUES
    (1, 3, 5, 'Amazing phone! Best purchase ever.'),
    (1, 4, 4, 'Great product but a bit pricey.'),
    (2, 3, 5, 'Perfect laptop for coding.');
```

### Rule #5: Migration Order Matters

Migrations run sequentially. Ensure dependencies are respected.

❌ **WRONG - Out of order:**

```sql
-- migrations/005_add_foreign_key.sql
ALTER TABLE product ADD CONSTRAINT fk_category
    FOREIGN KEY (category_id) REFERENCES category(id);

-- migrations/006_create_category_table.sql
CREATE TABLE category (...);  -- ❌ Too late! FK already referenced it
```

✅ **CORRECT - Proper order:**

```sql
-- migrations/005_create_category_table.sql
CREATE TABLE category (...);

-- migrations/006_add_foreign_key.sql
ALTER TABLE product ADD CONSTRAINT fk_category
    FOREIGN KEY (category_id) REFERENCES category(id);
```

### Rule #6: Test Migrations Locally

Before committing:

```bash
# 1. Backup current database
pg_dump ecommerce > backup.sql

# 2. Drop and recreate database
dropdb ecommerce
createdb ecommerce

# 3. Run ALL migrations from scratch
cd migrations
./run_migrations.sh

# 4. Verify seed data
psql ecommerce -c "SELECT COUNT(*) FROM product;"
psql ecommerce -c "SELECT COUNT(*) FROM product_variant;"

# 5. Run integration tests
go test ./test/integration/... -v

# 6. If all pass, commit your changes
```

### Migration Script Template

```sql
-- migrations/00X_descriptive_name.sql
-- Description: What this migration does
-- Author: Your Name
-- Date: 2025-XX-XX

-- Add your changes here
CREATE TABLE your_table (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Add indexes
CREATE INDEX idx_your_table_name ON your_table(name);

-- Add constraints
ALTER TABLE your_table ADD CONSTRAINT check_name_length
    CHECK (LENGTH(name) >= 3);
```

### Seed Data Template

```sql
-- seeds/00X_seed_your_data.sql
-- Description: Seed data for your_table
-- Purpose: Used by integration tests and local development
-- Author: Your Name
-- Date: 2025-XX-XX

-- Insert realistic data
INSERT INTO your_table (id, name, created_at, updated_at)
VALUES
    (1, 'Meaningful Name 1', NOW(), NOW()),
    (2, 'Meaningful Name 2', NOW(), NOW()),
    (3, 'Edge Case: Very Long Name With Many Characters', NOW(), NOW());

-- Verify relationships
INSERT INTO related_table (your_table_id, other_field)
VALUES
    (1, 'Related Data'),  -- ✅ References existing your_table.id
    (2, 'More Data');
```

---

## ✅ Code Review Checklist

Before submitting a PR, ensure:

### Architecture

- [ ] Does NOT bypass service layers (no direct repo-to-repo calls)
- [ ] Follows standard project structure
- [ ] Uses dependency injection correctly
- [ ] All dependencies are singletons

### Code Quality

- [ ] Methods ≤ 50 lines
- [ ] Files ≤ 500 lines
- [ ] No code duplication (DRY)
- [ ] Follows SOLID principles
- [ ] Appropriate design patterns used

### Model Validation

- [ ] Update models use pointers for optional fields
- [ ] Nullable fields use pointers (can distinguish null from empty)
- [ ] Array/nested structs use `dive` for validation
- [ ] Basic validation in models (length, format, range)
- [ ] Business validation in service layer

### Performance

- [ ] No N+1 queries
- [ ] No redundant loops
- [ ] Efficient database queries (indexes, pagination)
- [ ] Caching implemented where needed

### Error Handling

- [ ] Uses application-level errors (`AppError`)
- [ ] Errors wrapped with context
- [ ] Proper error propagation

### Logging

- [ ] Structured logging with context
- [ ] Appropriate log levels
- [ ] Correlation ID included
- [ ] No sensitive data logged

### Go Best Practices

- [ ] Idiomatic Go naming
- [ ] No global variables
- [ ] Proper context management
- [ ] Returns errors (no panics)
- [ ] Goroutines handled properly

### Security

- [ ] All inputs validated
- [ ] No sensitive data in logs
- [ ] Parameterized queries
- [ ] Outputs sanitized

### Testing

- [ ] Integration tests for all endpoints
- [ ] Edge cases covered
- [ ] Tests pass locally

### Documentation

- [ ] Constants defined (no hardcoded values)
- [ ] Complex logic commented
- [ ] API changes reflected in Postman collection

### Database Migrations

- [ ] New migration file created (never modify existing)
- [ ] Migration tested locally (drop DB, run all migrations)
- [ ] Seed data added/updated if needed
- [ ] Seed data follows API validation rules
- [ ] All foreign keys reference existing records
- [ ] Integration tests updated if seed data changed
- [ ] Migration order respects dependencies

---

## 📚 Quick Reference

### Common Anti-Patterns to Avoid

| ❌ Anti-Pattern                | ✅ Solution                         |
| ------------------------------ | ----------------------------------- |
| Bypassing service layers       | Always call service, not repository |
| Methods > 50 lines             | Extract to smaller methods          |
| Files > 500 lines              | Split into multiple files           |
| Global variables               | Use dependency injection            |
| N+1 queries                    | Use eager loading/joins             |
| Generic error strings          | Use AppError with codes             |
| String concatenation in logs   | Use structured logging              |
| Hardcoded values               | Define constants                    |
| Duplicate code                 | Extract to common module            |
| Fat handlers                   | Move logic to service layer         |
| Update models without pointers | Use pointers for nullable fields    |
| Nested validation missing      | Use `dive` for array/nested objects |
| Business logic in models       | Keep models for validation only     |
| Modifying old migrations       | Always create new migration files   |
| Random/meaningless seed data   | Use realistic, API-compliant data   |
| Breaking test seed data        | Add new data, don't modify existing |

### Standards Summary

```
✅ Structure: Follow product/ as reference
✅ DI: Constructor injection, singletons
✅ Methods: ≤ 50 lines
✅ Files: ≤ 500 lines
✅ Models: Pointers for nullable, dive for nested, validation tags
✅ Performance: Avoid N+1, optimize queries
✅ Errors: Use AppError
✅ Logging: Structured with correlation ID
✅ Go: Idiomatic, context-aware, error-first
✅ Security: Validate inputs, sanitize outputs
✅ Testing: Full integration test coverage
✅ Migrations: Never modify existing, test locally, add seed data
```

---

**Remember**: Quality over speed. Write clean, maintainable code that future developers (including yourself) will thank you for! 🎉
