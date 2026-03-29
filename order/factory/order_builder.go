package factory

import (
	"strings"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/mapper"
	"ecommerce-be/order/model"
	userModel "ecommerce-be/user/model"
)

// BuildOrderFromCartSnapshot creates the root order entity from an enriched cart snapshot.
func BuildOrderFromCartSnapshot(
	userID, sellerID uint,
	fulfillmentType entity.FulfillmentType,
	status entity.OrderStatus,
	now time.Time,
	metadata map[string]any,
	cart *model.CartResponse,
) *entity.Order {
	shipping := int64(0)
	if cart.Summary.Shipping != nil {
		shipping = *cart.Summary.Shipping
	}

	return mapper.BuildOrderEntity(
		userID,
		sellerID,
		fulfillmentType,
		status,
		metadata,
		cart.Summary.Subtotal,
		cart.Summary.TotalDiscount,
		shipping,
		cart.Summary.Tax,
		cart.Summary.Total,
		now,
	)
}

// BuildOrderItemsFromCartSnapshot snapshots cart lines into immutable order item rows.
func BuildOrderItemsFromCartSnapshot(
	orderID uint,
	cart *model.CartResponse,
) []entity.OrderItem {
	result := make([]entity.OrderItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		productID := item.Variant.Product.ID
		variantID := item.Variant.ID
		sku := toPtr(item.Variant.SKU)
		imageURL := firstImage(item.Variant.Images)
		variantName := buildVariantName(item.Variant.Options)

		result = append(result, entity.OrderItem{
			OrderID:        orderID,
			ProductID:      &productID,
			VariantID:      &variantID,
			SKU:            sku,
			ProductName:    item.Variant.Product.Name,
			VariantName:    variantName,
			ImageURL:       imageURL,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPrice,
			LineTotalCents: item.LineTotal,
			Attributes:     db.JSONMap{},
		})
	}
	return result
}

// BuildOrderAddressesFromUserAddresses snapshots shipping/billing addresses for order immutability.
func BuildOrderAddressesFromUserAddresses(
	orderID uint,
	shipping *userModel.AddressResponse,
	billing *userModel.AddressResponse,
) []entity.OrderAddress {
	return []entity.OrderAddress{
		{
			OrderID:   orderID,
			Type:      entity.ORDER_ADDR_SHIPPING,
			Address:   shipping.Address,
			Landmark:  shipping.Landmark,
			City:      shipping.City,
			State:     shipping.State,
			ZipCode:   shipping.ZipCode,
			CountryID: shipping.CountryID,
			Latitude:  shipping.Latitude,
			Longitude: shipping.Longitude,
		},
		{
			OrderID:   orderID,
			Type:      entity.ORDER_ADDR_BILLING,
			Address:   billing.Address,
			Landmark:  billing.Landmark,
			City:      billing.City,
			State:     billing.State,
			ZipCode:   billing.ZipCode,
			CountryID: billing.CountryID,
			Latitude:  billing.Latitude,
			Longitude: billing.Longitude,
		},
	}
}

// BuildOrderAppliedPromotionsFromCartSnapshot snapshots cart-level applied promotions.
func BuildOrderAppliedPromotionsFromCartSnapshot(
	orderID uint,
	cart *model.CartResponse,
) []entity.OrderAppliedPromotion {
	result := make([]entity.OrderAppliedPromotion, 0, len(cart.AppliedPromotions))
	for _, promo := range cart.AppliedPromotions {
		promoID := promo.PromotionID
		result = append(result, entity.OrderAppliedPromotion{
			OrderID:               orderID,
			PromotionID:           &promoID,
			PromotionName:         promo.Name,
			PromotionType:         promo.Type,
			DiscountCents:         promo.Discount,
			ShippingDiscountCents: promo.ShippingDiscount,
			IsStackable:           nil,
			Priority:              0,
			Metadata:              db.JSONMap{},
		})
	}
	return result
}

// BuildOrderItemAppliedPromotionsFromCartSnapshot snapshots item-level promotion breakdown.
func BuildOrderItemAppliedPromotionsFromCartSnapshot(
	orderID uint,
	cart *model.CartResponse,
	orderItems []entity.OrderItem,
) []entity.OrderItemAppliedPromotion {
	result := make([]entity.OrderItemAppliedPromotion, 0)
	limit := len(cart.Items)
	if len(orderItems) < limit {
		limit = len(orderItems)
	}

	for i := 0; i < limit; i++ {
		cartItem := cart.Items[i]
		orderItem := orderItems[i]
		for _, promo := range cartItem.AppliedPromotions {
			promotionID := promo.PromotionID
			result = append(result, entity.OrderItemAppliedPromotion{
				OrderID:       orderID,
				OrderItemID:   orderItem.ID,
				PromotionID:   &promotionID,
				PromotionName: promo.Name,
				PromotionType: promo.Type,
				DiscountCents: promo.Discount,
				OriginalCents: cartItem.LineTotal,
				FinalCents:    cartItem.DiscountedLineTotal,
				FreeQuantity:  0,
				Metadata:      db.JSONMap{},
			})
		}
	}
	return result
}

