package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonHandler "ecommerce-be/common/handler"
	"ecommerce-be/product/model"
	"ecommerce-be/product/service"
	"ecommerce-be/product/utils"

	"github.com/gin-gonic/gin"
)

// CollectionHandler handles HTTP requests related to collections
type CollectionHandler struct {
	*commonHandler.BaseHandler
	collectionService        service.CollectionService
	collectionProductService service.CollectionProductService
}

// NewCollectionHandler creates a new CollectionHandler
func NewCollectionHandler(
	collectionService service.CollectionService,
	collectionProductService service.CollectionProductService,
) *CollectionHandler {
	return &CollectionHandler{
		BaseHandler:              commonHandler.NewBaseHandler(),
		collectionService:        collectionService,
		collectionProductService: collectionProductService,
	}
}

func (h *CollectionHandler) CreateCollection(c *gin.Context) {
	var req model.CollectionCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.collectionService.CreateCollection(c, req, roleLevel, sellerID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_CREATE_COLLECTION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		utils.COLLECTION_CREATED_MSG,
		utils.COLLECTION_FIELD_NAME,
		response,
	)
}

func (h *CollectionHandler) UpdateCollection(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var req model.CollectionUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.collectionService.UpdateCollection(c, collectionID, req, roleLevel, sellerID)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_UPDATE_COLLECTION_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.COLLECTION_UPDATED_MSG,
		utils.COLLECTION_FIELD_NAME,
		response,
	)
}

func (h *CollectionHandler) DeleteCollection(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.collectionService.DeleteCollection(c, collectionID, roleLevel, sellerID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_DELETE_COLLECTION_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTION_DELETED_MSG, nil)
}

func (h *CollectionHandler) GetAllCollections(c *gin.Context) {
	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	response, err := h.collectionService.GetAllCollections(c, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_COLLECTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTIONS_RETRIEVED_MSG, response)
}

func (h *CollectionHandler) GetCollectionByID(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	response, err := h.collectionService.GetCollectionByID(c, collectionID, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_COLLECTIONS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		utils.COLLECTIONS_RETRIEVED_MSG,
		utils.COLLECTION_FIELD_NAME,
		response,
	)
}

func (h *CollectionHandler) AddProducts(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var req model.AddCollectionProductsRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.collectionProductService.AddProducts(c, collectionID, req, roleLevel, sellerID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_ADD_COLLECTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTION_PRODUCTS_ADDED_MSG, nil)
}

func (h *CollectionHandler) RemoveProducts(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var req model.RemoveCollectionProductsRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.collectionProductService.RemoveProducts(c, collectionID, req, roleLevel, sellerID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_REMOVE_COLLECTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTION_PRODUCTS_REMOVED_MSG, nil)
}

func (h *CollectionHandler) GetProducts(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var params model.GetCollectionProductsQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	req := params.ToRequest()

	var sellerIDPtr *uint
	if sellerID, exists := auth.GetSellerIDFromContext(c); exists {
		sellerIDPtr = &sellerID
	}

	response, err := h.collectionProductService.GetProducts(c, collectionID, req, sellerIDPtr)
	if err != nil {
		h.HandleError(c, err, utils.FAILED_TO_GET_COLLECTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTION_PRODUCTS_RETRIEVED_MSG, response)
}

func (h *CollectionHandler) ReorderProducts(c *gin.Context) {
	collectionID, err := h.ParseUintParam(c, "collectionId")
	if err != nil {
		h.HandleError(c, err, "Invalid collection ID")
		return
	}

	var req model.ReorderCollectionProductsRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	roleLevel, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.collectionProductService.ReorderProducts(c, collectionID, req, roleLevel, sellerID); err != nil {
		h.HandleError(c, err, utils.FAILED_TO_REORDER_COLLECTION_PRODUCTS_MSG)
		return
	}

	h.Success(c, http.StatusOK, utils.COLLECTION_PRODUCTS_REORDERED_MSG, nil)
}
