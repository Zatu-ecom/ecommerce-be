package utils

import "ecommerce-be/order/entity"

// ValidTransitions defines allowed order status transitions.
var ValidTransitions = map[entity.OrderStatus][]entity.OrderStatus{
	entity.ORDER_STATUS_PENDING: {
		entity.ORDER_STATUS_CONFIRMED,
		entity.ORDER_STATUS_CANCELLED,
		entity.ORDER_STATUS_FAILED,
	},
	entity.ORDER_STATUS_CONFIRMED: {
		entity.ORDER_STATUS_COMPLETED,
		entity.ORDER_STATUS_CANCELLED,
	},
	entity.ORDER_STATUS_COMPLETED: {
		entity.ORDER_STATUS_RETURNED,
	},
}

// IsValidTransition checks if an order status transition is allowed.
func IsValidTransition(from, to entity.OrderStatus) bool {
	next, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, candidate := range next {
		if candidate == to {
			return true
		}
	}
	return false
}

// RequiredFieldsForTransition returns required request fields for a transition.
func RequiredFieldsForTransition(from, to entity.OrderStatus) []string {
	switch {
	case from == entity.ORDER_STATUS_PENDING && to == entity.ORDER_STATUS_CONFIRMED:
		return []string{"transactionId"}
	case from == entity.ORDER_STATUS_PENDING && to == entity.ORDER_STATUS_FAILED:
		return []string{"failureReason"}
	default:
		return []string{}
	}
}
