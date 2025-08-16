# Backend API PRD - Datun E-commerce Platform

## 1. Project Overview

### 1.1 Purpose
This document outlines the backend API requirements for the Datun e-commerce platform, designed to support a React-based frontend application selling natural dental care products and related items.

### 1.2 Technology Stack
- **Language**: Go (Golang)
- **Framework**: Gin/Echo (recommended)
- **Database**: PostgreSQL (Primary recommendation)
- **Cache**: Redis for session management and caching
- **Authentication**: JWT (JSON Web Tokens)
- **Architecture**: RESTful API
- **ORM**: GORM or SQLx for database operations
- **Additional**: Docker for containerization, nginx for reverse proxy

#### 1.2.1 Database Selection Rationale

**PostgreSQL is recommended for this e-commerce platform because:**

✅ **ACID Compliance**: Essential for financial transactions and inventory management  
✅ **Complex Relationships**: Native support for foreign keys and complex joins  
✅ **JSON Support**: JSONB for flexible product attributes and metadata  
✅ **Full-text Search**: Built-in search capabilities for product discovery  
✅ **Performance**: Excellent query optimization and indexing  
✅ **Go Integration**: Outstanding support with libraries like pgx and GORM  
✅ **Scalability**: Proven performance in large-scale applications  
✅ **Data Integrity**: Strong consistency guarantees for critical operations  

**Alternative Considerations:**
- **MySQL**: Good alternative, slightly simpler but less advanced JSON support
- **MongoDB**: Consider only if you need extreme flexibility, but lacks ACID guarantees needed for e-commerce

## 2. Core Entities & Data Models

### 2.1 User Entity
```go
type User struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    FirstName   string    `json:"firstName" binding:"required"`
    LastName    string    `json:"lastName" binding:"required"`
    Email       string    `json:"email" binding:"required,email" gorm:"unique"`
    Password    string    `json:"-" binding:"required,min=6"`
    Phone       string    `json:"phone"`
    DateOfBirth string    `json:"dateOfBirth"`
    Gender      string    `json:"gender"`
    IsActive    bool      `json:"isActive" gorm:"default:true"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
    
    // Relationships
    Addresses []Address `json:"addresses" gorm:"foreignKey:UserID"`
    Orders    []Order   `json:"orders" gorm:"foreignKey:UserID"`
}

type Address struct {
    ID       uint   `json:"id" gorm:"primaryKey"`
    UserID   uint   `json:"userId"`
    Street   string `json:"street" binding:"required"`
    City     string `json:"city" binding:"required"`
    State    string `json:"state" binding:"required"`
    ZipCode  string `json:"zipCode" binding:"required"`
    Country  string `json:"country" binding:"required"`
    IsDefault bool  `json:"isDefault" gorm:"default:false"`
}
```

### 2.2 Product Entity
```go
type Product struct {
    ID               uint                `json:"id" gorm:"primaryKey"`
    Name             string              `json:"name" binding:"required"`
    Category         string              `json:"category" binding:"required"`
    Price            float64             `json:"price" binding:"required"`
    ShortDescription string              `json:"shortDescription"`
    LongDescription  string              `json:"longDescription"`
    Images           []string            `json:"images" gorm:"type:text[]"`
    InStock          bool                `json:"inStock" gorm:"default:true"`
    IsPopular        bool                `json:"isPopular" gorm:"default:false"`
    Discount         int                 `json:"discount" gorm:"default:0"`
    PackageOptions   []PackageOption     `json:"packageOptions" gorm:"foreignKey:ProductID"`
    NutritionalInfo  *NutritionalInfo    `json:"nutritionalInfo" gorm:"embedded"`
    Ingredients      []string            `json:"ingredients" gorm:"type:text[]"`
    Benefits         []string            `json:"benefits" gorm:"type:text[]"`
    Instructions     string              `json:"instructions"`
    CreatedAt        time.Time           `json:"createdAt"`
    UpdatedAt        time.Time           `json:"updatedAt"`
}

type PackageOption struct {
    ID        uint    `json:"id" gorm:"primaryKey"`
    ProductID uint    `json:"productId"`
    Size      string  `json:"size" binding:"required"`
    Price     float64 `json:"price" binding:"required"`
}

type NutritionalInfo struct {
    Calories      string `json:"calories"`
    Fat           string `json:"fat"`
    SaturatedFat  string `json:"saturatedFat"`
    Cholesterol   string `json:"cholesterol"`
    Sodium        string `json:"sodium"`
}
```

### 2.3 Cart Entity
```go
type Cart struct {
    ID        uint       `json:"id" gorm:"primaryKey"`
    UserID    uint       `json:"userId" binding:"required"`
    Items     []CartItem `json:"items" gorm:"foreignKey:CartID"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
}

type CartItem struct {
    ID          uint    `json:"id" gorm:"primaryKey"`
    CartID      uint    `json:"cartId"`
    ProductID   uint    `json:"productId" binding:"required"`
    PackageSize string  `json:"packageSize"`
    Quantity    int     `json:"quantity" binding:"required,min=1"`
    Price       float64 `json:"price" binding:"required"`
    
    // Relationships
    Product Product `json:"product" gorm:"foreignKey:ProductID"`
}
```

### 2.4 Order Entity
```go
type Order struct {
    ID          uint        `json:"id" gorm:"primaryKey"`
    UserID      uint        `json:"userId" binding:"required"`
    OrderNumber string      `json:"orderNumber" gorm:"unique"`
    Status      string      `json:"status" gorm:"default:pending"`
    Subtotal    float64     `json:"subtotal"`
    Tax         float64     `json:"tax"`
    Shipping    float64     `json:"shipping"`
    Total       float64     `json:"total"`
    Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
    
    // Shipping Address
    ShippingAddress Address `json:"shippingAddress" gorm:"embedded;embeddedPrefix:shipping_"`
    
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type OrderItem struct {
    ID          uint    `json:"id" gorm:"primaryKey"`
    OrderID     uint    `json:"orderId"`
    ProductID   uint    `json:"productId"`
    PackageSize string  `json:"packageSize"`
    Quantity    int     `json:"quantity"`
    Price       float64 `json:"price"`
    
    // Relationships
    Product Product `json:"product" gorm:"foreignKey:ProductID"`
}
```

## 3. API Endpoints Specification

### 3.1 Authentication APIs

#### 3.1.1 User Registration
- **Endpoint**: `POST /api/auth/register`
- **Description**: Register a new user account
- **Headers**: `Content-Type: application/json`
- **Request Body**:
```json
{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john.doe@example.com",
  "password": "securePassword123",
  "phone": "+1234567890",
  "dateOfBirth": "1990-01-01",
  "gender": "male"
}
```
- **Response (201 Created)**:
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "user": {
      "id": 1,
      "firstName": "John",
      "lastName": "Doe",
      "email": "john.doe@example.com",
      "phone": "+1234567890",
      "dateOfBirth": "1990-01-01",
      "gender": "male",
      "isActive": true,
      "createdAt": "2025-01-15T10:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "field": "email",
      "message": "Email already exists"
    },
    {
      "field": "password",
      "message": "Password must be at least 6 characters"
    }
  ],
  "code": "VALIDATION_ERROR"
}
```
- **Error Response (409 Conflict)**:
```json
{
  "success": false,
  "message": "User with this email already exists",
  "code": "USER_EXISTS"
}
```

#### 3.1.2 User Login
- **Endpoint**: `POST /api/auth/login`
- **Description**: Authenticate user and return JWT token
- **Headers**: `Content-Type: application/json`
- **Request Body**:
```json
{
  "email": "john.doe@example.com",
  "password": "securePassword123"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "user": {
      "id": 1,
      "firstName": "John",
      "lastName": "Doe",
      "email": "john.doe@example.com",
      "phone": "+1234567890",
      "dateOfBirth": "1990-01-01",
      "gender": "male",
      "isActive": true,
      "createdAt": "2025-01-15T10:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "24h"
  }
}
```
- **Error Response (401 Unauthorized)**:
```json
{
  "success": false,
  "message": "Invalid email or password",
  "code": "INVALID_CREDENTIALS"
}
```
- **Error Response (403 Forbidden)**:
```json
{
  "success": false,
  "message": "Account is deactivated",
  "code": "ACCOUNT_DEACTIVATED"
}
```

#### 3.1.3 Token Refresh
- **Endpoint**: `POST /api/auth/refresh`
- **Description**: Refresh JWT token
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**: `{}` (empty object)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "24h"
  }
}
```
- **Error Response (401 Unauthorized)**:
```json
{
  "success": false,
  "message": "Invalid or expired token",
  "code": "TOKEN_INVALID"
}
```

