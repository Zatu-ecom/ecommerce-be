package payment_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"ecommerce-be/payment/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

const (
	// Fake Razorpay test credentials (must match what the suite seeds into the DB).
	TestKeyID         = "rzp_test_aaaaaaaaaaaaaaaa"
	TestKeySecret     = "test_key_secret_00000000000000000000"
	TestWebhookSecret = "test_webhook_secret_0000000000000000"
)

// PaymentSuite holds shared state for payment integration tests.
type PaymentSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	customerClient *helpers.APIClient
	sellerClient   *helpers.APIClient
	webhookClient  *helpers.APIClient

	fakeRazorpay *fakeRazorpayServer
}

// fakeRazorpayServer mimics the Razorpay Orders API for integration tests.
type fakeRazorpayServer struct {
	server *httptest.Server
	orders map[string]string // receipt → order id
}

func newFakeRazorpay() *fakeRazorpayServer {
	f := &fakeRazorpayServer{orders: map[string]string{}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/orders":
			var req struct {
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
				Receipt  string `json:"receipt"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			orderID := fmt.Sprintf("order_%s", req.Receipt)
			f.orders[req.Receipt] = orderID
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"id":%q,"amount":%d,"currency":%q,"status":"created","receipt":%q}`,
				orderID, req.Amount, req.Currency, req.Receipt)))
		case r.Method == http.MethodPost && len(r.URL.Path) > len("/payments/") &&
			r.URL.Path[len(r.URL.Path)-len("/refund"):] == "/refund":
			// POST /payments/{id}/refund → return a refund id.
			var req struct {
				Amount int64 `json:"amount"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"id":"refund_fake_%d","amount":%d,"currency":"INR","status":"processing"}`,
				req.Amount, req.Amount)))
		default:
			http.NotFound(w, r)
		}
	}))
	return f
}

func (f *fakeRazorpayServer) close() {
	f.server.Close()
}

// signWebhook computes the Razorpay X-Razorpay-Signature for a raw body.
func signWebhook(rawBody []byte) string {
	mac := hmac.New(sha256.New, []byte(TestWebhookSecret))
	mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *PaymentSuite) SetupSuite() {
	// 32-byte key so credential encryption/decryption round-trips (same as file module tests).
	_ = os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	s.fakeRazorpay = newFakeRazorpay()

	// Point the gateway adapter at the fake server (must be set before server init).
	_ = os.Setenv("RAZORPAY_BASE_URL", s.fakeRazorpay.server.URL)

	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	// Seed the seller's Razorpay gateway config with encrypted test credentials.
	s.seedSellerGatewayConfig()

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(), s.customerClient, helpers.CustomerEmail, helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)

	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(), s.sellerClient, helpers.SellerEmail, helpers.SellerPassword,
	)
	s.sellerClient.SetToken(sellerToken)

	// Webhook client carries no auth token; signature is the auth.
	s.webhookClient = helpers.NewAPIClient(s.server)
}

func (s *PaymentSuite) TearDownSuite() {
	if s.fakeRazorpay != nil {
		s.fakeRazorpay.close()
	}
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *PaymentSuite) SetupTest() {
	s.cleanupPaymentDomainData()
	// Restore the seller-2 active config (used by US1 and US2 tests).
	s.seedSellerGatewayConfig()
}

// seedSellerGatewayConfig inserts an active Razorpay config for the test seller.
// The customer (Alice, user 5) is associated with seller 2 (John), whose base
// currency is INR and business country is IN — matching Razorpay support.
func (s *PaymentSuite) seedSellerGatewayConfig() {
	var gateway entity.PaymentGateway
	err := s.container.DB.Where("code = ?", "razorpay").First(&gateway).Error
	s.Require().NoError(err)

	// Credentials are field-level encrypted at rest; the adapter's DecryptSensitive
	// uses the same ENCRYPTION_KEY. For tests we store the values via the same
	// code path used in production (encrypted), so decryption round-trips.
	encrypted := map[string]any{
		"key_id":         TestKeyID,
		"key_secret":     encryptForTest(TestKeySecret),
		"webhook_secret": encryptForTest(TestWebhookSecret),
	}

	config := entity.PaymentGatewayConfig{
		SellerID:    helpers.Seller2UserID, // seller id 2 (IN/INR) — the customer's seller
		GatewayID:   gateway.ID,
		Environment: entity.EnvironmentSandbox,
		Credentials: encrypted,
		IsActive:    true,
		Priority:    1,
		Country:     "IN",
	}
	s.Require().NoError(s.container.DB.Create(&config).Error)
}

func (s *PaymentSuite) cleanupPaymentDomainData() {
	// Orders created by the suite reference carts; clear order graph + carts for
	// the involved users so each test starts with a clean slate.
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_history`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_item_applied_promotion`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_applied_coupon`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_applied_promotion`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_address`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM order_item`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM "order"`).Error)

	s.Require().NoError(s.container.DB.Exec(`DELETE FROM payment_transaction_event`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM payment_webhook_log`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM payment_refund`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM payment_transaction`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM payment_gateway_config`).Error)

	// Clear carts (children first) for the test customers.
	var cartIDs []uint
	s.Require().NoError(
		s.container.DB.Table("cart").
			Where("user_id IN ?", []uint{helpers.CustomerUserID, helpers.Customer2UserID}).
			Pluck("id", &cartIDs).Error,
	)
	if len(cartIDs) > 0 {
		s.Require().NoError(s.container.DB.Exec(
			`DELETE FROM cart_item_promotion WHERE cart_item_id IN (SELECT id FROM cart_item WHERE cart_id IN ?)`,
			cartIDs).Error)
		s.Require().NoError(s.container.DB.Exec(
			`DELETE FROM cart_applied_coupon WHERE cart_id IN ?`, cartIDs).Error)
		s.Require().NoError(s.container.DB.Exec(
			`DELETE FROM cart_item WHERE cart_id IN ?`, cartIDs).Error)
		s.Require().NoError(s.container.DB.Exec(`DELETE FROM cart WHERE id IN ?`, cartIDs).Error)
	}

	s.Require().NoError(s.container.DB.Exec(`DELETE FROM inventory_reservation`).Error)
	s.Require().NoError(s.container.DB.Exec(`DELETE FROM inventory_transaction`).Error)
}

func TestPaymentSuite(t *testing.T) {
	suite.Run(t, new(PaymentSuite))
}
