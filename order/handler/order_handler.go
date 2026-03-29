package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

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

type OrderHandler struct {
	*handler.BaseHandler
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{
		BaseHandler:  handler.NewBaseHandler(),
		orderService: orderService,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	var req model.CreateOrderRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.orderService.CreateOrder(c, userID, sellerID, req)
	if err != nil {
		log.ErrorWithContext(c, "createOrder: failed", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_CREATE_ORDER_MSG)
		return
	}

	h.Success(c, http.StatusCreated, orderConstants.ORDER_CREATED_MSG, resp)
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}
	_, role, exists := auth.GetUserRoleFromContext(c)
	if !exists {
		h.HandleError(c, errs.ErrRoleDataMissing, constants.ROLE_DATA_MISSING_MSG)
		return
	}

	orderID, err := parseOrderIDParam(c)
	if err != nil {
		h.HandleValidationError(c, errs.ErrInvalidID) 
		return
	}

	resp, serviceErr := h.orderService.GetOrderByID(c, userID, role, orderID)
	if serviceErr != nil {
		log.ErrorWithContext(c, "getOrderByID: failed", serviceErr)
		h.HandleError(c, serviceErr, orderConstants.FAILED_TO_FETCH_ORDER_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.ORDER_FETCHED_MSG, resp)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}
	_, role, exists := auth.GetUserRoleFromContext(c)
	if !exists {
		h.HandleError(c, errs.ErrRoleDataMissing, constants.ROLE_DATA_MISSING_MSG)
		return
	}

	var req model.ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.orderService.ListOrders(c, userID, role, req)
	if err != nil {
		log.ErrorWithContext(c, "listOrders: failed", err)
		h.HandleError(c, err, orderConstants.FAILED_TO_LIST_ORDERS_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.ORDERS_LISTED_MSG, resp)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	orderID, err := parseOrderIDParam(c)
	if err != nil {
		h.HandleValidationError(c, errs.ErrInvalidID)
		return
	}

	var req model.UpdateOrderStatusRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, serviceErr := h.orderService.UpdateOrderStatus(c, sellerID, orderID, req)
	if serviceErr != nil {
		log.ErrorWithContext(c, "updateOrderStatus: failed", serviceErr)
		h.HandleError(c, serviceErr, orderConstants.FAILED_TO_UPDATE_ORDER_STATUS_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.ORDER_STATUS_UPDATED_MSG, resp)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, errs.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}

	orderID, err := parseOrderIDParam(c)
	if err != nil {
		h.HandleValidationError(c, errs.ErrInvalidID)
		return
	}

	var req model.CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		h.HandleValidationError(c, err)
		return
	}

	resp, serviceErr := h.orderService.CancelOrder(c, userID, orderID, req)
	if serviceErr != nil {
		log.ErrorWithContext(c, "cancelOrder: failed", serviceErr)
		h.HandleError(c, serviceErr, orderConstants.FAILED_TO_CANCEL_ORDER_MSG)
		return
	}

	h.Success(c, http.StatusOK, orderConstants.ORDER_CANCELLED_MSG, resp)
}

func parseOrderIDParam(c *gin.Context) (uint, error) {
	orderIDRaw := c.Param("id")
	orderID64, err := strconv.ParseUint(orderIDRaw, 10, 64)
	if err != nil || orderID64 == 0 {
		return 0, errs.ErrInvalidID
	}
	return uint(orderID64), nil
}
