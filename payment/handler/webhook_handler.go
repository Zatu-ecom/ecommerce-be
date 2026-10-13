package handler

import (
	"io"
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	"ecommerce-be/common/model"
	paymentService "ecommerce-be/payment/service"
	paymentConstant "ecommerce-be/payment/utils/constant"

	"github.com/gin-gonic/gin"
)

// WebhookHandler receives inbound gateway callbacks.
type WebhookHandler struct {
	*handler.BaseHandler
	webhookService paymentService.WebhookService
}

// NewWebhookHandler creates the webhook handler.
func NewWebhookHandler(base *handler.BaseHandler, svc paymentService.WebhookService) *WebhookHandler {
	return &WebhookHandler{BaseHandler: base, webhookService: svc}
}

// HandleWebhook processes a provider webhook for the :code in the path.
// The route is public; authenticity comes from the provider signature, which
// the adapter verifies against the transacting seller's secret. The handler
// reads the raw body and forwards headers untouched — no provider-specific
// signature parsing lives at this layer.
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.HandleError(c, err, paymentConstant.FAILED_TO_READ_WEBHOOK_BODY_MSG)
		return
	}

	ipAddress := c.ClientIP()

	// The adapter reads its own signature/event headers from the raw request;
	// the handler stays provider-agnostic (no signature parsing here).
	err = h.webhookService.HandleWebhook(
		c, c.Param(paymentConstant.PARAM_CODE), rawBody, c.Request.Header, ipAddress)
	if err != nil {
		// A verification failure is a client error (401); an unknown gateway
		// code stays 404. Everything else is logged and answered 200:
		// verified deliveries must not be retried by the provider, and
		// processing errors are recorded in the webhook log.
		if isWebhookAuthError(err) {
			model.ErrorWithCode(c, http.StatusUnauthorized, err.Error(), paymentConstant.INVALID_WEBHOOK_SIGNATURE_CODE)
			return
		}
		if appErr, ok := commonError.AsAppError(err); ok && isWebhookNotFoundError(err) {
			model.ErrorWithCode(c, http.StatusNotFound, appErr.Message, appErr.Code)
			return
		}
		log.ErrorWithContext(c, "webhook processing failed", err)
		// Idempotent contract: verified-but-unknown events still return 200 so the
		// provider stops retrying. Processing errors are recorded in the webhook log.
		model.SuccessResponse(c, http.StatusOK, paymentConstant.WEBHOOK_RECEIVED_MSG, nil)
		return
	}

	model.SuccessResponse(c, http.StatusOK, paymentConstant.WEBHOOK_RECEIVED_MSG, nil)
}
