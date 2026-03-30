package mapper

import (
	"strings"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
	orderUtils "ecommerce-be/order/utils"
)

// BuildOrderEntity maps checkout snapshot totals into the persistent order root entity.
func BuildOrderEntity(
	userID, sellerID uint,
	fulfillmentType entity.FulfillmentType,
	status entity.OrderStatus,
	metadata map[string]any,
	subtotalCents, discountCents, shippingCents, taxCents, totalCents int64,
	now time.Time,
) *entity.Order {
	return &entity.Order{
		UserID:          userID,
		SellerID:        &sellerID,
		OrderNumber:     orderUtils.GenerateOrderNumber(sellerID),
		Status:          status,
		SubtotalCents:   subtotalCents,
		TaxCents:        taxCents,
		ShippingCents:   shippingCents,
		DiscountCents:   discountCents,
		TotalCents:      totalCents,
		PlacedAt:        &now,
		Metadata:        toJSONMap(metadata),
		TransactionID:   "",
		FulfillmentType: fulfillmentType,
	}
}

// BuildOrderCreatedHistory maps initial order creation audit entry based on the order's initial status.
func BuildOrderCreatedHistory(
	orderID, userID uint,
	role, initialStatus string,
) *entity.OrderHistory {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	return &entity.OrderHistory{
		OrderID:         orderID,
		FromStatus:      nil,
		ToStatus:        initialStatus,
		ChangedByUserID: &userID,
		ChangedByRole:   &normalizedRole,
		Metadata:        db.JSONMap{},
	}
}

// BuildOrderTransitionHistory maps status transition details into immutable history record.
func BuildOrderTransitionHistory(
	orderID uint,
	fromStatus, toStatus entity.OrderStatus,
	actorUserID uint,
	actorRole string,
	transactionID, failureReason, note *string,
	metadata map[string]any,
) *entity.OrderHistory {
	normalizedRole := strings.ToLower(strings.TrimSpace(actorRole))
	return &entity.OrderHistory{
		OrderID:         orderID,
		FromStatus:      strPtr(fromStatus.String()),
		ToStatus:        toStatus.String(),
		ChangedByUserID: &actorUserID,
		ChangedByRole:   &normalizedRole,
		TransactionID:   transactionID,
		FailureReason:   failureReason,
		Note:            note,
		Metadata:        toJSONMap(metadata),
	}
}

func toJSONMap(in map[string]any) db.JSONMap {
	if in == nil {
		return db.JSONMap{}
	}
	return db.JSONMap(in)
}

func strPtr(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