#### 3.1.4 User Logout
- **Endpoint**: `POST /api/auth/logout`
- **Description**: Logout user and blacklist JWT token
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**: `{}` (empty object)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```
- **Error Response (401 Unauthorized)**:
```json
{
  "success": false,
  "message": "Invalid token",
  "code": "TOKEN_INVALID"
}
```

### 3.2 User Management APIs

#### 3.2.1 Get User Profile
- **Endpoint**: `GET /api/users/profile`
- **Description**: Get current user's profile information
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "user": {
      "id": 1,
      "firstName": "John",
      "lastName": "Doe",
      "email": "john.doe@example.com",
      "phone": "+1234567890",
      "dateOfBirth": "1990-01-01",
      "gender": "male",
      "isActive": true,
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T10:30:00Z",
      "addresses": [
        {
          "id": 1,
          "street": "123 Main St",
          "city": "New York",
          "state": "NY",
          "zipCode": "10001",
          "country": "USA",
          "isDefault": true
        }
      ]
    }
  }
}
```
- **Error Response (401 Unauthorized)**:
```json
{
  "success": false,
  "message": "Authentication required",
  "code": "AUTH_REQUIRED"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "User not found",
  "code": "USER_NOT_FOUND"
}
```

#### 3.2.2 Update User Profile
- **Endpoint**: `PUT /api/users/profile`
- **Description**: Update user profile information
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "firstName": "John",
  "lastName": "Doe",
  "phone": "+1234567890",
  "dateOfBirth": "1990-01-01",
  "gender": "male"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Profile updated successfully",
  "data": {
    "user": {
      "id": 1,
      "firstName": "John",
      "lastName": "Doe",
      "email": "john.doe@example.com",
      "phone": "+1234567890",
      "dateOfBirth": "1990-01-01",
      "gender": "male",
      "isActive": true,
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T11:45:00Z"
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "field": "firstName",
      "message": "First name is required"
    },
    {
      "field": "phone",
      "message": "Invalid phone number format"
    }
  ],
  "code": "VALIDATION_ERROR"
}
```

#### 3.2.3 Get User Addresses
- **Endpoint**: `GET /api/users/addresses`
- **Description**: Get all addresses for the current user
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Addresses retrieved successfully",
  "data": {
    "addresses": [
      {
        "id": 1,
        "street": "123 Main St",
        "city": "New York",
        "state": "NY",
        "zipCode": "10001",
        "country": "USA",
        "isDefault": true
      },
      {
        "id": 2,
        "street": "456 Oak Ave",
        "city": "Los Angeles",
        "state": "CA",
        "zipCode": "90210",
        "country": "USA",
        "isDefault": false
      }
    ]
  }
}
```

#### 3.2.4 Add Address
- **Endpoint**: `POST /api/users/addresses`
- **Description**: Add a new address for the user
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "street": "456 Oak Ave",
  "city": "Los Angeles",
  "state": "CA",
  "zipCode": "90210",
  "country": "USA",
  "isDefault": false
}
```
- **Response (201 Created)**:
```json
{
  "success": true,
  "message": "Address added successfully",
  "data": {
    "address": {
      "id": 2,
      "street": "456 Oak Ave",
      "city": "Los Angeles",
      "state": "CA",
      "zipCode": "90210",
      "country": "USA",
      "isDefault": false
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "field": "street",
      "message": "Street address is required"
    },
    {
      "field": "zipCode",
      "message": "Invalid zip code format"
    }
  ],
  "code": "VALIDATION_ERROR"
}
```

#### 3.2.5 Update Address
- **Endpoint**: `PUT /api/users/addresses/:id`
- **Description**: Update an existing address
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Path Parameters**: `id` (integer) - Address ID
- **Request Body**:
```json
{
  "street": "456 Oak Ave Apt 2B",
  "city": "Los Angeles",
  "state": "CA",
  "zipCode": "90210",
  "country": "USA",
  "isDefault": true
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Address updated successfully",
  "data": {
    "address": {
      "id": 2,
      "street": "456 Oak Ave Apt 2B",
      "city": "Los Angeles",
      "state": "CA",
      "zipCode": "90210",
      "country": "USA",
      "isDefault": true
    }
  }
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Address not found",
  "code": "ADDRESS_NOT_FOUND"
}
```
- **Error Response (403 Forbidden)**:
```json
{
  "success": false,
  "message": "You don't have permission to update this address",
  "code": "PERMISSION_DENIED"
}
```

#### 3.2.6 Delete Address
- **Endpoint**: `DELETE /api/users/addresses/:id`
- **Description**: Delete an address
- **Headers**: `Authorization: Bearer <token>`
- **Path Parameters**: `id` (integer) - Address ID
- **Request Body**: None (DELETE request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Address deleted successfully"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Address not found",
  "code": "ADDRESS_NOT_FOUND"
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Cannot delete default address. Please set another address as default first.",
  "code": "CANNOT_DELETE_DEFAULT"
}
```

