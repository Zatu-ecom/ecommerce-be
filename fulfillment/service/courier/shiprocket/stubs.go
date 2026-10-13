package shiprocket

import (
	courier "ecommerce-be/fulfillment/service/courier"
)

// This package has no remaining stubs: every contract method is implemented
// (NDR/return capabilities arrive as separate optional interfaces in US5).

// compile-time guard: Adapter must satisfy the full contract at every phase.
var _ courier.CourierPartner = (*Adapter)(nil)
