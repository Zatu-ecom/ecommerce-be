package promotion_test

const (
	// PromotionAPIEndpoint is the base endpoint for Promotion creation
	PromotionAPIEndpoint = "/api/promotion"

	// PromotionVariantsEndpoint is the endpoint for linking variants to promotions.
	// Request body must include `promotionId` and `variantIds`.
	PromotionVariantsEndpoint = "/api/promotion/scope/variant"

	// CartAPIEndpoint is the base cart endpoint.
	// Current order module mounts cart handlers under /api/order.
	CartAPIEndpoint = "/api/order"

	// CartItemAPIEndpoint is the endpoint to add items to cart
	CartItemAPIEndpoint = "/api/order/item"
)
