package payment_test

import (
	"ecommerce-be/common/helper"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// encryptForTest encrypts a value using the resolved ENCRYPTION_KEY so the
// adapter's DecryptSensitive can round-trip it during tests. If no key is
// configured, it stores plaintext (round-trip still works for sandbox tests).
func encryptForTest(plaintext string) string {
	key := gateway.ResolveEncryptionKey()
	if key == "" {
		return plaintext
	}
	enc, err := helper.Encrypt(plaintext, key)
	if err != nil {
		return plaintext
	}
	return enc
}
