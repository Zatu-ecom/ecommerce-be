# Promotion API — Implementation TODO

> **Phase 1**: Finish the core Promotion CRUD APIs  
> **Phase 2** (next): Discount Code / Coupon APIs

---

## ✅ Already Implemented

| Area | Details |
|---|---|
| `POST /api/v1/promotion` | Create promotion (service implemented, **no HTTP handler/route yet** — see gap below) |
| `POST /api/v1/promotion/scope/product` | Add products to promotion |
| `DELETE /api/v1/promotion/scope/product` | Remove specific products |
| `DELETE /api/v1/promotion/scope/:promotionId/product` | Remove all products |
| `GET /api/v1/promotion/scope/:promotionId/product` | List products in scope |
| Same 4 routes for `/variant`, `/category`, `/collection` | ✅ All scope handlers, services, repositories done |
| `PromotionService.ApplyPromotionsToCart()` | Validation logic exists in `promotion_validator_service.go` |
| Strategy Pattern — `ValidateConfig` | ✅ Already called in `CreatePromotion`. All 7 strategy files exist (`percentage`, `fixed_amount`, `buy_x_get_y`, `free_shipping`, `bundle`, `tiered`, `flash_sale`) |
| Factory / Singleton | mapper, service factory, repo factory, handler factory |
| Entity | `Promotion`, `PromotionProduct`, `PromotionCategory`, `PromotionCollection`, `PromotionVariant`, `sale.go`, `usage.go`, `discount_code.go` |
| Error Package | `promotion/error/` |

---

## ❌ Missing — Must Implement Before Coupon Phase

### 1. `PromotionHandler` (HTTP layer for main promotion resource)

**File to create**: `promotion/handler/promotion_handler.go`

```
type PromotionHandler struct {
    *commonHandler.BaseHandler
    service service.PromotionService
}
```

Methods to add:
- [ ] `CreatePromotion` — POST body → `CreatePromotionRequest` → call `service.CreatePromotion`
- [ ] `GetPromotion` — GET `/:promotionId` → call `service.GetPromotionByID`
- [ ] `ListPromotions` — GET `/` (with query filters) → call `service.ListPromotions`
- [ ] `UpdatePromotion` — PUT/PATCH `/:promotionId` → call `service.UpdatePromotion`
- [ ] `UpdateStatus` — PATCH `/:promotionId/status` → activate / pause / end / cancel
- [ ] `DeletePromotion` — DELETE `/:promotionId` (soft delete)

---

### 2. Missing Service Methods on `PromotionService` interface

**File**: `promotion/service/promotion_service.go`

Add to the `PromotionService` interface and implement:

- [ ] `GetPromotionByID(ctx, id, sellerID) (*PromotionResponse, error)`
  - Call `repo.FindByID` + verify sellerID ownership
- [ ] `ListPromotions(ctx, req ListPromotionsRequest) (*ListPromotionsResponse, error)`
  - Filters: `status`, `promotionType`, `appliesTo`, `startsAfter`, `startsBefore`, pagination (`page`, `limit`)
  - Uses `repo.List` (see below)
- [ ] `UpdatePromotion(ctx, id, req UpdatePromotionRequest, sellerID) (*PromotionResponse, error)`
  - Partial update (only provided fields)
  - **Re-call `strategy.ValidateConfig(req.DiscountConfig)`** if `discountConfig` or `promotionType` is being changed (same pattern as `CreatePromotion`)
  - Re-validate date ranges if dates changed
- [ ] `UpdateStatus(ctx, id uint, status CampaignStatus, sellerID uint) (*PromotionResponse, error)`
  - Valid transitions: `draft → scheduled/active`, `active → paused/ended/cancelled`, `paused → active/cancelled`
  - Reject invalid transitions with clear error
- [ ] `DeletePromotion(ctx, id, sellerID) error`
  - Soft-delete only (set `deleted_at` via GORM)
  - Guard: cannot delete an `active` promotion — must deactivate first

---

### 3. Missing Repository Methods

**File**: `promotion/repository/promotion_repository.go`

Add to `PromotionRepository` interface and implement:

- [ ] `Update(ctx, promotion *entity.Promotion) error`
  - Use `db.Save()` or `db.Updates()` for partial update
- [ ] `UpdateStatus(ctx, id uint, status entity.CampaignStatus) error`
  - Targeted column update: `UPDATE promotion SET status = ? WHERE id = ?`
- [ ] `Delete(ctx, id uint) error`
  - Soft delete: `db.Delete(&entity.Promotion{}, id)`
- [ ] `List(ctx, filters ListPromotionFilters) ([]*entity.Promotion, int64, error)`
  - Dynamic WHERE building: `sellerID`, `status`, `promotionType`, `appliesTo`
  - Date range filter: `starts_at >= ?`, `ends_at <= ?`
  - Pagination: `LIMIT/OFFSET`
  - Returns total count for pagination response

---

### 4. New Model Types

**File**: `promotion/model/promotion_request_model.go` — add:

