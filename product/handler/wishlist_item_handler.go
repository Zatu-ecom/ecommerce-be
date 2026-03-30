package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"
)

// WishlistItemHandler handles HTTP requests related to wishlist item management
type WishlistItemHandler struct {
	*handler.BaseHandler
	wishlistItemService service.WishlistItemService
}

// NewWishlistItemHandler creates a new instance of WishlistItemHandler
func NewWishlistItemHandler(wishlistItemService service.WishlistItemService) *WishlistItemHandler {
	return &WishlistItemHandler{
		BaseHandler:         handler.NewBaseHandler(),
		wishlistItemService: wishlistItemService,
	}
}

// AddItem handles adding an item to a wishlist
// POST /api/wishlist/:id/item
func (h *WishlistItemHandler) AddItem(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_ITEM_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	var req model.WishlistItemCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.wishlistItemService.AddItem(c, userID, wishlistID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_ADD_WISHLIST_ITEM_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.WISHLIST_ITEM_ADDED_MSG,
		utils.WISHLIST_ITEM_FIELD_NAME,
		response,
	)
}

// RemoveItem handles removing an item from a wishlist
// DELETE /api/wishlist/:id/item/:itemId
func (h *WishlistItemHandler) RemoveItem(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_ITEM_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	itemID, err := h.ParseUintParam(c, "itemId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ITEM_ID_MSG)
		return
	}

	if err := h.wishlistItemService.RemoveItem(c, userID, wishlistID, itemID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_REMOVE_WISHLIST_ITEM_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.WISHLIST_ITEM_REMOVED_MSG, nil)
}

// MoveItem handles moving an item to another wishlist
// POST /api/wishlist/:id/item/:itemId/move
func (h *WishlistItemHandler) MoveItem(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		h.HandleError(c, nil, utils.UNAUTHORIZED_WISHLIST_ITEM_MSG)
		return
	}

	wishlistID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ID_MSG)
		return
	}

	itemID, err := h.ParseUintParam(c, "itemId")
	if err != nil {
		h.HandleError(c, err, utils.INVALID_WISHLIST_ITEM_ID_MSG)
		return
	}

	var req model.WishlistItemMoveRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	response, err := h.wishlistItemService.MoveItem(c, userID, wishlistID, itemID, req)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_MOVE_WISHLIST_ITEM_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.WISHLIST_ITEM_MOVED_MSG,
		utils.WISHLIST_ITEM_FIELD_NAME,
		response,
	)
}
