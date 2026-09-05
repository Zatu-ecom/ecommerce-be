package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ecommerce-be/common/log"
	paymenterrors "ecommerce-be/payment/error"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/utils/constant"
)

// RazorpayGateway implements PaymentGateway for Razorpay (India / INR).
type RazorpayGateway struct {
	code    string
	BaseURL string
	Client  *http.Client
}

// NewRazorpayGateway builds the adapter with a configurable base URL and HTTP client.
// baseURL defaults to the production Razorpay API; tests override it.
func NewRazorpayGateway(baseURL string) *RazorpayGateway {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = constant.RAZORPAY_BASE_URL
	}
	return &RazorpayGateway{
		code:    constant.GATEWAY_CODE_RAZORPAY,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ─── Interface implementation ──────────────────────────────────────────────────

func (g *RazorpayGateway) Code() string {
	return g.code
}

// InitiatePayment maps to POST {base}/orders. `receipt` is our internal transaction_id,
// which Razorpay treats as an idempotency key on the account.
func (g *RazorpayGateway) InitiatePayment(
	ctx context.Context,
	in paymentModel.InitiatePaymentInput,
) (*paymentModel.InitiatePaymentOutput, error) {
	creds, err := ParseRazorpayCredentials(in.Credentials)
	if err != nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured.WithMessagef(
			"[razorpay] %v", err)
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

	rawResp, err := g.doJSON(ctx, creds, http.MethodPost, "/orders", body)
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
	}, nil
}

// FetchPayment maps to GET {base}/payments/{id}.
func (g *RazorpayGateway) FetchPayment(
	ctx context.Context,
	gatewayPaymentID string,
	credentials map[string]any,
) (*paymentModel.PaymentStatusOutput, error) {
	creds, err := ParseRazorpayCredentials(credentials)
	if err != nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured.WithMessagef(
			"[razorpay] %v", err)
	}

	rawResp, err := g.doJSON(ctx, creds, http.MethodGet,
		fmt.Sprintf("/payments/%s", gatewayPaymentID), nil)
	if err != nil {
		return nil, err
	}

	status, _ := rawResp["status"].(string)
	amount, _ := rawResp["amount"].(float64)
	currency, _ := rawResp["currency"].(string)
	fee, _ := rawResp["fee"].(float64)
	method, _ := rawResp["method"].(string)

	feeCents := int64(fee)
	return &paymentModel.PaymentStatusOutput{
		GatewayPaymentID: gatewayPaymentID,
		Status:           status,
		AmountCents:      int64(amount),
		Currency:         currency,
		GatewayFeeCents:  &feeCents,
		PaymentMethod:    method,
		GatewayResponse:  rawResp,
	}, nil
}

// Refund maps to POST {base}/payments/{id}/refund.
func (g *RazorpayGateway) Refund(
	ctx context.Context,
	refundType RefundType,
	in paymentModel.RefundInput,
) (*paymentModel.RefundOutput, error) {
	creds, err := ParseRazorpayCredentials(in.Credentials)
	if err != nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured.WithMessagef(
			"[razorpay] %v", err)
	}

	body := map[string]any{
		"amount": in.AmountCents,
		"notes":  in.Notes,
	}
	rawResp, err := g.doJSON(ctx, creds, http.MethodPost,
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
		Currency:        currency,
		GatewayResponse: rawResp,
	}, nil
}

// VerifyWebhook verifies X-Razorpay-Signature = HMAC-SHA256(rawBody, webhook_secret) hex.
func (g *RazorpayGateway) VerifyWebhook(
	rawBody []byte,
	signature string,
	credentials map[string]any,
) (bool, error) {
	if strings.TrimSpace(signature) == "" {
		return false, nil
	}
	creds, err := ParseRazorpayCredentials(credentials)
	if err != nil {
		return false, err
	}

	mac := hmac.New(sha256.New, []byte(creds.WebhookSecret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))), nil
}

// ParseWebhook normalizes a Razorpay webhook body into a gateway-agnostic WebhookEvent.
func (g *RazorpayGateway) ParseWebhook(rawBody []byte) (*paymentModel.WebhookEvent, error) {
	var envelope struct {
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return nil, fmt.Errorf("razorpay: parse webhook body: %w", err)
	}
	if envelope.Event == "" {
		return nil, fmt.Errorf("razorpay: webhook missing event type")
	}

	event := &paymentModel.WebhookEvent{
		GatewayCode: g.code,
		EventType:   envelope.Event,
		Payload:     envelope.Payload,
	}

	// Extract the entity for the event family: payment.* → payload.payment.entity,
	// refund.* → payload.refund.entity.
	entityKey := "payment"
	if strings.HasPrefix(envelope.Event, "refund.") {
		entityKey = "refund"
	}
	entity := map[string]any{}
	if container, ok := envelope.Payload[entityKey].(map[string]any); ok {
		entity = container
	}
	if container, ok := envelope.Payload[entityKey+"_entity"].(map[string]any); ok {
		entity = container
	}
	// Razorpay nests the entity under payload.<key>.entity — unwrap one level.
	if nested, ok := entity["entity"].(map[string]any); ok {
		entity = nested
	}

	event.EventID, _ = envelope.Payload["id"].(string)
	if event.EventID == "" {
		event.EventID, _ = entity["id"].(string)
	}

	if id, ok := entity["id"].(string); ok {
		if entityKey == "refund" {
			event.RefundID = id
		} else {
			event.PaymentID = id
		}
	}
	if orderID, ok := entity["order_id"].(string); ok {
		event.SessionID = orderID
	}
	if paymentID, ok := entity["payment_id"].(string); ok {
		event.PaymentID = paymentID
	}
	if amount, ok := entity["amount"].(float64); ok {
		event.AmountCents = int64(amount)
	}
	if currency, ok := entity["currency"].(string); ok {
		event.Currency = currency
	}
	if fee, ok := entity["fee"].(float64); ok {
		feeCents := int64(fee)
		event.FeeCents = &feeCents
	}
	if failureCode, ok := entity["error_code"].(string); ok {
		event.FailureCode = failureCode
	}
	if failureMessage, ok := entity["error_description"].(string); ok {
		event.FailureMessage = failureMessage
	}

	// notes.transaction_id carries our internal reference when present.
	if notes, ok := entity["notes"].(map[string]any); ok {
		if txID, ok := notes["transaction_id"].(string); ok {
			event.TransactionID = txID
		}
	}

	return event, nil
}

// ─── HTTP helper ───────────────────────────────────────────────────────────────

// doJSON performs an authenticated request and returns the decoded JSON body.
func (g *RazorpayGateway) doJSON(
	ctx context.Context,
	creds *paymentModel.RazorpayCredentials,
	method, path string,
	body any,
) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("razorpay: marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("razorpay: build request: %w", err)
	}
	req.SetBasicAuth(creds.KeyID, creds.KeySecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("razorpay: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("razorpay: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.WarnWithContext(ctx,
			"razorpay api error status="+resp.Status+" path="+path+" body="+string(respBody))
		return nil, fmt.Errorf("razorpay: api error status=%d path=%s", resp.StatusCode, path)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("razorpay: parse response: %w", err)
	}
	return result, nil
}
