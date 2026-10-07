package fulfillment_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
)

// FulfillmentSuite holds shared state for fulfillment integration tests.
// Story phases extend coverage in their own files against this suite:
// plan (US2), book (US3), webhook/track (US4), NDR/returns (US5).
type FulfillmentSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	sellerClient   *helpers.APIClient
	customerClient *helpers.APIClient
	webhookClient  *helpers.APIClient

	fakeShiprocket *fakeShiprocketServer
	// awbSeq makes fake AWBs unique per call (the (provider, awb) unique
	// index requires what real couriers guarantee).
	awbSeq int64
}

// fakeShiprocketServer mimics the Shiprocket v1/external API for tests.
// The adapter base URL is overridden via SHIPROCKET_BASE_URL (must be set
// before server init). Handlers are programmable per test; unmapped paths
// answer 404 so missing stubs fail loudly instead of passing silently.
type fakeShiprocketServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc // "METHOD path" -> handler
}

// newFakeShiprocket builds the fake courier server.
func newFakeShiprocket() *fakeShiprocketServer {
	f := &fakeShiprocketServer{handlers: map[string]http.HandlerFunc{}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		h, ok := f.handlers[r.Method+" "+r.URL.Path]
		f.mu.Unlock()
		if !ok {
			http.Error(w, `{"message":"unstubbed courier path"}`, http.StatusNotFound)
			return
		}
		h(w, r)
	}))
	return f
}

// stub registers a handler for one method + path.
func (f *fakeShiprocketServer) stub(method, path string, h http.HandlerFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[method+" "+path] = h
}

func (f *fakeShiprocketServer) close() {
	f.server.Close()
}

// SetupSuite boots containers, migrations, seeds, the fake courier, and clients.
func (s *FulfillmentSuite) SetupSuite() {
	// 32-byte key so credential encryption/decryption round-trips.
	_ = os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	s.fakeShiprocket = newFakeShiprocket()

	// Point the Shiprocket adapter at the fake server (read in Phase 2+).
	_ = os.Setenv("SHIPROCKET_BASE_URL", s.fakeShiprocket.server.URL)

	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(), s.sellerClient, helpers.SellerEmail, helpers.SellerPassword,
	)
	s.sellerClient.SetToken(sellerToken)

	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(), s.customerClient, helpers.CustomerEmail, helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)

	// Webhook client carries no auth token; signature is the auth.
	s.webhookClient = helpers.NewAPIClient(s.server)
}

// TearDownSuite releases the fake courier and containers.
func (s *FulfillmentSuite) TearDownSuite() {
	if s.fakeShiprocket != nil {
		s.fakeShiprocket.close()
	}
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

// TestFulfillmentSuite is the single entry point; story phases add methods.
func TestFulfillmentSuite(t *testing.T) {
	suite.Run(t, new(FulfillmentSuite))
}
