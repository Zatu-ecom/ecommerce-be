package entity

// PaymentMethodType represents the type of payment method used for a transaction.
// (Kept after the saved-cards payment_method table was dropped: the type is
// still recorded per transaction for filtering and capture snapshots.)
type PaymentMethodType string

const (
	PaymentMethodTypeCard        PaymentMethodType = "card"
	PaymentMethodTypeUPI         PaymentMethodType = "upi"
	PaymentMethodTypeWallet      PaymentMethodType = "wallet"
	PaymentMethodTypeBankAccount PaymentMethodType = "bank_account"
)
