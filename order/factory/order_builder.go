package factory

import (
	"net/url"
	"strings"
	"time"

	"ecommerce-be/common/db"
	commonModel "ecommerce-be/common/model"
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
		shipping = cart.Summary.Shipping.AmountCents
	}

	return mapper.BuildOrderEntity(
		userID,
		sellerID,
		fulfillmentType,
		status,
		metadata,
		cart.Summary.Subtotal.AmountCents,
		cart.Summary.TotalDiscount.AmountCents,
		shipping,
		cart.Summary.Tax.AmountCents,
		cart.Summary.Total.AmountCents,
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
		imageURL := snapshotImageURL(item.Variant.Images)
		imageFileID := item.Variant.ImageFileID
		variantName := buildVariantName(item.Variant.Options)

		result = append(result, entity.OrderItem{
			OrderID:        orderID,
			ProductID:      &productID,
			VariantID:      &variantID,
			SKU:            sku,
			ProductName:    item.Variant.Product.Name,
			VariantName:    variantName,
			ImageURL:       imageURL,
			ImageFileID:    imageFileID,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPrice.AmountCents,
			LineTotalCents: item.LineTotal.AmountCents,
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
			DiscountCents:         promo.Discount.AmountCents,
			ShippingDiscountCents: promo.ShippingDiscount.AmountCents,
			IsStackable:           nil,
			Priority:              0,
			Metadata:              db.JSONMap{},
		})
	}
	return result
}

// BuildOrderAppliedCouponsFromCartSnapshot snapshots cart-level applied coupons.
func BuildOrderAppliedCouponsFromCartSnapshot(
	orderID uint,
	cart *model.CartResponse,
) []entity.OrderAppliedCoupon {
	result := make([]entity.OrderAppliedCoupon, 0, len(cart.AppliedCoupons))
	for _, coupon := range cart.AppliedCoupons {
		codeID := coupon.DiscountCodeID
		title := coupon.Title
		var titlePtr *string
		if title != "" {
			titlePtr = &title
		}
		result = append(result, entity.OrderAppliedCoupon{
			OrderID:               orderID,
			DiscountCodeID:        &codeID,
			CouponCode:            coupon.Code,
			CouponTitle:           titlePtr,
			DiscountType:          coupon.DiscountType,
			DiscountCents:         coupon.Discount.AmountCents,
			ShippingDiscountCents: coupon.ShippingDiscount.AmountCents,
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
				DiscountCents: promo.Discount.AmountCents,
				OriginalCents: cartItem.LineTotal.AmountCents,
				FinalCents:    cartItem.DiscountedLineTotal.AmountCents,
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
	ccy commonModel.CurrencyInfo,
) *model.OrderResponse {
	resp := &model.OrderResponse{
		ID:                order.ID,
		OrderNumber:       order.OrderNumber,
		Status:            order.Status,
		Currency:          ccy,
		Subtotal:          commonModel.NewMoney(order.SubtotalCents, ccy),
		Discount:          commonModel.NewMoney(order.DiscountCents, ccy),
		Shipping:          commonModel.NewMoney(order.ShippingCents, ccy),
		Tax:               commonModel.NewMoney(order.TaxCents, ccy),
		Total:             commonModel.NewMoney(order.TotalCents, ccy),
		FulfillmentType:   order.FulfillmentType,
		PlacedAt:          order.PlacedAt,
		PaidAt:            order.PaidAt,
		TransactionID:     order.TransactionID,
		Metadata:          map[string]any(order.Metadata),
		Customer:          customer,
		Items:             make([]model.OrderItemResponse, 0, len(order.Items)),
		Addresses:         make([]model.OrderAddressResponse, 0, len(order.Addresses)),
		AppliedPromotions: make([]model.OrderPromotionResponse, 0, len(order.AppliedPromotions)),
		AppliedCoupons:    make([]model.OrderCouponResponse, 0, len(order.AppliedCoupons)),
	}

	itemPromoByItemID := map[uint][]model.ItemPromotionBreakdownResponse{}
	for _, p := range order.ItemAppliedPromotions {
		itemPromoByItemID[p.OrderItemID] = append(
			itemPromoByItemID[p.OrderItemID],
			model.ItemPromotionBreakdownResponse{
				PromotionID:   p.PromotionID,
				PromotionName: p.PromotionName,
				PromotionType: p.PromotionType,
				Discount:      commonModel.NewMoney(p.DiscountCents, ccy),
				Original:      commonModel.NewMoney(p.OriginalCents, ccy),
				Final:         commonModel.NewMoney(p.FinalCents, ccy),
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
			ImageFileID:               item.ImageFileID,
			Quantity:                  item.Quantity,
			UnitPrice:                 commonModel.NewMoney(item.UnitPriceCents, ccy),
			LineTotal:                 commonModel.NewMoney(item.LineTotalCents, ccy),
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
			PromotionID:      promo.PromotionID,
			PromotionName:    promo.PromotionName,
			PromotionType:    promo.PromotionType,
			Discount:         commonModel.NewMoney(promo.DiscountCents, ccy),
			ShippingDiscount: commonModel.NewMoney(promo.ShippingDiscountCents, ccy),
			IsStackable:      promo.IsStackable,
			Priority:         promo.Priority,
		})
	}

	for _, coupon := range order.AppliedCoupons {
		resp.AppliedCoupons = append(resp.AppliedCoupons, model.OrderCouponResponse{
			DiscountCodeID:   coupon.DiscountCodeID,
			CouponCode:       coupon.CouponCode,
			CouponTitle:      coupon.CouponTitle,
			DiscountType:     coupon.DiscountType,
			DiscountValue:    moneyPtr(coupon.DiscountValue, ccy),
			Discount:         commonModel.NewMoney(coupon.DiscountCents, ccy),
			ShippingDiscount: commonModel.NewMoney(coupon.ShippingDiscountCents, ccy),
			IsCombinable:     coupon.IsCombinable,
		})
	}

	return resp
}

// moneyPtr wraps an optional cents value as a Money pointer.
func moneyPtr(cents *int64, ccy commonModel.CurrencyInfo) *commonModel.Money {
	if cents == nil {
		return nil
	}
	m := commonModel.NewMoney(*cents, ccy)
	return &m
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

// snapshotImageURL stores a stable object path for order immutability.
// Presigned URLs (GCS/S3) embed signatures in query params, often exceed
// order_item.image_url limits, and expire — image_file_id is the canonical ref.
func snapshotImageURL(images []string) *string {
	raw := firstImage(images)
	if raw == nil {
		return nil
	}
	parsed, err := url.Parse(*raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	stripped := parsed.String()
	if stripped == "" {
		return raw
	}
	return &stripped
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
