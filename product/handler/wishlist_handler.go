package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"
)

// WishlistHandler handles HTTP requests related to wishlist management
type WishlistHandler struct {
	*handler.BaseHandler
	wishlistService service.WishlistService
}

// NewWishlistHandler creates a new instance of WishlistHandler
func NewWishlistHandler(wishlistService service.WishlistService) *WishlistHandler {
	return &WishlistHandler{
		BaseHandler:     handler.NewBaseHandler(),
		wishlistService: wishlistService,
	}
}

// GetAllWishlists handles getting all wishlists for the authenticated user
// GET /api/wishlist
func (h *WishlistHandler) GetAllWishlists(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_MSG)
		return
	}

	response, err := h.wishlistService.GetAllWishlists(c, userID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_WISHLISTS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.WISHLISTS_RETRIEVED_MSG,
		utils.WISHLISTS_FIELD_NAME,
		response.Wishlists,
	)
}

// CreateWishlist handles creating a new wishlist
// POST /api/wishlist
func (h *WishlistHandler) CreateWishlist(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_MSG)
		return
	}

	var req model.WishlistCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.wishlistService.CreateWishlist(c, userID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_WISHLIST_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.WISHLIST_CREATED_MSG,
		utils.WISHLIST_FIELD_NAME,
		response,
	)
}

// GetWishlistByID handles getting a wishlist with products (paginated)
// GET /api/wishlist/:id?page=1&pageSize=20
func (h *WishlistHandler) GetWishlistByID(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	// Parse pagination params
	var params common.BaseListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.wishlistService.GetWishlistByID(c, userID, wishlistID, params)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_WISHLIST_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.WISHLIST_RETRIEVED_MSG,
		utils.WISHLIST_FIELD_NAME,
		response,
	)
}

// UpdateWishlist handles updating a wishlist (name and/or default)
// PUT /api/wishlist/:id
func (h *WishlistHandler) UpdateWishlist(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	var req model.WishlistUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.wishlistService.UpdateWishlist(c, userID, wishlistID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_WISHLIST_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.WISHLIST_UPDATED_MSG,
		utils.WISHLIST_FIELD_NAME,
		response,
	)
}

// DeleteWishlist handles deleting a wishlist
// DELETE /api/wishlist/:id
func (h *WishlistHandler) DeleteWishlist(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	if err := h.wishlistService.DeleteWishlist(c, userID, wishlistID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_WISHLIST_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.WISHLIST_DELETED_MSG, nil)
}
