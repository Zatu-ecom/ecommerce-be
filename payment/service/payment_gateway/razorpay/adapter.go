package razorpay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	paymentModel "ecommerce-be/payment/model"
	gateway "ecommerce-be/payment/service/payment_gateway"
	"ecommerce-be/payment/utils/constant"
)

// defaultBaseURL is the production Razorpay API endpoint. Tests override it
// through New (service_factory wires RAZORPAY_BASE_URL).
const defaultBaseURL = "https://api.razorpay.com/v1"

// Adapter implements gateway.PaymentGateway for Razorpay (India / INR).
type Adapter struct {
	code    string
	BaseURL string
	client  *http.Client
}

// New builds the adapter with a configurable base URL and a pooled HTTP client.
// An empty baseURL falls back to the production Razorpay API.
func New(baseURL string) *Adapter {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Adapter{
		code:    constant.GATEWAY_CODE_RAZORPAY,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// Code returns the provider code ("razorpay").
func (a *Adapter) Code() string {
	return a.code
}

// InitiatePayment maps to POST {base}/orders. `receipt` is our internal
// transaction_id, which Razorpay treats as an idempotency key on the account.
// The checkout payload (keyId + orderId) is adapter-defined: other providers
// return their own fields (e.g. clientSecret) without touching orchestrators.
func (a *Adapter) InitiatePayment(
	ctx context.Context,
	in paymentModel.InitiatePaymentInput,
) (*paymentModel.InitiatePaymentOutput, error) {
	creds, err := parseCredentials(in.Credentials)
	if err != nil {
		return nil, fmt.Errorf("[razorpay] %w", err)
	}

	body := map[string]any{
		"amount":   in.AmountCents,
		"currency": in.Currency,
		"receipt":  in.TransactionID,
		"notes": map[string]any{
			"transaction_id": in.TransactionID,
			"reference_type": in.ReferenceType,
			"reference_id":   in.ReferenceID,
		},
	}

	rawResp, err := a.doJSON(ctx, creds, http.MethodPost, "/orders", body)
	if err != nil {
		return nil, err
	}

	orderID, _ := rawResp["id"].(string)
	if orderID == "" {
		return nil, fmt.Errorf("razorpay: order id missing in create-order response")
	}
	status, _ := rawResp["status"].(string)

	return &paymentModel.InitiatePaymentOutput{
		GatewaySessionID: orderID,
		Status:           status,
		AmountCents:      in.AmountCents,
		Currency:         in.Currency,
		GatewayResponse:  rawResp,
		Checkout: paymentModel.CheckoutPayload{
			GatewayCode: a.code,
			Fields: map[string]any{
				"keyId":   creds.KeyID,
				"orderId": orderID,
			},
		},
	}, nil
}

// Refund maps to POST {base}/payments/{id}/refund. No retry: the provider
// dedupes by amount+payment, and a retried POST risks double refunds.
func (a *Adapter) Refund(
	ctx context.Context,
	in paymentModel.RefundInput,
) (*paymentModel.RefundOutput, error) {
	creds, err := parseCredentials(in.Credentials)
	if err != nil {
		return nil, fmt.Errorf("[razorpay] %w", err)
	}

	body := map[string]any{
		"amount": in.AmountCents,
		"notes":  in.Notes,
	}
	rawResp, err := a.doJSON(ctx, creds, http.MethodPost,
		fmt.Sprintf("/payments/%s/refund", in.GatewayPaymentID), body)
	if err != nil {
		return nil, err
	}

	refundID, _ := rawResp["id"].(string)
	status, _ := rawResp["status"].(string)
	amount, _ := rawResp["amount"].(float64)
	currency, _ := rawResp["currency"].(string)

	return &paymentModel.RefundOutput{
		GatewayRefundID: refundID,
		Status:          status,
		AmountCents:     int64(amount),
		Currency:        strings.ToUpper(currency),
		GatewayResponse: rawResp,
	}, nil
}

// TestConnection verifies key_id/key_secret with an authenticated
// GET {base}/orders?count=1. Provider 401 surfaces as a validation error so
// the handler maps it to 400 (bad keys), not 500. The webhook_secret is
// checked for length/format only — it cannot be verified without the provider
// calling back. Nothing is persisted.
func (a *Adapter) TestConnection(ctx context.Context, creds map[string]any) error {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return fmt.Errorf("[razorpay] %w", err)
	}
	if secret := strings.TrimSpace(parsed.WebhookSecret); secret != "" && len(secret) < 8 {
		return fmt.Errorf("razorpay: webhook_secret looks truncated (length/format check)")
	}

	if _, err := a.doJSON(ctx, parsed, http.MethodGet, "/orders?count=1", nil); err != nil {
		return err
	}
	return nil
}

// FetchRemoteStatus polls the provider for reconciliation. With a session id
// it reads GET {base}/orders/{sessionID}: paid → payment_completed, still
// open (created/attempted) → ignore, anything else terminal → payment_failed.
// With only a payment id it reads GET {base}/payments/{id}. With only a
// refund id it reads GET {base}/refunds/{id}. EventID is a system composite so
// cron-generated applies stay idempotent.
func (a *Adapter) FetchRemoteStatus(
	ctx context.Context,
	sessionID, paymentID string,
	creds map[string]any,
) (*gateway.NormalizedWebhook, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[razorpay] %w", err)
	}

	n := &gateway.NormalizedWebhook{
		ProviderEvent: "fetch_remote_status",
		Payload:       map[string]any{},
	}

	switch {
	case strings.TrimSpace(sessionID) != "":
		rawResp, err := a.doJSON(ctx, parsed, http.MethodGet,
			fmt.Sprintf("/orders/%s", sessionID), nil)
		if err != nil {
			return nil, err
		}
		n.SessionID = sessionID
		n.Payload = rawResp
		n.AmountCents = int64(numberValue(rawResp["amount"]))
		if currency, _ := rawResp["currency"].(string); currency != "" {
			n.Currency = strings.ToUpper(currency)
		}
		switch status, _ := rawResp["status"].(string); status {
		case "paid":
			n.Action = gateway.WebhookActionPaymentCompleted
		case "created", "attempted", "":
			n.Action = gateway.WebhookActionIgnore
		default:
			n.Action = gateway.WebhookActionPaymentFailed
			n.FailureCode = "REMOTE_" + strings.ToUpper(status)
			n.FailureMessage = "razorpay order " + sessionID + " is " + status
		}
	case strings.TrimSpace(paymentID) != "":
		rawResp, err := a.doJSON(ctx, parsed, http.MethodGet,
			fmt.Sprintf("/payments/%s", paymentID), nil)
		if err != nil {
			return nil, err
		}
		n.PaymentID = paymentID
		n.Payload = rawResp
		n.AmountCents = int64(numberValue(rawResp["amount"]))
		if currency, _ := rawResp["currency"].(string); currency != "" {
			n.Currency = strings.ToUpper(currency)
		}
		if orderID, _ := rawResp["order_id"].(string); orderID != "" {
			n.SessionID = orderID
		}
		switch status, _ := rawResp["status"].(string); status {
		case "captured":
			n.Action = gateway.WebhookActionPaymentCompleted
		case "authorized":
			n.Action = gateway.WebhookActionAuthorized
		case "failed":
			n.Action = gateway.WebhookActionPaymentFailed
			n.FailureCode, _ = rawResp["error_code"].(string)
			n.FailureMessage, _ = rawResp["error_description"].(string)
		default:
			n.Action = gateway.WebhookActionIgnore
		}
	default:
		return nil, fmt.Errorf("razorpay: fetch_remote_status needs a session or payment id")
	}

	n.EventID = "system:" + firstNonEmpty(n.SessionID, n.PaymentID) + ":" + string(n.Action)
	return n, nil
}

