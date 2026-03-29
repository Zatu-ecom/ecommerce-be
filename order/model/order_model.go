package model

import (
	"strings"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/order/entity"
)

// PaginationResponse alias for common pagination response.
type PaginationResponse = common.PaginationResponse

// ============================================================================
// Request Models
// ============================================================================

type CreateOrderRequest struct {
	ShippingAddressID uint                   `json:"shippingAddressId" binding:"required,gt=0"`
	BillingAddressID  uint                   `json:"billingAddressId"  binding:"required,gt=0"`
	FulfillmentType   entity.FulfillmentType `json:"fulfillmentType"`
	Metadata          map[string]any         `json:"metadata"`
}

type UpdateOrderStatusRequest struct {
	Status        entity.OrderStatus `json:"status"        binding:"required"`
	TransactionID *string            `json:"transactionId"`
	Note          *string            `json:"note"`
	FailureReason *string            `json:"failureReason"`
	Metadata      map[string]any     `json:"metadata"`
}

type CancelOrderRequest struct {
	Reason *string `json:"reason"`
}

// ListOrdersRequest is used for query binding from list order endpoints.
type ListOrdersRequest struct {
	common.BaseListParams
	Status   *string `form:"status"`
	FromDate *string `form:"fromDate"`
	ToDate   *string `form:"toDate"`
	Search   *string `form:"search"`
}

// ListOrdersFilter is a parsed/sanitized filter object for repository usage.
type ListOrdersFilter struct {
	common.BaseListParams
	Status   *entity.OrderStatus
	FromDate *time.Time
	ToDate   *time.Time
	Search   *string
}

func (r *ListOrdersRequest) ToFilter() ListOrdersFilter {
	r.SetDefaults()

	filter := ListOrdersFilter{
		BaseListParams: r.BaseListParams,
		Search:         r.Search,
	}

	if r.Status != nil {
		status := entity.OrderStatus(strings.TrimSpace(strings.ToLower(*r.Status)))
		if status.IsValid() {
			filter.Status = &status
		}
	}

	if r.FromDate != nil {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.FromDate)); err == nil {
			filter.FromDate = &parsed
		}
	}

	if r.ToDate != nil {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.ToDate)); err == nil {
			filter.ToDate = &parsed
		}
	}

	return filter
}

// ============================================================================
// Response Models
// ============================================================================

type OrderCustomerResponse struct {
	ID        uint    `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone"`
}

type ItemPromotionBreakdownResponse struct {
	PromotionID   *uint  `json:"promotionId"`
	PromotionName string `json:"promotionName"`
	PromotionType string `json:"promotionType"`
	DiscountCents int64  `json:"discountCents"`
	OriginalCents int64  `json:"originalCents"`
	FinalCents    int64  `json:"finalCents"`
	FreeQuantity  int    `json:"freeQuantity"`
}

type OrderItemResponse struct {
	ID                        uint                             `json:"id"`
	ProductID                 *uint                            `json:"productId"`
	VariantID                 *uint                            `json:"variantId"`
	ProductName               string                           `json:"productName"`
	VariantName               *string                          `json:"variantName"`
	SKU                       *string                          `json:"sku"`
	ImageURL                  *string                          `json:"imageUrl"`
	Quantity                  int                              `json:"quantity"`
	UnitPriceCents            int64                            `json:"unitPriceCents"`
	LineTotalCents            int64                            `json:"lineTotalCents"`
	Attributes                map[string]any                   `json:"attributes"`
	AppliedPromotionBreakdown []ItemPromotionBreakdownResponse `json:"appliedPromotionBreakdown"`
}

type OrderAddressResponse struct {
	Type      entity.OrderAddressType `json:"type"`
	Address   string                  `json:"address"`
	Landmark  string                  `json:"landmark"`
	City      string                  `json:"city"`
	State     string                  `json:"state"`
	ZipCode   string                  `json:"zipCode"`
	CountryID uint                    `json:"countryId"`
	Latitude  *float64                `json:"latitude"`
	Longitude *float64                `json:"longitude"`
}

type OrderPromotionResponse struct {
	PromotionID           *uint  `json:"promotionId"`
	PromotionName         string `json:"promotionName"`
	PromotionType         string `json:"promotionType"`
	DiscountCents         int64  `json:"discountCents"`
	ShippingDiscountCents int64  `json:"shippingDiscountCents"`
	IsStackable           *bool  `json:"isStackable"`
	Priority              int    `json:"priority"`
}

type OrderResponse struct {
	ID                uint                   `json:"id"`
	OrderNumber       string                 `json:"orderNumber"`
	Status            entity.OrderStatus     `json:"status"`
	SubtotalCents     int64                  `json:"subtotalCents"`
	DiscountCents     int64                  `json:"discountCents"`
	ShippingCents     int64                  `json:"shippingCents"`
	TaxCents          int64                  `json:"taxCents"`
	TotalCents        int64                  `json:"totalCents"`
	FulfillmentType   entity.FulfillmentType `json:"fulfillmentType"`
	PlacedAt          *time.Time             `json:"placedAt"`
	PaidAt            *time.Time             `json:"paidAt"`
	TransactionID     string                 `json:"transactionId"`
	Metadata          map[string]any         `json:"metadata"`
	Customer          *OrderCustomerResponse `json:"customer,omitempty"`
	Items             []OrderItemResponse    `json:"items"`
	Addresses         []OrderAddressResponse `json:"addresses"`
	AppliedPromotions []OrderPromotionResponse `json:"appliedPromotions"`
}

// OrderListResponse is a lightweight order summary for list APIs.
type OrderListResponse struct {
	ID              uint                   `json:"id"`
	OrderNumber     string                 `json:"orderNumber"`
	Status          entity.OrderStatus     `json:"status"`
	TotalCents      int64                  `json:"totalCents"`
	SubtotalCents   int64                  `json:"subtotalCents"`
	DiscountCents   int64                  `json:"discountCents"`
	FulfillmentType entity.FulfillmentType `json:"fulfillmentType"`
	PlacedAt        *time.Time             `json:"placedAt"`
	PaidAt          *time.Time             `json:"paidAt"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type UpdateStatusResponse struct {
	ID            uint               `json:"id"`
	OrderNumber   string             `json:"orderNumber"`
	PreviousStatus entity.OrderStatus `json:"previousStatus"`
	Status        entity.OrderStatus `json:"status"`
	TransactionID *string            `json:"transactionId"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

type PaginatedOrdersResponse struct {
	Orders     []OrderListResponse `json:"orders"`
	Pagination PaginationResponse  `json:"pagination"`
}