#### 3.2.7 Set Default Address
- **Endpoint**: `PATCH /api/users/addresses/:id/default`
- **Description**: Set an address as the default shipping address
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Path Parameters**: `id` (integer) - Address ID
- **Request Body**: `{}` (empty object)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Default address updated successfully",
  "data": {
    "address": {
      "id": 2,
      "street": "456 Oak Ave",
      "city": "Los Angeles",
      "state": "CA",
      "zipCode": "90210",
      "country": "USA",
      "isDefault": true
    }
  }
}
```

#### 3.2.8 Change Password
- **Endpoint**: `PATCH /api/users/password`
- **Description**: Change user password
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "currentPassword": "oldPassword123",
  "newPassword": "newSecurePassword456",
  "confirmPassword": "newSecurePassword456"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Password changed successfully"
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Current password is incorrect",
  "code": "INVALID_CURRENT_PASSWORD"
}
```

### 3.3 Product APIs

#### 3.3.1 Get All Products
- **Endpoint**: `GET /api/products`
- **Description**: Get paginated list of products with filtering and search
- **Headers**: None required
- **Query Parameters**:
  - `page`: Page number (default: 1, integer)
  - `limit`: Items per page (default: 20, max: 100, integer)
  - `category`: Filter by category (string)
  - `search`: Search in product name and description (string)
  - `inStock`: Filter by stock status (boolean: true/false)
  - `isPopular`: Filter popular products (boolean: true/false)
  - `sortBy`: Sort field (name, price, createdAt) (default: createdAt)
  - `sortOrder`: Sort order (asc, desc) (default: desc)
  - `minPrice`: Minimum price filter (number)
  - `maxPrice`: Maximum price filter (number)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Products retrieved successfully",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "Premium Ghee",
        "category": "Dairy",
        "price": 850,
        "shortDescription": "Pure and aromatic cow ghee made from fresh cream",
        "images": [
          "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
        ],
        "inStock": true,
        "isPopular": true,
        "discount": 10,
        "packageOptions": [
          {
            "id": 1,
            "size": "6-pack",
            "price": 850
          },
          {
            "id": 2,
            "size": "12-pack",
            "price": 1600
          }
        ],
        "createdAt": "2025-01-15T10:30:00Z",
        "updatedAt": "2025-01-15T10:30:00Z"
      },
      {
        "id": 2,
        "name": "Organic Honey",
        "category": "Natural Care",
        "price": 450,
        "shortDescription": "Raw, unprocessed organic honey with natural enzymes",
        "images": [
          "https://images.unsplash.com/photo-1587049016823-c90bb2e77f43?w=400"
        ],
        "inStock": true,
        "isPopular": false,
        "discount": 0,
        "packageOptions": [
          {
            "id": 3,
            "size": "250g",
            "price": 450
          },
          {
            "id": 4,
            "size": "500g",
            "price": 850
          }
        ],
        "createdAt": "2025-01-15T09:20:00Z",
        "updatedAt": "2025-01-15T09:20:00Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 5,
      "totalItems": 100,
      "itemsPerPage": 20,
      "hasNextPage": true,
      "hasPrevPage": false
    },
    "filters": {
      "appliedCategory": "Dairy",
      "appliedSearch": null,
      "priceRange": {
        "min": 450,
        "max": 850
      }
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Invalid query parameters",
  "errors": [
    {
      "field": "page",
      "message": "Page must be a positive integer"
    },
    {
      "field": "limit",
      "message": "Limit cannot exceed 100"
    }
  ],
  "code": "INVALID_QUERY_PARAMS"
}
```

#### 3.3.2 Get Product by ID
- **Endpoint**: `GET /api/products/:id`
- **Description**: Get detailed product information by ID
- **Headers**: None required
- **Path Parameters**: `id` (integer) - Product ID
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product retrieved successfully",
  "data": {
    "product": {
      "id": 1,
      "name": "Premium Ghee",
      "category": "Dairy",
      "price": 850,
      "shortDescription": "Pure and aromatic cow ghee made from fresh cream",
      "longDescription": "Our Premium Ghee is made from the finest quality cow milk cream, churned using traditional methods to preserve its natural aroma and nutritional value. Rich in vitamins A, D, E, and K, this ghee is perfect for cooking, baking, and Ayurvedic preparations.",
      "images": [
        "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400",
        "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=600",
        "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=800"
      ],
      "inStock": true,
      "isPopular": true,
      "discount": 10,
      "packageOptions": [
        {
          "id": 1,
          "size": "6-pack",
          "price": 850
        },
        {
          "id": 2,
          "size": "12-pack",
          "price": 1600
        },
        {
          "id": 3,
          "size": "24-pack",
          "price": 3000
        }
      ],
      "nutritionalInfo": {
        "calories": "900 per 100g",
        "fat": "100g",
        "saturatedFat": "60g",
        "cholesterol": "300mg",
        "sodium": "0mg"
      },
      "ingredients": [
        "Pure Cow Milk Cream"
      ],
      "benefits": [
        "Rich in vitamins A, D, E, and K",
        "Supports healthy digestion",
        "Boosts immunity",
        "Good for bone health",
        "Contains healthy fats"
      ],
      "instructions": "Use for cooking, baking, or as a spread. Store in a cool, dry place. No refrigeration required.",
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T10:30:00Z"
    }
  }
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Product not found",
  "code": "PRODUCT_NOT_FOUND"
}
```

