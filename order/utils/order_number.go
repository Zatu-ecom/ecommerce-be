package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const (
	orderNumberPrefix = "ORD"
	randomCharset     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomPartLength  = 6
)

// GenerateOrderNumber generates a unique order number in format:
// ORD-<epoch_ms>-<seller_b36>-<random>.
func GenerateOrderNumber(sellerID uint) string {
	epochMillis := time.Now().UTC().UnixMilli()
	sellerPart := EncodeSellerID(sellerID)
	randomPart := generateRandomAlphanumeric(randomPartLength)

	return fmt.Sprintf("%s-%d-%s-%s", orderNumberPrefix, epochMillis, sellerPart, randomPart)
}

// EncodeSellerID encodes seller ID in base36 (uppercase).
func EncodeSellerID(sellerID uint) string {
	return strings.ToUpper(strconv.FormatUint(uint64(sellerID), 36))
}

// DecodeSellerID decodes an encoded base36 seller ID.
func DecodeSellerID(encoded string) (uint, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return 0, fmt.Errorf("encoded seller id is empty")
	}

	parsed, err := strconv.ParseUint(trimmed, 36, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid encoded seller id %q: %w", encoded, err)
	}

	return uint(parsed), nil
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
