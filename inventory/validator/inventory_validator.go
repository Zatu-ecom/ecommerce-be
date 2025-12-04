package validator

import (
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
)

// ValidateAdjustmentType validates that the adjustment type is valid
func ValidateAdjustmentType(adjustmentType entity.AdjustmentType) error {
	if !adjustmentType.IsValid() {
		return invErrors.ErrInvalidAdjustmentType
	}
	return nil
}

// ValidateTransactionType validates that the transaction type is valid
func ValidateTransactionType(transactionType entity.TransactionType) error {
	if !transactionType.IsValid() {
		return invErrors.ErrInvalidTransactionType
	}
	return nil
}

// ValidateManualTransactionType validates that the transaction type is allowed for manual adjustments
func ValidateManualTransactionType(transactionType entity.TransactionType) error {
	if !transactionType.IsManualType() {
		return invErrors.ErrNotManualTransaction
	}
	return nil
}

// ValidateAdjustmentRequest validates the adjustment request combination
func ValidateAdjustmentRequest(
	transactionType entity.TransactionType,
	direction *entity.AdjustmentType,
) error {
	// Validate transaction type is allowed
	if err := ValidateManualTransactionType(transactionType); err != nil {
		return err
	}

	// For ADJUSTMENT type, direction is REQUIRED
	if transactionType.RequiresDirection() {
		if direction == nil {
			return invErrors.ErrDirectionRequired
		}
		// Validate the direction value
		if err := ValidateAdjustmentType(*direction); err != nil {
			return err
		}
	} else {
		// For other types, direction should NOT be provided
		if direction != nil {
			return invErrors.ErrDirectionNotAllowed
		}
	}

	return nil
}

// ValidateQuantityForOperation validates if the operation is allowed based on threshold
func ValidateQuantityForOperation(
	currentQuantity int,
	quantityChange int,
	threshold int,
	transactionType entity.TransactionType,
) error {
	newQuantity := currentQuantity + quantityChange

	// Check if operation would result in quantity below threshold
	if newQuantity < threshold {
		return invErrors.ErrBelowThreshold
	}

	return nil
}

// ValidateReservedQuantityForOperation validates reservation operations
func ValidateReservedQuantityForOperation(
	currentReserved int,
	quantityChange int,
) error {
	newReserved := currentReserved + quantityChange

	// Reserved quantity cannot be negative
	if newReserved < 0 {
		return invErrors.ErrInsufficientReservedStock
	}

	return nil
}

// ValidateReferenceRequired validates that reference is provided for system operations
func ValidateReferenceRequired(
	transactionType entity.TransactionType,
	reference *string,
) error {
	// System operations require a reference ID
	if transactionType.RequiresReference() && (reference == nil || *reference == "") {
		return invErrors.ErrReferenceRequired
	}

	return nil
}