package handler

import (
	"net/http"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonHandler "ecommerce-be/common/handler"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/service"
	promotionConstants "ecommerce-be/promotion/utils/constant"

	"github.com/gin-gonic/gin"
)

// SaleHandler handles HTTP requests for sales
type SaleHandler struct {
	*commonHandler.BaseHandler
	service service.SaleService
}

// NewSaleHandler creates a new SaleHandler
func NewSaleHandler(service service.SaleService) *SaleHandler {
	return &SaleHandler{
		BaseHandler: commonHandler.NewBaseHandler(),
		service:     service,
	}
}

func (h *SaleHandler) CreateSale(c *gin.Context) {
	var req model.CreateSaleRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.CreateSale(c, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_CREATE_SALE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusCreated,
		promotionConstants.SALE_CREATED_MSG,
		promotionConstants.SALE_FIELD,
		response,
	)
}

func (h *SaleHandler) UpdateSale(c *gin.Context) {
	saleID, err := h.ParseUintParam(c, "saleId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_SALE_ID_MSG)
		return
	}

	var req model.UpdateSaleRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.UpdateSale(c, saleID, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_SALE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.SALE_UPDATED_MSG,
		promotionConstants.SALE_FIELD,
		response,
	)
}

func (h *SaleHandler) DeleteSale(c *gin.Context) {
	saleID, err := h.ParseUintParam(c, "saleId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_SALE_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	if err := h.service.DeleteSale(c, saleID, sellerID); err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_DELETE_SALE_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.SALE_DELETED_MSG, nil)
}

func (h *SaleHandler) GetSale(c *gin.Context) {
	saleID, err := h.ParseUintParam(c, "saleId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_SALE_ID_MSG)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.GetSaleByID(c, saleID, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_GET_SALE_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.SALE_RETRIEVED_MSG,
		promotionConstants.SALE_FIELD,
		response,
	)
}

func (h *SaleHandler) ListSales(c *gin.Context) {
	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.ListSales(c, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_LIST_SALES_MSG)
		return
	}

	h.Success(c, http.StatusOK, promotionConstants.SALES_LISTED_MSG, response)
}

func (h *SaleHandler) UpdateStatus(c *gin.Context) {
	saleID, err := h.ParseUintParam(c, "saleId")
	if err != nil {
		h.HandleError(c, err, promotionConstants.INVALID_SALE_ID_MSG)
		return
	}

	var req model.UpdateSaleStatusRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	_, sellerID, err := auth.ValidateUserHasSellerRoleOrHigherAndReturnAuthData(c)
	if err != nil {
		h.HandleError(c, err, constants.UNAUTHORIZED_ERROR_MSG)
		return
	}

	response, err := h.service.UpdateStatus(c, saleID, req, sellerID)
	if err != nil {
		h.HandleError(c, err, promotionConstants.FAILED_TO_UPDATE_SALE_STATUS_MSG)
		return
	}

	h.SuccessWithData(
		c,
		http.StatusOK,
		promotionConstants.SALE_STATUS_UPDATED_MSG,
		promotionConstants.SALE_FIELD,
		response,
	)
}