```go
// UpdatePromotionRequest — all fields optional (partial update)
type UpdatePromotionRequest struct {
    Name           *string                `json:"name"           binding:"omitempty,min=3,max=255"`
    DisplayName    *string                `json:"displayName"    binding:"omitempty,max=255"`
    Description    *string                `json:"description"`
    DiscountConfig map[string]interface{} `json:"discountConfig" binding:"omitempty"`
    // ... same optional fields as CreatePromotionRequest
    StartsAt *string `json:"startsAt" binding:"omitempty"`
    EndsAt   *string `json:"endsAt"   binding:"omitempty"`
    Status   *entity.CampaignStatus `json:"status" binding:"omitempty"`
}

// UpdateStatusRequest
type UpdateStatusRequest struct {
    Status entity.CampaignStatus `json:"status" binding:"required,oneof=draft scheduled active paused ended cancelled"`
}

// ListPromotionsRequest
type ListPromotionsRequest struct {
    SellerID      uint
    Status        *entity.CampaignStatus `form:"status"`
    PromotionType *entity.PromotionType  `form:"promotionType"`
    AppliesTo     *entity.ScopeType      `form:"appliesTo"`
    Page          int                    `form:"page,default=1"`
    Limit         int                    `form:"limit,default=20"`
}

// ListPromotionsResponse
type ListPromotionsResponse struct {
    Promotions []*PromotionResponse `json:"promotions"`
    Total      int64                `json:"total"`
    Page       int                  `json:"page"`
    Limit      int                  `json:"limit"`
}
```

---

### 5. Promotion Routes (main resource)

**File to create**: `promotion/route/promotion_routes.go`

Routes to add (all `sellerAuth` protected):

```
GET    /api/v1/promotion              → ListPromotions
POST   /api/v1/promotion              → CreatePromotion
GET    /api/v1/promotion/:promotionId → GetPromotion
PUT    /api/v1/promotion/:promotionId → UpdatePromotion
PATCH  /api/v1/promotion/:promotionId/status → UpdateStatus
DELETE /api/v1/promotion/:promotionId → DeletePromotion
```

---

### 6. Wire Everything in Container & Factory

- [ ] Add `PromotionHandler` to `HandlerFactory` (`handler_factory.go`)
- [ ] Add `GetPromotionHandler()` to `HandlerFactory`
- [ ] Register `NewPromotionModule()` in `container.go` alongside the existing `NewPromotionScopeModule()`

---

### 7. Constants — Add for new operations

**File**: `promotion/utils/constant/promotion_constants.go`

```go
PROMOTION_CREATED_MSG          = "Promotion created successfully"
PROMOTION_RETRIEVED_MSG        = "Promotion retrieved successfully"
PROMOTIONS_RETRIEVED_MSG       = "Promotions retrieved successfully"
PROMOTION_UPDATED_MSG          = "Promotion updated successfully"
PROMOTION_STATUS_UPDATED_MSG   = "Promotion status updated successfully"
PROMOTION_DELETED_MSG          = "Promotion deleted successfully"
FAILED_TO_CREATE_PROMOTION_MSG = "Failed to create promotion"
FAILED_TO_GET_PROMOTION_MSG    = "Failed to get promotion"
FAILED_TO_LIST_PROMOTIONS_MSG  = "Failed to list promotions"
FAILED_TO_UPDATE_PROMOTION_MSG = "Failed to update promotion"
FAILED_TO_DELETE_PROMOTION_MSG = "Failed to delete promotion"
PROMOTION_FIELD                = "promotion"
PROMOTIONS_FIELD               = "promotions"
```

---

### 8. Factory mapper — Add `UpdatePromotionRequestToEntity`

**File**: `promotion/factory/promotion_mapper.go`

```go
func ApplyUpdatePromotionRequest(existing *entity.Promotion, req model.UpdatePromotionRequest) *entity.Promotion
```

---

### 9. Strategy `CalculateDiscount` — wire it into `ApplyPromotionsToCart`

The `PromotionStrategy` interface has two methods:
- `ValidateConfig` ✅ — already called in `CreatePromotion` (and should also be called in `UpdatePromotion`)
- `CalculateDiscount` ❌ — **not wired anywhere yet**

This is the core discount engine. It must be called inside `ApplyPromotionsToCart` in `promotion_validator_service.go` to actually compute discounts per cart item.  
Nothing new to create — just wire `promotionStrategy.GetPromotionStrategy(p.PromotionType).CalculateDiscount(...)` inside the validator loop.

`ApplyPromotionsToCart` itself will remain an **internal service call** from the cart service (no HTTP route needed).

---

## 📋 Implementation Order (Recommended)

```
1. Models (UpdatePromotionRequest, ListPromotionsRequest, ListPromotionsResponse)
2. Repository  (Update, UpdateStatus, Delete, List)
3. Service  (GetPromotionByID, ListPromotions, UpdatePromotion, UpdateStatus, DeletePromotion)
4. Factory mapper  (ApplyUpdatePromotionRequest)
5. Handler  (promotion_handler.go)
6. Constants  (add new ones)
7. Routes  (promotion_routes.go)
8. Wire factory (handler_factory.go, container.go)
9. Build & smoke test
```

---

## 🚀 Next Phase: Discount Code / Coupon

Once the above is ✅ complete, start the discount code phase.  
Entity `discount_code.go` and `discount_code_scope.go` already exist in `promotion/entity/`.  
Full implementation plan will be in a separate `DISCOUNT_CODE_TODO.md`.
