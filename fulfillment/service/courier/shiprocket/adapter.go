package shiprocket

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"ecommerce-be/fulfillment/utils/constant"

	"golang.org/x/sync/singleflight"
)

// Adapter implements courier.CourierPartner for Shiprocket.
// Operational methods (rates, booking, tracking, webhook, NDR) land with
// the user-story phases; this file owns the struct, construction, and Code.
type Adapter struct {
	code    string
	BaseURL string
	client  *http.Client

	tokenMu sync.RWMutex
	tokens  map[string]tokenEntry
	// loginFlight collapses concurrent logins for the same credential key
	// (thundering herd on expiry) to one provider POST per pod.
	loginFlight singleflight.Group
}

// New builds the adapter with a configurable base URL and a pooled HTTP client.
// An empty baseURL falls back to the production Shiprocket API.
func New(baseURL string) *Adapter {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Adapter{
		code:    constant.COURIER_CODE_SHIPROCKET,
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second, // outer bound; per-call contexts enforce GET 5s / POST 10s
		},
	}
}

// Code returns the provider code ("shiprocket").
func (a *Adapter) Code() string {
	return a.code
}
