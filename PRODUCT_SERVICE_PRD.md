# Product Service API - Product Requirement Document (PRD)

## 1. Overview

This document outlines the API specifications for the Product Service in our e-commerce platform. The Product Service is designed with a generalized, industry-agnostic approach that supports dynamic attributes, making it suitable for any type of product across various industries.

## 2. Architecture Overview

### 2.1 Key Components

- **Category Management**: Hierarchical category structure with dynamic attribute configuration
- **Product Management**: Core product information with dynamic attributes
- **Attribute System**: Flexible attribute definitions and validations
- **Package Options**: Product variants and packaging

### 2.2 Database Tables

- `category` - Hierarchical product categories (singular table names)
- `attribute_definition` - Master attribute definitions  
- `category_attribute` - Category-specific attribute configurations
- `product` - Core product information
- `product_attribute` - Dynamic product attribute values
- `package_option` - Product variants and packages

**Note**: All table names use singular form as per the GORM configuration with `SingularTable: true`.

## 3. API Specifications

### 3.1 Category Management APIs

#### 3.1.1 Get All Categories
- **Endpoint**: `GET /api/categories`
- **Description**: Get hierarchical list of all product categories
- **Headers**: 
  - `Authorization`: Bearer token (optional for public access)
- **Query Parameters**:
  - `parentId`: Filter by parent category ID (integer, optional)
