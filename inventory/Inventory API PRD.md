# Inventory Management API PRD

## 1. Overview

This document defines the API specifications for the Inventory Management module. It covers multi-location support, inventory adjustments, and stock transfers.

**Base URL**: `/api` **Authentication**: Required (Seller/Admin) for all endpoints. **Response Format**: Standard JSON response (see 

common/response.go).

---

## 2. Common Data Structures

### Location Object

```json
{
  "id": 1,
  "name": "Main Warehouse",
  "type": "WAREHOUSE",
  "isActive": true,
  "priority": 1,
  "address": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "zipCode": "10001",
    "country": "USA"
  }
}

```
### Inventory Object

```json
{
  "id": 101,
  "variantId": 55,
  "locationId": 1,
  "quantity": 100,
  "reservedQuantity": 5,
  "availableQuantity": 95,
  "lowStockThreshold": 10
}
```

---

## 3. Location Management APIs

### 3.1 Create Location

**Endpoint**: `POST /inventory/locations` **Description**: Create a new physical location for inventory storage.

**Request Body**:

```json
{
  "name": "Downtown Store",
  "type": "STORE", // Enum: WAREHOUSE, STORE, RETURN_CENTER
  "priority": 2,
  "address": {
    "street": "456 Market St",
    "city": "San Francisco",
    "state": "CA",
    "zipCode": "94103",
    "country": "USA"
  }
}
```

**Validation**:

- `name`: Required, Min 3 chars.
- `type`: Required, Must be one of `WAREHOUSE`, `STORE`, `RETURN_CENTER`.
- `address`: Required (all fields inside).

**Response (201 Created)**:

```json
{
  "success": true,
  "message": "Location created successfully",
  "data": { "id": 2, ... } // Location Object
}
```

### 3.2 List Locations

**Endpoint**: `GET /inventory/locations` **Description**: Get all locations for the seller.

**Query Params**:

- `isActive`: `true` (default) or `false`

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Locations retrieved successfully",
  "data": [
    { "id": 1, "name": "Main Warehouse", ... },
    { "id": 2, "name": "Downtown Store", ... }
  ]
}

```
### 3.3 Update Location

**Endpoint**: `PUT /inventory/locations/:id` **Description**: Update location details (Name, Priority, Address).

**Request Body**:

```json
{
  "name": "Downtown Flagship Store",
  "priority": 1
}
```

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Location updated successfully",
  "data": { ... }
}
```

---

## 4. Inventory Management APIs

### 4.1 Get Inventory for Variant

**Endpoint**: `GET /inventory/products/:variantId` **Description**: Get stock levels for a specific product variant across all locations.

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Inventory retrieved successfully",
  "data": {
    "variantId": 55,
    "totalAvailable": 150,
    "locations": [
      {
        "locationId": 1,
        "locationName": "Main Warehouse",
        "quantity": 100,
        "reserved": 5,
        "available": 95
      },
      {
        "locationId": 2,
        "locationName": "Downtown Store",
        "quantity": 55,
        "reserved": 0,
        "available": 55
      }
    ]
  }
}
```

### 4.2 Adjust Inventory

**Endpoint**: `POST /inventory/adjust` **Description**: Manually adjust stock levels (e.g., new stock arrival, stock correction, damage).

**Request Body**:

```json
{
  "variantId": 55,
  "locationId": 1,
  "quantity": 10,
  "type": "ADD", // Enum: ADD, REMOVE, SET
  "reason": "New shipment received",
  "reference": "PO-12345" // Optional
}
```

**Validation**:

- `variantId`, `locationId`: Required.
- `quantity`: Required, Must be > 0.
- `type`: Required, Must be `ADD`, `REMOVE`, or `SET`.
- `reason`: Required, Min 5 chars.

**Response (200 OK)**:

```json
{
  "success": true,
  "message": "Inventory adjusted successfully",
  "data": {
    "inventoryId": 101,
    "previousQuantity": 100,
    "newQuantity": 110
  }
}
```

---

## 5. Stock Transfer APIs

### 5.1 Create Transfer Request

**Endpoint**: `POST /inventory/transfers` **Description**: Initiate a stock movement from one location to another.

**Request Body**:

```json
{
  "sourceLocationId": 1,
  "destinationLocationId": 2,
  "items": [
    { "variantId": 55, "quantity": 20 },
    { "variantId": 56, "quantity": 5 }
  ],
  "notes": "Restocking downtown store for weekend sale"
}
```

**Validation**:

- `sourceLocationId`, `destinationLocationId`: Required, Must be different.
- `items`: Required, Non-empty list.
- `quantity`: Must be <= Available stock at Source.

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Transfer created successfully",
  "data": {
    "id": 501,
    "status": "PENDING",
    "referenceNumber": "TRF-2025-001"
  }
}
```

