package service

import (
	"context"
	"strings"

	"ecommerce-be/common/log"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	userModel "ecommerce-be/user/model"
)

// CurrencyResolver narrows the user surface this file needs: the
// seller's display currency for money rendering. Production wires the real
// user service; tests stub it.
type CurrencyResolver interface {
	GetSellerDefaultCurrency(ctx context.Context, sellerID uint) (*userModel.CurrencyResponse, error)
}

// ProfileResolver narrows the user surface for recipient identity
// (courier payloads). Same wiring story as the currency resolver.
type ProfileResolver interface {
	GetProfile(ctx context.Context, userID uint) (*userModel.ProfileResponse, error)
}

// GetOrderForFulfillment reads the planning inputs for an order. It is the
// read half of the fulfillment contract: lines with variant linkage,
// fulfillment type gate data, the delivery address snapshot source, and the
// COD amount. No state is changed.
func (h *OrderFulfillmentHooksImpl) GetOrderForFulfillment(
	ctx context.Context,
	orderID uint,
) (*fulfillmentmodel.FulfillmentOrderView, error) {
	order, err := h.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, orderError.ErrOrderNotFound
	}

	view := &fulfillmentmodel.FulfillmentOrderView{
		OrderID:         order.ID,
		UserID:          order.UserID,
		FulfillmentType: string(order.FulfillmentType),
		Status:          string(order.Status),
	}
	if order.SellerID != nil {
		view.SellerID = *order.SellerID
	}

	for _, item := range order.Items {
		sku := ""
		if item.SKU != nil {
			sku = *item.SKU
		}
		view.Items = append(view.Items, fulfillmentmodel.FulfillmentOrderItemView{
			OrderItemID:    item.ID,
			VariantID:      item.VariantID,
			Quantity:       item.Quantity,
			ProductName:    item.ProductName,
			SKU:            sku,
			UnitPriceCents: item.UnitPriceCents,
		})
	}

	if addr := shippingAddress(order.Addresses); addr != nil {
		view.DeliveryAddressID = addr.ID
		view.DeliveryAddressUpdatedAt = addr.UpdatedAt
		view.DeliveryPincode = addr.ZipCode
		view.DeliveryStreet = addr.Address
		view.DeliveryCity = addr.City
		view.DeliveryState = addr.State
	}

	// Recipient identity for courier payloads (name + phone from profile).
	if h.profileResolver != nil {
		if profile, err := h.profileResolver.GetProfile(ctx, order.UserID); err == nil && profile != nil {
			name := strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName))
			view.RecipientName = name
			view.RecipientPhone = strings.TrimSpace(profile.Phone)
		} else if err != nil {
			log.ErrorWithContext(ctx, "fulfillment view: profile unresolved", err)
		}
	}

	// COD when unpaid (PaidAt nil): the courier collects the order total.
	// Prepaid orders (PaidAt set) collect nothing.
	if order.PaidAt == nil {
		view.CodCents = order.TotalCents
	}

	// Display currency only: a resolution failure logs and leaves the code
	// empty (callers fall back to INR display) rather than failing planning.
	if h.currencyResolver != nil {
		if ccy, err := h.currencyResolver.GetSellerDefaultCurrency(ctx, view.SellerID); err == nil && ccy != nil {
			view.CurrencyCode = strings.ToUpper(strings.TrimSpace(ccy.Code))
		} else if err != nil {
			log.ErrorWithContext(ctx, "fulfillment view: currency unresolved, display falls back", err)
		}
	}

	return view, nil
}

// shippingAddress prefers the shipping snapshot, falling back to the first
// address row when no explicit shipping type exists.
func shippingAddress(addresses []entity.OrderAddress) *entity.OrderAddress {
	if len(addresses) == 0 {
		return nil
	}
	for i := range addresses {
		if addresses[i].Type == entity.ORDER_ADDR_SHIPPING {
			return &addresses[i]
		}
	}
	return &addresses[0]
}
