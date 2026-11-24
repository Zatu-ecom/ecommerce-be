package razorpay_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ecommerce-be/common/config"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/service/payment_gateway"
	"ecommerce-be/payment/service/payment_gateway/razorpay"
)

const (
	testKeyID         = "rzp_test_aaaaaaaaaaaaaaaa"
	testKeySecret     = "test_key_secret_00000000000000000000"
	testWebhookSecret = "test_webhook_secret_0000000000000000"
)

func testCreds() map[string]any {
	return map[string]any{
		"key_id":         testKeyID,
		"key_secret":     testKeySecret,
		"webhook_secret": testWebhookSecret,
		"account_id":     "acc_test123",
	}
}

func signBody(t *testing.T, rawBody []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

func webhookBody(t *testing.T, event string, entity map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"event": event,
		"payload": map[string]any{
			"payment": map[string]any{"entity": entity},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func headersWith(sig, eventID string) http.Header {
	h := http.Header{}
	if sig != "" {
		h.Set("X-Razorpay-Signature", sig)
	}
	if eventID != "" {
		h.Set("X-Razorpay-Event-Id", eventID)
	}
	return h
}

// ─── HMAC verification ───────────────────────────────────────────────────────

func TestNormalizeWebhookAcceptsValidSignature(t *testing.T) {
	adapter := razorpay.New("")
	rawBody := webhookBody(t, "payment.captured", map[string]any{
		"id":       "pay_valid1",
		"order_id": "order_valid1",
		"amount":   float64(10000),
		"currency": "INR",
	})

	n, err := adapter.NormalizeWebhook(rawBody, headersWith(signBody(t, rawBody, testWebhookSecret), ""), testCreds())
	if err != nil {
		t.Fatalf("valid signature must verify: %v", err)
	}
	if n.Action != gateway.WebhookActionPaymentCompleted {
		t.Fatalf("action = %q, want payment_completed", n.Action)
	}
	if n.PaymentID != "pay_valid1" || n.SessionID != "order_valid1" {
		t.Fatalf("locators not extracted: %+v", n)
	}
	if n.EventID == "" {
		t.Fatal("EventID must always be set for idempotency")
	}
}

func TestNormalizeWebhookRejectsWrongSellerSecret(t *testing.T) {
	adapter := razorpay.New("")
	rawBody := webhookBody(t, "payment.captured", map[string]any{
		"id":       "pay_other1",
		"order_id": "order_other1",
	})

	// Signed with a DIFFERENT seller's secret: must fail closed, no state.
	if _, err := adapter.NormalizeWebhook(
		rawBody,
		headersWith(signBody(t, rawBody, "another_sellers_secret_0000000000"), ""),
		testCreds(),
	); err == nil {
		t.Fatal("wrong-seller signature must be rejected")
	}
}

func TestNormalizeWebhookRejectsMissingSignature(t *testing.T) {
	adapter := razorpay.New("")
	rawBody := webhookBody(t, "payment.captured", map[string]any{"id": "pay_nosig"})

	if _, err := adapter.NormalizeWebhook(rawBody, headersWith("", ""), testCreds()); err == nil {
		t.Fatal("missing signature must be rejected")
	}
}

func TestNormalizeWebhookRejectsTamperedBody(t *testing.T) {
	adapter := razorpay.New("")
	rawBody := webhookBody(t, "payment.captured", map[string]any{"id": "pay_tamper"})
	sig := signBody(t, rawBody, testWebhookSecret)

	tampered := append(append([]byte{}, rawBody...), ' ')
	if _, err := adapter.NormalizeWebhook(tampered, headersWith(sig, ""), testCreds()); err == nil {
		t.Fatal("tampered body must fail HMAC")
	}
}

// ─── Event map ───────────────────────────────────────────────────────────────

func TestNormalizeWebhookEventMap(t *testing.T) {
	adapter := razorpay.New("")
	cases := []struct {
		event string
		want  gateway.WebhookAction
	}{
		{"payment.authorized", gateway.WebhookActionAuthorized},
		{"payment.captured", gateway.WebhookActionPaymentCompleted},
		{"payment.failed", gateway.WebhookActionPaymentFailed},
		{"refund.created", gateway.WebhookActionRefundPending},
		{"refund.processed", gateway.WebhookActionRefundCompleted},
		{"refund.failed", gateway.WebhookActionRefundFailed},
		{"order.paid", gateway.WebhookActionIgnore},
		{"something.entirely_new", gateway.WebhookActionIgnore},
	}
	for _, tc := range cases {
		var rawBody []byte
		if strings.HasPrefix(tc.event, "refund.") {
			rawBody, _ = json.Marshal(map[string]any{
				"event":   tc.event,
				"payload": map[string]any{"refund": map[string]any{"entity": map[string]any{"id": "rf_1"}}},
			})
		} else {
			rawBody = webhookBody(t, tc.event, map[string]any{"id": "pay_1"})
		}
		n, err := adapter.NormalizeWebhook(rawBody, headersWith(signBody(t, rawBody, testWebhookSecret), ""), testCreds())
		if err != nil {
			t.Fatalf("event %q: unexpected error: %v", tc.event, err)
		}
		if n.Action != tc.want {
			t.Errorf("event %q: action = %q, want %q", tc.event, n.Action, tc.want)
		}
		if n.EventID == "" {
			t.Errorf("event %q: EventID must be set", tc.event)
		}
	}
}

func TestNormalizeWebhookPrefersEventIDHeader(t *testing.T) {
	adapter := razorpay.New("")
	rawBody := webhookBody(t, "payment.authorized", map[string]any{"id": "pay_auth1"})

	n, err := adapter.NormalizeWebhook(
		rawBody,
		headersWith(signBody(t, rawBody, testWebhookSecret), "evt_header_123"),
		testCreds(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if n.EventID != "evt_header_123" {
		t.Fatalf("EventID = %q, want header value", n.EventID)
	}
}

func TestNormalizeWebhookCompositeEventIDFallback(t *testing.T) {
	adapter := razorpay.New("")
	// authorized vs captured for the same payment must NOT collide.
	authBody := webhookBody(t, "payment.authorized", map[string]any{"id": "pay_same"})
	capBody := webhookBody(t, "payment.captured", map[string]any{"id": "pay_same"})

	auth, err := adapter.NormalizeWebhook(authBody, headersWith(signBody(t, authBody, testWebhookSecret), ""), testCreds())
	if err != nil {
		t.Fatal(err)
	}
	cap, err := adapter.NormalizeWebhook(capBody, headersWith(signBody(t, capBody, testWebhookSecret), ""), testCreds())
	if err != nil {
		t.Fatal(err)
	}
	if auth.EventID == cap.EventID {
		t.Fatalf("authorized and captured must have distinct EventIDs, both = %q", auth.EventID)
	}
}

// ─── PeekLocators (untrusted, no HMAC) ───────────────────────────────────────

func TestPeekLocatorsExtractsHintsWithoutVerifying(t *testing.T) {
	adapter := razorpay.New("")
	rawBody, _ := json.Marshal(map[string]any{
		"event": "payment.captured",
		"payload": map[string]any{
			"payment": map[string]any{"entity": map[string]any{
				"id":       "pay_peek1",
				"order_id": "order_peek1",
				"notes":    map[string]any{"transaction_id": "TXN_PEEK1"},
			}},
		},
	})

	loc := adapter.PeekLocators(rawBody)
	if loc.PaymentID != "pay_peek1" || loc.SessionID != "order_peek1" || loc.TransactionID != "TXN_PEEK1" {
		t.Fatalf("unexpected locators: %+v", loc)
	}
}

func TestPeekLocatorsToleratesGarbage(t *testing.T) {
	adapter := razorpay.New("")
	loc := adapter.PeekLocators([]byte("not json at all{{{"))
	if loc != (gateway.Locators{}) {
		t.Fatalf("garbage body must yield empty locators, got %+v", loc)
	}
}

// ─── MaskHints ───────────────────────────────────────────────────────────────

func TestMaskHintsNeverLeaksSecrets(t *testing.T) {
	adapter := razorpay.New("")
	stored := map[string]any{
		"key_id":         testKeyID,
		"key_secret":     "ENCRYPTED_BLOB_1",
		"webhook_secret": "ENCRYPTED_BLOB_2",
		"account_id":     "acc_test123",
	}

	hints := adapter.MaskHints(stored)
	for _, k := range []string{"key_secret", "webhook_secret"} {
		if _, ok := hints[k]; ok {
			t.Fatalf("mask must never include %q", k)
		}
		if strings.Contains(strings.ToLower(stringMust(hints)), strings.ToLower("ENCRYPTED_BLOB")) {
			t.Fatalf("encrypted blobs must not appear in hints: %v", hints)
		}
	}
	if hints["account_id"] != "acc_test123" {
		t.Fatalf("account_id must be shown fully, got %v", hints["account_id"])
	}
	keyID, _ := hints["key_id"].(string)
	if !strings.Contains(keyID, testKeyID[:8]) || !strings.Contains(keyID, testKeyID[len(testKeyID)-4:]) {
		t.Fatalf("key_id hint must carry first8+last4, got %q", keyID)
	}
}

// ─── MergePartial ────────────────────────────────────────────────────────────

func TestMergePartialKeepsStoredSecretsOnEmpty(t *testing.T) {
	config.Reset()
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	adapter := razorpay.New("")

	stored, err := adapter.Encrypt(testCreds())
	if err != nil {
		t.Fatal(err)
	}

	merged, err := adapter.MergePartial(stored, map[string]any{"key_id": "rzp_test_newkey0000000000"})
	if err != nil {
		t.Fatal(err)
	}
	// Merged output is RAW: the pipeline is MergePartial → Validate → Encrypt.
	if err := adapter.Validate(merged, false); err != nil {
		t.Fatalf("merged credentials must validate: %v", err)
	}
	restored, err := adapter.Encrypt(merged)
	if err != nil {
		t.Fatal(err)
	}
	// Empty/omitted sensitive keys keep existing values; decrypt must round-trip.
	decrypted, err := adapter.Decrypt(restored)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted["key_secret"] != testKeySecret || decrypted["webhook_secret"] != testWebhookSecret {
		t.Fatalf("stored secrets must survive partial update: %v", decrypted)
	}
	if decrypted["key_id"] != "rzp_test_newkey0000000000" {
		t.Fatalf("key_id must update, got %v", decrypted["key_id"])
	}
}

func TestMergePartialReplacesProvidedSecrets(t *testing.T) {
	config.Reset()
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	adapter := razorpay.New("")

	stored, err := adapter.Encrypt(testCreds())
	if err != nil {
		t.Fatal(err)
	}

	merged, err := adapter.MergePartial(stored, map[string]any{"webhook_secret": "brand_new_webhook_secret_000000"})
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Validate(merged, false); err != nil {
		t.Fatalf("merged credentials must validate: %v", err)
	}
	restored, err := adapter.Encrypt(merged)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := adapter.Decrypt(restored)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted["webhook_secret"] != "brand_new_webhook_secret_000000" {
		t.Fatalf("provided secret must replace stored one: %v", decrypted)
	}
	if decrypted["key_secret"] != testKeySecret {
		t.Fatalf("untouched secret must survive: %v", decrypted)
	}
}

// ─── Validate ────────────────────────────────────────────────────────────────

func TestValidateRejectsMissingSecretsOnFullSave(t *testing.T) {
	adapter := razorpay.New("")
	err := adapter.Validate(map[string]any{"key_id": testKeyID}, false)
	if err == nil {
		t.Fatal("full save without secrets must fail validation")
	}
}

func TestValidateAllowsPartialUpdate(t *testing.T) {
	adapter := razorpay.New("")
	if err := adapter.Validate(map[string]any{"key_id": "rzp_test_partial000000"}, true); err != nil {
		t.Fatalf("partial update with only key_id must pass: %v", err)
	}
}

// ─── Encrypt/Decrypt round-trip + fail-closed ─────────────────────────────────

func TestEncryptDecryptRoundTrip(t *testing.T) {
	config.Reset()
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	adapter := razorpay.New("")

	stored, err := adapter.Encrypt(testCreds())
	if err != nil {
		t.Fatal(err)
	}
	if stored["key_id"] != testKeyID || stored["account_id"] != "acc_test123" {
		t.Fatalf("public fields must stay plaintext: %v", stored)
	}
	if stored["key_secret"] == testKeySecret || stored["webhook_secret"] == testWebhookSecret {
		t.Fatal("secrets must be encrypted at rest")
	}

	decrypted, err := adapter.Decrypt(stored)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range testCreds() {
		if decrypted[k] != v {
			t.Fatalf("round-trip mismatch for %q: got %v", k, decrypted[k])
		}
	}
}

func TestEncryptFailsClosedWithoutKey(t *testing.T) {
	config.Reset()
	t.Setenv("ENCRYPTION_KEY", "")
	adapter := razorpay.New("")

	if _, err := adapter.Encrypt(testCreds()); err == nil {
		t.Fatal("encrypt without ENCRYPTION_KEY must fail closed (no plaintext secrets)")
	}
	if _, err := adapter.Decrypt(map[string]any{"key_secret": "x"}); err == nil {
		t.Fatal("decrypt without ENCRYPTION_KEY must fail closed")
	}
}

// ─── HTTP behavior (fake server, no real Razorpay) ───────────────────────────

func TestTestConnectionOKAndUnauthorized(t *testing.T) {
	var gotAuth string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path == "/orders" && r.URL.Query().Get("count") == "1" {
			user, pass, _ := r.BasicAuth()
			if user != testKeyID || pass != testKeySecret {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"entity":"collection","count":1,"items":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer fake.Close()

	adapter := razorpay.New(fake.URL)
	if err := adapter.TestConnection(t.Context(), testCreds()); err != nil {
		t.Fatalf("valid creds must pass test connection: %v", err)
	}
	if gotAuth == "" {
		t.Fatal("test connection must use basic auth")
	}

	bad := testCreds()
	bad["key_secret"] = "wrong_secret_0000000000000000000000"
	if err := adapter.TestConnection(t.Context(), bad); err == nil {
		t.Fatal("401 from provider must surface as an error")
	}
}

func TestInitiatePaymentReturnsCheckoutFields(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/orders" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"order_chk1","amount":10000,"currency":"INR","status":"created"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer fake.Close()

	config.Reset()
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	adapter := razorpay.New(fake.URL)
	stored, err := adapter.Encrypt(testCreds())
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := adapter.Decrypt(stored)
	if err != nil {
		t.Fatal(err)
	}

	out, err := adapter.InitiatePayment(t.Context(), paymentModel.InitiatePaymentInput{
		TransactionID: "TXN_CHK1",
		AmountCents:   10000,
		Currency:      "INR",
		ReferenceType: "order",
		ReferenceID:   42,
		CustomerID:    5,
		SellerID:      2,
		Credentials:   decrypted,
	})
	if err != nil {
		t.Fatalf("initiate must succeed against fake: %v", err)
	}
	if out.GatewaySessionID != "order_chk1" {
		t.Fatalf("session id = %q, want order_chk1", out.GatewaySessionID)
	}
	if out.Checkout.GatewayCode != "razorpay" {
		t.Fatalf("checkout gateway code = %q", out.Checkout.GatewayCode)
	}
	if out.Checkout.Fields["keyId"] != testKeyID || out.Checkout.Fields["orderId"] != "order_chk1" {
		t.Fatalf("checkout fields must carry keyId+orderId, got %v", out.Checkout.Fields)
	}
}

func stringMust(v map[string]any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