- **Request Body**: None

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Categories retrieved successfully",
  "data": {
    "categories": [
      {
        "id": 1,
        "name": "Electronics",
        "parentId": null,
        "description": "Electronic products and devices",
        "children": [
          {
            "id": 2,
            "name": "Smartphones",
            "parentId": 1,
            "description": "Mobile phones and accessories",
            "children": []
          },
          {
            "id": 3,
            "name": "Laptops",
            "parentId": 1,
            "description": "Portable computers",
            "children": []
          }
        ]
      }
    ]
  }
}
```

#### 3.1.2 Create Category
- **Endpoint**: `POST /api/categories`
- **Description**: Create a new product category
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
  - `Content-Type`: application/json
- **Request Body**:
```json
{
  "name": "Gaming Laptops",
  "parentId": 3,
  "description": "High-performance laptops for gaming"
}
```

**Validation Rules**:
- `name`: Required, 3-100 characters, unique within parent
- `parentId`: Optional, must exist if provided
- `description`: Optional, max 500 characters

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Category created successfully",
  "data": {
    "category": {
      "id": 4,
      "name": "Gaming Laptops",
      "parentId": 3,
      "description": "High-performance laptops for gaming",
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.1.3 Update Category
- **Endpoint**: `PUT /api/categories/{categoryId}`
- **Description**: Update an existing category
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `categoryId`: Category ID to update
- **Request Body**:
```json
{
  "name": "Premium Gaming Laptops",
  "description": "High-end gaming laptops with premium features"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Category updated successfully",
  "data": {
    "category": {
      "id": 4,
      "name": "Premium Gaming Laptops",
      "parentId": 3,
      "description": "High-end gaming laptops with premium features",
      "updatedAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

#### 3.1.4 Delete Category
- **Endpoint**: `DELETE /api/categories/{categoryId}`
- **Description**: Permanently delete a category
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
- **Path Parameters**:
  - `categoryId`: Category ID to delete

**Business Rules**:
- Cannot delete category with products
- Cannot delete category with child categories
- Permanent deletion (hard delete)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Category deleted successfully"
}
```

#### 3.1.5 Get Category Attributes with Inheritance
- **Endpoint**: `GET /api/categories/{categoryId}/attributes`
- **Description**: Get all attributes for a category including inherited attributes from parent categories
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
- **Path Parameters**:
  - `categoryId`: Category ID

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Category attributes retrieved successfully",
  "data": {
    "attributes": [
      {
        "id": 1,
        "key": "warranty_period",
        "name": "Warranty Period",
        "dataType": "number",
        "unit": "months",
        "description": "Product warranty duration",
        "allowedValues": null,
        "createdAt": "2024-01-15T10:30:00Z"
      },
      {
        "id": 2,
        "key": "color",
        "name": "Color",
        "dataType": "string",
        "unit": null,
        "description": "Product color",
        "allowedValues": ["Red", "Blue", "Green", "Black", "White"],
        "createdAt": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

### 3.2 Attribute Definition APIs

#### 3.2.1 Get All Attribute Definitions
- **Endpoint**: `GET /api/attributes`
- **Description**: Get all attribute definitions
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
- **Query Parameters**:
  - `dataType`: Filter by data type (string, number, boolean, array)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Attribute definitions retrieved successfully",
  "data": {
    "attributes": [
      {
        "id": 1,
        "key": "warranty_period",
        "name": "Warranty Period",
        "dataType": "number",
        "unit": "months",
        "description": "Product warranty duration",
        "allowedValues": null,
        "createdAt": "2024-01-15T10:30:00Z"
      },
      {
        "id": 2,
        "key": "color",
        "name": "Color",
        "dataType": "string",
        "unit": null,
        "description": "Product color",
        "allowedValues": ["Red", "Blue", "Green", "Black", "White"],
        "createdAt": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

#### 3.2.2 Create Attribute Definition
- **Endpoint**: `POST /api/attributes`
- **Description**: Create a new attribute definition
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
  - `Content-Type`: application/json
- **Request Body**:
```json
{
  "key": "screen_size",
  "name": "Screen Size",
  "dataType": "string",
  "unit": "inches",
  "description": "Display screen size",
  "allowedValues": ["13.3", "15.6", "17.3"]
}
```

**Validation Rules**:
- `key`: Required, 3-50 characters, lowercase with underscores, unique
- `name`: Required, 3-100 characters
- `dataType`: Required, enum (string, number, boolean, array)
- `unit`: Optional, max 20 characters
- `description`: Optional, max 500 characters
- `allowedValues`: Optional array, only for predefined options

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Attribute definition created successfully",
  "data": {
    "attribute": {
      "id": 3,
      "key": "screen_size",
      "name": "Screen Size",
      "dataType": "string",
      "unit": "inches",
      "description": "Display screen size",
      "allowedValues": ["13.3", "15.6", "17.3"],
      "isActive": true,
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.2.3 Create Category-Specific Attribute Definition
- **Endpoint**: `POST /api/attributes/{categoryId}`
- **Description**: Create a new attribute definition and associate it with a specific category
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `categoryId`: Category ID to associate the attribute with
- **Request Body**:
```json
{
  "key": "gpu_memory",
  "name": "GPU Memory",
  "dataType": "string",
  "unit": "GB",
  "description": "Graphics card memory size",
  "allowedValues": ["4GB", "6GB", "8GB", "12GB", "16GB"]
}
```

**Validation Rules**:
- Same as Create Attribute Definition
- `categoryId`: Must exist and be active
- `key`: Must be unique across all attributes

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Category attribute definition created successfully",
  "data": {
    "attribute": {
      "id": 4,
      "key": "gpu_memory",
      "name": "GPU Memory",
      "dataType": "string",
      "unit": "GB",
      "description": "Graphics card memory size",
      "allowedValues": ["4GB", "6GB", "8GB", "12GB", "16GB"],
      "isActive": true,
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

### 3.3 Category Attribute Configuration APIs

> **Note**: The category attribute configuration APIs have been moved to the Category Management section (3.1.5) and Attribute Definition section (3.2.3) for better organization. The inheritance-based approach provides a more flexible and maintainable solution.

#### 3.3.1 Get Category Attributes
- **Endpoint**: `GET /api/categories/{categoryId}/attributes` (Now returns inherited attributes)
- **Description**: Get all attributes for a category including inherited attributes from parent categories
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
- **Path Parameters**:
  - `categoryId`: Category ID

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Category attributes retrieved successfully",
  "data": {
    "attributes": [
      {
        "id": 1,
        "key": "warranty_period",
        "name": "Warranty Period",
        "dataType": "number",
        "unit": "months",
        "description": "Product warranty duration",
        "allowedValues": null,
        "createdAt": "2024-01-15T10:30:00Z"
      },
      {
        "id": 2,
        "key": "color",
        "name": "Color",
        "dataType": "string",
        "unit": null,
        "description": "Product color",
        "allowedValues": ["Red", "Blue", "Green", "Black", "White"],
        "createdAt": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

#### 3.3.2 Configure Category Attributes
- **Endpoint**: `POST /api/categories/{categoryId}/attributes` (Now creates category-specific attributes)
- **Description**: Create a new attribute definition and associate it with a specific category
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `categoryId`: Category ID to associate the attribute with
- **Request Body**:
```json
{
  "key": "gpu_memory",
  "name": "GPU Memory",
  "dataType": "string",
  "unit": "GB",
  "description": "Graphics card memory size",
  "allowedValues": ["4GB", "6GB", "8GB", "12GB", "16GB"]
}
```

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Category attribute definition created successfully",
  "data": {
    "attribute": {
      "id": 4,
      "key": "gpu_memory",
      "name": "GPU Memory",
      "dataType": "string",
      "unit": "GB",
      "description": "Graphics card memory size",
      "allowedValues": ["4GB", "6GB", "8GB", "12GB", "16GB"],
      "isActive": true,
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

### 3.4 Product Management APIs

#### 3.4.1 Get All Products
- **Endpoint**: `GET /api/products`
- **Description**: Get paginated list of products with filtering and search
- **Headers**: None required
- **Query Parameters**:
  - `page`: Page number (default: 1, integer)
  - `limit`: Items per page (default: 20, max: 100, integer)
  - `categoryId`: Filter by category ID (integer)
  - `search`: Search in product name and description (string)
  - `inStock`: Filter by stock status (boolean: true/false)
  - `isPopular`: Filter popular products (boolean: true/false)
  - `isActive`: Filter active products (boolean, default: true)
  - `sortBy`: Sort field (name, price, createdAt) (default: createdAt)
  - `sortOrder`: Sort order (asc, desc) (default: desc)
  - `minPrice`: Minimum price filter (number)
  - `maxPrice`: Maximum price filter (number)
  - `brand`: Filter by brand (string)
  - `attributes`: Filter by attributes (format: key:value,key2:value2)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Products retrieved successfully",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "iPhone 15 Pro",
        "categoryId": 2,
        "category": {
          "id": 2,
          "name": "Smartphones"
        },
        "brand": "Apple",
        "sku": "IP15P-256-BLK",
        "price": 999.99,
        "currency": "USD",
        "shortDescription": "Latest iPhone with advanced features",
        "images": [
          "https://example.com/iphone15pro1.jpg",
          "https://example.com/iphone15pro2.jpg"
        ],
        "inStock": true,
        "isPopular": true,
        "discount": 0,
        "tags": ["smartphone", "apple", "5g"],
        "attributes": {
          "warranty_period": "12",
          "color": "Black",
          "storage_capacity": "256GB"
        },
        "packageOptions": [
          {
            "id": 1,
            "name": "128GB Model",
            "price": 999.99,
            "quantity": 1,
            "isActive": true
          }
        ],
        "createdAt": "2024-01-15T10:30:00Z",
        "updatedAt": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 5,
      "totalItems": 98,
      "itemsPerPage": 20,
      "hasNext": true,
      "hasPrev": false
    }
  }
}
```

#### 3.4.2 Get Product by ID
- **Endpoint**: `GET /api/products/{productId}`
- **Description**: Get detailed information about a specific product
- **Headers**: None required
- **Path Parameters**:
  - `productId`: Product ID

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product retrieved successfully",
  "data": {
    "product": {
      "id": 1,
      "name": "iPhone 15 Pro",
      "categoryId": 2,
      "category": {
        "id": 2,
        "name": "Smartphones",
        "parent": {
          "id": 1,
          "name": "Electronics"
        }
      },
      "brand": "Apple",
      "sku": "IP15P-256-BLK",
      "price": 999.99,
      "currency": "USD",
      "shortDescription": "Latest iPhone with advanced features",
      "longDescription": "Experience the future with iPhone 15 Pro featuring...",
      "images": [
        "https://example.com/iphone15pro1.jpg",
        "https://example.com/iphone15pro2.jpg"
      ],
      "inStock": true,
      "isPopular": true,
      "isActive": true,
      "discount": 0,
      "tags": ["smartphone", "apple", "5g"],
      "attributes": [
        {
          "key": "warranty_period",
          "value": "12",
          "definition": {
            "name": "Warranty Period",
            "dataType": "number",
            "unit": "months"
          }
        },
        {
          "key": "color",
          "value": "Black",
          "definition": {
            "name": "Color",
            "dataType": "string"
          }
        }
      ],
      "packageOptions": [
        {
          "id": 1,
          "name": "128GB Model",
          "description": "iPhone 15 Pro with 128GB storage",
          "price": 999.99,
          "quantity": 1,
          "isActive": true
        },
        {
          "id": 2,
          "name": "256GB Model",
          "description": "iPhone 15 Pro with 256GB storage",
          "price": 1099.99,
          "quantity": 1,
          "isActive": true
        }
      ],
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.4.3 Create Product
- **Endpoint**: `POST /api/products`
- **Description**: Create a new product
- **Headers**: 
  - `Authorization`: Bearer token (Admin/Seller required)
  - `Content-Type`: application/json
- **Request Body**:
```json
{
  "name": "Samsung Galaxy S24",
  "categoryId": 2,
  "brand": "Samsung",
  "sku": "SGS24-128-BLK",
  "price": 899.99,
  "currency": "USD",
  "shortDescription": "Latest Samsung flagship smartphone",
  "longDescription": "Experience premium Android with Galaxy S24...",
  "images": [
    "https://example.com/galaxy-s24-1.jpg",
    "https://example.com/galaxy-s24-2.jpg"
  ],
  "isPopular": false,
  "discount": 5,
  "tags": ["smartphone", "samsung", "android"],
  "attributes": [
    {
      "key": "warranty_period",
      "value": "24"
    },
    {
      "key": "color",
      "value": "Black"
    },
    {
      "key": "storage_capacity",
      "value": "128GB"
    }
  ],
  "packageOptions": [
    {
      "name": "128GB Model",
      "description": "Galaxy S24 with 128GB storage",
      "price": 899.99,
      "quantity": 1
    },
    {
      "name": "256GB Model",
      "description": "Galaxy S24 with 256GB storage",
      "price": 999.99,
      "quantity": 1
    }
  ]
}
```

**Validation Rules**:
- `name`: Required, 3-200 characters
- `categoryId`: Required, must exist and be active
- `brand`: Optional, max 100 characters
- `sku`: Required, unique, 3-50 characters
- `price`: Required, positive number
- `currency`: Optional, 3-character ISO code (default: USD)
- `shortDescription`: Optional, max 500 characters
- `longDescription`: Optional, max 5000 characters
- `images`: Optional array, max 10 URLs
- `discount`: Optional, 0-100 integer
- `tags`: Optional array, max 20 tags
- `attributes`: Required based on category configuration
- `packageOptions`: Optional array, max 50 options

**Business Rules**:
- Validate all required attributes for the category
- Validate attribute values against allowed values (if defined)
- SKU must be unique across all products
- Price must be positive

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Product created successfully",
  "data": {
    "product": {
      "id": 2,
      "name": "Samsung Galaxy S24",
      "categoryId": 2,
      "brand": "Samsung",
      "sku": "SGS24-128-BLK",
      "price": 899.99,
      "currency": "USD",
      "shortDescription": "Latest Samsung flagship smartphone",
      "inStock": true,
      "isPopular": false,
      "isActive": true,
      "discount": 5,
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.4.4 Update Product
- **Endpoint**: `PUT /api/products/{productId}`
- **Description**: Update an existing product
- **Headers**: 
  - `Authorization`: Bearer token (Admin/Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID to update
- **Request Body**: Same as create product (all fields optional)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product updated successfully",
  "data": {
    "product": {
      "id": 2,
      "name": "Samsung Galaxy S24 Ultra",
      "price": 999.99,
      "updatedAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

#### 3.4.5 Delete Product
- **Endpoint**: `DELETE /api/products/{productId}`
- **Description**: Soft delete a product (set isActive to false)
- **Headers**: 
  - `Authorization`: Bearer token (Admin required)
- **Path Parameters**:
  - `productId`: Product ID to delete

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product deleted successfully"
}
```

#### 3.4.6 Update Product Stock Status
- **Endpoint**: `PATCH /api/products/{productId}/stock`
- **Description**: Update product stock status
- **Headers**: 
  - `Authorization`: Bearer token (Admin/Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID
- **Request Body**:
```json
{
  "inStock": false
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product stock status updated successfully"
}
```

### 3.5 Search and Filter APIs

#### 3.5.1 Search Products
- **Endpoint**: `GET /api/products/search`
- **Description**: Advanced product search with full-text search and filters
- **Headers**: None required
- **Query Parameters**:
  - `q`: Search query (string, required)
  - `categoryId`: Filter by category (integer)
  - `page`: Page number (default: 1)
  - `limit`: Items per page (default: 20, max: 100)
  - `sortBy`: Sort field (relevance, name, price, createdAt)
  - `sortOrder`: Sort order (asc, desc)
  - `filters`: Additional filters (format: attribute1:value1,attribute2:value2)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Products found successfully",
  "data": {
    "query": "smartphone apple",
    "results": [
      {
        "id": 1,
        "name": "iPhone 15 Pro",
        "price": 999.99,
        "shortDescription": "Latest iPhone with advanced features",
        "images": ["https://example.com/iphone15pro1.jpg"],
        "relevanceScore": 0.95,
        "matchedFields": ["name", "brand", "tags"]
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 1,
      "totalItems": 1
    },
    "searchTime": "0.05s"
  }
}
```

#### 3.5.2 Get Product Filters
- **Endpoint**: `GET /api/products/filters`
- **Description**: Get available filters for product search
- **Headers**: None required
- **Query Parameters**:
  - `categoryId`: Get filters for specific category (integer)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Filters retrieved successfully",
  "data": {
    "filters": {
      "categories": [
        {
          "id": 1,
          "name": "Electronics",
          "productCount": 150
        }
      ],
      "brands": [
        {
          "name": "Apple",
          "productCount": 25
        },
        {
          "name": "Samsung",
          "productCount": 30
        }
      ],
      "priceRanges": [
        {
          "range": "0-500",
          "productCount": 45
        },
        {
          "range": "500-1000",
          "productCount": 78
        }
      ],
      "attributes": [
        {
          "key": "color",
          "name": "Color",
          "values": [
            {
              "value": "Black",
              "productCount": 35
            },
            {
              "value": "White",
              "productCount": 28
            }
          ]
        }
      ]
    }
  }
}
```

#### 3.5.3 Get Related Products
- **Endpoint**: `GET /api/products/{productId}/related`
- **Description**: Get products related to a specific product
- **Headers**: None required
- **Path Parameters**:
  - `productId`: Product ID
- **Query Parameters**:
  - `limit`: Number of related products (default: 5, max: 20)

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Related products retrieved successfully",
  "data": {
    "relatedProducts": [
      {
        "id": 2,
        "name": "Samsung Galaxy S24",
        "price": 899.99,
        "shortDescription": "Latest Samsung flagship",
        "images": ["https://example.com/galaxy-s24-1.jpg"],
        "relationReason": "Same category"
      }
    ]
  }
}
```

### 3.6 Package Option APIs

#### 3.6.1 Get Product Package Options
- **Endpoint**: `GET /api/products/{productId}/packages`
- **Description**: Get all package options for a product
- **Headers**: None required
- **Path Parameters**:
  - `productId`: Product ID

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Package options retrieved successfully",
  "data": {
    "packageOptions": [
      {
        "id": 1,
        "name": "128GB Model",
        "description": "iPhone 15 Pro with 128GB storage",
        "price": 999.99,
        "quantity": 1,
        "isActive": true
      }
    ]
  }
}
```

#### 3.6.2 Add Package Option
- **Endpoint**: `POST /api/products/{productId}/packages`
- **Description**: Add a new package option to a product
- **Headers**: 
  - `Authorization`: Bearer token (Admin/Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID
- **Request Body**:
```json
{
  "name": "512GB Model",
  "description": "iPhone 15 Pro with 512GB storage",
  "price": 1299.99,
  "quantity": 1
}
```

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Package option added successfully",
  "data": {
    "packageOption": {
      "id": 3,
      "name": "512GB Model",
      "price": 1299.99,
      "quantity": 1,
      "isActive": true
    }
  }
}
```

## 4. Error Handling

### 4.1 Common Error Responses

#### 400 Bad Request
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "field": "name",
      "message": "Product name is required"
    },
    {
      "field": "price",
      "message": "Price must be a positive number"
    }
  ]
}
```

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "Authentication required",
  "errorCode": "AUTH_REQUIRED"
}
```

#### 403 Forbidden
```json
{
  "success": false,
  "message": "Insufficient permissions",
  "errorCode": "INSUFFICIENT_PERMISSIONS"
}
```

#### 404 Not Found
```json
{
  "success": false,
  "message": "Product not found",
  "errorCode": "PRODUCT_NOT_FOUND"
}
```

#### 409 Conflict
```json
{
  "success": false,
  "message": "SKU already exists",
  "errorCode": "SKU_CONFLICT"
}
```

#### 422 Unprocessable Entity
```json
{
  "success": false,
  "message": "Category attribute validation failed",
  "errors": [
    {
      "attribute": "warranty_period",
      "message": "This attribute is required for Electronics category"
    }
  ]
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "Internal server error",
  "errorCode": "INTERNAL_ERROR"
}
```

## 5. Security & Authorization

### 5.1 Authentication
- JWT-based authentication for protected endpoints
- Public access for product listing and details
- Admin/Seller roles for product management

### 5.2 Authorization Levels
- **Public**: Product listing, details, search, filters
- **Authenticated**: Access to user-specific features
- **Seller**: Create/update own products
- **Admin**: Full access to all product management features

### 5.3 Rate Limiting
- **Public APIs**: 1000 requests per hour per IP
- **Authenticated APIs**: 5000 requests per hour per user
- **Admin APIs**: 10000 requests per hour per user

## 6. Performance Considerations

### 6.1 Caching Strategy
- **Product Lists**: Cache for 5 minutes
- **Product Details**: Cache for 15 minutes
- **Category Lists**: Cache for 1 hour
- **Filters**: Cache for 30 minutes

### 6.2 Pagination
- Default: 20 items per page
- Maximum: 100 items per page
- Use cursor-based pagination for large datasets

### 6.3 Database Optimization
- Proper indexing on search fields
- Composite indexes for filter combinations
- Query optimization for attribute lookups

### 6.4 Category Hierarchy Performance
- **Get All Categories API**: Uses optimized queries to avoid N+1 problems
- **Inheritance-based Attributes**: Reduces data duplication and improves maintainability
- **Single Query Approach**: Fetches all categories in one query and builds hierarchy in memory
- **Avoids Duplicate Data**: Prevents fetching the same category multiple times in nested relationships

**Performance Benefits**:
- Reduced database queries for category hierarchies
- Eliminated duplicate category data in responses
- Improved response times for large category trees
- Better memory efficiency for nested category structures

### 6.5 Simplified Data Model
- **Removed `isActive` Fields**: All entities now use hard delete instead of soft delete
- **Cleaner API Responses**: No more `isActive` fields in JSON responses
- **Simplified Queries**: No need to filter by `isActive` status in database queries
- **Reduced Complexity**: Less conditional logic in business rules and validation

## 7. Implementation Notes

### 7.1 Code Validation Requirements

1. **Input Validation**:
   - Implement struct validation tags in Go
   - Custom validators for business rules
   - Sanitize all user inputs

2. **Database Constraints**:
   - Foreign key constraints
   - Unique constraints on SKU
   - Check constraints for positive prices

3. **Business Logic Validation**:
   - Category-specific attribute validation
   - Attribute value validation against allowed values
   - Stock level consistency checks

4. **Error Handling**:
   - Consistent error response format
   - Proper HTTP status codes
   - Detailed validation error messages

5. **Logging & Monitoring**:
   - Log all CRUD operations
   - Performance monitoring
   - Error tracking and alerting

### 7.2 Testing Requirements

1. **Unit Tests**: 90% code coverage
2. **Integration Tests**: API endpoint testing
3. **Performance Tests**: Load testing for search APIs
4. **Security Tests**: Authentication and authorization testing

This PRD provides a comprehensive foundation for implementing a scalable, industry-agnostic product service that can handle any type of product across various business domains.
