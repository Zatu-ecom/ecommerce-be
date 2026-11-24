package get_product_by_id

// moneyAmount returns the major-unit amount from a nested Money node.
// The response `price` is now a Money object: {amount, amountCents, formatted}.
func moneyAmount(node any) float64 {
	m, ok := node.(map[string]any)
	if !ok {
		return 0
	}
	amt, _ := m["amount"].(float64)
	return amt
}
