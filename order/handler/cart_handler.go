package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"

	"ecommerce-be/order/model"
	"ecommerce-be/order/service"
	orderConstants "ecommerce-be/order/utils/constant"

	"github.com/gin-gonic/gin"
)

type CartHandler struct {
	*handler.BaseHandler
	cartService service.CartService
}

func NewCartHandler(cartService service.CartService) *CartHandler {
	return &CartHandler{
		BaseHandler: handler.NewBaseHandler(),
		cartService: cartService,
	}
}

// AddToCart API handler to add an item to user's cart
// @Summary Add item to cart
// @Description Adds a product variant to the active user's cart and returns full cart with applied promotions.
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.AddCartItemRequest true "Add Cart Item Request"
// @Success 201 {object} common.StandardResponse{data=model.CartResponse}
// @Failure 401 {object} common.ErrorResponse
// @Failure 400 {object} common.ErrorResponse
// @Router /api/cart/item [post]
func (h *CartHandler) AddToCart(c *gin.Context) {
	// 1. Get user context
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		log.ErrorWithContext(c, "addToCart: user ID missing from context", nil)
		h.HandleError(c, errs.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}

	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		log.ErrorWithContext(c, "addToCart: seller ID missing from context", nil)
		h.HandleError(c, errs.UnauthorizedError, orderConstants.SELLER_CONTEXT_REQUIRED_MSG)
		return
	}

	// 2. Bind request body
	var req model.AddCartItemRequest
	if err := h.BindJSON(c, &req); err != nil {
		log.WarnWithContext(c, "addToCart: validation failed: "+err.Error())
		h.HandleValidationError(c, err)
		return
	}

	// 3. Call Service layer
	resp, err := h.cartService.AddToCart(c, userID, sellerID, req)
	if err != nil {
		log.ErrorWithContext(c, "addToCart: failed to add item", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_ADD_ITEM_TO_CART_MSG)
		return
	}

	// 4. Return success response (using CartResponse containing the full promotion summary)
	h.Success(c, http.StatusCreated, orderConstants.ITEM_ADDED_TO_CART_MSG, resp)
}