#### 3.3.3 Get Product Categories
- **Endpoint**: `GET /api/products/categories`
- **Description**: Get list of all available product categories
- **Headers**: None required
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Categories retrieved successfully",
  "data": {
    "categories": [
      {
        "name": "Dairy",
        "count": 15,
        "description": "Fresh dairy products including milk, ghee, and cheese"
      },
      {
        "name": "Organic",
        "count": 23,
        "description": "Certified organic products grown without pesticides"
      },
      {
        "name": "Natural Care",
        "count": 18,
        "description": "Natural health and beauty care products"
      },
      {
        "name": "Supplements",
        "count": 12,
        "description": "Health supplements and vitamins"
      }
    ],
    "totalCategories": 4
  }
}
```

#### 3.3.4 Get Product Filters
- **Endpoint**: `GET /api/products/filters`
- **Description**: Get all available filter attributes and their possible values for product filtering UI
- **Headers**: None required
- **Query Parameters**:
  - `category`: Get filters specific to a category (optional, string)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Product filters retrieved successfully",
  "data": {
    "filters": {
      "categories": {
        "type": "select",
        "label": "Category",
        "options": [
          {
            "value": "Dairy",
            "label": "Dairy Products",
            "count": 15
          },
          {
            "value": "Organic",
            "label": "Organic Products",
            "count": 23
          },
          {
            "value": "Natural Care",
            "label": "Natural Care",
            "count": 18
          },
          {
            "value": "Supplements",
            "label": "Health Supplements",
            "count": 12
          }
        ]
      },
      "priceRange": {
        "type": "range",
        "label": "Price Range",
        "min": 50,
        "max": 5000,
        "step": 50,
        "currency": "INR",
        "defaultRange": [100, 2000]
      },
      "availability": {
        "type": "checkbox",
        "label": "Availability",
        "options": [
          {
            "value": "in_stock",
            "label": "In Stock",
            "count": 65
          },
          {
            "value": "out_of_stock",
            "label": "Out of Stock",
            "count": 3
          }
        ]
      },
      "popularity": {
        "type": "checkbox",
        "label": "Product Type",
        "options": [
          {
            "value": "popular",
            "label": "Popular Products",
            "count": 25
          },
          {
            "value": "new_arrivals",
            "label": "New Arrivals",
            "count": 12
          }
        ]
      },
      "discount": {
        "type": "select",
        "label": "Discount",
        "options": [
          {
            "value": "any",
            "label": "Any Discount",
            "count": 35
          },
          {
            "value": "5_plus",
            "label": "5% and above",
            "count": 30
          },
          {
            "value": "10_plus",
            "label": "10% and above",
            "count": 20
          },
          {
            "value": "20_plus",
            "label": "20% and above",
            "count": 8
          }
        ]
      },
      "packageSize": {
        "type": "select",
        "label": "Package Size",
        "options": [
          {
            "value": "Individual",
            "label": "Individual Pack",
            "count": 25
          },
          {
            "value": "6-pack",
            "label": "6 Pack Bundle",
            "count": 35
          },
          {
            "value": "12-pack",
            "label": "12 Pack Bundle",
            "count": 30
          },
          {
            "value": "24-pack",
            "label": "24 Pack Bundle",
            "count": 15
          }
        ]
      },
      "sortBy": {
        "type": "select",
        "label": "Sort By",
        "options": [
          {
            "value": "relevance",
            "label": "Relevance"
          },
          {
            "value": "name_asc",
            "label": "Name (A to Z)"
          },
          {
            "value": "name_desc",
            "label": "Name (Z to A)"
          },
          {
            "value": "price_asc",
            "label": "Price (Low to High)"
          },
          {
            "value": "price_desc",
            "label": "Price (High to Low)"
          },
          {
            "value": "newest",
            "label": "Newest First"
          },
          {
            "value": "popular",
            "label": "Most Popular"
          }
        ]
      }
    },
    "metadata": {
      "totalProducts": 68,
      "appliedFilters": {},
      "generatedAt": "2025-01-15T10:30:00Z"
    }
  }
}
```
- **Response with Category Filter (200 OK)**:
```json
{
  "success": true,
  "message": "Category-specific filters retrieved successfully",
  "data": {
    "filters": {
      "priceRange": {
        "type": "range",
        "label": "Price Range",
        "min": 200,
        "max": 3000,
        "step": 100,
        "currency": "INR",
        "defaultRange": [300, 1500]
      },
      "packageSize": {
        "type": "select",
        "label": "Package Size",
        "options": [
          {
            "value": "250g",
            "label": "250g",
            "count": 8
          },
          {
            "value": "500g",
            "label": "500g",
            "count": 12
          },
          {
            "value": "1kg",
            "label": "1kg",
            "count": 6
          }
        ]
      }
    },
    "metadata": {
      "category": "Dairy",
      "totalProducts": 15,
      "generatedAt": "2025-01-15T10:30:00Z"
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Invalid category provided",
  "code": "INVALID_CATEGORY"
}
```

#### 3.3.5 Search Products
- **Endpoint**: `GET /api/products/search`
- **Description**: Advanced product search with multiple filters
- **Headers**: None required
- **Query Parameters**:
  - `q`: Search query (required, string)
  - `category`: Filter by category (string)
  - `page`: Page number (default: 1, integer)
  - `limit`: Items per page (default: 20, integer)
  - `sortBy`: Sort field (relevance, name, price, createdAt) (default: relevance)
  - `sortOrder`: Sort order (asc, desc) (default: desc)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Search completed successfully",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "Premium Ghee",
        "category": "Dairy",
        "price": 850,
        "shortDescription": "Pure and aromatic cow ghee made from fresh cream",
        "images": [
          "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
        ],
        "inStock": true,
        "isPopular": true,
        "discount": 10,
        "relevanceScore": 0.95
      }
    ],
    "searchInfo": {
      "query": "ghee",
      "totalResults": 3,
      "searchTime": "0.045s"
    },
    "pagination": {
      "currentPage": 1,
      "totalPages": 1,
      "totalItems": 3,
      "itemsPerPage": 20
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Search query is required",
  "code": "MISSING_SEARCH_QUERY"
}
```

#### 3.3.6 Get Related Products
- **Endpoint**: `GET /api/products/:id/related`
- **Description**: Get products related to a specific product
- **Headers**: None required
- **Path Parameters**: `id` (integer) - Product ID
- **Query Parameters**:
  - `limit`: Number of related products (default: 4, max: 10, integer)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Related products retrieved successfully",
  "data": {
    "relatedProducts": [
      {
        "id": 2,
        "name": "Organic Butter",
        "category": "Dairy",
        "price": 650,
        "shortDescription": "Fresh organic butter made from cream",
        "images": [
          "https://images.unsplash.com/photo-1589985269047-0dbf5e294199?w=400"
        ],
        "inStock": true,
        "isPopular": false,
        "discount": 5
      },
      {
        "id": 3,
        "name": "Pure Cow Milk",
        "category": "Dairy",
        "price": 80,
        "shortDescription": "Fresh cow milk delivered daily",
        "images": [
          "https://images.unsplash.com/photo-1563636619-e9143da7973b?w=400"
        ],
        "inStock": true,
        "isPopular": true,
        "discount": 0
      }
    ],
    "baseProduct": {
      "id": 1,
      "name": "Premium Ghee",
      "category": "Dairy"
    }
  }
}
```

### 3.4 Cart APIs