// FetchRefundStatus polls GET {base}/refunds/{id} for stuck-refund
// reconciliation: processed → refund_completed, failed → refund_failed,
// anything still open → refund_pending. Implements the optional
// gateway.RefundStatusFetcher capability.
func (a *Adapter) FetchRefundStatus(
	ctx context.Context,
	refundID string,
	creds map[string]any,
) (*gateway.NormalizedWebhook, error) {
	parsed, err := parseCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("[razorpay] %w", err)
	}
	if strings.TrimSpace(refundID) == "" {
		return nil, fmt.Errorf("razorpay: fetch_refund_status needs a refund id")
	}

	rawResp, err := a.doJSON(ctx, parsed, http.MethodGet,
		fmt.Sprintf("/refunds/%s", refundID), nil)
	if err != nil {
		return nil, err
	}

	n := &gateway.NormalizedWebhook{
		ProviderEvent: "fetch_refund_status",
		RefundID:      refundID,
		Payload:       rawResp,
		AmountCents:   int64(numberValue(rawResp["amount"])),
	}
	if currency, _ := rawResp["currency"].(string); currency != "" {
		n.Currency = strings.ToUpper(currency)
	}
	if paymentID, _ := rawResp["payment_id"].(string); paymentID != "" {
		n.PaymentID = paymentID
	}
	switch status, _ := rawResp["status"].(string); status {
	case "processed":
		n.Action = gateway.WebhookActionRefundCompleted
	case "failed":
		n.Action = gateway.WebhookActionRefundFailed
		n.FailureMessage, _ = rawResp["error_description"].(string)
	default:
		n.Action = gateway.WebhookActionRefundPending
	}
	n.EventID = "system:" + refundID + ":" + string(n.Action)
	return n, nil
}

// numberValue coerces provider JSON numbers (float64, json.Number, int) to float64.
func numberValue(v any) float64 {
	switch num := v.(type) {
	case float64:
		return num
	case float32:
		return float64(num)
	case int:
		return float64(num)
	case int64:
		return float64(num)
	case json.Number:
		f, _ := num.Float64()
		return f
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "unknown"
}
