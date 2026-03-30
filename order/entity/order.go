package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// Order Status Enum
// ============================================================================

type OrderStatus string

const (
	ORDER_STATUS_PENDING   OrderStatus = "pending"
	ORDER_STATUS_CONFIRMED OrderStatus = "confirmed"
	// ORDER_STATUS_PROCESSING OrderStatus = "processing"
	// ORDER_STATUS_SHIPPED    OrderStatus = "shipped"
	// ORDER_STATUS_DELIVERED  OrderStatus = "delivered"
	ORDER_STATUS_CANCELLED OrderStatus = "cancelled"
	// ORDER_STATUS_REFUNDED   OrderStatus = "refunded"
	ORDER_STATUS_FAILED    OrderStatus = "failed"
	ORDER_STATUS_RETURNED  OrderStatus = "returned"
	ORDER_STATUS_COMPLETED OrderStatus = "completed"
)

// ValidOrderStatuses returns all valid order status values
func ValidOrderStatuses() []OrderStatus {
	return []OrderStatus{
		ORDER_STATUS_PENDING,
		ORDER_STATUS_CONFIRMED,
		// ORDER_STATUS_PROCESSING,
		// ORDER_STATUS_SHIPPED,
		// ORDER_STATUS_DELIVERED,
		ORDER_STATUS_CANCELLED,
		// ORDER_STATUS_REFUNDED,
		ORDER_STATUS_FAILED,
		ORDER_STATUS_RETURNED,
		ORDER_STATUS_COMPLETED,
	}
}

// String returns the string representation
func (s OrderStatus) String() string {
	return string(s)
}

// IsValid checks if the order status is valid
func (s OrderStatus) IsValid() bool {
	switch s {
	case ORDER_STATUS_PENDING,
		ORDER_STATUS_CONFIRMED,
		// ORDER_STATUS_PROCESSING,
		// ORDER_STATUS_SHIPPED,
		// ORDER_STATUS_DELIVERED,
		ORDER_STATUS_CANCELLED,
		// ORDER_STATUS_REFUNDED,
		ORDER_STATUS_FAILED,
		ORDER_STATUS_RETURNED,
		ORDER_STATUS_COMPLETED:
		return true
	}
	return false
}

// ============================================================================
// Fulfillment Type Enum
// ============================================================================

type FulfillmentType string

const (
	// Buy Online, Pick Up In Store
	BOPIS     FulfillmentType = "bopis"
	// Direct Ship to CustomeR, for online order this is the default fulfillment type
	DIRECTSHIP FulfillmentType = "directship"
	// Delivery to Customer, this is for local delivery or third-party delivery service
	DELIVERY  FulfillmentType = "delivery"
	// Transfer to another store
	TRANSFER  FulfillmentType = "transfer"
)

func ValidFulfillmentTypes() []FulfillmentType {
	return []FulfillmentType{
		BOPIS,
		DIRECTSHIP,
		DELIVERY,
		TRANSFER,
	}
}

func (f FulfillmentType) String() string {
	return string(f)
}

func (f FulfillmentType) IsValid() bool {
	switch f {
	case BOPIS, DIRECTSHIP, DELIVERY, TRANSFER:
		return true
	}
	return false
}

// ============================================================================
// Order Entity
// ============================================================================

type Order struct {
	db.BaseEntity
	UserID          uint            `json:"userId"          gorm:"column:user_id;not null;index"`
	SellerID        *uint           `json:"sellerId"        gorm:"column:seller_id;index"`
	OrderNumber     string          `json:"orderNumber"     gorm:"column:order_number;uniqueIndex"`
	Status          OrderStatus     `json:"status"          gorm:"column:status;size:32;default:pending;index"`
	SubtotalCents   int64           `json:"subtotalCents"   gorm:"column:subtotal_cents;default:0"`
	TaxCents        int64           `json:"taxCents"        gorm:"column:tax_cents;default:0"`
	ShippingCents   int64           `json:"shippingCents"   gorm:"column:shipping_cents;default:0"`
	DiscountCents   int64           `json:"discountCents"   gorm:"column:discount_cents;default:0"`
	TotalCents      int64           `json:"totalCents"      gorm:"column:total_cents;default:0"`
	PlacedAt        *time.Time      `json:"placedAt"        gorm:"column:placed_at"`
	PaidAt          *time.Time      `json:"paidAt"          gorm:"column:paid_at"`
	Metadata        db.JSONMap      `json:"metadata"        gorm:"column:metadata;type:jsonb;default:'{}'"`
	TransactionID   string          `json:"transactionId"   gorm:"column:transaction_id"`
	FulfillmentType FulfillmentType `json:"fulfillmentType" gorm:"column:fulfillment_type;size:32;default:'directship'"`

	// Associations for query preloading.
	Items                  []OrderItem                 `json:"items,omitempty"                  gorm:"foreignKey:OrderID"`
	Addresses              []OrderAddress              `json:"addresses,omitempty"              gorm:"foreignKey:OrderID"`
	AppliedPromotions      []OrderAppliedPromotion     `json:"appliedPromotions,omitempty"      gorm:"foreignKey:OrderID"`
	AppliedCoupons         []OrderAppliedCoupon        `json:"appliedCoupons,omitempty"         gorm:"foreignKey:OrderID"`
	ItemAppliedPromotions  []OrderItemAppliedPromotion `json:"itemAppliedPromotions,omitempty"  gorm:"foreignKey:OrderID"`
}