#### 3.4.1 Get User Cart
- **Endpoint**: `GET /api/cart`
- **Description**: Get current user's shopping cart with all items
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Cart retrieved successfully",
  "data": {
    "cart": {
      "id": 1,
      "userId": 1,
      "items": [
        {
          "id": 1,
          "productId": 1,
          "packageSize": "6-pack",
          "quantity": 2,
          "price": 850,
          "total": 1700,
          "product": {
            "id": 1,
            "name": "Premium Ghee",
            "category": "Dairy",
            "images": [
              "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
            ],
            "inStock": true,
            "discount": 10
          }
        },
        {
          "id": 2,
          "productId": 2,
          "packageSize": "500g",
          "quantity": 1,
          "price": 850,
          "total": 850,
          "product": {
            "id": 2,
            "name": "Organic Honey",
            "category": "Natural Care",
            "images": [
              "https://images.unsplash.com/photo-1587049016823-c90bb2e77f43?w=400"
            ],
            "inStock": true,
            "discount": 0
          }
        }
      ],
      "summary": {
        "subtotal": 2550,
        "discount": 170,
        "tax": 238,
        "shipping": 50,
        "total": 2668,
        "totalItems": 3,
        "totalProducts": 2
      },
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T11:45:00Z"
    }
  }
}
```
- **Error Response (401 Unauthorized)**:
```json
{
  "success": false,
  "message": "Authentication required",
  "code": "AUTH_REQUIRED"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Cart not found",
  "code": "CART_NOT_FOUND"
}
```

#### 3.4.2 Add Item to Cart
- **Endpoint**: `POST /api/cart/items`
- **Description**: Add a product item to user's cart
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "productId": 1,
  "packageSize": "6-pack",
  "quantity": 2
}
```
- **Response (201 Created)**:
```json
{
  "success": true,
  "message": "Item added to cart successfully",
  "data": {
    "cartItem": {
      "id": 1,
      "productId": 1,
      "packageSize": "6-pack",
      "quantity": 2,
      "price": 850,
      "total": 1700,
      "product": {
        "id": 1,
        "name": "Premium Ghee",
        "category": "Dairy",
        "images": [
          "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
        ]
      }
    },
    "cartSummary": {
      "totalItems": 2,
      "subtotal": 1700
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "field": "productId",
      "message": "Product ID is required"
    },
    {
      "field": "quantity",
      "message": "Quantity must be at least 1"
    }
  ],
  "code": "VALIDATION_ERROR"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Product not found",
  "code": "PRODUCT_NOT_FOUND"
}
```
- **Error Response (409 Conflict)**:
```json
{
  "success": false,
  "message": "Item already exists in cart. Use update endpoint to modify quantity.",
  "code": "ITEM_ALREADY_IN_CART"
}
```

#### 3.4.3 Update Cart Item
- **Endpoint**: `PUT /api/cart/items/:id`
- **Description**: Update quantity of an existing cart item
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Path Parameters**: `id` (integer) - Cart item ID
- **Request Body**:
```json
{
  "quantity": 3
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Cart item updated successfully",
  "data": {
    "cartItem": {
      "id": 1,
      "productId": 1,
      "packageSize": "6-pack",
      "quantity": 3,
      "price": 850,
      "total": 2550,
      "product": {
        "id": 1,
        "name": "Premium Ghee"
      }
    },
    "cartSummary": {
      "totalItems": 3,
      "subtotal": 2550
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Quantity must be greater than 0",
  "code": "INVALID_QUANTITY"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Cart item not found",
  "code": "CART_ITEM_NOT_FOUND"
}
```

#### 3.4.4 Remove Cart Item
- **Endpoint**: `DELETE /api/cart/items/:id`
- **Description**: Remove a specific item from cart
- **Headers**: `Authorization: Bearer <token>`
- **Path Parameters**: `id` (integer) - Cart item ID
- **Request Body**: None (DELETE request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Item removed from cart successfully",
  "data": {
    "cartSummary": {
      "totalItems": 1,
      "subtotal": 850
    }
  }
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Cart item not found",
  "code": "CART_ITEM_NOT_FOUND"
}
```
- **Error Response (403 Forbidden)**:
```json
{
  "success": false,
  "message": "You don't have permission to remove this item",
  "code": "PERMISSION_DENIED"
}
```

#### 3.4.5 Clear Cart
- **Endpoint**: `DELETE /api/cart`
- **Description**: Remove all items from user's cart
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: None (DELETE request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Cart cleared successfully",
  "data": {
    "cartSummary": {
      "totalItems": 0,
      "subtotal": 0
    }
  }
}
```

#### 3.4.6 Get Cart Summary
- **Endpoint**: `GET /api/cart/summary`
- **Description**: Get cart summary without full item details
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Cart summary retrieved successfully",
  "data": {
    "summary": {
      "totalItems": 3,
      "totalProducts": 2,
      "subtotal": 2550,
      "estimatedTax": 238,
      "estimatedShipping": 50,
      "estimatedTotal": 2838
    }
  }
}
```

#### 3.4.7 Apply Coupon to Cart
- **Endpoint**: `POST /api/cart/coupon`
- **Description**: Apply a discount coupon to the cart
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "couponCode": "SAVE10"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Coupon applied successfully",
  "data": {
    "coupon": {
      "code": "SAVE10",
      "discountType": "percentage",
      "discountValue": 10,
      "discountAmount": 255
    },
    "cartSummary": {
      "subtotal": 2550,
      "couponDiscount": 255,
      "tax": 214,
      "shipping": 50,
      "total": 2559
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Invalid or expired coupon code",
  "code": "INVALID_COUPON"
}
```

### 3.5 Order APIs

#### 3.5.1 Create Order
- **Endpoint**: `POST /api/orders`
- **Description**: Create a new order from current cart items
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Request Body**:
```json
{
  "shippingAddressId": 1,
  "paymentMethod": "card",
  "paymentDetails": {
    "cardNumber": "****-****-****-1234",
    "cardType": "visa"
  },
  "couponCode": "SAVE10",
  "notes": "Please deliver in the evening after 6 PM"
}
```
- **Response (201 Created)**:
```json
{
  "success": true,
  "message": "Order created successfully",
  "data": {
    "order": {
      "id": 1,
      "orderNumber": "ORD-2025-001",
      "userId": 1,
      "status": "pending",
      "paymentStatus": "pending",
      "paymentMethod": "card",
      "subtotal": 2550,
      "discount": 255,
      "tax": 214,
      "shipping": 50,
      "total": 2559,
      "notes": "Please deliver in the evening after 6 PM",
      "items": [
        {
          "id": 1,
          "productId": 1,
          "packageSize": "6-pack",
          "quantity": 2,
          "price": 850,
          "total": 1700,
          "product": {
            "id": 1,
            "name": "Premium Ghee",
            "category": "Dairy",
            "images": [
              "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
            ]
          }
        },
        {
          "id": 2,
          "productId": 2,
          "packageSize": "500g",
          "quantity": 1,
          "price": 850,
          "total": 850,
          "product": {
            "id": 2,
            "name": "Organic Honey",
            "category": "Natural Care",
            "images": [
              "https://images.unsplash.com/photo-1587049016823-c90bb2e77f43?w=400"
            ]
          }
        }
      ],
      "shippingAddress": {
        "street": "123 Main St",
        "city": "New York",
        "state": "NY",
        "zipCode": "10001",
        "country": "USA"
      },
      "tracking": {
        "trackingNumber": null,
        "carrier": null,
        "estimatedDelivery": "2025-01-20T18:00:00Z"
      },
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T10:30:00Z"
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Cart is empty",
  "code": "EMPTY_CART"
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Shipping address not found",
  "code": "ADDRESS_NOT_FOUND"
}
```
- **Error Response (422 Unprocessable Entity)**:
```json
{
  "success": false,
  "message": "Some items in cart are out of stock",
  "errors": [
    {
      "productId": 1,
      "productName": "Premium Ghee",
      "message": "Only 1 item available, but 2 requested"
    }
  ],
  "code": "INSUFFICIENT_STOCK"
}
```

