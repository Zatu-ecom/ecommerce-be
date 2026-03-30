# 🏠 Inventory Home Page - UI/UX Design Document

> **Purpose**: Guide frontend development for Inventory Management Dashboard  
> **Target Users**: Sellers with multiple locations  
> **Last Updated**: December 7, 2025

---

## 📋 Table of Contents

1. [User Journey Overview](#user-journey-overview)
2. [Page Structure & Layout](#page-structure--layout)
3. [Stage 1: Location List View](#stage-1-location-list-view)
4. [Stage 2: Product List View](#stage-2-product-list-view)
5. [Stage 3: Variant Details View](#stage-3-variant-details-view)
6. [API Endpoints Mapping](#api-endpoints-mapping)
7. [State Management](#state-management)
8. [Error Handling & Edge Cases](#error-handling--edge-cases)
9. [Mobile Responsiveness](#mobile-responsiveness)

---

## 🎯 User Journey Overview

### Primary Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     INVENTORY DASHBOARD                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 1: LOCATION LIST                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Warehouse A  │  │ Store NYC    │  │ Return Ctr   │          │
│  │ 245 Products │  │ 89 Products  │  │ 12 Products  │          │
│  │ ⚠️  15 Low   │  │ ✅ In Stock  │  │ ⚠️  3 Low    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │ (Click Location)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 2: PRODUCT LIST (For Selected Location)                  │
│  Location: Warehouse A                          [🔍 Search] [📊]│
│  ─────────────────────────────────────────────────────────────  │
│  📦 iPhone 15 Pro                                               │
│  │   Stock: 250 units (3 variants)                             │
│  │   Reserved: 15 | Available: 235                             │
│  └─► [▼ View Variants]                                          │
│                                                                  │
│  📦 Samsung Galaxy S24                                          │
│  │   Stock: 89 units (4 variants)                              │
│  │   Reserved: 5 | Available: 84                               │
│  │   ⚠️  2 variants low stock                                  │
│  └─► [▼ View Variants]                                          │
└─────────────────────────────────────────────────────────────────┘
                              │ (Click View Variants)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 3: VARIANT DETAILS (Expanded Dropdown)                   │
│                                                                  │
│  📦 iPhone 15 Pro                                               │
│  └─► Variants:                                                  │
│      ┌─────────────────────────────────────────────────────┐   │
│      │ 🔵 Black - 256GB                                     │   │
│      │ SKU: IPH15PRO-BLK-256                               │   │
│      │ Quantity: 120 | Reserved: 5 | Available: 115       │   │
│      │ Bin: A-12-03 | Threshold: 10                       │   │
│      │ [➕ Adjust] [📦 Transfer] [📊 History]              │   │
│      ├─────────────────────────────────────────────────────┤   │
│      │ ⚪ White - 256GB                                    │   │
│      │ SKU: IPH15PRO-WHT-256                               │   │
│      │ Quantity: 80 | Reserved: 7 | Available: 73         │   │
│      │ ⚠️  Low Stock (Threshold: 20)                      │   │
│      │ [➕ Adjust] [📦 Transfer] [📊 History]              │   │
│      ├─────────────────────────────────────────────────────┤   │
│      │ 🔵 Black - 512GB                                    │   │
│      │ SKU: IPH15PRO-BLK-512                               │   │
│      │ Quantity: 50 | Reserved: 3 | Available: 47         │   │
│      │ [➕ Adjust] [📦 Transfer] [📊 History]              │   │
│      └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🏗️ Page Structure & Layout

### Layout Type: **Three-Stage Progressive Disclosure**

This approach reduces cognitive load and allows users to drill down from high-level to detailed view.

### Global Navigation

```
┌─────────────────────────────────────────────────────────────────┐
│  [🏠 Inventory] > {Current Stage Breadcrumb}                    │
│                                          [🔔] [👤 John Seller]   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📍 Stage 1: Location List View

### Purpose

Give sellers a **bird's-eye view** of all their locations and overall inventory health.

### Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  Inventory Dashboard                                             │
│  ─────────────────────────────────────────────────────────────  │
│  Overview                                                        │
│  📊 Total Locations: 5 | Total Products: 1,245 | Low Stock: 23 │
│                                                                  │
│  [+ Add Location]                          [🔍 Search Locations]│
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ┌──────────────────────────────┐  ┌─────────────────────────┐ │
│  │ 🏢 Main Warehouse             │  │ 🏪 NYC Flagship Store   │ │
│  │ Type: WAREHOUSE               │  │ Type: STORE             │ │
│  │ ─────────────────────────────│  │ ─────────────────────────│ │
│  │ 📦 Products: 845              │  │ 📦 Products: 234        │ │
│  │ 📊 Total Stock: 12,450 units  │  │ 📊 Total Stock: 1,890   │ │
│  │ ⚠️  Low Stock Items: 15       │  │ ✅ All Items in Stock   │ │
│  │ 🔴 Out of Stock: 3            │  │ ⚠️  Low Stock Items: 5  │ │
│  │ 🎯 Active                     │  │ 🎯 Active               │ │
│  │                               │  │                         │ │
│  │ 📍 123 Warehouse Rd           │  │ 📍 456 5th Ave          │ │
│  │    Brooklyn, NY 11201         │  │    New York, NY 10001   │ │
│  │                               │  │                         │ │
│  │ [View Inventory] [⚙️ Settings]│  │ [View Inventory] [⚙️]   │ │
│  └──────────────────────────────┘  └─────────────────────────┘ │
│                                                                  │
│  ┌──────────────────────────────┐  ┌─────────────────────────┐ │
│  │ 📦 Returns Center             │  │ 🏪 LA Store             │ │
│  │ Type: RETURN_CENTER           │  │ Type: STORE             │ │
│  │ ─────────────────────────────│  │ ─────────────────────────│ │
│  │ 📦 Products: 45               │  │ 📦 Products: 189        │ │
│  │ 📊 Total Stock: 234 units     │  │ 📊 Total Stock: 2,100   │ │
│  │ 🟡 Pending Inspection: 12     │  │ ✅ All Items in Stock   │ │
│  │ 🎯 Active                     │  │ 🎯 Active               │ │
│  │                               │  │                         │ │
│  │ 📍 789 Return Ave             │  │ 📍 321 Sunset Blvd      │ │
│  │    Queens, NY 11375           │  │    Los Angeles, CA 90028│ │
│  │                               │  │                         │ │
│  │ [View Inventory] [⚙️ Settings]│  │ [View Inventory] [⚙️]   │ │
│  └──────────────────────────────┘  └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Data to Display (Per Location Card)

| Field                  | Source                             | Display Logic                                      |
| ---------------------- | ---------------------------------- | -------------------------------------------------- |
| **Location Name**      | `location.name`                    | Always show                                        |
| **Type**               | `location.type`                    | Icon + Text (🏢 WAREHOUSE, 🏪 STORE, 📦 RETURN)    |
| **Status**             | `location.isActive`                | 🎯 Active / 🚫 Inactive                            |
| **Product Count**      | Aggregate from inventories         | Count distinct `variantId` for this location       |
| **Total Stock**        | Sum of `quantity`                  | `SUM(inventory.quantity)` for location             |
| **Low Stock Count**    | Count where `quantity < threshold` | Red badge if > 0                                   |
| **Out of Stock Count** | Count where `quantity = 0`         | Red badge if > 0                                   |
| **Address**            | `location.address`                 | Format: Street, City, State ZIP                    |
| **Priority**           | `location.priority`                | Sort locations by this (optional visual indicator) |

### API Endpoint

```
GET /api/inventory/locations/summary

Response:
{
  "success": true,
  "data": {
    "overview": {
      "totalLocations": 5,
      "totalProducts": 1245,
      "totalStock": 18674,
      "lowStockCount": 23,
      "outOfStockCount": 8
    },
    "locations": [
      {
        "id": 1,
        "name": "Main Warehouse",
        "type": "WAREHOUSE",
        "isActive": true,
        "priority": 1,
        "address": {
          "street": "123 Warehouse Rd",
          "city": "Brooklyn",
          "state": "NY",
          "zipCode": "11201",
          "country": "USA"
        },
        "inventorySummary": {
          "productCount": 845,
          "totalStock": 12450,
          "lowStockCount": 15,
          "outOfStockCount": 3
        }
      },
      // ... more locations
    ]
  }
}
```

### User Actions

- **Click "View Inventory"** → Navigate to Stage 2 (Product List for that location)
- **Click "⚙️ Settings"** → Open location settings modal (edit address, priority, etc.)
- **Click "+ Add Location"** → Open "Add New Location" form
- **Search** → Filter locations by name/type

---

## 📦 Stage 2: Product List View

### Purpose

Show all products at the selected location with aggregated inventory data.

### Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  [🏠 Inventory] > Main Warehouse                                │
│  ─────────────────────────────────────────────────────────────  │
│  📍 Main Warehouse (123 Warehouse Rd, Brooklyn, NY)             │
│  [← Back to Locations]                          [📊 Export] [+] │
│                                                                  │
│  [🔍 Search Products]  [Filter: All | Low Stock | Out of Stock] │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 📱 iPhone 15 Pro                           [▼ View Variants]│  │
│  │ Category: Electronics > Smartphones                       │  │
│  │ ─────────────────────────────────────────────────────────│  │
│  │ Total Stock: 250 units (across 3 variants)               │  │
│  │ Reserved: 15 | Available: 235                            │  │
│  │ ✅ All variants in stock                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 📱 Samsung Galaxy S24                      [▼ View Variants]│  │
│  │ Category: Electronics > Smartphones                       │  │
│  │ ─────────────────────────────────────────────────────────│  │
│  │ Total Stock: 89 units (across 4 variants)                │  │
│  │ Reserved: 5 | Available: 84                              │  │
│  │ ⚠️  2 variants low stock                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 💻 MacBook Pro 14"                         [▼ View Variants]│  │
│  │ Category: Electronics > Laptops                           │  │
│  │ ─────────────────────────────────────────────────────────│  │
│  │ Total Stock: 45 units (across 2 variants)                │  │
│  │ Reserved: 8 | Available: 37                              │  │
│  │ ✅ All variants in stock                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 👕 Nike Air Max Sneakers                   [▼ View Variants]│  │
│  │ Category: Fashion > Footwear                              │  │
│  │ ─────────────────────────────────────────────────────────│  │
│  │ Total Stock: 12 units (across 8 variants)                │  │
│  │ Reserved: 0 | Available: 12                              │  │
│  │ 🔴 5 variants out of stock | ⚠️  2 low stock             │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  [Load More] or [Pagination: 1 2 3 ... 42]                      │
└─────────────────────────────────────────────────────────────────┘
```

### Data to Display (Per Product Row)

| Field                  | Source                         | Display Logic                              |
| ---------------------- | ------------------------------ | ------------------------------------------ |
| **Product Name**       | `product.name`                 | Always show with product icon/image        |
| **Category**           | `category.name` (breadcrumb)   | Full category path                         |
| **Total Stock**        | Sum of all variant quantities  | `SUM(inventory.quantity)` for all variants |
| **Variant Count**      | Count of variants              | "across X variants"                        |
| **Reserved**           | Sum of `reservedQuantity`      | `SUM(inventory.reservedQuantity)`          |
| **Available**          | Total - Reserved               | `totalStock - totalReserved`               |
| **Low Stock Alert**    | Count variants below threshold | ⚠️ badge if any variant is low             |
| **Out of Stock Alert** | Count variants with 0 quantity | 🔴 badge if any variant is out             |
| **Status Indicator**   | Overall health                 | ✅ Good / ⚠️ Warning / 🔴 Critical         |

### API Endpoint

```
GET /api/inventory/locations/{locationId}/products?page=1&pageSize=20&filter=all

Response:
{
  "success": true,
  "data": {
    "locationInfo": {
      "id": 1,
      "name": "Main Warehouse",
      "address": "123 Warehouse Rd, Brooklyn, NY"
    },
    "products": [
      {
        "productId": 10,
        "productName": "iPhone 15 Pro",
        "category": "Electronics > Smartphones",
        "totalStock": 250,
        "variantCount": 3,
        "reserved": 15,
        "available": 235,
        "lowStockVariants": 0,
        "outOfStockVariants": 0,
        "status": "GOOD" // GOOD | WARNING | CRITICAL
      },
      // ... more products
    ],
    "pagination": {
      "currentPage": 1,
      "pageSize": 20,
      "totalPages": 42,
      "totalItems": 845
    }
  }
}
```

### User Actions

- **Click "▼ View Variants"** → Expand dropdown to show Stage 3 (Variant Details)
- **Click "← Back to Locations"** → Return to Stage 1
- **Click "📊 Export"** → Download inventory report (CSV/Excel)
- **Search** → Filter products by name
- **Filter** → Show only low stock or out of stock items

---

## 🎨 Stage 3: Variant Details View

### Purpose

Show detailed inventory information for each variant within a product (expanded dropdown/accordion).

### Layout (Expanded Under Product)

```
┌──────────────────────────────────────────────────────────────┐
│ 📱 iPhone 15 Pro                              [▲ Hide Variants]│
│ Category: Electronics > Smartphones                           │
│ ─────────────────────────────────────────────────────────────│
│ Total Stock: 250 units (across 3 variants)                   │
│ Reserved: 15 | Available: 235                                │
│                                                               │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Variant 1/3                                              │ │
│ │ ┌────────┐  🔵 Black - 256GB                            │ │
│ │ │ [IMG]  │  SKU: IPH15PRO-BLK-256                       │ │
│ │ └────────┘  Price: $999.99                              │ │
│ │                                                          │ │
│ │ 📊 Stock Details:                                        │ │
│ │ ┌──────────────┬──────────────┬──────────────┐          │ │
│ │ │ Total: 120   │ Reserved: 5  │ Available: 115 │         │ │
│ │ └──────────────┴──────────────┴──────────────┘          │ │
│ │                                                          │ │
│ │ 📍 Bin Location: A-12-03                                │ │
│ │ ⚠️  Threshold: 10 (Currently: 120 - Well Stocked)       │ │
│ │                                                          │ │
│ │ 📈 Recent Activity:                                     │ │
│ │ • Dec 6: +50 units (Purchase Order #1234)               │ │
│ │ • Dec 5: -25 units (Sales Order #5678)                  │ │
│ │                                                          │ │
│ │ [➕ Adjust Stock] [📦 Transfer] [📊 Full History]       │ │
│ │ [✏️ Edit Bin] [🔔 Set Alert]                            │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ Variant 2/3                                              │ │
│ │ ┌────────┐  ⚪ White - 256GB                           │ │
│ │ │ [IMG]  │  SKU: IPH15PRO-WHT-256                       │ │
│ │ └────────┘  Price: $999.99                              │ │
│ │                                                          │ │
│ │ 📊 Stock Details:                                        │ │
│ │ ┌──────────────┬──────────────┬──────────────┐          │ │
│ │ │ Total: 80    │ Reserved: 7  │ Available: 73  │         │ │
│ │ └──────────────┴──────────────┴──────────────┘          │ │
│ │                                                          │ │
│ │ 📍 Bin Location: A-12-04                                │ │
│ │ ⚠️  LOW STOCK! Threshold: 20 (Currently: 80)           │ │
│ │                                                          │ │
│ │ 📈 Recent Activity:                                     │ │
│ │ • Dec 6: -15 units (Sales Order #5680)                  │ │
│ │ • Dec 4: +30 units (Purchase Order #1230)               │ │
│ │                                                          │ │
│ │ [➕ Adjust Stock] [📦 Transfer] [📊 Full History]       │ │
│ │ [✏️ Edit Bin] [🔔 Set Alert]                            │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ Variant 3/3                                              │ │
│ │ ┌────────┐  🔵 Black - 512GB                            │ │
│ │ │ [IMG]  │  SKU: IPH15PRO-BLK-512                       │ │
│ │ └────────┘  Price: $1,199.99                            │ │
│ │                                                          │ │
│ │ 📊 Stock Details:                                        │ │
│ │ ┌──────────────┬──────────────┬──────────────┐          │ │
│ │ │ Total: 50    │ Reserved: 3  │ Available: 47  │         │ │
│ │ └──────────────┴──────────────┴──────────────┘          │ │
│ │                                                          │ │
│ │ 📍 Bin Location: A-12-05                                │ │
│ │ ✅ Threshold: 10 (Currently: 50 - In Stock)            │ │
│ │                                                          │ │
│ │ [➕ Adjust Stock] [📦 Transfer] [📊 Full History]       │ │
│ │ [✏️ Edit Bin] [🔔 Set Alert]                            │ │
│ └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### Data to Display (Per Variant)

| Field                   | Source                        | Display Logic                                    |
| ----------------------- | ----------------------------- | ------------------------------------------------ |
| **Variant Name**        | Variant options (Color, Size) | Concatenate option values with product name      |
| **SKU**                 | `variant.sku`                 | Always show                                      |
| **Price**               | `variant.price`               | Formatted currency                               |
| **Image**               | `variant.image` or product    | Thumbnail (120x120px)                            |
| **Total Quantity**      | `inventory.quantity`          | Current stock level                              |
| **Reserved Quantity**   | `inventory.reservedQuantity`  | Stock allocated for orders                       |
| **Available Quantity**  | `quantity - reservedQuantity` | What can be sold now                             |
| **Threshold**           | `inventory.threshold`         | Low stock alert level                            |
| **Bin Location**        | `inventory.binLocation`       | Physical location in warehouse (e.g., "A-12-03") |
| **Stock Status**        | Calculated                    | ✅ In Stock / ⚠️ Low / 🔴 Out / 🔙 Backorder     |
| **Recent Transactions** | Last 3 transactions           | Date, type, quantity, reference                  |

### API Endpoint

```
GET /api/inventory/product/{productId}/variant?locationId=1

Response:
{
  "success": true,
  "data": {
    "productInfo": {
      "id": 10,
      "name": "iPhone 15 Pro",
      "category": "Electronics > Smartphones"
    },
    "variants": [
      {
        "variantId": 55,
        "variantName": "Black - 256GB",
        "sku": "IPH15PRO-BLK-256",
        "price": 999.99,
        "imageUrl": "https://cdn.example.com/...",
        "inventory": {
          "id": 101,
          "locationId": 1,
          "quantity": 120,
          "reservedQuantity": 5,
          "availableQuantity": 115,
          "threshold": 10,
          "binLocation": "A-12-03",
          "status": "IN_STOCK"
        },
        "recentTransactions": [
          {
            "date": "2025-12-06",
            "type": "PURCHASE_ORDER",
            "quantity": 50,
            "reference": "PO-1234",
            "note": "Restocking"
          },
          {
            "date": "2025-12-05",
            "type": "SALES_ORDER",
            "quantity": -25,
            "reference": "SO-5678",
            "note": "Order fulfillment"
          }
        ]
      },
      // ... more variants
    ]
  }
}
```

### User Actions

- **Click "➕ Adjust Stock"** → Open modal to add/subtract inventory (with reason)
- **Click "📦 Transfer"** → Transfer stock to another location
- **Click "📊 Full History"** → View all transactions for this variant
- **Click "✏️ Edit Bin"** → Update bin location
- **Click "🔔 Set Alert"** → Configure low stock notifications
- **Click "▲ Hide Variants"** → Collapse variant details

---

## 🔗 API Endpoints Mapping

### Summary of Required Endpoints

| Endpoint                                     | Purpose                            | Stage |
| -------------------------------------------- | ---------------------------------- | ----- |
| `GET /api/inventory/locations/summary`       | Location list with inventory stats | 1     |
| `GET /api/inventory/locations/{id}/products` | Product list for a location        | 2     |
| `GET /api/inventory/product/{id}/variant`    | Variant details with inventory     | 3     |
| `POST /api/inventory/adjustments`            | Adjust stock quantity              | 3     |
| `POST /api/inventory/transfers`              | Transfer stock between locations   | 3     |
| `GET /api/inventory/transactions`            | Transaction history                | 3     |
| `PATCH /api/inventory/{id}`                  | Update bin location/threshold      | 3     |

---

## 💾 State Management

### Frontend State Structure (React/Vue Example)

```javascript
// Global inventory state
const inventoryState = {
  // Stage 1: Locations
  locations: {
    data: [],
    loading: false,
    error: null,
    overview: {
      totalLocations: 0,
      totalProducts: 0,
      lowStockCount: 0,
    },
  },

  // Stage 2: Products
  selectedLocation: null,
  products: {
    data: [],
    loading: false,
    error: null,
    pagination: { currentPage: 1, totalPages: 1 },
  },

  // Stage 3: Variants (expanded product ID)
  expandedProductId: null,
  variants: {
    data: [],
    loading: false,
    error: null,
  },

  // Filters & Search
  filters: {
    searchQuery: "",
    stockStatus: "all", // all | low | out
    sortBy: "name", // name | stock | priority
  },
};
```

---

## ❌ Error Handling & Edge Cases

### Empty States

#### No Locations

```
┌─────────────────────────────────────────┐
│  📍 No Locations Found                   │
│                                          │
│  You haven't set up any inventory        │
│  locations yet. Add your first location  │
│  to start tracking inventory.            │
│                                          │
│  [+ Add Your First Location]             │
└─────────────────────────────────────────┘
```

#### No Products at Location

```
┌─────────────────────────────────────────┐
│  📦 No Products at This Location         │
│                                          │
│  This location doesn't have any          │
│  inventory yet. Transfer stock or add    │
│  new products to get started.            │
│                                          │
│  [📦 Transfer Stock] [+ Add Product]     │
└─────────────────────────────────────────┘
```

#### No Variants (Edge Case - Shouldn't Happen)

```
┌─────────────────────────────────────────┐
│  ⚠️  Product Configuration Error         │
│                                          │
│  This product has no variants. Please    │
│  contact support or add variants in      │
│  Product Management.                     │
│                                          │
│  [Go to Product Management]              │
└─────────────────────────────────────────┘
```

### Error States

#### API Error

```
┌─────────────────────────────────────────┐
│  ❌ Failed to Load Data                  │
│                                          │
│  Something went wrong. Please try again. │
│  If the problem persists, contact        │
│  support with error code: INV-500-1234   │
│                                          │
│  [🔄 Retry]                               │
└─────────────────────────────────────────┘
```

#### Low Stock Warning

```
┌─────────────────────────────────────────┐
│  ⚠️  Low Stock Alert                     │
│                                          │
│  23 products across all locations are    │
│  running low. Review and restock soon.   │
│                                          │
│  [View Low Stock Items]                  │
└─────────────────────────────────────────┘
```

### Edge Cases

- **Negative Stock** (Backorder): Display with 🔙 icon and "Backorder: -5 units"
- **High Reserved Quantity**: Show warning if reserved > 80% of total
- **Inactive Location**: Gray out and show "🚫 Inactive" badge
- **Missing Bin Location**: Show "Not assigned" with option to set

---

## 📱 Mobile Responsiveness

### Stage 1 (Mobile): Stack Cards Vertically

```
┌───────────────────────────┐
│  🏠 Inventory              │
│  [☰ Menu]      [🔍 Search]│
├───────────────────────────┤
│ Overview                  │
│ 5 Locations | 1,245 Items │
│ ⚠️  23 Low Stock           │
├───────────────────────────┤
│ 🏢 Main Warehouse         │
│ WAREHOUSE | Active        │
│                           │
│ 845 Products | 12.4K Stock│
│ ⚠️  15 Low | 🔴 3 Out     │
│                           │
│ [View] [⚙️]                │
├───────────────────────────┤
│ 🏪 NYC Store              │
│ STORE | Active            │
│                           │
│ 234 Products | 1.9K Stock │
│ ✅ In Stock               │
│                           │
│ [View] [⚙️]                │
└───────────────────────────┘
```

### Stage 2 & 3 (Mobile): Accordion Style

- Use collapsible accordions for products
- Swipe gestures to navigate
- Bottom sheet for actions (Adjust, Transfer)

---

## 🎨 Design Recommendations

### Color Scheme

- **Primary**: Blue (#2563EB) - Actions, links
- **Success**: Green (#10B981) - In stock, positive actions
- **Warning**: Orange (#F59E0B) - Low stock alerts
- **Error**: Red (#EF4444) - Out of stock, critical alerts
- **Neutral**: Gray (#6B7280) - Text, borders

### Typography

- **Headings**: 24px, 20px, 18px (Bold)
- **Body**: 16px (Regular)
- **Small Text**: 14px (Metadata, SKU)
- **Captions**: 12px (Hints, helper text)

### Icons

- 🏢 Warehouse, 🏪 Store, 📦 Returns
- ⚠️ Warning, 🔴 Error, ✅ Success
- 📊 Charts, 📦 Products, 📍 Location
- ➕ Add, ✏️ Edit, 🔍 Search

### Spacing

- Card padding: 20px
- Section gap: 16px
- Element margin: 8px

---

## ✅ Summary of Your Initial Approach

### ✅ What You Got Right

1. **Three-stage drill-down** (Location → Product → Variant) - Perfect! ✅
2. **Location list first** - Great starting point ✅
3. **Dropdown for variants** - Good UX, keeps page clean ✅

### 🔄 Recommendations

| Your Idea                   | Recommendation                               |
| --------------------------- | -------------------------------------------- |
| "Show product count"        | ✅ YES + add stock count & alerts            |
| "Variants in dropdown"      | ✅ YES + add actions (Adjust, Transfer)      |
| "Click location → products" | ✅ YES + add filters & search                |
| Not mentioned               | 💡 Add overview dashboard at Stage 1         |
| Not mentioned               | 💡 Show recent transactions in Stage 3       |
| Not mentioned               | 💡 Add bin location for warehouse management |

### Key Additions to Your Design

1. **Overview Stats** at Stage 1 (total products, low stock alerts)
2. **Status Indicators** (✅ ⚠️ 🔴) for quick health checks
3. **Reserved vs Available** distinction (important for order management)
4. **Recent Activity** in variant view (context for inventory changes)
5. **Quick Actions** (Adjust, Transfer, History) at variant level
6. **Bin Location** field for warehouse organization

---

## 🚀 Next Steps

1. **Backend**: Implement the three summary endpoints listed above
2. **Frontend**: Build Stage 1 (Location List) first
3. **Testing**: Test with real data (100+ products, multiple locations)
4. **Feedback**: Get seller feedback on Stage 1 before building Stage 2 & 3
5. **Iteration**: Add filters, search, and advanced features based on usage

---

## 📞 Questions to Clarify

1. **Do you want real-time updates** (WebSocket) when inventory changes?
2. **Bulk actions** - Should sellers be able to adjust multiple variants at once?
3. **Export formats** - CSV, Excel, or PDF for reports?
4. **Mobile app** - Will this be mobile-first or desktop-primary?
5. **Permissions** - Can all sellers adjust inventory, or only certain roles?

---

**Your initial design was solid! The three-stage approach is exactly right. The enhancements above add professional polish and essential data points that sellers need for effective inventory management.** 🎯
