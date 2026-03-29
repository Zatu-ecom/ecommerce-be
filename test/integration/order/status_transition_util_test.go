package order_test

import (
	"reflect"
	"testing"

	"ecommerce-be/order/entity"
	orderUtils "ecommerce-be/order/utils"
)

func TestStatusTransitionUtilIsValidTransition(t *testing.T) {
	cases := []struct {
		name string
		from entity.OrderStatus
		to   entity.OrderStatus
		want bool
	}{
		{"pending to confirmed", entity.ORDER_STATUS_PENDING, entity.ORDER_STATUS_CONFIRMED, true},
		{"pending to failed", entity.ORDER_STATUS_PENDING, entity.ORDER_STATUS_FAILED, true},
		{"pending to completed invalid", entity.ORDER_STATUS_PENDING, entity.ORDER_STATUS_COMPLETED, false},
		{"terminal cancelled to confirmed invalid", entity.ORDER_STATUS_CANCELLED, entity.ORDER_STATUS_CONFIRMED, false},
		{"completed to returned", entity.ORDER_STATUS_COMPLETED, entity.ORDER_STATUS_RETURNED, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := orderUtils.IsValidTransition(tc.from, tc.to)
			if got != tc.want {
				t.Fatalf("IsValidTransition(%s,%s)=%v want=%v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestStatusTransitionUtilRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		from entity.OrderStatus
		to   entity.OrderStatus
		want []string
	}{
		{
			name: "pending to confirmed requires transactionId",
			from: entity.ORDER_STATUS_PENDING,
			to:   entity.ORDER_STATUS_CONFIRMED,
			want: []string{"transactionId"},
		},
		{
			name: "pending to failed requires failureReason",
			from: entity.ORDER_STATUS_PENDING,
			to:   entity.ORDER_STATUS_FAILED,
			want: []string{"failureReason"},
		},
		{
			name: "confirmed to cancelled no required fields",
			from: entity.ORDER_STATUS_CONFIRMED,
			to:   entity.ORDER_STATUS_CANCELLED,
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := orderUtils.RequiredFieldsForTransition(tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("RequiredFieldsForTransition(%s,%s)=%v want=%v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

