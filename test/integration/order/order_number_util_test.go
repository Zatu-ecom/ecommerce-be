package order_test

import (
	"regexp"
	"strings"
	"testing"

	orderUtils "ecommerce-be/order/utils"
)

func TestOrderNumberUtilGenerateFormat(t *testing.T) {
	orderNumber := orderUtils.GenerateOrderNumber(12345)

	pattern := regexp.MustCompile(`^ORD-\d{13}-[0-9A-Z]{10}-[0-9A-Z]{4}$`)
	if !pattern.MatchString(orderNumber) {
		t.Fatalf("order number format mismatch: %s", orderNumber)
	}
}

func TestOrderNumberUtilEncodeSellerIDDeterministic(t *testing.T) {
	input := uint(987654)
	encodedOne := orderUtils.EncodeSellerID(input)
	encodedTwo := orderUtils.EncodeSellerID(input)

	if encodedOne == "" {
		t.Fatalf("encoded seller id must not be empty")
	}
	if encodedOne != strings.ToUpper(encodedOne) {
		t.Fatalf("encoded seller id must be uppercase: %s", encodedOne)
	}
	if encodedOne != encodedTwo {
		t.Fatalf("hash encoding must be deterministic: %s != %s", encodedOne, encodedTwo)
	}
	if len(encodedOne) != 10 {
		t.Fatalf("expected hash length 10, got %d (%s)", len(encodedOne), encodedOne)
	}
}

func TestOrderNumberUtilDecodeNotSupported(t *testing.T) {
	_, err := orderUtils.DecodeSellerID("ANYVALUE")
	if err == nil {
		t.Fatalf("expected error for non-reversible seller hash")
	}
}
