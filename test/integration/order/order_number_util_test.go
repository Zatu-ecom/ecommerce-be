package order_test

import (
	"regexp"
	"strings"
	"testing"

	orderUtils "ecommerce-be/order/utils"
)

func TestOrderNumberUtilGenerateFormat(t *testing.T) {
	orderNumber := orderUtils.GenerateOrderNumber(12345)

	pattern := regexp.MustCompile(`^ORD-\d{13}-[0-9A-Z]+-[0-9A-Z]{6}$`)
	if !pattern.MatchString(orderNumber) {
		t.Fatalf("order number format mismatch: %s", orderNumber)
	}
}

func TestOrderNumberUtilEncodeDecodeSellerID(t *testing.T) {
	input := uint(987654)
	encoded := orderUtils.EncodeSellerID(input)

	if encoded == "" {
		t.Fatalf("encoded seller id must not be empty")
	}
	if encoded != strings.ToUpper(encoded) {
		t.Fatalf("encoded seller id must be uppercase: %s", encoded)
	}

	decoded, err := orderUtils.DecodeSellerID(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if decoded != input {
		t.Fatalf("decoded seller id mismatch: got=%d want=%d", decoded, input)
	}
}

func TestOrderNumberUtilDecodeInvalid(t *testing.T) {
	_, err := orderUtils.DecodeSellerID("!@#")
	if err == nil {
		t.Fatalf("expected error for invalid encoded seller id")
	}
}