### 5.2 Update Transfer Status

**Endpoint**: `PUT /inventory/transfers/:id/status` **Description**: Move transfer through its lifecycle (PENDING -> SHIPPED -> RECEIVED).

**Request Body**:

```json
{
  "status": "SHIPPED", // Enum: SHIPPED, RECEIVED, CANCELLED
  "notes": "Picked up by courier"
}
```

**Logic**:

- **SHIPPED**: Deducts stock from Source (if not already reserved).
- **RECEIVED**: Adds stock to Destination.
- **CANCELLED**: Releases reserved stock at Source.

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Transfer status updated to SHIPPED",
  "data": { "id": 501, "status": "SHIPPED", ... }
}
```


---

## 6. Validation Rules

### 6.1 Location Validations

- **Name**:
    - Required.
    - Min length: 3 characters.
    - Max length: 255 characters.
    - Must be unique per seller (case-insensitive).
- **Type**:
    - Required.
    - Must be one of: `WAREHOUSE`, `STORE`, `RETURN_CENTER`.
- **Priority**:
    - Optional (default: 0).
    - Must be >= 0.
- **Address**:
    - Required.
    - `street`: Required, min 5 chars.
    - `city`: Required, min 2 chars.
    - `state`: Required, min 2 chars.
    - `zipCode`: Required, valid format check (regex).
    - `country`: Required, min 2 chars.

### 6.2 Inventory Adjustment Validations

- **VariantID**:
    - Required.
    - Must exist in `product_variants` table.
- **LocationID**:
    - Required.
    - Must exist in `locations` table.
    - Location must be `isActive: true`.
- **Quantity**:
    - Required.
    - Must be > 0.
    - For `REMOVE` type: Must be <= `AvailableQuantity` (cannot remove more than what's available).
- **Type**:
    - Required.
    - Must be one of: `ADD`, `REMOVE`, `SET`.
- **Reason**:
    - Required.
    - Min length: 5 characters (e.g., "Stock count correction", "Damaged goods").

### 6.3 Stock Transfer Validations

- **SourceLocationID**:
    - Required.
    - Must exist and be active.
- **DestinationLocationID**:
    - Required.
    - Must exist and be active.
    - **Constraint**: `SourceLocationID` != `DestinationLocationID`.
- **Items**:
    - Required.
    - Must contain at least 1 item.
    - **Duplicate Check**: Same `variantId` cannot appear twice in the list.
- **Item Quantity**:
    - Must be > 0.
    - **Stock Check**: Source location must have `AvailableQuantity` >= `TransferQuantity` for each item.
- **Status Transitions**:
    - `PENDING` -> `SHIPPED`: Allowed.
    - `SHIPPED` -> `RECEIVED`: Allowed.
    - `PENDING` -> `CANCELLED`: Allowed.
    - `SHIPPED` -> `CANCELLED`: Allowed (requires restocking logic).
    - `RECEIVED` -> `CANCELLED`: **Not Allowed** (transfer is final).

---

## 7. Error Codes

| Code                 | Message                     | Description                                                             |
| -------------------- | --------------------------- | ----------------------------------------------------------------------- |
| `LOC_NOT_FOUND`      | Location not found          | The specified location ID does not exist.                               |
| `VAR_NOT_FOUND`      | Variant not found           | The specified product variant ID does not exist.                        |
| `INSUFFICIENT_STOCK` | Insufficient stock          | Source location does not have enough available stock.                   |
| `INVALID_TRANSITION` | Invalid status transition   | The requested status change is not allowed (e.g., RECEIVED -> PENDING). |
| `DUPLICATE_LOC_NAME` | Location name exists        | A location with this name already exists for the seller.                |
| `SAME_SRC_DEST`      | Source and Destination same | Cannot transfer stock to the same location.                             |