#### 3.5.2 Get User Orders
- **Endpoint**: `GET /api/orders`
- **Description**: Get paginated list of user's orders
- **Headers**: `Authorization: Bearer <token>`
- **Query Parameters**:
  - `page`: Page number (default: 1, integer)
  - `limit`: Items per page (default: 10, max: 50, integer)
  - `status`: Filter by order status (pending, processing, shipped, delivered, cancelled)
  - `startDate`: Filter orders from date (YYYY-MM-DD format)
  - `endDate`: Filter orders to date (YYYY-MM-DD format)
  - `sortBy`: Sort field (createdAt, total, status) (default: createdAt)
  - `sortOrder`: Sort order (asc, desc) (default: desc)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Orders retrieved successfully",
  "data": {
    "orders": [
      {
        "id": 1,
        "orderNumber": "ORD-2025-001",
        "status": "processing",
        "paymentStatus": "paid",
        "total": 2559,
        "itemCount": 3,
        "totalProducts": 2,
        "estimatedDelivery": "2025-01-20T18:00:00Z",
        "createdAt": "2025-01-15T10:30:00Z"
      },
      {
        "id": 2,
        "orderNumber": "ORD-2025-002",
        "status": "delivered",
        "paymentStatus": "paid",
        "total": 1200,
        "itemCount": 2,
        "totalProducts": 1,
        "deliveredAt": "2025-01-10T14:30:00Z",
        "createdAt": "2025-01-08T09:15:00Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 3,
      "totalItems": 25,
      "itemsPerPage": 10,
      "hasNextPage": true,
      "hasPrevPage": false
    },
    "summary": {
      "totalOrders": 25,
      "totalSpent": 45750,
      "averageOrderValue": 1830
    }
  }
}
```

#### 3.5.3 Get Order by ID
- **Endpoint**: `GET /api/orders/:id`
- **Description**: Get detailed information about a specific order
- **Headers**: `Authorization: Bearer <token>`
- **Path Parameters**: `id` (integer) - Order ID
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Order retrieved successfully",
  "data": {
    "order": {
      "id": 1,
      "orderNumber": "ORD-2025-001",
      "userId": 1,
      "status": "processing",
      "paymentStatus": "paid",
      "paymentMethod": "card",
      "paymentDetails": {
        "cardType": "visa",
        "last4": "1234",
        "transactionId": "txn_1234567890"
      },
      "subtotal": 2550,
      "discount": 255,
      "tax": 214,
      "shipping": 50,
      "total": 2559,
      "notes": "Please deliver in the evening after 6 PM",
      "items": [
        {
          "id": 1,
          "productId": 1,
          "packageSize": "6-pack",
          "quantity": 2,
          "price": 850,
          "total": 1700,
          "product": {
            "id": 1,
            "name": "Premium Ghee",
            "category": "Dairy",
            "images": [
              "https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"
            ]
          }
        }
      ],
      "shippingAddress": {
        "street": "123 Main St",
        "city": "New York",
        "state": "NY",
        "zipCode": "10001",
        "country": "USA"
      },
      "tracking": {
        "trackingNumber": "TRK123456789",
        "carrier": "FedEx",
        "estimatedDelivery": "2025-01-20T18:00:00Z",
        "trackingUrl": "https://fedex.com/track/TRK123456789"
      },
      "timeline": [
        {
          "status": "pending",
          "timestamp": "2025-01-15T10:30:00Z",
          "description": "Order placed successfully"
        },
        {
          "status": "confirmed",
          "timestamp": "2025-01-15T11:00:00Z",
          "description": "Payment confirmed"
        },
        {
          "status": "processing",
          "timestamp": "2025-01-15T14:30:00Z",
          "description": "Order is being prepared"
        }
      ],
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T14:30:00Z"
    }
  }
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Order not found",
  "code": "ORDER_NOT_FOUND"
}
```
- **Error Response (403 Forbidden)**:
```json
{
  "success": false,
  "message": "You don't have permission to view this order",
  "code": "PERMISSION_DENIED"
}
```

#### 3.5.4 Cancel Order
- **Endpoint**: `PATCH /api/orders/:id/cancel`
- **Description**: Cancel an order (only if status is pending or confirmed)
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Path Parameters**: `id` (integer) - Order ID
- **Request Body**:
```json
{
  "reason": "Changed my mind",
  "refundMethod": "original_payment"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Order cancelled successfully",
  "data": {
    "order": {
      "id": 1,
      "orderNumber": "ORD-2025-001",
      "status": "cancelled",
      "cancellationReason": "Changed my mind",
      "refundStatus": "processing",
      "refundAmount": 2559,
      "cancelledAt": "2025-01-15T16:30:00Z"
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Order cannot be cancelled. It has already been shipped.",
  "code": "CANNOT_CANCEL_ORDER"
}
```

#### 3.5.5 Track Order
- **Endpoint**: `GET /api/orders/:id/track`
- **Description**: Get tracking information for an order
- **Headers**: `Authorization: Bearer <token>`
- **Path Parameters**: `id` (integer) - Order ID
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Tracking information retrieved successfully",
  "data": {
    "tracking": {
      "orderNumber": "ORD-2025-001",
      "status": "in_transit",
      "trackingNumber": "TRK123456789",
      "carrier": "FedEx",
      "estimatedDelivery": "2025-01-20T18:00:00Z",
      "trackingUrl": "https://fedex.com/track/TRK123456789",
      "currentLocation": "New York Distribution Center",
      "updates": [
        {
          "status": "picked_up",
          "location": "Warehouse - Mumbai",
          "timestamp": "2025-01-16T08:00:00Z",
          "description": "Package picked up from warehouse"
        },
        {
          "status": "in_transit",
          "location": "Distribution Center - Delhi",
          "timestamp": "2025-01-17T12:00:00Z",
          "description": "Package in transit"
        },
        {
          "status": "in_transit",
          "location": "Distribution Center - New York",
          "timestamp": "2025-01-18T09:30:00Z",
          "description": "Package arrived at destination facility"
        }
      ]
    }
  }
}
```
- **Error Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Tracking information not available",
  "code": "TRACKING_NOT_AVAILABLE"
}
```

#### 3.5.6 Reorder
- **Endpoint**: `POST /api/orders/:id/reorder`
- **Description**: Add all items from a previous order to the current cart
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: application/json`
- **Path Parameters**: `id` (integer) - Order ID
- **Request Body**: `{}` (empty object)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Items added to cart successfully",
  "data": {
    "addedItems": [
      {
        "productId": 1,
        "packageSize": "6-pack",
        "quantity": 2,
        "added": true
      },
      {
        "productId": 2,
        "packageSize": "500g",
        "quantity": 1,
        "added": false,
        "reason": "Product out of stock"
      }
    ],
    "cartSummary": {
      "totalItems": 2,
      "subtotal": 1700
    }
  }
}
```