// BuildOrderResponseFromEntity converts preloaded order entities to API response.
func BuildOrderResponseFromEntity(
	order *entity.Order,
	customer *model.OrderCustomerResponse,
) *model.OrderResponse {
	resp := &model.OrderResponse{
		ID:                order.ID,
		OrderNumber:       order.OrderNumber,
		Status:            order.Status,
		SubtotalCents:     order.SubtotalCents,
		DiscountCents:     order.DiscountCents,
		ShippingCents:     order.ShippingCents,
		TaxCents:          order.TaxCents,
		TotalCents:        order.TotalCents,
		FulfillmentType:   order.FulfillmentType,
		PlacedAt:          order.PlacedAt,
		PaidAt:            order.PaidAt,
		TransactionID:     order.TransactionID,
		Metadata:          map[string]any(order.Metadata),
		Customer:          customer,
		Items:             make([]model.OrderItemResponse, 0, len(order.Items)),
		Addresses:         make([]model.OrderAddressResponse, 0, len(order.Addresses)),
		AppliedPromotions: make([]model.OrderPromotionResponse, 0, len(order.AppliedPromotions)),
	}

	itemPromoByItemID := map[uint][]model.ItemPromotionBreakdownResponse{}
	for _, p := range order.ItemAppliedPromotions {
		itemPromoByItemID[p.OrderItemID] = append(
			itemPromoByItemID[p.OrderItemID],
			model.ItemPromotionBreakdownResponse{
				PromotionID:   p.PromotionID,
				PromotionName: p.PromotionName,
				PromotionType: p.PromotionType,
				DiscountCents: p.DiscountCents,
				OriginalCents: p.OriginalCents,
				FinalCents:    p.FinalCents,
				FreeQuantity:  p.FreeQuantity,
			},
		)
	}

	for _, item := range order.Items {
		resp.Items = append(resp.Items, model.OrderItemResponse{
			ID:                        item.ID,
			ProductID:                 item.ProductID,
			VariantID:                 item.VariantID,
			ProductName:               item.ProductName,
			VariantName:               item.VariantName,
			SKU:                       item.SKU,
			ImageURL:                  item.ImageURL,
			Quantity:                  item.Quantity,
			UnitPriceCents:            item.UnitPriceCents,
			LineTotalCents:            item.LineTotalCents,
			Attributes:                map[string]any(item.Attributes),
			AppliedPromotionBreakdown: itemPromoByItemID[item.ID],
		})
	}

	for _, addr := range order.Addresses {
		resp.Addresses = append(resp.Addresses, model.OrderAddressResponse{
			Type:      addr.Type,
			Address:   addr.Address,
			Landmark:  addr.Landmark,
			City:      addr.City,
			State:     addr.State,
			ZipCode:   addr.ZipCode,
			CountryID: addr.CountryID,
			Latitude:  addr.Latitude,
			Longitude: addr.Longitude,
		})
	}

	for _, promo := range order.AppliedPromotions {
		resp.AppliedPromotions = append(resp.AppliedPromotions, model.OrderPromotionResponse{
			PromotionID:           promo.PromotionID,
			PromotionName:         promo.PromotionName,
			PromotionType:         promo.PromotionType,
			DiscountCents:         promo.DiscountCents,
			ShippingDiscountCents: promo.ShippingDiscountCents,
			IsStackable:           promo.IsStackable,
			Priority:              promo.Priority,
		})
	}

	return resp
}

func buildVariantName(options []model.VariantOptionInfo) *string {
	if len(options) == 0 {
		return nil
	}
	parts := make([]string, 0, len(options))
	for _, opt := range options {
		if strings.TrimSpace(opt.Value) != "" {
			parts = append(parts, strings.TrimSpace(opt.Value))
		}
	}
	if len(parts) == 0 {
		return nil
	}
	joined := strings.Join(parts, " / ")
	return &joined
}

func firstImage(images []string) *string {
	if len(images) == 0 || strings.TrimSpace(images[0]) == "" {
		return nil
	}
	return &images[0]
}

func toJSONMap(in map[string]any) db.JSONMap {
	if in == nil {
		return db.JSONMap{}
	}
	return db.JSONMap(in)
}

func toPtr(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
