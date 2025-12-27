# 📊 Inventory Summary APIs - Detailed Design

> **Purpose**: Complete API specifications for inventory dashboard endpoints  
> **Last Updated**: December 7, 2025  
> **Status**: Ready for Implementation

---

## 📋 Table of Contents

1. [API 1: Get Location Summary](#api-1-get-location-summary)
2. [API 2: Get Products at Location](#api-2-get-products-at-location)
3. [API 3: Get Variant Details with Inventory](#api-3-get-variant-details-with-inventory)
4. [Common Response Structure](#common-response-structure)
5. [Error Codes](#error-codes)
6. [Database Query Optimization](#database-query-optimization)
7. [Caching Strategy](#caching-strategy)

---

## 🏢 API 1: Get Location Summary

**Purpose**: Get all locations with aggregated inventory statistics for the seller

### Endpoint

```
GET /api/inventory/locations/summary
```

### Authentication

- **Required**: YES
- **Roles**: `SELLER`, `ADMIN`
- **Headers**:
  - `Authorization: Bearer <JWT_TOKEN>` (required)
  - `X-Correlation-ID: <UUID>` (required)

### Request Parameters

#### Query Parameters

| Parameter   | Type     | Required | Default    | Validation                            | Description                      |
| ----------- | -------- | -------- | ---------- | ------------------------------------- | -------------------------------- |
| `isActive`  | `bool`   | No       | `true`     | `true` or `false`                     | Filter active/inactive locations |
| `type`      | `string` | No       | all        | `WAREHOUSE`, `STORE`, `RETURN_CENTER` | Filter by location type          |
| `sortBy`    | `string` | No       | `priority` | `name`, `priority`, `productCount`    | Sort field                       |
| `sortOrder` | `string` | No       | `asc`      | `asc`, `desc`                         | Sort direction                   |

#### Example Requests

```bash
# Get all active locations (default)
GET /api/inventory/locations/summary

# Get all locations including inactive
GET /api/inventory/locations/summary?isActive=false

# Get only warehouses
GET /api/inventory/locations/summary?type=WAREHOUSE

# Sort by product count descending
GET /api/inventory/locations/summary?sortBy=productCount&sortOrder=desc
```

### Request Validation

```go
type LocationSummaryQuery struct {
    IsActive  *bool  `form:"isActive"`
    Type      string `form:"type" binding:"omitempty,oneof=WAREHOUSE STORE RETURN_CENTER"`
    SortBy    string `form:"sortBy" binding:"omitempty,oneof=name priority productCount"`
    SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

// Validation rules
func (q *LocationSummaryQuery) Validate() error {
    // Type validation (if provided)
    if q.Type != "" {
        validTypes := []string{"WAREHOUSE", "STORE", "RETURN_CENTER"}
        if !contains(validTypes, q.Type) {
            return errors.New("invalid location type")
        }
    }

    // SortBy default
    if q.SortBy == "" {
        q.SortBy = "priority"
    }

    // SortOrder default
    if q.SortOrder == "" {
        q.SortOrder = "asc"
    }

    return nil
}
```

### Response Structure

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Location summary fetched successfully",
  "data": {
    "locations": [
      {
        "id": 1,
        "name": "Main Warehouse",
        "type": "WAREHOUSE",
        "isActive": true,
        "priority": 1,
        "address": {
          "id": 101,
          "street": "123 Warehouse Road",
          "city": "Brooklyn",
          "state": "NY",
          "zipCode": "11201",
          "country": "USA",
          "latitude": 40.6782,
          "longitude": -73.9442
        },
        "inventorySummary": {
          "productCount": 845,
          "variantCount": 2134,
          "totalStock": 12450,
          "totalReserved": 156,
          "totalAvailable": 12294,
          "lowStockCount": 15,
          "outOfStockCount": 3,
          "averageStockValue": 145.67,
          "stockHealth": "GOOD"
        },
        "createdAt": "2025-01-15T10:30:00Z",
        "updatedAt": "2025-12-06T14:22:00Z"
      },
      {
        "id": 2,
        "name": "NYC Flagship Store",
        "type": "STORE",
        "isActive": true,
        "priority": 2,
        "address": {
          "id": 102,
          "street": "456 Fifth Avenue",
          "city": "New York",
          "state": "NY",
          "zipCode": "10001",
          "country": "USA",
          "latitude": 40.7484,
          "longitude": -73.9857
        },
        "inventorySummary": {
          "productCount": 234,
          "variantCount": 567,
          "totalStock": 1890,
          "totalReserved": 45,
          "totalAvailable": 1845,
          "lowStockCount": 5,
          "outOfStockCount": 0,
          "averageStockValue": 8.08,
          "stockHealth": "GOOD"
        },
        "createdAt": "2025-02-10T09:15:00Z",
        "updatedAt": "2025-12-07T08:30:00Z"
      },
      {
        "id": 3,
        "name": "Returns Processing Center",
        "type": "RETURN_CENTER",
        "isActive": true,
        "priority": 5,
        "address": {
          "id": 103,
          "street": "789 Return Avenue",
          "city": "Queens",
          "state": "NY",
          "zipCode": "11375",
          "country": "USA",
          "latitude": 40.7282,
          "longitude": -73.832
        },
        "inventorySummary": {
          "productCount": 45,
          "variantCount": 89,
          "totalStock": 234,
          "totalReserved": 12,
          "totalAvailable": 222,
          "lowStockCount": 3,
          "outOfStockCount": 5,
          "averageStockValue": 5.2,
          "stockHealth": "WARNING"
        },
        "createdAt": "2025-03-01T11:00:00Z",
        "updatedAt": "2025-12-06T16:45:00Z"
      }
    ],
    "metadata": {
      "filters": {
        "isActive": true,
        "type": null,
        "sortBy": "priority",
        "sortOrder": "asc"
      },
      "timestamp": "2025-12-07T10:30:00Z"
    }
  }
}
```

#### Stock Health Calculation

```go
// Stock health is determined by lowStockCount and outOfStockCount ratio
func CalculateStockHealth(lowStockCount, outOfStockCount, totalProducts int) string {
    if totalProducts == 0 {
        return "UNKNOWN"
    }

    criticalRatio := float64(outOfStockCount) / float64(totalProducts)
    warningRatio := float64(lowStockCount) / float64(totalProducts)

    if criticalRatio > 0.1 || outOfStockCount > 10 {
        return "CRITICAL" // >10% out of stock
    }
    if warningRatio > 0.1 || lowStockCount > 20 {
        return "WARNING" // >10% low stock
    }
    return "GOOD"
}
```

#### Empty State Response (200 OK)

```json
{
  "success": true,
  "message": "No locations found",
  "data": {
    "overview": {
      "totalLocations": 0,
      "activeLocations": 0,
      "inactiveLocations": 0,
      "totalProducts": 0,
      "totalStock": 0,
      "totalReserved": 0,
      "totalAvailable": 0,
      "lowStockCount": 0,
      "outOfStockCount": 0,
      "locationTypes": {}
    },
    "locations": [],
    "metadata": {
      "filters": {
        "isActive": true,
        "type": null,
        "sortBy": "priority",
        "sortOrder": "asc"
      },
      "timestamp": "2025-12-07T10:30:00Z"
    }
  }
}
```

### Error Responses

#### 401 Unauthorized

```json
{
  "success": false,
  "message": "Unauthorized: Invalid or missing JWT token",
  "error": {
    "code": "UNAUTHORIZED",
    "details": "Please login to access this resource"
  }
}
```

#### 400 Bad Request

```json
{
  "success": false,
  "message": "Invalid request parameters",
  "error": {
    "code": "INVALID_PARAMS",
    "details": "Invalid location type. Must be one of: WAREHOUSE, STORE, RETURN_CENTER"
  }
}
```

#### 500 Internal Server Error

```json
{
  "success": false,
  "message": "Failed to fetch location summary",
  "error": {
    "code": "INTERNAL_ERROR",
    "details": "Database query failed"
  }
}
```

### Business Logic

```go
func (s *LocationService) GetLocationSummary(
    ctx context.Context,
    sellerID uint,
    query LocationSummaryQuery,
) (*LocationSummaryResponse, error) {
    // 1. Validate query parameters
    if err := query.Validate(); err != nil {
        return nil, errors.NewBadRequest(err.Error())
    }

    // 2. Fetch all locations for seller with filters
    locations, err := s.locationRepo.FindBySellerID(ctx, sellerID, query)
    if err != nil {
        return nil, errors.NewInternalError("Failed to fetch locations")
    }

    // 3. For each location, aggregate inventory data
    var locationSummaries []LocationWithSummary
    var overview OverviewStats

    for _, location := range locations {
        // Aggregate inventory for this location
        summary, err := s.inventoryRepo.GetLocationInventorySummary(
            ctx,
            location.ID,
        )
        if err != nil {
            logger.Error("Failed to get inventory summary",
                "locationId", location.ID,
                "error", err)
            continue
        }

        // Calculate stock health
        summary.StockHealth = CalculateStockHealth(
            summary.LowStockCount,
            summary.OutOfStockCount,
            summary.ProductCount,
        )

        locationSummaries = append(locationSummaries, LocationWithSummary{
            Location:         location,
            InventorySummary: summary,
        })

        // Update overview stats
        overview.TotalProducts += summary.ProductCount
        overview.TotalStock += summary.TotalStock
        overview.TotalReserved += summary.TotalReserved
        overview.LowStockCount += summary.LowStockCount
        overview.OutOfStockCount += summary.OutOfStockCount
    }

    // 4. Calculate overview
    overview.TotalLocations = len(locations)
    overview.ActiveLocations = countActive(locations)
    overview.InactiveLocations = overview.TotalLocations - overview.ActiveLocations
    overview.TotalAvailable = overview.TotalStock - overview.TotalReserved
    overview.LocationTypes = countByType(locations)

    // 5. Build response
    response := &LocationSummaryResponse{
        Overview:  overview,
        Locations: locationSummaries,
        Metadata: Metadata{
            Filters:   query,
            Timestamp: time.Now(),
        },
    }

    return response, nil
}
```

**Service Layer Logic:**

````go
// 1. Get inventory summary from repository (inventory module only)
summary, variantIDs, err := s.inventoryRepo.GetLocationInventorySummary(locationID)

// 2. Call Product Service to get product count (cross-service call)
productCount, err := s.productService.GetProductCountByVariantIDs(variantIDs, sellerID)

// 3. Add product count to summary
summary.ProductCount = productCount
```---

## 📦 API 2: Get Products at Location

**Purpose**: Get all products with aggregated inventory data for a specific location

### Endpoint

````

GET /api/inventory/locations/{locationId}/products

````

### Authentication

- **Required**: YES
- **Roles**: `SELLER`, `ADMIN`
- **Headers**:
  - `Authorization: Bearer <JWT_TOKEN>` (required)
  - `X-Correlation-ID: <UUID>` (required)

### Request Parameters

#### Path Parameters

| Parameter    | Type   | Required | Validation     | Description         |
| ------------ | ------ | -------- | -------------- | ------------------- |
| `locationId` | `uint` | Yes      | Must exist, >0 | Location identifier |

#### Query Parameters

| Parameter     | Type     | Required | Default | Validation                                 | Description                 |
| ------------- | -------- | -------- | ------- | ------------------------------------------ | --------------------------- |
| `page`        | `int`    | No       | `1`     | >=1                                        | Page number                 |
| `pageSize`    | `int`    | No       | `20`    | 1-100                                      | Items per page              |
| `search`      | `string` | No       | -       | max 100 chars                              | Search in product name, SKU |
| `categoryId`  | `uint`   | No       | -       | Must exist                                 | Filter by category          |
| `stockStatus` | `string` | No       | `all`   | `all`, `inStock`, `lowStock`, `outOfStock` | Filter by stock status      |
| `sortBy`      | `string` | No       | `name`  | `name`, `stock`, `lowStock`                | Sort field                  |
| `sortOrder`   | `string` | No       | `asc`   | `asc`, `desc`                              | Sort direction              |

#### Example Requests

```bash
# Get first page of products
GET /api/inventory/locations/1/products

# Search for products
GET /api/inventory/locations/1/products?search=iPhone

# Filter by category
GET /api/inventory/locations/1/products?categoryId=5

# Show only low stock items
GET /api/inventory/locations/1/products?stockStatus=lowStock

# Pagination
GET /api/inventory/locations/1/products?page=2&pageSize=50

# Sort by total stock descending
GET /api/inventory/locations/1/products?sortBy=stock&sortOrder=desc

# Combined filters
GET /api/inventory/locations/1/products?categoryId=5&stockStatus=lowStock&sortBy=name&page=1
````

### Request Validation

```go
type ProductsAtLocationQuery struct {
    Page        int    `form:"page" binding:"omitempty,gte=1"`
    PageSize    int    `form:"pageSize" binding:"omitempty,gte=1,lte=100"`
    Search      string `form:"search" binding:"omitempty,max=100"`
    CategoryID  *uint  `form:"categoryId" binding:"omitempty,gte=1"`
    StockStatus string `form:"stockStatus" binding:"omitempty,oneof=all inStock lowStock outOfStock"`
    SortBy      string `form:"sortBy" binding:"omitempty,oneof=name stock lowStock"`
    SortOrder   string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

func (q *ProductsAtLocationQuery) Validate() error {
    // Set defaults
    if q.Page == 0 {
        q.Page = 1
    }
    if q.PageSize == 0 {
        q.PageSize = 20
    }
    if q.StockStatus == "" {
        q.StockStatus = "all"
    }
    if q.SortBy == "" {
        q.SortBy = "name"
    }
    if q.SortOrder == "" {
        q.SortOrder = "asc"
    }

    return nil
}
```

### Response Structure

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Products fetched successfully",
  "data": {
    "locationInfo": {
      "id": 1,
      "name": "Main Warehouse",
      "type": "WAREHOUSE",
      "address": "123 Warehouse Road, Brooklyn, NY 11201"
    },
    "products": [
      {
        "productId": 10,
        "productName": "iPhone 15 Pro",
        "productImage": "https://cdn.example.com/products/iphone-15-pro.jpg",
        "category": {
          "id": 5,
          "name": "Smartphones",
        },
        "inventorySummary": {
          "variantCount": 3,
          "totalStock": 250,
          "totalReserved": 15,
          "totalAvailable": 235,
          "lowStockVariants": 0,
          "outOfStockVariants": 0,
          "stockStatus": "IN_STOCK",
        }
      },
      {
        "productId": 11,
        "productName": "Samsung Galaxy S24",
        "productSlug": "samsung-galaxy-s24",
        "productImage": "https://cdn.example.com/products/samsung-s24.jpg",
        "category": {
          "id": 5,
          "name": "Smartphones",
          "breadcrumb": "Electronics > Mobile Phones > Smartphones"
        },
        "inventorySummary": {
          "variantCount": 4,
          "totalStock": 89,
          "totalReserved": 5,
          "totalAvailable": 84,
          "lowStockVariants": 2,
          "outOfStockVariants": 0,
          "stockStatus": "LOW_STOCK",
        }
      },
      {
        "productId": 12,
        "productName": "MacBook Pro 14\"",
        "productSlug": "macbook-pro-14",
        "productImage": "https://cdn.example.com/products/macbook-pro-14.jpg",
        "category": {
          "id": 8,
          "name": "Laptops",
          "breadcrumb": "Electronics > Computers > Laptops"
        },
        "inventorySummary": {
          "variantCount": 2,
          "totalStock": 45,
          "totalReserved": 8,
          "totalAvailable": 37,
          "lowStockVariants": 0,
          "outOfStockVariants": 0,
          "stockStatus": "IN_STOCK",
        }
      },
      {
        "productId": 15,
        "productName": "Nike Air Max Sneakers",
        "productSlug": "nike-air-max-sneakers",
        "productImage": "https://cdn.example.com/products/nike-air-max.jpg",
        "category": {
          "id": 20,
          "name": "Footwear",
          "breadcrumb": "Fashion > Shoes > Footwear"
        },
        "inventorySummary": {
          "variantCount": 8,
          "totalStock": 12,
          "totalReserved": 0,
          "totalAvailable": 12,
          "lowStockVariants": 2,
          "outOfStockVariants": 5,
          "stockStatus": "OUT_OF_STOCK",
        }
      }
    ],
    "pagination": {
      "currentPage": 1,
      "pageSize": 20,
      "totalPages": 43,
      "totalItems": 845,
      "hasNextPage": true,
      "hasPreviousPage": false
    },
    "filters": {
      "search": null,
      "categoryId": null,
      "stockStatus": "all",
      "sortBy": "name",
      "sortOrder": "asc"
    }
  }
}
```

#### Stock Status Logic

```go
// Stock status is determined by variant-level stock
func DetermineStockStatus(
    outOfStockVariants int,
    lowStockVariants int,
    variantCount int,
) string {
    if outOfStockVariants == variantCount {
        return "OUT_OF_STOCK" // All variants out of stock
    }
    if outOfStockVariants > 0 || lowStockVariants > 0 {
        return "LOW_STOCK" // At least one variant low/out
    }
    return "IN_STOCK" // All variants in stock
}

func DetermineStockHealth(
    outOfStockVariants int,
    lowStockVariants int,
    variantCount int,
) string {
    outRatio := float64(outOfStockVariants) / float64(variantCount)
    lowRatio := float64(lowStockVariants) / float64(variantCount)

    if outRatio > 0.5 {
        return "CRITICAL" // >50% variants out
    }
    if outRatio > 0.0 || lowRatio > 0.3 {
        return "WARNING" // Any out or >30% low
    }
    return "GOOD"
}
```

#### Empty State Response (200 OK)

```json
{
  "success": true,
  "message": "No products found at this location",
  "data": {
    "locationInfo": {
      "id": 1,
      "name": "Main Warehouse",
      "type": "WAREHOUSE",
      "address": "123 Warehouse Road, Brooklyn, NY 11201"
    },
    "products": [],
    "pagination": {
      "currentPage": 1,
      "pageSize": 20,
      "totalPages": 0,
      "totalItems": 0,
      "hasNextPage": false,
      "hasPreviousPage": false
    },
    "filters": {
      "search": null,
      "categoryId": null,
      "stockStatus": "all",
      "sortBy": "name",
      "sortOrder": "asc"
    }
  }
}
```

### Error Responses

#### 404 Not Found

```json
{
  "success": false,
  "message": "Location not found",
  "error": {
    "code": "LOCATION_NOT_FOUND",
    "details": "Location with ID 999 does not exist or does not belong to this seller"
  }
}
```

#### 400 Bad Request

```json
{
  "success": false,
  "message": "Invalid request parameters",
  "error": {
    "code": "INVALID_PARAMS",
    "details": "pageSize must be between 1 and 100"
  }
}
```

### Business Logic

```go
func (s *InventoryService) GetProductsAtLocation(
    ctx context.Context,
    locationID uint,
    sellerID uint,
    query ProductsAtLocationQuery,
) (*ProductsAtLocationResponse, error) {
    // 1. Validate query
    if err := query.Validate(); err != nil {
        return nil, errors.NewBadRequest(err.Error())
    }

    // 2. Verify location exists and belongs to seller
    location, err := s.locationRepo.FindByID(ctx, locationID)
    if err != nil || location.SellerID != sellerID {
        return nil, errors.LocationNotFound
    }

    // 3. Build filters
    filters := buildProductFilters(query)

    // 4. Get total count for pagination
    totalItems, err := s.inventoryRepo.CountProductsAtLocation(
        ctx,
        locationID,
        filters,
    )
    if err != nil {
        return nil, errors.NewInternalError("Failed to count products")
    }

    // 5. Fetch products with aggregated inventory
    products, err := s.inventoryRepo.GetProductsAtLocationWithInventory(
        ctx,
        locationID,
        filters,
        query.Page,
        query.PageSize,
    )
    if err != nil {
        return nil, errors.NewInternalError("Failed to fetch products")
    }

    // 6. Calculate stock status and health for each product
    for i := range products {
        products[i].InventorySummary.StockStatus = DetermineStockStatus(
            products[i].InventorySummary.OutOfStockVariants,
            products[i].InventorySummary.LowStockVariants,
            products[i].InventorySummary.VariantCount,
        )
        products[i].InventorySummary.StockHealth = DetermineStockHealth(
            products[i].InventorySummary.OutOfStockVariants,
            products[i].InventorySummary.LowStockVariants,
            products[i].InventorySummary.VariantCount,
        )
    }

    // 7. Build pagination
    pagination := buildPagination(query.Page, query.PageSize, totalItems)

    // 8. Get summary stats
    summary, err := s.inventoryRepo.GetLocationProductSummary(ctx, locationID)
    if err != nil {
        logger.Error("Failed to get summary", "error", err)
        // Continue without summary
    }

    // 9. Build response
    response := &ProductsAtLocationResponse{
        LocationInfo: buildLocationInfo(location),
        Products:     products,
        Pagination:   pagination,
        Filters:      query,
        Summary:      summary,
    }

    return response, nil
}
```

### SQL Queries (Microservice-Ready Approach)

#### Step 1: Inventory Repository Query (Inventory Module Only)

```sql
-- Get inventory aggregated by variant at a location (NO product joins)
SELECT
    i.variant_id,
    COUNT(i.id) as inventory_records,
    COALESCE(SUM(i.quantity), 0) as total_stock,
    COALESCE(SUM(i.reserved_quantity), 0) as total_reserved,
    COALESCE(SUM(i.quantity - i.reserved_quantity), 0) as total_available,
    MAX(CASE WHEN i.quantity > 0 AND i.quantity <= i.threshold THEN 1 ELSE 0 END) as is_low_stock,
    MAX(CASE WHEN i.quantity = 0 THEN 1 ELSE 0 END) as is_out_of_stock
FROM inventory i
WHERE i.location_id = $1
    AND i.deleted_at IS NULL
GROUP BY i.variant_id
ORDER BY i.variant_id;
-- Returns: List of variants with inventory stats at this location
```

#### Step 2: Service Layer Orchestration

```go
func (s *InventoryService) GetProductsAtLocation(
    ctx context.Context,
    locationID uint,
    sellerID uint,
    query ProductsAtLocationQuery,
) (*ProductsAtLocationResponse, error) {
    // 1. Get inventory data from inventory repository (inventory module only)
    variantInventories, err := s.inventoryRepo.GetVariantInventoriesAtLocation(locationID)
    if err != nil {
        return nil, err
    }

    // Extract variant IDs
    variantIDs := make([]uint, len(variantInventories))
    for i, inv := range variantInventories {
        variantIDs[i] = inv.VariantID
    }

    // 2. Call Product Service to get variant → product mapping (cross-service)
    // This returns: map[variantID]ProductBasicInfo
    productsByVariant, err := s.productService.GetProductsByVariantIDs(
        ctx,
        variantIDs,
        sellerID,
    )
    if err != nil {
        return nil, err
    }

    // 3. Group inventory by product ID (in-memory aggregation)
    productInventoryMap := make(map[uint]*ProductInventoryAgg)

    for _, varInv := range variantInventories {
        productInfo := productsByVariant[varInv.VariantID]
        if productInfo == nil {
            continue // Skip if product not found (shouldn't happen)
        }

        productID := productInfo.ID

        // Initialize if first time seeing this product
        if productInventoryMap[productID] == nil {
            productInventoryMap[productID] = &ProductInventoryAgg{
                ProductID:    productID,
                ProductName:  productInfo.Name,
                ProductSlug:  productInfo.Slug,
                ProductImage: productInfo.ImageURL,
                Category:     productInfo.Category,
                PriceRange:   productInfo.PriceRange,
                InventorySummary: InventorySummary{
                    VariantCount: 0,
                    TotalStock: 0,
                    TotalReserved: 0,
                    TotalAvailable: 0,
                    LowStockVariants: 0,
                    OutOfStockVariants: 0,
                },
            }
        }

        // Aggregate inventory stats for this product
        agg := productInventoryMap[productID]
        agg.InventorySummary.VariantCount++
        agg.InventorySummary.TotalStock += varInv.TotalStock
        agg.InventorySummary.TotalReserved += varInv.TotalReserved
        agg.InventorySummary.TotalAvailable += varInv.TotalAvailable

        if varInv.IsLowStock {
            agg.InventorySummary.LowStockVariants++
        }
        if varInv.IsOutOfStock {
            agg.InventorySummary.OutOfStockVariants++
        }
    }

    // 4. Convert map to slice and apply filters/sorting
    products := convertMapToSlice(productInventoryMap)
    products = applyFilters(products, query)
    products = applySorting(products, query)

    // 5. Apply pagination
    totalItems := len(products)
    paginatedProducts := applyPagination(products, query.Page, query.PageSize)

    // 6. Calculate stock status and health
    for i := range paginatedProducts {
        paginatedProducts[i].InventorySummary.StockStatus = DetermineStockStatus(
            paginatedProducts[i].InventorySummary.OutOfStockVariants,
            paginatedProducts[i].InventorySummary.LowStockVariants,
            paginatedProducts[i].InventorySummary.VariantCount,
        )
        paginatedProducts[i].InventorySummary.StockHealth = DetermineStockHealth(
            paginatedProducts[i].InventorySummary.OutOfStockVariants,
            paginatedProducts[i].InventorySummary.LowStockVariants,
            paginatedProducts[i].InventorySummary.VariantCount,
        )
    }

    // 7. Build response
    return &ProductsAtLocationResponse{
        LocationInfo: buildLocationInfo(location),
        Products:     paginatedProducts,
        Pagination:   buildPagination(query.Page, query.PageSize, totalItems),
        Filters:      query,
        Summary:      calculateSummary(products),
    }, nil
}
```

#### Step 3: Product Service Method (NEW - To be added)

```go
// In product/service/variant_query_service.go

// GetProductsByVariantIDs returns product basic info grouped by variant IDs
// This enables inventory service to group inventory by product without DB joins
func (s *VariantQueryServiceImpl) GetProductsByVariantIDs(
    ctx context.Context,
    variantIDs []uint,
    sellerID *uint,
) (map[uint]*model.ProductBasicInfo, error) {
    // 1. Batch query variants with product info
    variants, err := s.variantRepo.FindVariantsByIDsWithProduct(variantIDs)
    if err != nil {
        return nil, err
    }

    // 2. Build map: variantID → ProductBasicInfo
    result := make(map[uint]*model.ProductBasicInfo)

    for _, variant := range variants {
        // Validate seller ownership
        if sellerID != nil && variant.Product.SellerID != *sellerID {
            continue // Skip variants not owned by seller
        }

        result[variant.ID] = &model.ProductBasicInfo{
            ID:       variant.Product.ID,
            Name:     variant.Product.Name,
            Slug:     variant.Product.Slug,
            ImageURL: variant.Product.ImageURL,
            Category: buildCategoryInfo(variant.Product.Category),
            Brand:    variant.Product.Brand,
            PriceRange: PriceRange{
                Min: variant.Price, // Individual variant price
                Max: variant.Price, // Will be aggregated by inventory service
            },
        }
    }

    return result, nil
}
```

**Why This Approach is Better:**

✅ **Microservice-Ready**: Inventory module only queries `inventory` table  
✅ **Clear Separation**: Product data comes from Product Service  
✅ **No DB Joins**: Each service queries its own database  
✅ **Scalable**: Can split into separate databases/services anytime  
✅ **Performance**: 2 queries total (inventory + product batch lookup) vs N+1---

## 🎨 VariantInventoryInfo

**Purpose**: Get detailed inventory information for all variants of a product at a specific location

### Endpoint

```
GET /api/inventory/products/{productId}/variants
```

### Authentication

- **Required**: YES
- **Roles**: `SELLER`, `ADMIN`
- **Headers**:
  - `Authorization: Bearer <JWT_TOKEN>` (required)
  - `X-Correlation-ID: <UUID>` (required)

### Request Parameters

#### Path Parameters

| Parameter   | Type   | Required | Validation     | Description        |
| ----------- | ------ | -------- | -------------- | ------------------ |
| `productId` | `uint` | Yes      | Must exist, >0 | Product identifier |

#### Query Parameters

| Parameter             | Type   | Required | Validation        | Description                                |
| --------------------- | ------ | -------- | ----------------- | ------------------------------------------ |
| `locationId`          | `uint` | Yes      | Must exist, >0    | Location to check inventory                |
| `includeTransactions` | `bool` | No       | `true` or `false` | Include recent transactions                |
| `transactionLimit`    | `int`  | No       | 1-50              | Number of recent transactions (default: 3) |

#### Example Requests

```bash
# Get variants with inventory at location
GET /api/inventory/products/10/variants?locationId=1

# Include recent transactions
GET /api/inventory/products/10/variants?locationId=1&includeTransactions=true

# Get more transaction history
GET /api/inventory/products/10/variants?locationId=1&includeTransactions=true&transactionLimit=10
```

### Request Validation

```go
type VariantInventoryQuery struct {
    LocationID           uint `form:"locationId" binding:"required,gte=1"`
    IncludeTransactions  bool `form:"includeTransactions"`
    TransactionLimit     int  `form:"transactionLimit" binding:"omitempty,gte=1,lte=50"`
}

func (q *VariantInventoryQuery) Validate() error {
    if q.LocationID == 0 {
        return errors.New("locationId is required")
    }
    if q.TransactionLimit == 0 {
        q.TransactionLimit = 3 // Default
    }
    return nil
}
```

### Response Structure

#### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Variant inventory fetched successfully",
  "data": {
    "productInfo": {
      "id": 10,
      "name": "iPhone 15 Pro",
      "slug": "iphone-15-pro",
      "description": "Latest iPhone with advanced features",
      "basePrice": 999.99,
      "category": {
        "id": 5,
        "name": "Smartphones",
        "breadcrumb": "Electronics > Mobile Phones > Smartphones"
      },
      "brand": "Apple",
      "sellerId": 2
    },
    "locationInfo": {
      "id": 1,
      "name": "Main Warehouse",
      "type": "WAREHOUSE",
      "address": "123 Warehouse Road, Brooklyn, NY 11201"
    },
    "variants": [
      {
        "variantId": 55,
        "sku": "IPH15PRO-BLK-256",
        "variantName": "Black - 256GB",
        "imageUrl": "https://cdn.example.com/variants/iph15pro-black-256.jpg",
        "options": [
          {
            "name": "Color",
            "value": "Black"
          },
          {
            "name": "Storage",
            "value": "256GB"
          }
        ],
        "inventory": {
          "inventoryId": 101,
          "quantity": 120,
          "reservedQuantity": 5,
          "availableQuantity": 115,
          "threshold": 10,
          "binLocation": "A-12-03",
          "stockStatus": "IN_STOCK",
          "stockLevel": "HEALTHY",
          "lastRestocked": "2025-12-06T10:30:00Z",
          "lastSold": "2025-12-07T09:15:00Z",
          "turnoverDays": 15
        },
        "recentTransactions": [
          {
            "id": 501,
            "type": "PURCHASE_ORDER",
            "transactionType": "IN",
            "quantity": 50,
            "quantityBefore": 70,
            "quantityAfter": 120,
            "reference": "PO-1234",
            "note": "Restocking from supplier",
            "performedBy": {
              "userId": 2,
              "name": "John Seller",
              "email": "john.seller@example.com"
            },
            "createdAt": "2025-12-06T10:30:00Z"
          },
          {
            "id": 498,
            "type": "SALES_ORDER",
            "transactionType": "OUT",
            "quantity": -25,
            "quantityBefore": 95,
            "quantityAfter": 70,
            "reference": "SO-5678",
            "note": "Order fulfillment - Order #12345",
            "performedBy": {
              "userId": 2,
              "name": "John Seller",
              "email": "john.seller@example.com"
            },
            "createdAt": "2025-12-05T14:22:00Z"
          },
          {
            "id": 495,
            "type": "ADJUSTMENT",
            "transactionType": "IN",
            "quantity": 5,
            "quantityBefore": 90,
            "quantityAfter": 95,
            "reference": "ADJ-789",
            "note": "Found extra units during cycle count",
            "performedBy": {
              "userId": 5,
              "name": "Warehouse Manager",
              "email": "manager@example.com"
            },
            "createdAt": "2025-12-04T11:00:00Z"
          }
        ]
      },
      {
        "variantId": 56,
        "sku": "IPH15PRO-WHT-256",
        "variantName": "White - 256GB",
        "price": 999.99,
        "compareAtPrice": 1099.99,
        "costPrice": 750.0,
        "imageUrl": "https://cdn.example.com/variants/iph15pro-white-256.jpg",
        "barcode": "1234567890124",
        "options": [
          {
            "name": "Color",
            "value": "White"
          },
          {
            "name": "Storage",
            "value": "256GB"
          }
        ],
        "inventory": {
          "inventoryId": 102,
          "quantity": 18,
          "reservedQuantity": 7,
          "availableQuantity": 11,
          "threshold": 20,
          "binLocation": "A-12-04",
          "stockStatus": "LOW_STOCK",
          "lastRestocked": "2025-12-04T08:00:00Z",
          "lastSold": "2025-12-07T10:30:00Z",
          "turnoverDays": 8
        },
        "recentTransactions": [
          {
            "id": 502,
            "type": "SALES_ORDER",
            "transactionType": "OUT",
            "quantity": -15,
            "quantityBefore": 33,
            "quantityAfter": 18,
            "reference": "SO-5680",
            "note": "Bulk order fulfillment",
            "performedBy": {
              "userId": 2,
              "name": "John Seller",
              "email": "john.seller@example.com"
            },
            "createdAt": "2025-12-06T16:45:00Z"
          },
          {
            "id": 496,
            "type": "PURCHASE_ORDER",
            "transactionType": "IN",
            "quantity": 30,
            "quantityBefore": 3,
            "quantityAfter": 33,
            "reference": "PO-1230",
            "note": "Emergency restock",
            "performedBy": {
              "userId": 2,
              "name": "John Seller",
              "email": "john.seller@example.com"
            },
            "createdAt": "2025-12-04T08:00:00Z"
          }
        ]
      },
      {
        "variantId": 57,
        "sku": "IPH15PRO-BLK-512",
        "variantName": "Black - 512GB",
        "price": 1199.99,
        "compareAtPrice": 1299.99,
        "costPrice": 900.0,
        "imageUrl": "https://cdn.example.com/variants/iph15pro-black-512.jpg",
        "barcode": "1234567890125",
        "options": [
          {
            "name": "Color",
            "value": "Black"
          },
          {
            "name": "Storage",
            "value": "512GB"
          }
        ],
        "inventory": {
          "inventoryId": 103,
          "quantity": 50,
          "reservedQuantity": 3,
          "availableQuantity": 47,
          "threshold": 10,
          "binLocation": "A-12-05",
          "stockStatus": "IN_STOCK",
          "stockLevel": "HEALTHY",
          "lastRestocked": "2025-12-01T12:00:00Z",
          "lastSold": "2025-12-06T15:30:00Z",
          "turnoverDays": 20
        },
        "recentTransactions": [
          {
            "id": 499,
            "type": "SALES_ORDER",
            "transactionType": "OUT",
            "quantity": -8,
            "quantityBefore": 58,
            "quantityAfter": 50,
            "reference": "SO-5679",
            "note": "Regular order",
            "performedBy": {
              "userId": 2,
              "name": "John Seller",
              "email": "john.seller@example.com"
            },
            "createdAt": "2025-12-05T09:20:00Z"
          }
        ]
      }
    ],
    "aggregatedSummary": {
      "totalVariants": 3,
      "totalStock": 188,
      "totalReserved": 15,
      "totalAvailable": 173,
      "lowStockVariants": 1,
      "outOfStockVariants": 0,
      "totalValue": 187810.12,
      "averageStockPerVariant": 62.67,
      "stockHealth": "WARNING"
    }
  }
}
```

#### Stock Level Calculation

```go
// Stock level indicates urgency
func DetermineStockLevel(quantity, threshold int) string {
    if quantity == 0 {
        return "OUT_OF_STOCK"
    }
    if quantity < 0 {
        return "BACKORDER"
    }
    if quantity <= threshold {
        return "WARNING" // At or below threshold
    }
    if quantity <= threshold*2 {
        return "LOW" // Within 2x threshold
    }
    return "HEALTHY"
}

func DetermineStockStatus(quantity int) string {
    if quantity < 0 {
        return "BACKORDER"
    }
    if quantity == 0 {
        return "OUT_OF_STOCK"
    }
    return "IN_STOCK"
}
```

#### Empty State (No Inventory at Location)

```json
{
  "success": true,
  "message": "No inventory found for this product at the specified location",
  "data": {
    "productInfo": {
      "id": 10,
      "name": "iPhone 15 Pro",
      "slug": "iphone-15-pro",
      "category": {
        "id": 5,
        "name": "Smartphones",
        "breadcrumb": "Electronics > Mobile Phones > Smartphones"
      }
    },
    "locationInfo": {
      "id": 1,
      "name": "Main Warehouse",
      "type": "WAREHOUSE",
      "address": "123 Warehouse Road, Brooklyn, NY 11201"
    },
    "variants": [],
    "aggregatedSummary": {
      "totalVariants": 0,
      "totalStock": 0,
      "totalReserved": 0,
      "totalAvailable": 0,
      "lowStockVariants": 0,
      "outOfStockVariants": 0,
      "totalValue": 0,
      "averageStockPerVariant": 0,
      "stockHealth": "UNKNOWN"
    }
  }
}
```

### Error Responses

#### 404 Not Found - Product

```json
{
  "success": false,
  "message": "Product not found",
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "details": "Product with ID 999 does not exist or does not belong to this seller"
  }
}
```

#### 404 Not Found - Location

```json
{
  "success": false,
  "message": "Location not found",
  "error": {
    "code": "LOCATION_NOT_FOUND",
    "details": "Location with ID 999 does not exist or does not belong to this seller"
  }
}
```

#### 400 Bad Request

```json
{
  "success": false,
  "message": "Invalid request parameters",
  "error": {
    "code": "INVALID_PARAMS",
    "details": "locationId is required"
  }
}
```

### Business Logic

```go
func (s *InventoryService) GetProductVariantInventory(
    ctx context.Context,
    productID uint,
    sellerID uint,
    query VariantInventoryQuery,
) (*VariantInventoryResponse, error) {
    // 1. Validate query
    if err := query.Validate(); err != nil {
        return nil, errors.NewBadRequest(err.Error())
    }

    // 2. Verify product exists and belongs to seller
    product, err := s.productRepo.FindByID(ctx, productID)
    if err != nil || product.SellerID != sellerID {
        return nil, errors.ProductNotFound
    }

    // 3. Verify location exists and belongs to seller
    location, err := s.locationRepo.FindByID(ctx, query.LocationID)
    if err != nil || location.SellerID != sellerID {
        return nil, errors.LocationNotFound
    }

    // 4. Get all variants for the product
    variants, err := s.variantRepo.FindByProductID(ctx, productID)
    if err != nil {
        return nil, errors.NewInternalError("Failed to fetch variants")
    }

    // 5. For each variant, get inventory at location
    var variantInventories []VariantWithInventory
    var aggregated AggregatedSummary

    for _, variant := range variants {
        // Get inventory for this variant at location
        inventory, err := s.inventoryRepo.FindByVariantAndLocation(
            ctx,
            variant.ID,
            query.LocationID,
        )
        if err != nil {
            // No inventory record for this variant at this location
            continue
        }

        // Calculate derived fields
        inventory.AvailableQuantity = inventory.Quantity - inventory.ReservedQuantity
        inventory.StockStatus = DetermineStockStatus(inventory.Quantity)
        inventory.StockLevel = DetermineStockLevel(inventory.Quantity, inventory.Threshold)

        // Get recent transactions if requested
        var transactions []Transaction
        if query.IncludeTransactions {
            transactions, err = s.transactionRepo.GetRecentByInventory(
                ctx,
                inventory.ID,
                query.TransactionLimit,
            )
            if err != nil {
                logger.Error("Failed to fetch transactions",
                    "inventoryId", inventory.ID,
                    "error", err)
                transactions = []Transaction{}
            }
        }

        variantInventories = append(variantInventories, VariantWithInventory{
            Variant:             variant,
            Inventory:           inventory,
            RecentTransactions: transactions,
        })

        // Update aggregated summary
        aggregated.TotalVariants++
        aggregated.TotalStock += inventory.Quantity
        aggregated.TotalReserved += inventory.ReservedQuantity
        aggregated.TotalAvailable += inventory.AvailableQuantity
        if inventory.StockLevel == "WARNING" || inventory.StockLevel == "LOW" {
            aggregated.LowStockVariants++
        }
        if inventory.StockStatus == "OUT_OF_STOCK" {
            aggregated.OutOfStockVariants++
        }
        aggregated.TotalValue += float64(inventory.Quantity) * variant.CostPrice
    }

    // Calculate averages
    if aggregated.TotalVariants > 0 {
        aggregated.AverageStockPerVariant = float64(aggregated.TotalStock) / float64(aggregated.TotalVariants)
    }

    // Determine overall health
    aggregated.StockHealth = DetermineStockHealth(
        aggregated.OutOfStockVariants,
        aggregated.LowStockVariants,
        aggregated.TotalVariants,
    )

    // 6. Build response
    response := &VariantInventoryResponse{
        ProductInfo:       buildProductInfo(product),
        LocationInfo:      buildLocationInfo(location),
        Variants:          variantInventories,
        AggregatedSummary: aggregated,
    }

    return response, nil
}
```

### SQL Queries

#### Get Inventory for Variant at Location

```sql
SELECT
    i.id,
    i.variant_id,
    i.location_id,
    i.quantity,
    i.reserved_quantity,
    i.threshold,
    i.bin_location,
    i.created_at,
    i.updated_at,
    -- Get last restock date
    (SELECT MAX(created_at) FROM inventory_transaction
     WHERE inventory_id = i.id AND type = 'PURCHASE_ORDER') as last_restocked,
    -- Get last sale date
    (SELECT MAX(created_at) FROM inventory_transaction
     WHERE inventory_id = i.id AND type = 'SALES_ORDER') as last_sold
FROM inventory i
WHERE i.variant_id = $1
    AND i.location_id = $2
    AND i.deleted_at IS NULL;
```

#### Get Recent Transactions

```sql
SELECT
    it.id,
    it.type,
    it.transaction_type,
    it.quantity,
    it.quantity_before,
    it.quantity_after,
    it.reference_number as reference,
    it.note,
    it.performed_by,
    it.created_at,
    u.id as user_id,
    u.name as user_name,
    u.email as user_email
FROM inventory_transaction it
LEFT JOIN "user" u ON it.performed_by = u.id
WHERE it.inventory_id = $1
    AND it.deleted_at IS NULL
ORDER BY it.created_at DESC
LIMIT $2;
```

---

## 🔄 Common Response Structure

All APIs follow this structure:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Details string `json:"details"`
}
```

---

## ❌ Error Codes

| HTTP Status | Error Code           | Description                  | Example                              |
| ----------- | -------------------- | ---------------------------- | ------------------------------------ |
| 400         | `INVALID_PARAMS`     | Invalid query parameters     | pageSize > 100                       |
| 400         | `MISSING_REQUIRED`   | Missing required parameter   | locationId not provided              |
| 401         | `UNAUTHORIZED`       | Missing or invalid JWT       | No Authorization header              |
| 403         | `FORBIDDEN`          | User doesn't have permission | Accessing another seller's inventory |
| 404         | `LOCATION_NOT_FOUND` | Location doesn't exist       | Location ID 999 not found            |
| 404         | `PRODUCT_NOT_FOUND`  | Product doesn't exist        | Product ID 999 not found             |
| 404         | `CATEGORY_NOT_FOUND` | Category doesn't exist       | Category ID 999 not found            |
| 500         | `INTERNAL_ERROR`     | Server error                 | Database connection failed           |
| 500         | `DATABASE_ERROR`     | Database query failed        | SQL syntax error                     |

---

## ⚡ Database Query Optimization

### Indexes Required

```sql
-- Location queries
CREATE INDEX idx_location_seller_active ON location(seller_id, is_active);
CREATE INDEX idx_location_type ON location(type);

-- Inventory aggregation
CREATE INDEX idx_inventory_location ON inventory(location_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_variant ON inventory(variant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_location_variant ON inventory(location_id, variant_id) WHERE deleted_at IS NULL;

-- Product queries
CREATE INDEX idx_product_variant_product ON product_variant(product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_product_category ON product(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_product_seller ON product(seller_id) WHERE deleted_at IS NULL;

-- Transaction history
CREATE INDEX idx_transaction_inventory_created ON inventory_transaction(inventory_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_transaction_type ON inventory_transaction(type);

-- Search optimization
CREATE INDEX idx_product_name_gin ON product USING gin(to_tsvector('english', name));
CREATE INDEX idx_variant_sku_gin ON product_variant USING gin(to_tsvector('english', sku));
```

### Query Performance Tips

1. **Use Aggregation in SQL**: Aggregate in database, not in application
2. **Limit Joins**: Only join necessary tables
3. **Pagination**: Always use LIMIT/OFFSET
4. **Selective Fields**: SELECT only needed columns
5. **Prepared Statements**: Reuse query plans

---

## 🗄️ Caching Strategy

### Cache Keys

```go
// Location summary
cacheKey := fmt.Sprintf("inventory:location:summary:seller:%d:active:%t:type:%s",
    sellerID, isActive, locationType)
ttl := 5 * time.Minute

// Products at location
cacheKey := fmt.Sprintf("inventory:location:%d:products:page:%d:size:%d:filter:%s",
    locationID, page, pageSize, filterHash)
ttl := 3 * time.Minute

// Variant inventory
cacheKey := fmt.Sprintf("inventory:product:%d:location:%d:variants",
    productID, locationID)
ttl := 2 * time.Minute
```

### Cache Invalidation Rules

```go
// Invalidate when:
// 1. Location created/updated/deleted
InvalidatePattern("inventory:location:summary:*")

// 2. Inventory adjusted
InvalidatePattern(fmt.Sprintf("inventory:location:%d:*", locationID))
InvalidatePattern(fmt.Sprintf("inventory:product:%d:*", productID))

// 3. Stock transfer
InvalidatePattern(fmt.Sprintf("inventory:location:%d:*", fromLocationID))
InvalidatePattern(fmt.Sprintf("inventory:location:%d:*", toLocationID))

// 4. Product/variant updated
InvalidatePattern(fmt.Sprintf("inventory:product:%d:*", productID))
```

---

## 🧪 Testing Checklist

### API 1: Location Summary

- [ ] Returns correct overview stats
- [ ] Filters by isActive correctly
- [ ] Filters by type correctly
- [ ] Sorts by different fields
- [ ] Handles empty state
- [ ] Validates seller ownership
- [ ] Handles large datasets (1000+ locations)
- [ ] Cache works correctly

### API 2: Products at Location

- [ ] Pagination works correctly
- [ ] Search finds products by name and SKU
- [ ] Filters by category
- [ ] Filters by stock status (in/low/out)
- [ ] Sorting works for all fields
- [ ] Handles location not found
- [ ] Calculates stock health correctly
- [ ] Returns correct price range
- [ ] Handles empty state

### API 3: Variant Details

- [ ] Returns all variants with inventory
- [ ] Calculates available quantity correctly
- [ ] Determines stock status correctly
- [ ] Recent transactions sorted by date
- [ ] Handles missing location/product
- [ ] Aggregated summary is accurate
- [ ] Handles variants without inventory
- [ ] Transaction limit works correctly

---

## 📝 Implementation Checklist

### Backend Tasks

- [ ] Create repository methods for aggregation queries
- [ ] Implement service layer with business logic
- [ ] Create handler methods with validation
- [ ] Add route definitions with middleware
- [ ] Write integration tests for all endpoints
- [ ] Add database indexes for performance
- [ ] Implement caching layer
- [ ] Add error handling and logging
- [ ] Document in Postman collection
- [ ] Add API rate limiting

### Frontend Integration

- [ ] Create API client methods
- [ ] Implement state management
- [ ] Add loading and error states
- [ ] Create UI components for each stage
- [ ] Add search and filter functionality
- [ ] Implement pagination
- [ ] Add real-time updates (optional)
- [ ] Mobile responsive design

---

**This design is production-ready and follows all architectural guidelines. You can start implementation immediately!** 🚀