#### 3.5.7 Get Order Invoice
- **Endpoint**: `GET /api/orders/:id/invoice`
- **Description**: Get invoice details for an order
- **Headers**: `Authorization: Bearer <token>`
- **Path Parameters**: `id` (integer) - Order ID
- **Query Parameters**:
  - `format`: Response format (json, pdf) (default: json)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Invoice retrieved successfully",
  "data": {
    "invoice": {
      "invoiceNumber": "INV-2025-001",
      "orderNumber": "ORD-2025-001",
      "invoiceDate": "2025-01-15T10:30:00Z",
      "dueDate": "2025-01-15T10:30:00Z",
      "status": "paid",
      "customer": {
        "name": "John Doe",
        "email": "john.doe@example.com",
        "address": {
          "street": "123 Main St",
          "city": "New York",
          "state": "NY",
          "zipCode": "10001",
          "country": "USA"
        }
      },
      "items": [
        {
          "description": "Premium Ghee (6-pack)",
          "quantity": 2,
          "unitPrice": 850,
          "total": 1700
        }
      ],
      "summary": {
        "subtotal": 2550,
        "discount": 255,
        "tax": 214,
        "shipping": 50,
        "total": 2559
      },
      "paymentDetails": {
        "method": "card",
        "transactionId": "txn_1234567890",
        "paidAt": "2025-01-15T10:35:00Z"
      }
    }
  }
}
```

### 3.6 Additional Utility APIs

#### 3.6.1 Health Check
- **Endpoint**: `GET /health`
- **Description**: Check API and system health status
- **Headers**: None required
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "version": "1.0.0",
  "services": {
    "database": {
      "status": "connected",
      "responseTime": "5ms"
    },
    "redis": {
      "status": "connected",
      "responseTime": "2ms"
    },
    "fileStorage": {
      "status": "available",
      "responseTime": "12ms"
    }
  },
  "system": {
    "uptime": "72h 15m 30s",
    "memory": {
      "used": "256MB",
      "total": "1GB",
      "percentage": 25
    },
    "cpu": {
      "usage": "15%"
    }
  }
}
```
- **Response (503 Service Unavailable)**:
```json
{
  "status": "unhealthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "errors": [
    {
      "service": "database",
      "status": "disconnected",
      "error": "Connection timeout"
    }
  ]
}
```

#### 3.6.2 Upload Image
- **Endpoint**: `POST /api/upload/image`
- **Description**: Upload product or user images
- **Headers**: 
  - `Authorization: Bearer <token>`
  - `Content-Type: multipart/form-data`
- **Request Body**: Form data with image file
```
file: [binary image data]
category: product | profile | other
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Image uploaded successfully",
  "data": {
    "imageUrl": "https://cdn.datun.com/images/products/abc123.jpg",
    "publicId": "products/abc123",
    "size": 245760,
    "format": "jpg",
    "dimensions": {
      "width": 800,
      "height": 600
    }
  }
}
```
- **Error Response (400 Bad Request)**:
```json
{
  "success": false,
  "message": "Invalid file format. Only JPEG, PNG, and WebP are allowed.",
  "code": "INVALID_FILE_FORMAT"
}
```

#### 3.6.3 Get Application Statistics
- **Endpoint**: `GET /api/stats`
- **Description**: Get application statistics (admin only)
- **Headers**: `Authorization: Bearer <admin-token>`
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Statistics retrieved successfully",
  "data": {
    "users": {
      "total": 1250,
      "active": 980,
      "newThisMonth": 85
    },
    "products": {
      "total": 150,
      "inStock": 140,
      "outOfStock": 10,
      "categories": 8
    },
    "orders": {
      "total": 3450,
      "thisMonth": 320,
      "revenue": {
        "total": 2850000,
        "thisMonth": 285000,
        "currency": "INR"
      }
    },
    "cart": {
      "activeUsers": 120,
      "averageItems": 2.5,
      "totalValue": 456000
    }
  }
}
```

#### 3.6.4 Search Suggestions
- **Endpoint**: `GET /api/search/suggestions`
- **Description**: Get search suggestions based on query
- **Headers**: None required
- **Query Parameters**:
  - `q`: Search query (required, minimum 2 characters)
  - `limit`: Number of suggestions (default: 10, max: 20)
- **Request Body**: None (GET request)
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Suggestions retrieved successfully",
  "data": {
    "suggestions": [
      {
        "type": "product",
        "text": "Premium Ghee",
        "category": "Dairy",
        "matches": 3
      },
      {
        "type": "category",
        "text": "Dairy Products",
        "matches": 15
      },
      {
        "type": "brand",
        "text": "Organic Valley",
        "matches": 8
      }
    ],
    "totalSuggestions": 3,
    "query": "ghee"
  }
}
```

#### 3.6.5 Contact Support
- **Endpoint**: `POST /api/support/contact`
- **Description**: Submit a contact/support request
- **Headers**: 
  - `Content-Type: application/json`
  - `Authorization: Bearer <token>` (optional, for logged-in users)
- **Request Body**:
```json
{
  "name": "John Doe",
  "email": "john.doe@example.com",
  "subject": "Product Inquiry",
  "message": "I have a question about the Premium Ghee product...",
  "category": "product_inquiry",
  "orderId": 123
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Support request submitted successfully",
  "data": {
    "ticketId": "TICK-2025-001",
    "status": "open",
    "estimatedResponse": "24 hours",
    "submittedAt": "2025-01-15T10:30:00Z"
  }
}
```

#### 3.6.6 Newsletter Subscription
- **Endpoint**: `POST /api/newsletter/subscribe`
- **Description**: Subscribe to newsletter
- **Headers**: `Content-Type: application/json`
- **Request Body**:
```json
{
  "email": "john.doe@example.com",
  "preferences": {
    "productUpdates": true,
    "promotions": true,
    "newsletters": false
  }
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Successfully subscribed to newsletter",
  "data": {
    "email": "john.doe@example.com",
    "subscriptionId": "sub_abc123",
    "subscribedAt": "2025-01-15T10:30:00Z"
  }
}
```

#### 3.6.7 Unsubscribe from Newsletter
- **Endpoint**: `DELETE /api/newsletter/unsubscribe`
- **Description**: Unsubscribe from newsletter
- **Headers**: `Content-Type: application/json`
- **Request Body**:
```json
{
  "email": "john.doe@example.com",
  "token": "unsubscribe_token_abc123"
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Successfully unsubscribed from newsletter"
}
```

### 4.1 Standard Error Response Format
```json
{
  "success": false,
  "message": "Error description",
  "errors": [
    {
      "field": "fieldName",
      "message": "Specific error message"
    }
  ],
  "code": "ERROR_CODE"
}
```

### 4.2 HTTP Status Codes
- **200 OK**: Successful GET, PUT, DELETE requests
- **201 Created**: Successful POST requests
- **400 Bad Request**: Validation errors, malformed requests
- **401 Unauthorized**: Authentication required or invalid token
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource already exists
- **422 Unprocessable Entity**: Validation errors
- **500 Internal Server Error**: Server errors

## 5. Authentication & Security

