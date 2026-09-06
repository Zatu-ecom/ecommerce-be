package handler

import (
	"net/http"
	"strings"

	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	paymentModel "ecommerce-be/payment/model"
	paymentService "ecommerce-be/payment/service"
	paymentConstant "ecommerce-be/payment/utils/constant"

	"github.com/gin-gonic/gin"
)

// PaymentHandler exposes customer/seller payment endpoints.
type PaymentHandler struct {
	*handler.BaseHandler
	paymentService paymentService.PaymentService
}

// NewPaymentHandler creates the payment handler.
func NewPaymentHandler(base *handler.BaseHandler, svc paymentService.PaymentService) *PaymentHandler {
	return &PaymentHandler{BaseHandler: base, paymentService: svc}
}

// InitiatePayment starts a payment for a pending order.
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}
	sellerID, sellerExists := auth.GetSellerIDFromContext(c)
	if !sellerExists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	var req paymentModel.InitiatePaymentRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.paymentService.InitiatePayment(c, userID, sellerID, req)
	if err != nil {
		log.ErrorWithContext(c, "initiatePayment: failed", err)
		h.HandleError(c, err, paymentConstant.FAILED_TO_INITIATE_PAYMENT_MSG)
		return
	}

	h.Success(c, http.StatusOK, paymentConstant.PAYMENT_INITIATED_MSG, resp)
}

// GetPaymentStatus returns the current state of a payment.
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.UnauthorizedError, constants.AUTHENTICATION_REQUIRED_MSG)
		return
	}

	_, roleName, _ := auth.GetUserRoleFromContext(c)

	transactionID := c.Param(paymentConstant.PARAM_TRANSACTION_ID)
	if strings.TrimSpace(transactionID) == "" {
		h.HandleValidationError(c, nil)
		return
	}

	resp, err := h.paymentService.GetPaymentStatus(c, userID, roleName, transactionID)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_GET_PAYMENT_STATUS_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.PAYMENT_STATUS_FETCHED_MSG, resp)
}

// ListSellerTransactions lists the authenticated seller's payments.
// Supports server-side ?status= filtering; list rows never carry events.
func (h *PaymentHandler) ListSellerTransactions(c *gin.Context) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	page := parsePositiveInt(c.Query(paymentConstant.QUERY_PAGE), 1)
	pageSize := parsePositiveInt(c.Query(paymentConstant.QUERY_PAGE_SIZE), 20)
	status := c.Query(paymentConstant.QUERY_STATUS)

	txns, total, err := h.paymentService.ListSellerTransactions(c, sellerID, page, pageSize, status)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_LIST_TRANSACTIONS_MSG)
		return
	}

	h.Success(c, http.StatusOK, paymentConstant.TRANSACTIONS_FETCHED_MSG, map[string]any{
		"items":    txns,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// ListWebhookLogs lists the authenticated seller's webhook deliveries.
func (h *PaymentHandler) ListWebhookLogs(c *gin.Context) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	page := parsePositiveInt(c.Query(paymentConstant.QUERY_PAGE), 1)
	pageSize := parsePositiveInt(c.Query(paymentConstant.QUERY_PAGE_SIZE), 20)
	status := c.Query(paymentConstant.QUERY_STATUS)
	eventType := c.Query(paymentConstant.QUERY_EVENT_TYPE)

	logs, total, err := h.paymentService.ListWebhookLogs(c, sellerID, page, pageSize, status, eventType)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_LIST_WEBHOOK_LOGS_MSG)
		return
	}

	h.Success(c, http.StatusOK, paymentConstant.WEBHOOK_LOGS_FETCHED_MSG, map[string]any{
		"items":    logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Refund requests a full or partial refund for a completed payment.
func (h *PaymentHandler) Refund(c *gin.Context) {
	sellerID, exists := auth.GetSellerIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrSellerDataMissing, constants.SELLER_DATA_MISSING_MSG)
		return
	}

	var req paymentModel.RefundRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	resp, err := h.paymentService.InitiateRefund(c, sellerID, req)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_REFUND_PAYMENT_MSG)
		return
	}
	h.Success(c, http.StatusOK, paymentConstant.REFUND_INITIATED_MSG, resp)
}

func parsePositiveInt(raw string, fallback int) int {
	value := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return fallback
		}
		value = value*10 + int(ch-'0')
	}
	if value <= 0 {
		return fallback
	}
	return value
}
