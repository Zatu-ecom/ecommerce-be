# Product & File Module Integration Pre-Spec (v2)

## 1. Overview
This document outlines the architectural decisions, database models, and API definitions for linking the newly developed **File Module** with the **Product Module**. It follows the approved **Hybrid DTO Approach**, returning domain-specific structures to the frontend while leveraging the internal File APIs for file metadata and access URLs. This approach natively supports both product images and videos by linking to their respective file IDs in the File Module.

## 2. Database Design & Migrations

### 2.1 Table Schema (Migration Logic)
The `product_media` table links products to files stored in the File Module (images, videos, etc.). We do not use hard foreign keys to the file module tables in order to preserve strict modular boundaries.

**File:** `db/migrations/000X_create_product_media_table.up.sql`
```sql
CREATE TABLE product_media (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    file_id VARCHAR(36) NOT NULL, -- UUIDv7 referencing File Module
    is_primary BOOLEAN DEFAULT FALSE,
    display_order INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_product_media_product FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE,
    -- Prevent the same file from being attached to the same product multiple times
    CONSTRAINT uq_product_media_product_file UNIQUE (product_id, file_id)
);

CREATE INDEX idx_product_media_product_id ON product_media(product_id);
```

**File:** `db/migrations/000X_create_product_media_table.down.sql`
```sql
DROP TABLE IF EXISTS product_media;
```

### 2.2 GORM Models
The corresponding Go model resides inside the Product module repository/domain layer.

```go
// product_media.go (inside product module)
type ProductMedia struct {
    ID           int64     `gorm:"primaryKey;autoIncrement"`
    ProductID    int64     `gorm:"not null;index"`
    FileID       string    `gorm:"type:varchar(36);not null"`
    IsPrimary    bool      `gorm:"default:false"`
    DisplayOrder int       `gorm:"default:0"`
    CreatedAt    time.Time `gorm:"autoCreateTime"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime"`

    // Product Product `gorm:"foreignKey:ProductID"` // if needed for joins
}
```

## 3. DTO Models (Hybrid Approach)

We map the generic `FileItem` from the File Module into a domain-specific `ProductMediaDTO`. This keeps the API response clean and relevant to storefronts/admins.

```go
// dto/product_response.go

type ProductMediaDTO struct {
    FileID       string `json:"fileId"`
    URL          string `json:"url"`                     // Main high-res image or video stream URL
    ThumbnailURL string `json:"thumbnailUrl,omitempty"`  // Auto-populated if variant exists (e.g. video poster)
    IsPrimary    bool   `json:"isPrimary"`
    DisplayOrder int    `json:"displayOrder"`
}