### 5.1 JWT Token Structure
```json
{
  "user_id": 1,
  "email": "john.doe@example.com",
  "exp": 1640995200,
  "iat": 1640908800
}
```

### 5.2 Security Requirements
- Password hashing using bcrypt
- JWT token expiration (24 hours)
- Rate limiting on authentication endpoints
- Input validation and sanitization
- CORS configuration
- SQL injection prevention using parameterized queries

## 6. Database Design

### 6.1 PostgreSQL Schema Design

#### 6.1.1 Core Tables Structure

```sql
-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    date_of_birth DATE,
    gender VARCHAR(10),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Addresses table
CREATE TABLE addresses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    street TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    zip_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'India',
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table with JSONB for flexible attributes
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    short_description TEXT,
    long_description TEXT,
    images JSONB, -- Array of image URLs
    in_stock BOOLEAN DEFAULT true,
    is_popular BOOLEAN DEFAULT false,
    discount INTEGER DEFAULT 0,
    nutritional_info JSONB, -- Flexible nutrition data
    ingredients JSONB, -- Array of ingredients
    benefits JSONB, -- Array of benefits
    instructions TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Package options table
CREATE TABLE package_options (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    size VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Shopping carts table
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Cart items table
CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER REFERENCES carts(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    package_size VARCHAR(100),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cart_id, product_id, package_size)
);

-- Orders table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    order_number VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    payment_status VARCHAR(50) DEFAULT 'pending',
    payment_method VARCHAR(50),
    subtotal DECIMAL(10,2) NOT NULL,
    discount DECIMAL(10,2) DEFAULT 0,
    tax DECIMAL(10,2) DEFAULT 0,
    shipping DECIMAL(10,2) DEFAULT 0,
    total DECIMAL(10,2) NOT NULL,
    notes TEXT,
    shipping_address JSONB NOT NULL, -- Embedded address
    tracking_info JSONB, -- Tracking details
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Order items table
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    package_size VARCHAR(100),
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 6.1.2 Key Relationships
- **User → Addresses**: One-to-Many (Users can have multiple addresses)
- **User → Orders**: One-to-Many (Users can have multiple orders)
- **User → Cart**: One-to-One (Each user has one active cart)
- **Cart → CartItems**: One-to-Many (Cart contains multiple items)
- **Product → PackageOptions**: One-to-Many (Products can have multiple package sizes)
- **Order → OrderItems**: One-to-Many (Orders contain multiple items)

#### 6.1.3 Indexes for Performance

```sql
-- Essential indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);
CREATE INDEX idx_addresses_default ON addresses(user_id, is_default);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_stock ON products(in_stock);
CREATE INDEX idx_products_popular ON products(is_popular);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_created ON products(created_at DESC);

-- JSONB indexes for flexible queries
CREATE INDEX idx_products_attributes ON products USING GIN(nutritional_info);
CREATE INDEX idx_products_ingredients ON products USING GIN(ingredients);

CREATE INDEX idx_package_options_product ON package_options(product_id);

CREATE INDEX idx_carts_user ON carts(user_id);
CREATE INDEX idx_cart_items_cart ON cart_items(cart_id);
CREATE INDEX idx_cart_items_product ON cart_items(product_id);

CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created ON orders(created_at DESC);
CREATE INDEX idx_orders_number ON orders(order_number);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_product ON order_items(product_id);
```

#### 6.1.4 Advanced Features

**Full-text Search:**
```sql
-- Add full-text search capability
ALTER TABLE products ADD COLUMN search_vector tsvector;
CREATE INDEX idx_products_search ON products USING GIN(search_vector);

-- Update search vector trigger
CREATE OR REPLACE FUNCTION update_product_search()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.short_description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.category, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_product_search
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_product_search();
```

**JSONB Queries Examples:**
```sql
-- Query products by nutritional info
SELECT * FROM products 
WHERE nutritional_info->>'calories' LIKE '%900%';

-- Query products by benefits
SELECT * FROM products 
WHERE benefits @> '["Rich in vitamins A, D, E, and K"]';

-- Query products by ingredient
SELECT * FROM products 
WHERE ingredients @> '["Pure Cow Milk Cream"]';
```

### 6.2 Data Migration Strategy

#### 6.2.1 Migration Files Structure
```
migrations/
├── 001_create_users_table.up.sql
├── 001_create_users_table.down.sql
├── 002_create_addresses_table.up.sql
├── 002_create_addresses_table.down.sql
├── 003_create_products_table.up.sql
├── 003_create_products_table.down.sql
└── ...
```

#### 6.2.2 Seed Data Strategy
```sql
-- Sample seed data for development
INSERT INTO products (name, category, price, short_description, images, nutritional_info) VALUES
('Premium Ghee', 'Dairy', 850.00, 'Pure and aromatic cow ghee', 
 '["https://images.unsplash.com/photo-1563379091339-03246a5d6690?w=400"]',
 '{"calories": "900 per 100g", "fat": "100g", "saturatedFat": "60g"}');
```

## 7. API Rate Limiting

### 7.1 Rate Limits
- Authentication endpoints: 5 requests per minute per IP
- General API endpoints: 100 requests per minute per user
- Product listing: 1000 requests per hour per IP

## 8. Caching Strategy

### 8.1 Redis Caching
- Product listings: Cache for 1 hour
- Product details: Cache for 30 minutes
- User sessions: Store JWT blacklist
- Cart data: Cache for 24 hours

## 9. Deployment Requirements

### 9.1 Environment Variables
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=datun_db
DB_USER=postgres
DB_PASSWORD=password

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRES_IN=24h

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Server
PORT=8080
GIN_MODE=release

# CORS
ALLOWED_ORIGINS=http://localhost:3000,https://yourdomain.com
```

### 9.2 Docker Configuration
- Multi-stage build for optimized image size
- Health checks for container monitoring
- Volume mounts for persistent data

## 10. Testing Requirements

### 10.1 Unit Tests
- All service layer functions
- Authentication middleware
- Input validation
- Error handling

### 10.2 Integration Tests
- API endpoint testing
- Database operations
- Authentication flows
- Cart and order workflows

## 11. Monitoring & Logging

### 11.1 Logging Requirements
- Request/response logging
- Error logging with stack traces
- Performance metrics
- Authentication attempts

### 11.2 Health Check Endpoint
- **Endpoint**: `GET /health`
- **Response**:
```json
{
  "status": "healthy",
  "database": "connected",
  "redis": "connected",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## 12. Future Enhancements

### 12.1 Phase 2 Features
- Payment gateway integration
- Order tracking and notifications
- Product reviews and ratings
- Wishlist functionality
- Admin panel APIs
- Inventory management
- Coupon and discount system
- Email notifications

### 12.2 Performance Optimizations
- Database query optimization
- API response caching
- Image optimization and CDN
- Pagination improvements
- Search functionality enhancement

---

This PRD provides a comprehensive foundation for developing the backend API in Go. The specification covers all the functionality required by your React frontend and follows REST API best practices.
