package handler

import (
	"net/http"
	"strconv"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	errs "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	commonModel "ecommerce-be/common/model"

	"ecommerce-be/order/model"
	"ecommerce-be/order/service"
	orderConstants "ecommerce-be/order/utils/constant"

	"github.com/gin-gonic/gin"
)

// GuestCartHandler handles HTTP requests for guest (device-identified) cart operations.
type GuestCartHandler struct {
	*handler.BaseHandler
	guestCartService service.GuestCartService
}

// NewGuestCartHandler creates a new GuestCartHandler.
func NewGuestCartHandler(guestCartService service.GuestCartService) *GuestCartHandler {
	return &GuestCartHandler{
		BaseHandler:      handler.NewBaseHandler(),
		guestCartService: guestCartService,
	}
}

// AddToGuestCart adds items to a guest cart.
// POST /api/order/cart/guest/item
func (h *GuestCartHandler) AddToGuestCart(c *gin.Context) {
	deviceID, ok := auth.ExtractAndValidateDeviceID(c)
	if !ok {
		commonModel.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constants.DEVICE_ID_REQUIRED_MSG,
			constants.DEVICE_ID_REQUIRED_CODE,
		)
		c.Abort()
		return
	}

	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		log.ErrorWithContext(c, "addToGuestCart: seller ID missing from context", nil)
		h.HandleError(c, errs.UnauthorizedError, orderConstants.SELLER_CONTEXT_REQUIRED_MSG)
		return
	}

	var req model.AddCartItemRequest
	if err := h.BindJSON(c, &req); err != nil {
		log.WarnWithContext(c, "addToGuestCart: validation failed: "+err.Error())
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.guestCartService.AddToGuestCart(c, deviceID, sellerID, req)
	if err != nil {
		log.ErrorWithContext(c, "addToGuestCart: failed to add item", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_ADD_ITEM_TO_CART_MSG)
		return
	}

	h.Success(c, http.StatusCreated, orderConstants.ITEM_ADDED_TO_CART_MSG, resp)
}

// GetGuestCart retrieves the guest cart.
// GET /api/order/cart/guest
func (h *GuestCartHandler) GetGuestCart(c *gin.Context) {
	deviceID, ok := auth.ExtractAndValidateDeviceID(c)
	if !ok {
		commonModel.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constants.DEVICE_ID_REQUIRED_MSG,
			constants.DEVICE_ID_REQUIRED_CODE,
		)
		c.Abort()
		return
	}

	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		log.ErrorWithContext(c, "getGuestCart: seller ID missing from context", nil)
		h.HandleError(c, errs.UnauthorizedError, orderConstants.SELLER_CONTEXT_REQUIRED_MSG)
		return
	}

	resp, err := h.guestCartService.GetGuestCart(c, deviceID, sellerID)
	if err != nil {
		log.ErrorWithContext(c, "getGuestCart: failed to fetch cart", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_GET_CART_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.CART_FETCHED_MSG, resp)
}

// DeleteGuestCart deletes a guest cart by cart ID.
// DELETE /api/order/cart/guest/:cartId
func (h *GuestCartHandler) DeleteGuestCart(c *gin.Context) {
	deviceID, ok := auth.ExtractAndValidateDeviceID(c)
	if !ok {
		commonModel.ErrorWithCode(
			c,
			http.StatusBadRequest,
			constants.DEVICE_ID_REQUIRED_MSG,
			constants.DEVICE_ID_REQUIRED_CODE,
		)
		c.Abort()
		return
	}

	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		log.ErrorWithContext(c, "deleteGuestCart: seller ID missing from context", nil)
		h.HandleError(c, errs.UnauthorizedError, orderConstants.SELLER_CONTEXT_REQUIRED_MSG)
		return
	}

	cartIDRaw := c.Param("cartId")
	cartID64, err := strconv.ParseUint(cartIDRaw, 10, 64)
	if err != nil || cartID64 == 0 {
		h.HandleValidationError(c, errs.ErrInvalidID)
		return
	}

	resp, err := h.guestCartService.DeleteGuestCart(c, deviceID, sellerID, uint(cartID64))
	if err != nil {
		log.ErrorWithContext(c, "deleteGuestCart: failed to delete cart", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_DELETE_CART_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.CART_DELETED_MSG, resp)
}
