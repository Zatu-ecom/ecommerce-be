# Product Variant System - Product Requirement Document (PRD)

## 1. Overview

This document outlines the complete product variant system for our SaaS e-commerce platform. The variant system enables merchants to sell products with multiple options (like color, size, storage) while maintaining proper inventory, pricing, and customer experience.

### 1.1 Purpose

The variant system replaces the simple `PackageOption` approach with a robust, flexible system that supports:

- Multiple product options (color, size, storage, style, etc.)
- Individual pricing and inventory per variant
- Complex variant combinations
- Structured data for filtering and search
- Seamless customer experience

### 1.2 Key Concepts

#### Product

The main product entity containing general information (name, description, category, brand).

#### Product Options

Configurable choices available for a product (e.g., Color, Size, Storage).

#### Product Option Values

Specific values for each option (e.g., Red, Blue, Black for Color).

#### Product Variants

Actual purchasable combinations of option values with individual SKU, price, and stock.

#### Variant Option Values

Junction table linking variants to their selected option values.

---

## 2. Database Schema

### 2.1 Core Tables

#### Product Table

```sql
CREATE TABLE product (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    category_id INTEGER NOT NULL REFERENCES category(id),
    brand VARCHAR(100),
    sku VARCHAR(50) UNIQUE NOT NULL,  -- Base SKU
    short_description TEXT,
    long_description TEXT,
    tags TEXT[],
    seller_id INTEGER NOT NULL REFERENCES user(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### Product Variant Table

```sql
CREATE TABLE product_variant (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    sku VARCHAR(50) UNIQUE NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    compare_at_price DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'USD',
    stock INTEGER DEFAULT 0,
    in_stock BOOLEAN DEFAULT true,
    images TEXT[],
    weight DECIMAL(10,2),
    is_default BOOLEAN DEFAULT false,
    is_popular BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### Product Option Table

```sql
CREATE TABLE product_option (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100),
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### Product Option Value Table

```sql
CREATE TABLE product_option_value (
    id SERIAL PRIMARY KEY,
    option_id INTEGER NOT NULL REFERENCES product_option(id) ON DELETE CASCADE,
    value VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    color_code VARCHAR(7),
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### Variant Option Value Table (Junction)

```sql
CREATE TABLE variant_option_value (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER NOT NULL REFERENCES product_variant(id) ON DELETE CASCADE,
    option_id INTEGER NOT NULL REFERENCES product_option(id) ON DELETE CASCADE,
    option_value_id INTEGER NOT NULL REFERENCES product_option_value(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(variant_id, option_id)  -- Each variant has one value per option
);
```

### 2.2 Entity Relationships

```
Product (1) ──→ (N) ProductOption ──→ (N) ProductOptionValue
   │                     │                        │
   │                     │                        │
   └──→ (N) ProductVariant ──→ (N) VariantOptionValue ──→ (1) ProductOptionValue
```

### 2.3 Indexes

```sql
-- Performance indexes
CREATE INDEX idx_product_variant_product_id ON product_variant(product_id);
CREATE INDEX idx_product_variant_sku ON product_variant(sku);
CREATE INDEX idx_product_option_product_id ON product_option(product_id);
CREATE INDEX idx_product_option_value_option_id ON product_option_value(option_id);
CREATE INDEX idx_variant_option_value_variant_id ON variant_option_value(variant_id);
CREATE INDEX idx_variant_option_value_option_value_id ON variant_option_value(option_value_id);
```

---

## 3. API Specifications

### 3.1 Product Management APIs (Updated)

#### 3.1.1 Get All Products

- **Endpoint**: `GET /api/products`
- **Description**: Get paginated list of products with variant information
- **Headers**: None required
- **Query Parameters**:
  - `page`: Page number (default: 1, integer)
  - `limit`: Items per page (default: 20, max: 100, integer)
  - `categoryId`: Filter by category ID (integer)
  - `search`: Search in product name and description (string)
  - `inStock`: Filter by stock status (boolean: true/false)
  - `isPopular`: Filter popular products (boolean: true/false)
  - `sortBy`: Sort field (name, price, createdAt) (default: createdAt)
  - `sortOrder`: Sort order (asc, desc) (default: desc)
  - `minPrice`: Minimum price filter (number)
  - `maxPrice`: Maximum price filter (number)
  - `brand`: Filter by brand (string)

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Products retrieved successfully",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "Premium Cotton T-Shirt",
        "categoryId": 2,
        "category": {
          "id": 2,
          "name": "Clothing"
        },
        "brand": "Nike",
        "sku": "TSHIRT-001",
        "shortDescription": "Comfortable premium cotton t-shirt",
        "longDescription": "Made with 100% organic cotton...",
        "tags": ["clothing", "cotton", "casual"],
        "sellerId": 101,
        "hasVariants": true,
        "priceRange": {
          "min": 25.0,
          "max": 35.0
        },
        "currency": "USD",
        "totalStock": 45,
        "inStock": true,
        "images": ["tshirt-main.jpg", "tshirt-alt.jpg"],
        "variantPreview": {
          "totalVariants": 9,
          "options": [
            {
              "name": "color",
              "displayName": "Color",
              "availableValues": ["Red", "Blue", "Black"]
            },
            {
              "name": "size",
              "displayName": "Size",
              "availableValues": ["S", "M", "L"]
            }
          ]
        },
        "createdAt": "2024-01-15T10:30:00Z",
        "updatedAt": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 5,
      "totalItems": 98,
      "itemsPerPage": 20
    }
  }
}
```

#### 3.1.2 Get Product by ID

- **Endpoint**: `GET /api/products/{productId}`
- **Description**: Get detailed product information with all variants and options
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
      "name": "Premium Cotton T-Shirt",
      "categoryId": 2,
      "category": {
        "id": 2,
        "name": "Clothing",
        "parent": {
          "id": 1,
          "name": "Fashion"
        }
      },
      "brand": "Nike",
      "sku": "TSHIRT-001",
      "shortDescription": "Comfortable premium cotton t-shirt",
      "longDescription": "Made with 100% organic cotton for ultimate comfort...",
      "tags": ["clothing", "cotton", "casual"],
      "sellerId": 101,
      "hasVariants": true,
      "attributes": [
        {
          "key": "material",
          "value": "100% Cotton",
          "definition": {
            "name": "Material",
            "unit": null
          }
        },
        {
          "key": "care_instructions",
          "value": "Machine wash cold",
          "definition": {
            "name": "Care Instructions",
            "unit": null
          }
        }
      ],
      "options": [
        {
          "id": 1,
          "name": "color",
          "displayName": "Color",
          "position": 1,
          "values": [
            {
              "id": 1,
              "value": "red",
              "displayName": "Red",
              "colorCode": "#FF0000",
              "position": 1
            },
            {
              "id": 2,
              "value": "blue",
              "displayName": "Blue",
              "colorCode": "#0000FF",
              "position": 2
            },
            {
              "id": 3,
              "value": "black",
              "displayName": "Black",
              "colorCode": "#000000",
              "position": 3
            }
          ]
        },
        {
          "id": 2,
          "name": "size",
          "displayName": "Size",
          "position": 2,
          "values": [
            {
              "id": 4,
              "value": "s",
              "displayName": "Small",
              "position": 1
            },
            {
              "id": 5,
              "value": "m",
              "displayName": "Medium",
              "position": 2
            },
            {
              "id": 6,
              "value": "l",
              "displayName": "Large",
              "position": 3
            }
          ]
        }
      ],
      "variants": [
        {
          "id": 1,
          "sku": "TSHIRT-001-RED-S",
          "price": 25.0,
          "compareAtPrice": null,
          "currency": "USD",
          "stock": 10,
          "inStock": true,
          "images": ["tshirt-red-s-front.jpg", "tshirt-red-s-back.jpg"],
          "isDefault": false,
          "selectedOptions": [
            {
              "optionId": 1,
              "optionName": "color",
              "optionDisplayName": "Color",
              "valueId": 1,
              "value": "red",
              "valueDisplayName": "Red"
            },
            {
              "optionId": 2,
              "optionName": "size",
              "optionDisplayName": "Size",
              "valueId": 4,
              "value": "s",
              "valueDisplayName": "Small"
            }
          ]
        },
        {
          "id": 2,
          "sku": "TSHIRT-001-RED-M",
          "price": 25.0,
          "compareAtPrice": null,
          "currency": "USD",
          "stock": 15,
          "inStock": true,
          "images": ["tshirt-red-m-front.jpg"],
          "isDefault": true,
          "selectedOptions": [
            {
              "optionId": 1,
              "optionName": "color",
              "optionDisplayName": "Color",
              "valueId": 1,
              "value": "red",
              "valueDisplayName": "Red"
            },
            {
              "optionId": 2,
              "optionName": "size",
              "optionDisplayName": "Size",
              "valueId": 5,
              "value": "m",
              "valueDisplayName": "Medium"
            }
          ]
        },
        {
          "id": 3,
          "sku": "TSHIRT-001-BLACK-L",
          "price": 30.0,
          "compareAtPrice": 35.0,
          "currency": "USD",
          "stock": 5,
          "inStock": true,
          "images": ["tshirt-black-l.jpg"],
          "isDefault": false,
          "selectedOptions": [
            {
              "optionId": 1,
              "optionName": "color",
              "optionDisplayName": "Color",
              "valueId": 3,
              "value": "black",
              "valueDisplayName": "Black"
            },
            {
              "optionId": 2,
              "optionName": "size",
              "optionDisplayName": "Size",
              "valueId": 6,
              "value": "l",
              "valueDisplayName": "Large"
            }
          ]
        }
      ],
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.1.3 Create Product

- **Endpoint**: `POST /api/products`
- **Description**: Create a new product with variants
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json

**Request Body**:

```json
{
  "name": "Premium Cotton T-Shirt",
  "categoryId": 2,
  "brand": "Nike",
  "sku": "TSHIRT-001",
  "shortDescription": "Comfortable premium cotton t-shirt",
  "longDescription": "Made with 100% organic cotton for ultimate comfort and style...",
  "tags": ["clothing", "cotton", "casual"],
  "attributes": [
    {
      "key": "material",
      "value": "100% Cotton"
    },
    {
      "key": "care_instructions",
      "value": "Machine wash cold"
    }
  ],
  "options": [
    {
      "name": "color",
      "displayName": "Color",
      "position": 1,
      "values": [
        {
          "value": "red",
          "displayName": "Red",
          "colorCode": "#FF0000",
          "position": 1
        },
        {
          "value": "blue",
          "displayName": "Blue",
          "colorCode": "#0000FF",
          "position": 2
        },
        {
          "value": "black",
          "displayName": "Black",
          "colorCode": "#000000",
          "position": 3
        }
      ]
    },
    {
      "name": "size",
      "displayName": "Size",
      "position": 2,
      "values": [
        {
          "value": "s",
          "displayName": "Small",
          "position": 1
        },
        {
          "value": "m",
          "displayName": "Medium",
          "position": 2
        },
        {
          "value": "l",
          "displayName": "Large",
          "position": 3
        }
      ]
    }
  ],
  "variants": [
    {
      "sku": "TSHIRT-001-RED-S",
      "price": 25.0,
      "images": ["tshirt-red-s-front.jpg", "tshirt-red-s-back.jpg"],
      "isDefault": false,
      "selectedOptions": [
        { "optionName": "color", "value": "red" },
        { "optionName": "size", "value": "s" }
      ]
    },
    {
      "sku": "TSHIRT-001-RED-M",
      "price": 25.0,
      "images": ["tshirt-red-m.jpg"],
      "isDefault": true,
      "selectedOptions": [
        { "optionName": "color", "value": "red" },
        { "optionName": "size", "value": "m" }
      ]
    },
    {
      "sku": "TSHIRT-001-BLACK-L",
      "price": 30.0,
      "images": ["tshirt-black-l.jpg"],
      "isDefault": false,
      "selectedOptions": [
        { "optionName": "color", "value": "black" },
        { "optionName": "size", "value": "l" }
      ]
    }
  ],
  "autoGenerateVariants": false
}
```

**Alternative Request (Auto-generate variants)**:

```json
{
  "name": "Premium Cotton T-Shirt",
  "categoryId": 2,
  "brand": "Nike",
  "sku": "TSHIRT-001",
  "shortDescription": "Comfortable premium cotton t-shirt",
  "options": [
    {
      "name": "color",
      "displayName": "Color",
      "values": [
        { "value": "red", "displayName": "Red", "colorCode": "#FF0000" },
        { "value": "blue", "displayName": "Blue", "colorCode": "#0000FF" }
      ]
    },
    {
      "name": "size",
      "displayName": "Size",
      "values": [
        { "value": "s", "displayName": "Small" },
        { "value": "m", "displayName": "Medium" }
      ]
    }
  ],
  "autoGenerateVariants": true,
  "defaultVariantSettings": {
    "price": 25.0,
    "stock": 10,
    "currency": "USD"
  }
}
```

**Validation Rules**:

- `name`: Required, 3-200 characters
- `categoryId`: Required, must exist and be active
- `brand`: Optional, max 100 characters
- `sku`: Required, unique, 3-50 characters (base SKU)
- `shortDescription`: Optional, max 500 characters
- `longDescription`: Optional, max 5000 characters
- `tags`: Optional array, max 20 tags
- `options`: Required if hasVariants=true, at least 1 option
- `variants`: Required, at least 1 variant
- Each variant SKU must be unique across all products

**Response (201 Created)**:

```json
{
  "success": true,
  "message": "Product created successfully",
  "data": {
    "product": {
      "id": 1,
      "name": "Premium Cotton T-Shirt",
      "sku": "TSHIRT-001",
      "hasVariants": true,
      "totalVariants": 6,
      "createdAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

#### 3.1.4 Update Product

- **Endpoint**: `PUT /api/products/{productId}`
- **Description**: Update an existing product (base information only, use variant endpoints for variants)
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID to update

**Request Body** (all fields optional):

```json
{
  "name": "Premium Organic Cotton T-Shirt",
  "shortDescription": "Updated description",
  "longDescription": "Updated long description...",
  "tags": ["organic", "cotton", "eco-friendly"]
}
```

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Product updated successfully",
  "data": {
    "product": {
      "id": 1,
      "name": "Premium Organic Cotton T-Shirt",
      "updatedAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

#### 3.1.5 Delete Product

- **Endpoint**: `DELETE /api/products/{productId}`
- **Description**: Delete a product (cascades to all variants)
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
- **Path Parameters**:
  - `productId`: Product ID to delete

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Product deleted successfully"
}
```

---

### 3.2 Variant Management APIs

#### 3.2.1 Get Specific Variant

- **Endpoint**: `GET /api/products/{productId}/variants/{variantId}`
- **Description**: Get detailed information about a specific variant
- **Headers**: None required
- **Path Parameters**:
  - `variantId`: Variant ID

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variant retrieved successfully",
  "data": {
    "variant": {
      "id": 2,
      "productId": 1,
      "sku": "TSHIRT-001-RED-M",
      "price": 25.0,
      "currency": "USD",
      "stock": 15,
      "inStock": true,
      "images": ["tshirt-red-m-front.jpg", "tshirt-red-m-back.jpg"],
      "weight": 0.2,
      "isDefault": true,
      "selectedOptions": [
        {
          "optionId": 1,
          "optionName": "color",
          "optionDisplayName": "Color",
          "valueId": 1,
          "value": "red",
          "valueDisplayName": "Red",
          "colorCode": "#FF0000"
        },
        {
          "optionId": 2,
          "optionName": "size",
          "optionDisplayName": "Size",
          "valueId": 5,
          "value": "m",
          "valueDisplayName": "Medium"
        }
      ],
      "product": {
        "id": 1,
        "name": "Premium Cotton T-Shirt",
        "brand": "Nike"
      }
    }
  }
}
```

#### 3.2.2 Find Variant by Options

- **Endpoint**: `GET /api/products/{productId}/variants/find`
- **Description**: Find specific variant by selected options
- **Headers**: None required
- **Path Parameters**:
  - `productId`: Product ID
- **Query Parameters**:
  - `color`: Selected color value (example)
  - `size`: Selected size value (example)
  - Or generic: `options`: JSON string of selected options

**Example**: `GET /api/products/1/variants/find?color=red&size=m`

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variant found successfully",
  "data": {
    "variant": {
      "id": 2,
      "sku": "TSHIRT-001-RED-M",
      "price": 25.0,
      "compareAtPrice": null,
      "stock": 15,
      "inStock": true,
      "images": ["tshirt-red-m.jpg"],
      "selectedOptions": [
        {
          "optionId": 1,
          "optionName": "color",
          "optionDisplayName": "Color",
          "valueId": 1,
          "value": "red",
          "valueDisplayName": "Red",
          "colorCode": "#FF0000"
        },
        {
          "optionId": 2,
          "optionName": "size",
          "optionDisplayName": "Size",
          "valueId": 5,
          "value": "m",
          "valueDisplayName": "Medium"
        }
      ]
    }
  }
}
```

**Error Response (404 Not Found)**:

```json
{
  "success": false,
  "message": "No variant found with selected options",
  "errorCode": "VARIANT_NOT_FOUND",
  "details": {
    "requestedOptions": {
      "color": "red",
      "size": "xl"
    },
    "availableOptions": {
      "color": ["red", "blue", "black"],
      "size": ["s", "m", "l"]
    }
  }
}
```

#### 3.2.3 Get Available Options for Product

- **Endpoint**: `GET /api/products/{productId}/options`
- **Description**: Get all available options and their values for a product
- **Headers**: None required
- **Path Parameters**:
  - `productId`: Product ID

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Product options retrieved successfully",
  "data": {
    "options": [
      {
        "id": 1,
        "name": "color",
        "displayName": "Color",
        "position": 1,
        "values": [
          {
            "id": 1,
            "value": "red",
            "displayName": "Red",
            "variantCount": 3
          },
          {
            "id": 2,
            "value": "blue",
            "displayName": "Blue",
            "variantCount": 3
          },
          {
            "id": 3,
            "value": "black",
            "displayName": "Black",
            "variantCount": 3
          }
        ]
      },
      {
        "id": 2,
        "name": "size",
        "displayName": "Size",
        "position": 2,
        "values": [
          {
            "id": 4,
            "value": "s",
            "displayName": "Small",
            "variantCount": 3
          },
          {
            "id": 5,
            "value": "m",
            "displayName": "Medium",
            "variantCount": 3
          },
          {
            "id": 6,
            "value": "l",
            "displayName": "Large",
            "variantCount": 3
          }
        ]
      }
    ]
  }
}
```

---

### 3.3 Seller Variant Management APIs

#### 3.3.1 Add Variant to Existing Product

- **Endpoint**: `POST /api/products/{productId}/variants`
- **Description**: Add a new variant to an existing product
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID

**Request Body**:

```json
{
  "sku": "TSHIRT-001-BLUE-XL",
  "price": 28.0,
  "compareAtPrice": null,
  "stock": 20,
  "images": ["tshirt-blue-xl.jpg"],
  "selectedOptions": [
    { "optionName": "color", "value": "blue" },
    { "optionName": "size", "value": "xl" }
  ]
}
```

**Validation Rules**:

- `sku`: Required, unique across all variants
- `price`: Required, positive number
- `stock`: Required, non-negative integer
- `selectedOptions`: Required, must match existing product options
- Each option in selectedOptions must reference valid option and value

**Response (201 Created)**:

```json
{
  "success": true,
  "message": "Variant added successfully",
  "data": {
    "variant": {
      "id": 10,
      "sku": "TSHIRT-001-BLUE-XL",
      "price": 28.0,
      "stock": 20,
      "createdAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

#### 3.3.2 Update Variant

- **Endpoint**: `PUT /api/products/{productId}/variants/{variantId}`
- **Description**: Update an existing variant
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID
  - `variantId`: Variant ID

**Request Body** (all fields optional):

```json
{
  "price": 27.0,
  "compareAtPrice": 30.0,
  "stock": 25,
  "images": ["tshirt-red-m-new.jpg"],
  "isDefault": true
}
```

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variant updated successfully",
  "data": {
    "variant": {
      "id": 2,
      "sku": "TSHIRT-001-RED-M",
      "price": 27.0,
      "compareAtPrice": 30.0,
      "stock": 25,
      "updatedAt": "2024-01-15T11:15:00Z"
    }
  }
}
```

#### 3.3.3 Delete Variant

- **Endpoint**: `DELETE /api/products/{productId}/variants/{variantId}`
- **Description**: Delete a specific variant
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
- **Path Parameters**:
  - `productId`: Product ID
  - `variantId`: Variant ID

**Business Rules**:

- Cannot delete the last variant of a product
- If deleting default variant, automatically sets another variant as default

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variant deleted successfully"
}
```

**Error Response (400 Bad Request)**:

```json
{
  "success": false,
  "message": "Cannot delete the last variant of a product",
  "errorCode": "LAST_VARIANT_DELETE_NOT_ALLOWED"
}
```

#### 3.3.4 Update Variant Stock

- **Endpoint**: `PATCH /api/products/{productId}/variants/{variantId}/stock`
- **Description**: Update stock for a specific variant
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID
  - `variantId`: Variant ID

**Request Body**:

```json
{
  "stock": 50,
  "operation": "set"
}
```

**Operations**:

- `set`: Set stock to exact value
- `add`: Increase stock by value
- `subtract`: Decrease stock by value

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variant stock updated successfully",
  "data": {
    "variant": {
      "id": 2,
      "sku": "TSHIRT-001-RED-M",
      "stock": 50,
      "inStock": true
    }
  }
}
```

#### 3.3.5 Bulk Update Variants

- **Endpoint**: `PUT /api/products/{productId}/variants/bulk`
- **Description**: Update multiple variants at once
- **Headers**:
  - `Authorization`: Bearer token (Seller required)
  - `Content-Type`: application/json
- **Path Parameters**:
  - `productId`: Product ID

**Request Body**:

```json
{
  "variants": [
    {
      "id": 1,
      "price": 26.0,
      "stock": 15
    },
    {
      "id": 2,
      "price": 26.0,
      "stock": 20
    },
    {
      "id": 3,
      "price": 32.0,
      "compareAtPrice": 38.0,
      "stock": 8
    }
  ]
}
```

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Variants updated successfully",
  "data": {
    "updatedCount": 3,
    "variants": [
      { "id": 1, "price": 26.0, "stock": 15 },
      { "id": 2, "price": 26.0, "stock": 20 },
      { "id": 3, "price": 32.0, "stock": 8 }
    ]
  }
}
```

---

### 3.4 Search and Filter APIs

#### 3.4.1 Search Products with Variant Filters

- **Endpoint**: `GET /api/products/search`
- **Description**: Advanced product search with variant-aware filtering
- **Headers**: None required
- **Query Parameters**:
  - `q`: Search query (string, required)
  - `categoryId`: Filter by category (integer)
  - `minPrice`: Minimum variant price (number)
  - `maxPrice`: Maximum variant price (number)
  - `inStock`: Only products with available variants (boolean)
  - `options`: Filter by variant options (e.g., `color:red,size:m`)
  - `page`: Page number (default: 1)
  - `limit`: Items per page (default: 20, max: 100)
  - `sortBy`: Sort field (relevance, name, price, createdAt)
  - `sortOrder`: Sort order (asc, desc)

**Example**: `GET /api/products/search?q=t-shirt&color=red&minPrice=20&maxPrice=50&page=1&limit=20`

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Products found successfully",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "Premium Cotton T-Shirt",
        "brand": "Nike",
        "priceRange": { "min": 25.0, "max": 35.0 },
        "availableOptions": {
          "color": ["Red", "Blue", "Black"],
          "size": ["S", "M", "L"]
        },
        "matchingVariants": 3,
        "totalVariants": 9,
        "images": ["tshirt-main.jpg"],
        "relevanceScore": 0.95
      }
    ],
    "filters": {
      "appliedFilters": {
        "color": "red",
        "priceRange": "20-50"
      },
      "availableFilters": {
        "colors": [
          { "value": "red", "displayName": "Red", "count": 15 },
          { "value": "blue", "displayName": "Blue", "count": 12 }
        ],
        "sizes": [
          { "value": "s", "displayName": "Small", "count": 8 },
          { "value": "m", "displayName": "Medium", "count": 20 }
        ]
      }
    },
    "pagination": {
      "currentPage": 1,
      "totalPages": 2,
      "totalItems": 25,
      "itemsPerPage": 20
    }
  }
}
```

#### 3.4.2 Get Product Filters with Variant Data

- **Endpoint**: `GET /api/products/filters`
- **Description**: Get available filters based on variant data
- **Headers**: None required
- **Query Parameters**:
  - `categoryId`: Get filters for specific category (integer, optional)

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
          "name": "Clothing",
          "productCount": 45
        },
        {
          "id": 2,
          "name": "Electronics",
          "productCount": 32
        }
      ],
      "brands": [
        {
          "name": "Nike",
          "productCount": 12
        },
        {
          "name": "Adidas",
          "productCount": 8
        }
      ],
      "priceRanges": [
        {
          "range": "0-25",
          "productCount": 15
        },
        {
          "range": "25-50",
          "productCount": 30
        },
        {
          "range": "50-100",
          "productCount": 18
        }
      ],
      "variantOptions": [
        {
          "optionName": "color",
          "displayName": "Color",
          "values": [
            {
              "value": "red",
              "displayName": "Red",
              "productCount": 25,
              "variantCount": 45
            },
            {
              "value": "blue",
              "displayName": "Blue",
              "productCount": 20,
              "variantCount": 38
            },
            {
              "value": "black",
              "displayName": "Black",
              "productCount": 18,
              "variantCount": 32
            }
          ]
        },
        {
          "optionName": "size",
          "displayName": "Size",
          "values": [
            {
              "value": "s",
              "displayName": "Small",
              "productCount": 18,
              "variantCount": 25
            },
            {
              "value": "m",
              "displayName": "Medium",
              "productCount": 22,
              "variantCount": 35
            },
            {
              "value": "l",
              "displayName": "Large",
              "productCount": 20,
              "variantCount": 30
            }
          ]
        }
      ]
    }
  }
}
```

---

### 3.5 Cart and Order Integration

#### 3.5.1 Add Variant to Cart

- **Endpoint**: `POST /api/cart/add`
- **Description**: Add specific variant to cart
- **Headers**:
  - `Authorization`: Bearer token (Customer required)
  - `Content-Type`: application/json

**Request Body**:

```json
{
  "variantId": 2,
  "quantity": 1
}
```

**Validation Rules**:

- `variantId`: Required, must exist
- `quantity`: Required, positive integer, must not exceed available stock

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Item added to cart successfully",
  "data": {
    "cartItem": {
      "id": 15,
      "variantId": 2,
      "quantity": 1,
      "variant": {
        "id": 2,
        "sku": "TSHIRT-001-RED-M",
        "price": 25.0,
        "images": ["tshirt-red-m.jpg"],
        "product": {
          "name": "Premium Cotton T-Shirt",
          "brand": "Nike"
        },
        "selectedOptions": [
          { "optionName": "color", "displayName": "Red" },
          { "optionName": "size", "displayName": "Medium" }
        ]
      },
      "subtotal": 25.0
    }
  }
}
```