type ProductResponseDTO struct {
    ID          int64             `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Price       float64           `json:"price"`
    Media       []ProductMediaDTO `json:"media"`
}
```

## 4. Service Logic & Inter-Module Communication

### 4.1 Dependency Injection
The Product Service will require interfaces for interacting with the File Module. To maintain decoupling, these interfaces will be defined inside the Product Module but implemented/satisfied by the actual File Services during application wiring.

```go
// service/product_service.go
type ProductService struct {
    productRepo    repository.ProductRepository
    fileReadAPI    file.ReadAPI     // Interface mapped to FileReadService
    fileDeleteAPI  file.DeleteAPI   // Interface mapped to FileDeleteService
}
```

### 4.2 Fetching Products with Media
When retrieving a product, the service layer maps `ProductMedia` database rows to `ProductMediaDTO` objects by querying the internal File API.

```go
// Example logic snippet inside ProductService.GetProduct(ctx, productID)

// 1. Fetch product and its media relations from DB
product, mediaItems, err := s.productRepo.GetProductWithMedia(ctx, productID)

// 2. Extract File IDs
var fileIDs []string
for _, item := range mediaItems {
    fileIDs = append(fileIDs, item.FileID)
}

// 3. Fetch full file details from File Module
filesResult, err := s.fileReadAPI.GetAllFiles(ctx, principal, model.GetFilesFilter{
    FileIDs:            fileIDs,
    IncludeDownloadURL: true,
    IncludeVariants:    true, // E.g., for thumbnails/posters
})

// 4. Create O(1) lookup map for easy mapping
fileMap := make(map[string]*file_model.FileItem)
for _, f := range filesResult.Files {
    fileMap[f.FileID] = f
}

// 5. Build DTOs
var mediaDTOs []ProductMediaDTO
for _, item := range mediaItems {
    if fileData, exists := fileMap[item.FileID]; exists {
        
        // Extract Thumbnail URL if variants exist (fallback to original URL)
        thumbURL := fileData.DownloadUrl
        for _, v := range fileData.Variants {
            if v.VariantCode == "thumb_200" || v.VariantCode == "poster" { // Convention for thumbnail/poster variant
                thumbURL = v.DownloadUrl
                break
            }
        }

        mediaDTOs = append(mediaDTOs, ProductMediaDTO{
            FileID:       item.FileID,
            URL:          fileData.DownloadUrl,
            ThumbnailURL: thumbURL,
            IsPrimary:    item.IsPrimary,
            DisplayOrder: item.DisplayOrder,
        })
    }
}
```
*Note: For the List (`GET /products`) endpoint, the logic will be similar but batched. All `fileIDs` across the entire page of products will be collected and fetched in a single internal API call to avoid N+1 query problems.*

### 4.3 Handling Detachment & Deletion
**Decision:** When a media file is detached/removed from a product via the Product API, we will synchronously call the File Module to hard-delete the underlying file.

```go
// Example logic for removing media from a product
func (s *ProductService) RemoveProductMedia(ctx context.Context, principal Principal, productID int64, fileID string) error {
    // 1. Delete the linking record from product_media table
    err := s.productRepo.DeleteProductMedia(ctx, productID, fileID)
    if err != nil {
        return err
    }

    // 2. Synchronously instruct File Module to delete the file blob and its metadata
    err = s.fileDeleteAPI.DeleteFile(ctx, principal, fileID)
    if err != nil {
        // If deletion fails, we log it. The product mapping is gone, so the file becomes 
        // orphaned. A background job can sweep orphaned files if strict consistency is needed.
        log.Errorf("Failed to delete file %s from file module: %v", fileID, err)
    }

    return nil
}
```

## 5. Detailed API Endpoint Logic

This section defines the strict step-by-step logic for each API endpoint to ensure no mistakes are made during implementation.

### 5.1 `POST /api/products/{id}/media` (Link Media)
**Purpose:** Associates an already uploaded file (from the File Module) with a Product. It does *not* accept multipart form data; it expects a JSON payload containing the `fileId`.

**Request Payload:**
```json
{
  "fileId": "018e6b...",
  "isPrimary": true,
  "displayOrder": 1
}
```

**Step-by-Step Logic:**
1. **Validation:** Ensure `fileId` is a valid format. Ensure the `productId` exists in the database.
2. **File Verification:** Call the internal File Module `fileReadAPI.GetFile(ctx, principal, fileId)` to confirm the file actually exists and belongs to this tenant/user. If it returns an error/404, return `400 Bad Request: Invalid fileId`.
3. **Primary Reset Logic:** If the payload specifies `isPrimary: true`, first execute an update query to unset primary status on all existing media for this product: `UPDATE product_media SET is_primary = false WHERE product_id = ?`.
4. **Insert Mapping:** Insert the new row into the `product_media` table. Handle the `UNIQUE (product_id, file_id)` constraint; if it triggers, return `409 Conflict`.
5. **Response:** Return `201 Created` with the newly linked `ProductMediaDTO`.

---

### 5.2 `GET /api/products/{id}` (Read Product)
**Purpose:** Retrieves a single product and embeds its associated media data.

**Step-by-Step Logic:**
1. **Fetch Product:** Fetch the product from the `product` table. If not found, return `404`.
2. **Fetch Media Mappings:** Fetch associated media rows from `product_media` where `product_id = ?` ORDER BY `display_order ASC`.
3. **Extract IDs:** Extract all `file_id`s into a slice.
4. **Fetch File Metadata:** Call `fileReadAPI.GetAllFiles(ctx, principal, GetFilesFilter{FileIDs: ids, IncludeDownloadURL: true, IncludeVariants: true})`.
5. **Map to DTO:** Iterate over the DB rows, look up the corresponding `FileItem` from step 4, and map them to `ProductMediaDTO`s. Extract the `ThumbnailURL` from the `thumb_200` or `poster` variant if available.
6. **Response:** Assemble and return the `ProductResponseDTO`.

---

### 5.3 `GET /api/products` (List Products)
**Purpose:** Retrieves a paginated list of products, efficiently embedding media for each without N+1 query problems.

**Step-by-Step Logic:**
1. **Fetch Products:** Fetch the page of products from the `product` table.
2. **Fetch Media Mappings (Batch):** Fetch all media rows for *all* retrieved products using an `IN` clause: `WHERE product_id IN (...)`.
3. **Extract Unique IDs:** Extract all unique `file_id`s across all products into a single slice.
4. **Fetch File Metadata (Batch):** Make a **single** batch call to `fileReadAPI.GetAllFiles(...)` with all extracted `file_id`s. *(Do not loop and call this per product)*.
5. **Build Lookup Map:** Build an O(1) lookup map of `file_id -> FileItem`.
6. **Assemble Response:** Loop through each product, map its associated media rows to `ProductMediaDTO`s using the lookup map, and attach them.
7. **Response:** Return the paginated array of `ProductResponseDTO`s.

---

### 5.4 `DELETE /api/products/{id}/media/{fileId}` (Unlink & Delete Media)
**Purpose:** Removes the media from the product AND hard-deletes the physical file from the File Module.

**Step-by-Step Logic:**
1. **Verify Mapping:** Check if `fileId` is linked to `productId` in `product_media`. If not, return `404 Not Found`.
2. **Delete Mapping:** Execute `DELETE FROM product_media WHERE product_id = ? AND file_id = ?`.
3. **Primary Fallback:** If the deleted media was `is_primary = true`, find the remaining media item with the lowest `display_order` and set it to primary (optional but recommended UX).
4. **Synchronous File Deletion:** Call `fileDeleteAPI.DeleteFile(ctx, principal, fileId)` to delete the actual blob and file database record.
   - *Error Handling Note:* If this file deletion fails, log a severe error (`log.Errorf(...)`) but do *not* fail the API request (return 204). The product state is now correct; the file is simply an orphaned blob that can be cleaned up later.
5. **Response:** Return `204 No Content`.

---

### 5.5 `PATCH /api/products/{id}/media/{fileId}` (Update Media Metadata)
**Purpose:** Updates the sorting order and primary status of a specific media link.

**Request Payload:**
```json
{
  "isPrimary": true,
  "displayOrder": 2
}
```

**Step-by-Step Logic:**
1. **Verify Mapping:** Ensure the link exists in `product_media`. If not, return `404`.
2. **Primary Reset Logic:** If `isPrimary` is `true`, execute `UPDATE product_media SET is_primary = false WHERE product_id = ? AND file_id != ?`.
3. **Update Link:** Update the `product_media` row with the new `is_primary` and `display_order`.
4. **Response:** Return `200 OK` with the updated `ProductMediaDTO`.

## 6. Next Steps for Specification
- Use the `speckit-specify` tool or write the formal `spec.md` for the Product/File Integration based on this document.
- Generate actionable tasks (`tasks.md`) based on the created spec.
