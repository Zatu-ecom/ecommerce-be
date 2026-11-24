package factory

import (
	fulfillmenterrors "ecommerce-be/fulfillment/error"
)

// errProviderNotSupported reports an unregistered provider code.
// WithMessagef returns a copy (never mutates the shared var), preserving
// the AppError type handlers map on — wrapping with %w would downgrade
// this 404 to a 500.
func errProviderNotSupported(code string) error {
	return fulfillmenterrors.ErrorProviderNotSupported.WithMessagef("unknown courier: %s", code)
}

// errProviderNotActive reports a registered but deactivated provider.
func errProviderNotActive(code string) error {
	return fulfillmenterrors.ErrorProviderNotConfigured.WithMessagef("courier deactivated: %s", code)
}