**Error Response (400 Bad Request)**:

```json
{
  "success": false,
  "message": "Insufficient stock for selected variant",
  "errorCode": "INSUFFICIENT_STOCK",
  "details": {
    "variantId": 2,
    "requestedQuantity": 20,
    "availableStock": 15
  }
}
```

#### 3.5.2 Get Cart with Variant Details

- **Endpoint**: `GET /api/cart`
- **Description**: Get cart with full variant information
- **Headers**:
  - `Authorization`: Bearer token (Customer required)

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Cart retrieved successfully",
  "data": {
    "cart": {
      "items": [
        {
          "id": 15,
          "quantity": 2,
          "variant": {
            "id": 2,
            "sku": "TSHIRT-001-RED-M",
            "price": 25.0,
            "compareAtPrice": null,
            "images": ["tshirt-red-m.jpg"],
            "inStock": true,
            "stock": 15,
            "product": {
              "id": 1,
              "name": "Premium Cotton T-Shirt",
              "brand": "Nike"
            },
            "selectedOptions": [
              { "optionName": "color", "displayName": "Red" },
              { "optionName": "size", "displayName": "Medium" }
            ]
          },
          "subtotal": 50.0
        },
        {
          "id": 16,
          "quantity": 1,
          "variant": {
            "id": 5,
            "sku": "IPHONE15PRO-256GB-BLUE",
            "price": 1099.0,
            "compareAtPrice": null,
            "images": ["iphone-blue.jpg"],
            "inStock": true,
            "stock": 25,
            "product": {
              "id": 2,
              "name": "iPhone 15 Pro",
              "brand": "Apple"
            },
            "selectedOptions": [
              { "optionName": "storage", "displayName": "256GB" },
              { "optionName": "color", "displayName": "Blue Titanium" }
            ]
          },
          "subtotal": 1099.0
        }
      ],
      "totalItems": 3,
      "totalAmount": 1149.0,
      "currency": "USD"
    }
  }
}
```

---

## 4. Error Handling

### 4.1 Common Error Responses

#### 4.1.1 Variant Not Found (404)

```json
{
  "success": false,
  "message": "Variant not found",
  "errorCode": "VARIANT_NOT_FOUND",
  "statusCode": 404
}
```

#### 4.1.2 Invalid Option Combination (400)

```json
{
  "success": false,
  "message": "Invalid option combination",
  "errorCode": "INVALID_OPTION_COMBINATION",
  "statusCode": 400,
  "details": {
    "requestedOptions": {
      "color": "red",
      "size": "xl"
    },
    "availableCombinations": [
      { "color": "red", "size": "s" },
      { "color": "red", "size": "m" },
      { "color": "red", "size": "l" }
    ]
  }
}
```

#### 4.1.3 Variant Out of Stock (400)

```json
{
  "success": false,
  "message": "Selected variant is out of stock",
  "errorCode": "VARIANT_OUT_OF_STOCK",
  "statusCode": 400,
  "details": {
    "variantId": 5,
    "sku": "TSHIRT-001-BLUE-L",
    "currentStock": 0,
    "alternativeVariants": [
      {
        "variantId": 2,
        "sku": "TSHIRT-001-RED-M",
        "inStock": true,
        "stock": 15
      }
    ]
  }
}
```

#### 4.1.4 Duplicate SKU (409)

```json
{
  "success": false,
  "message": "SKU already exists",
  "errorCode": "SKU_CONFLICT",
  "statusCode": 409,
  "details": {
    "sku": "TSHIRT-001-RED-M",
    "existingVariantId": 2
  }
}
```

#### 4.1.5 Invalid Variant Operation (400)

```json
{
  "success": false,
  "message": "Cannot delete the last variant of a product",
  "errorCode": "LAST_VARIANT_DELETE_NOT_ALLOWED",
  "statusCode": 400
}
```

#### 4.1.6 Insufficient Stock (400)

```json
{
  "success": false,
  "message": "Insufficient stock for selected variant",
  "errorCode": "INSUFFICIENT_STOCK",
  "statusCode": 400,
  "details": {
    "variantId": 2,
    "requestedQuantity": 20,
    "availableStock": 15
  }
}
```

---

## 5. Business Rules

### 5.1 Variant Management Rules

1. **Product Creation**

   - Every product must have at least one variant
   - If no options are provided, create a default variant automatically
   - Base SKU must be unique across all products for a seller
   - Each variant SKU must be unique across all products globally

2. **Variant Options**

   - Options are product-specific, not seller-specific
   - Each option can have unlimited values
   - Position field determines display order
   - Cannot delete an option if variants are using it

3. **Stock Management**

   - Stock is managed at variant level, not product level
   - Product is "in stock" if any variant has stock > 0
   - Stock can never be negative
   - Support three operations: set, add, subtract

4. **Pricing**

   - Each variant can have its own price
   - `compareAtPrice` is optional (for showing discounts)
   - All prices must be positive numbers
   - Currency is set at variant level (default: USD)

5. **Default Variants**

   - Only one variant per product can be marked as default
   - Default variant is used for product listing displays
   - If default variant is deleted, automatically assign another variant as default
   - If no default is specified, the first variant becomes default

6. **Variant Deletion**
   - Cannot delete the last variant of a product
   - Deleting a variant removes it from carts (with notification to customers)
   - Cascading delete: deleting product deletes all its variants

### 5.2 Search and Filter Rules

1. **Price Filtering**

   - Min/max price filters apply to variant prices
   - A product matches if any of its variants fall within the range

2. **Option Filtering**

   - Customers can filter by any product option (color, size, etc.)
   - Multiple filters use AND logic (e.g., red AND medium)
   - Show product count for each filter value

3. **Stock Filtering**
   - `inStock=true` shows only products with at least one in-stock variant
   - Out-of-stock variants are hidden but not deleted

### 5.3 Cart and Order Rules

1. **Cart Items**

   - Cart items reference specific variants, not products
   - If a variant is deleted, remove it from carts
   - If variant price changes, cart reflects new price
   - Stock is reserved only at checkout, not when added to cart

2. **Order Creation**
   - Orders contain variant IDs, not product IDs
   - Capture variant price at time of order
   - Reduce variant stock upon successful order
   - Orders store snapshot of variant data for historical reference

---

## 6. Performance Considerations

### 6.1 Database Optimization

1. **Indexes**

   - Index on `product_variant.product_id` for fast variant lookups
   - Index on `product_variant.sku` for SKU searches
   - Composite index on `variant_option_value(variant_id, option_id)` for option queries
   - Index on `product_option.product_id` for option lookups

2. **Query Optimization**

   - Use JOIN queries to fetch product + variants + options in single query
   - Implement pagination for large variant lists
   - Use lazy loading for variant images

3. **Caching Strategy**
   - Cache product options (rarely change): 1 hour
   - Cache product variants: 15 minutes
   - Cache product listings: 5 minutes
   - Invalidate cache on variant updates

### 6.2 API Performance

1. **Response Optimization**

   - Include only necessary fields in list endpoints
   - Use `variantPreview` for product listings (summary data)
   - Full variant data only in product detail endpoint
   - Support field selection via query params (future)

2. **Bulk Operations**
   - Support bulk variant updates to reduce API calls
   - Implement batch stock updates
   - Use database transactions for consistency

---

## 7. Multi-Tenant Considerations

### 7.1 Seller Isolation

1. **Data Separation**

   - All products and variants belong to specific sellers
   - Sellers can only manage their own products/variants
   - Middleware validates seller ownership before operations

2. **Category Management**
   - Global categories shared across all sellers
   - Sellers can create custom categories
   - Products linked to categories (global or custom)

### 7.2 Subscription-Based Access

1. **Feature Limits**
   - Plan-based limits on number of products
   - Plan-based limits on variants per product
   - Plan-based limits on product images
   - Enforce limits via middleware

---

## 8. Future Enhancements

### 8.1 Planned Features

1. **Variant Images**

   - Support multiple images per variant
   - Image position management
   - Image optimization and CDN integration

2. **Variant Inventory**

   - Integration with separate inventory service
   - Real-time stock updates
   - Low stock alerts
   - Inventory history tracking

3. **Smart Variant Generation**

   - AI-powered variant suggestions
   - Bulk variant creation from CSV
   - Variant templates for quick setup

4. **Advanced Pricing**

   - Tiered pricing (bulk discounts)
   - Time-based pricing (flash sales)
   - Customer group pricing
   - Dynamic pricing rules

5. **Variant Analytics**
   - Best-selling variant reports
   - Stock turnover analysis
   - Price optimization suggestions
   - Variant performance metrics

---

## 9. Appendix

### 9.1 Example Use Cases

#### Use Case 1: T-Shirt with Color and Size

```
Product: Premium Cotton T-Shirt
Options: Color (Red, Blue, Black), Size (S, M, L)
Variants: 9 variants (3 colors × 3 sizes)
Result: Customer selects Red + Medium, adds specific variant to cart
```

#### Use Case 2: Smartphone with Storage

```
Product: iPhone 15 Pro
Options: Storage (128GB, 256GB, 512GB), Color (Natural, Blue)
Variants: 6 variants (3 storage × 2 colors)
Result: Different pricing for different storage capacities
```

#### Use Case 3: Simple Product (No Variants)

```
Product: USB Cable
Options: None
Variants: 1 default variant
Result: Customer sees simple add-to-cart button
```

### 9.2 Sample Data

#### Sample Product with Variants (JSON)

```json
{
  "product": {
    "id": 1,
    "name": "Premium Cotton T-Shirt",
    "sku": "TSHIRT-001",
    "brand": "Nike",
    "options": [
      {
        "id": 1,
        "name": "color",
        "values": [
          { "id": 1, "value": "red", "displayName": "Red" },
          { "id": 2, "value": "blue", "displayName": "Blue" }
        ]
      },
      {
        "id": 2,
        "name": "size",
        "values": [
          { "id": 3, "value": "s", "displayName": "Small" },
          { "id": 4, "value": "m", "displayName": "Medium" }
        ]
      }
    ],
    "variants": [
      {
        "id": 1,
        "sku": "TSHIRT-001-RED-S",
        "price": 25.0,
        "stock": 10,
        "selectedOptions": [
          { "optionId": 1, "valueId": 1 },
          { "optionId": 2, "valueId": 3 }
        ]
      },
      {
        "id": 2,
        "sku": "TSHIRT-001-RED-M",
        "price": 25.0,
        "stock": 15,
        "isDefault": true,
        "selectedOptions": [
          { "optionId": 1, "valueId": 1 },
          { "optionId": 2, "valueId": 4 }
        ]
      }
    ]
  }
}
```

---

**Document Version**: 1.0  
**Last Updated**: 2024-01-15  
**Status**: Final  
**Author**: Product Team
