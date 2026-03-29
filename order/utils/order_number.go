package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"
	"time"

	"ecommerce-be/common/config"
)

const (
	orderNumberPrefix = "ORD"
	randomCharset     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomPartLength  = 4
	sellerHashLength  = 10
)

// GenerateOrderNumber generates a unique order number in format:
// ORD-<epoch_ms>-<seller_hash>-<random>.
func GenerateOrderNumber(sellerID uint) string {
	epochMillis := time.Now().UTC().UnixMilli()
	sellerPart := EncodeSellerID(sellerID)
	randomPart := generateRandomAlphanumeric(randomPartLength)

	return fmt.Sprintf("%s-%d-%s-%s", orderNumberPrefix, epochMillis, sellerPart, randomPart)
}

// EncodeSellerID returns a deterministic, non-reversible seller hash segment.
// Hash length can be configured using ORDER_NUMBER_SELLER_HASH_LEN (default 10).
func EncodeSellerID(sellerID uint) string {
	secret := resolveHashSecret()

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", sellerID)))
	sum := mac.Sum(nil)

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum)
	encoded = strings.ToUpper(encoded)
	if len(encoded) <= sellerHashLength {
		return encoded
	}
	return encoded[:sellerHashLength]
}

// DecodeSellerID is not supported for hash-based encoding.
func DecodeSellerID(encoded string) (uint, error) {
	return 0, fmt.Errorf("seller hash is non-reversible")
}

func generateRandomAlphanumeric(n int) string {
	if n <= 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(n)

	max := big.NewInt(int64(len(randomCharset)))
	for i := range n {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Fallback to a deterministic but still valid char in very rare entropy failures.
			b.WriteByte(randomCharset[i%len(randomCharset)])
			continue
		}
		b.WriteByte(randomCharset[v.Int64()])
	}

	return b.String()
}

func resolveHashSecret() string {
	cfg := config.Get()
	if cfg != nil && strings.TrimSpace(cfg.Auth.JWTSecret) != "" {
		secret := strings.TrimSpace(cfg.Auth.JWTSecret)
		return secret
	}
	return "order-number-default-secret"
}
