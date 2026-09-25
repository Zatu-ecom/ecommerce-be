package middleware

import "net/http"

// CorrelationIDWebhookSkipRules exempts inbound payment webhooks from the
// strict X-Correlation-ID requirement. External providers do not send that
// header; GenerateCorrelationID is used as the fallback instead.
var CorrelationIDWebhookSkipRules = []PathSkipRule{
	{
		Methods:    []string{http.MethodPost},
		PathPrefix: "/api/payment/webhooks/",
	},
}
