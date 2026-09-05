package route

import (
	"ecommerce-be/payment/factory/singleton"
	"ecommerce-be/payment/handler"

	"github.com/gin-gonic/gin"
)

// WebhookModule implements the Module interface for gateway webhook routes.
// Webhooks are public: authenticity is verified via the gateway signature.
type WebhookModule struct {
	webhookHandler *handler.WebhookHandler
}

// NewWebhookModule creates a new WebhookModule.
func NewWebhookModule() *WebhookModule {
	f := singleton.GetInstance()
	return &WebhookModule{
		webhookHandler: f.GetWebhookHandler(),
	}
}

// RegisterRoutes registers webhook routes under /api/payment/webhooks.
func (m *WebhookModule) RegisterRoutes(router *gin.Engine) {
	webhooks := router.Group("/api/payment/webhooks")
	{
		webhooks.POST("/razorpay", m.webhookHandler.HandleRazorpay)
	}
}
