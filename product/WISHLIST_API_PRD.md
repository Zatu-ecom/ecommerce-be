# Wishlist API - Product Requirements Document

> **Last Updated**: January 24, 2026  
> **Module**: Order Service  
> **Status**: Draft

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Entity Design](#entity-design)
3. [API Endpoints](#api-endpoints)
4. [Request/Response Specifications](#requestresponse-specifications)
5. [Business Rules](#business-rules)
6. [Error Codes](#error-codes)
7. [Database Schema](#database-schema)

---

## 🎯 Overview

### Features

| Feature                | Description                                                           |
| ---------------------- | --------------------------------------------------------------------- |
| **Multiple wishlist** | Users can have multiple wishlist (e.g., "My Wishlist", "Gift Ideas") |
| **Default Wishlist**   | One wishlist is marked as default for quick "add to wishlist" actions |
| **Variant-based**      | item are saved at variant level (specific color/size)                |
| **Move to Cart**       | Easily move wishlist item to shopping cart                           |

### User Stories

- As a customer, I want to save products for later so I can purchase them when ready
- As a customer, I want to create multiple wishlist to organize my saved item
- As a customer, I want to quickly toggle wishlist status from product pages
- As a customer, I want to move wishlist item to my cart with one click

---

## 🗃️ Entity Design

### Wishlist

```go
type Wishlist struct {
    ID        uint
    UserID    uint   // Owner of the wishlist
    Name      string // e.g., "My Wishlist", "Gift Ideas"
    IsDefault bool   // Only one can be default per user
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### WishlistItem

```go
type WishlistItem struct {
    ID         uint
    WishlistID uint // Parent wishlist
    VariantID  uint // Product variant (not product)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

---

## 🔗 API Endpoints

### Wishlist Management

| Method   | Endpoint                     | Description              | Auth        |
| -------- | ---------------------------- | ------------------------ | ----------- |
| `GET`    | `/api/wishlist`             | Get all user's wishlist | ✅ Customer |
| `POST`   | `/api/wishlist`             | Create new wishlist      | ✅ Customer |
| `GET`    | `/api/wishlist/:id`         | Get wishlist with item  | ✅ Customer |
| `PUT`    | `/api/wishlist/:id`         | Update wishlist (name)   | ✅ Customer |
| `DELETE` | `/api/wishlist/:id`         | Delete wishlist          | ✅ Customer |
| `PUT`    | `/api/wishlist/:id/default` | Set as default wishlist  | ✅ Customer |

### Wishlist Item Management

| Method   | Endpoint                                | Description                   | Auth        |
| -------- | --------------------------------------- | ----------------------------- | ----------- |
| `POST`   | `/api/wishlist/:id/item`              | Add item to wishlist          | ✅ Customer |
| `DELETE` | `/api/wishlist/:id/item/:itemId`      | Remove item from wishlist     | ✅ Customer |
| `POST`   | `/api/wishlist/:id/item/:itemId/move` | Move item to another wishlist | ✅ Customer |
| `POST`   | `/api/wishlist/:id/item/:itemId/cart` | Add item to cart              | ✅ Customer |

### Quick Actions (Uses Default Wishlist)

| Method | Endpoint                         | Description                      | Auth        |
| ------ | -------------------------------- | -------------------------------- | ----------- |
| `POST` | `/api/wishlist/toggle`           | Add/Remove from default wishlist | ✅ Customer |
| `GET`  | `/api/wishlist/check/:variantId` | Check if variant is wishlisted   | ✅ Customer |

---

## 📝 Request/Response Specifications

### 1. Get All wishlist

**`GET /api/wishlist`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Response 200:**

```json
{
  "success": true,
  "message": "wishlist retrieved successfully",
  "data": {
    "wishlist": [
      {
        "id": 1,
        "name": "My Wishlist",
        "isDefault": true,
        "itemCount": 5,
        "createdAt": "2026-01-20T10:00:00Z"
      },
      {
        "id": 2,
        "name": "Gift Ideas",
        "isDefault": false,
        "itemCount": 3,
        "createdAt": "2026-01-22T14:30:00Z"
      }
    ]
  }
}
```

---

### 2. Create Wishlist

**`POST /api/wishlist`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "name": "Birthday Gifts"
}
```

**Validation:**

- `name`: required, min=1, max=255

**Response 201:**

```json
{
  "success": true,
  "message": "Wishlist created successfully",
  "data": {
    "wishlist": {
      "id": 3,
      "name": "Birthday Gifts",
      "isDefault": false,
      "itemCount": 0,
      "createdAt": "2026-01-24T09:00:00Z"
    }
  }
}
```

---

### 3. Get Wishlist with item

**`GET /api/wishlist/:id`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Query Parameters:**
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `pageSize` | int | 20 | item per page (max 50) |

**Response 200:**

```json
{
  "success": true,
  "message": "Wishlist retrieved successfully",
  "data": {
    "wishlist": {
      "id": 1,
      "name": "My Wishlist",
      "isDefault": true,
      "itemCount": 2,
      "item": [
        {
          "id": 101,
          "variantId": 15,
          "createdAt": "2026-01-20T10:30:00Z",
          "variant": {
            "id": 15,
            "sku": "IPH15-BLK-256",
            "name": "iPhone 15 Pro - Black / 256GB",
            "priceCents": 99999,
            "originalPriceCents": 109999,
            "inStock": true,
            "stockQuantity": 25,
            "product": {
              "id": 5,
              "name": "iPhone 15 Pro",
              "slug": "iphone-15-pro",
              "imageUrl": "https://cdn.example.com/iphone-15-pro.jpg"
            }
          }
        },
        {
          "id": 102,
          "variantId": 22,
          "createdAt": "2026-01-21T15:00:00Z",
          "variant": {
            "id": 22,
            "sku": "NIKE-AIR-WHT-42",
            "name": "Nike Air Max - White / Size 42",
            "priceCents": 12999,
            "originalPriceCents": 12999,
            "inStock": false,
            "stockQuantity": 0,
            "product": {
              "id": 8,
              "name": "Nike Air Max",
              "slug": "nike-air-max",
              "imageUrl": "https://cdn.example.com/nike-air-max.jpg"
            }
          }
        }
      ]
    },
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "totalitem": 2,
      "totalPages": 1
    }
  }
}
```

---

### 4. Update Wishlist

**`PUT /api/wishlist/:id`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "name": "My Favorites"
}
```

**Validation:**

- `name`: required, min=1, max=255

**Response 200:**

```json
{
  "success": true,
  "message": "Wishlist updated successfully",
  "data": {
    "wishlist": {
      "id": 1,
      "name": "My Favorites",
      "isDefault": true,
      "itemCount": 2
    }
  }
}
```

---

### 5. Delete Wishlist

**`DELETE /api/wishlist/:id`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Response 200:**

```json
{
  "success": true,
  "message": "Wishlist deleted successfully",
  "data": null
}
```

**Response 400 (Cannot delete default):**

```json
{
  "success": false,
  "message": "Cannot delete default wishlist. Set another wishlist as default first.",
  "data": null
}
```

---

### 6. Set Default Wishlist

**`PUT /api/wishlist/:id/default`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Response 200:**

```json
{
  "success": true,
  "message": "Default wishlist updated successfully",
  "data": {
    "wishlist": {
      "id": 2,
      "name": "Gift Ideas",
      "isDefault": true
    }
  }
}
```

---

### 7. Add Item to Wishlist

**`POST /api/wishlist/:id/item`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "variantId": 15
}
```

**Validation:**

- `variantId`: required, must exist, must be active

**Response 201:**

```json
{
  "success": true,
  "message": "Item added to wishlist",
  "data": {
    "item": {
      "id": 103,
      "wishlistId": 1,
      "variantId": 15,
      "createdAt": "2026-01-24T10:00:00Z"
    }
  }
}
```

**Response 409 (Duplicate):**

```json
{
  "success": false,
  "message": "Item already in wishlist",
  "data": null
}
```

---

### 8. Remove Item from Wishlist

**`DELETE /api/wishlist/:id/item/:itemId`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Response 200:**

```json
{
  "success": true,
  "message": "Item removed from wishlist",
  "data": null
}
```

---

### 9. Move Item to Another Wishlist

**`POST /api/wishlist/:id/item/:itemId/move`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "targetWishlistId": 2
}
```

**Validation:**

- `targetWishlistId`: required, must exist, must belong to user

**Response 200:**

```json
{
  "success": true,
  "message": "Item moved successfully",
  "data": {
    "item": {
      "id": 103,
      "wishlistId": 2,
      "variantId": 15
    }
  }
}
```

**Response 409 (Already exists in target):**

```json
{
  "success": false,
  "message": "Item already exists in target wishlist",
  "data": null
}
```

---

### 10. Add Wishlist Item to Cart

**`POST /api/wishlist/:id/item/:itemId/cart`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "quantity": 1,
  "removeFromWishlist": false
}
```

**Validation:**

- `quantity`: optional (default=1), min=1
- `removeFromWishlist`: optional (default=false)

**Response 200:**

```json
{
  "success": true,
  "message": "Item added to cart",
  "data": {
    "cartItem": {
      "id": 50,
      "variantId": 15,
      "quantity": 1
    },
    "removedFromWishlist": false
  }
}
```

**Response 400 (Out of stock):**

```json
{
  "success": false,
  "message": "Item is out of stock",
  "data": null
}
```

---

### 11. Toggle Wishlist (Quick Action)

**`POST /api/wishlist/toggle`**

Uses the user's **default wishlist**. Creates one if none exists.

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
Content-Type: application/json
```

**Request:**

```json
{
  "variantId": 15
}
```

**Validation:**

- `variantId`: required, must exist

**Response 200 (Added):**

```json
{
  "success": true,
  "message": "Item added to wishlist",
  "data": {
    "wishlisted": true,
    "wishlistId": 1,
    "wishlistName": "My Wishlist",
    "itemId": 103
  }
}
```

**Response 200 (Removed):**

```json
{
  "success": true,
  "message": "Item removed from wishlist",
  "data": {
    "wishlisted": false
  }
}
```

---

### 12. Check if Variant is Wishlisted

**`GET /api/wishlist/check/:variantId`**

**Headers:**

```
Authorization: Bearer <token>
X-Correlation-ID: <uuid>
```

**Response 200 (In wishlist):**

```json
{
  "success": true,
  "data": {
    "wishlisted": true,
    "wishlistId": 1,
    "wishlistName": "My Wishlist",
    "itemId": 103
  }
}
```

**Response 200 (Not in wishlist):**

```json
{
  "success": true,
  "data": {
    "wishlisted": false
  }
}
```

---

## 📊 Business Rules

### Wishlist Rules

| Rule                      | Description                                                            |
| ------------------------- | ---------------------------------------------------------------------- |
| **Auto-create default**   | First wishlist created for user is automatically set as default        |
| **One default only**      | Setting a new default automatically removes default flag from previous |
| **Cannot delete default** | Must set another wishlist as default before deleting                   |
| **Cascade delete**        | Deleting a wishlist removes all its item                              |
| **Max wishlist**         | User can have maximum 10 wishlist                                     |

### Wishlist Item Rules

| Rule                       | Description                                             |
| -------------------------- | ------------------------------------------------------- |
| **No duplicates per list** | Same variant cannot be added twice to the same wishlist |
| **Cross-list allowed**     | Same variant can exist in multiple different wishlist  |
| **Validate variant**       | Variant must exist and be active when adding            |
| **Stock check on cart**    | Validate stock availability when moving item to cart    |
| **Max item per list**     | Maximum 100 item per wishlist                          |

### Quick Action Rules

| Rule                     | Description                                                     |
| ------------------------ | --------------------------------------------------------------- |
| **Auto-create wishlist** | If user has no wishlist, create "My Wishlist" as default        |
| **Toggle behavior**      | If item exists in default wishlist, remove it; otherwise add it |
| **Check any wishlist**   | Check endpoint returns true if item is in ANY user's wishlist   |

---

## ❌ Error Codes

| Code                      | HTTP Status | Message                             |
| ------------------------- | ----------- | ----------------------------------- |
| `WISHLIST_NOT_FOUND`      | 404         | Wishlist not found                  |
| `WISHLIST_ITEM_NOT_FOUND` | 404         | Wishlist item not found             |
| `WISHLIST_ITEM_DUPLICATE` | 409         | Item already in wishlist            |
| `CANNOT_DELETE_DEFAULT`   | 400         | Cannot delete default wishlist      |
| `MAX_wishlist_REACHED`   | 400         | Maximum number of wishlist reached |
| `MAX_item_REACHED`       | 400         | Maximum item per wishlist reached  |
| `VARIANT_NOT_FOUND`       | 404         | Product variant not found           |
| `VARIANT_NOT_ACTIVE`      | 400         | Product variant is not active       |
| `ITEM_OUT_OF_STOCK`       | 400         | Item is out of stock                |
| `UNAUTHORIZED`            | 401         | Authentication required             |
| `FORBIDDEN`               | 403         | Access denied to this wishlist      |

---

## 🗄️ Database Schema

```sql
-- Wishlist table
CREATE TABLE IF NOT EXISTS wishlist (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT 'My Wishlist',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_wishlist_user_id ON wishlist(user_id);
CREATE INDEX IF NOT EXISTS idx_wishlist_is_default ON wishlist(user_id, is_default) WHERE is_default = TRUE;

-- Wishlist item table
CREATE TABLE IF NOT EXISTS wishlist_item (
    id BIGSERIAL PRIMARY KEY,
    wishlist_id BIGINT NOT NULL REFERENCES wishlist(id) ON DELETE CASCADE,
    variant_id BIGINT NOT NULL REFERENCES product_variant(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Prevent duplicates in same wishlist
    CONSTRAINT uq_wishlist_item_variant UNIQUE (wishlist_id, variant_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_wishlist_item_wishlist_id ON wishlist_item(wishlist_id);
CREATE INDEX IF NOT EXISTS idx_wishlist_item_variant_id ON wishlist_item(variant_id);
```

---

## 🔒 Authorization

All wishlist APIs require:

1. **Valid JWT token** with `customer` role
2. **User ownership validation** - Users can only access their own wishlist
3. **X-Correlation-ID header** - Required for all requests

### Middleware Chain

```
CorrelationID → Logger → AuthMiddleware → CustomerAuth → Handler
```

---

## 📈 Future Enhancements

- [ ] Share wishlist with others (public link)
- [ ] Wishlist collaboration (shared editing)
- [ ] Price drop notifications
- [ ] Back-in-stock notifications
- [ ] Export wishlist to PDF/CSV
- [ ] Wishlist analytics (most wishlisted item)
