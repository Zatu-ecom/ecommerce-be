package helper

import (
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/validator"
)

// ValidateManageRequest validates the manage inventory request
func ValidateManageRequest(req model.ManageInventoryRequest) error {
	if err := validator.ValidateAdjustmentRequest(req.TransactionType, req.Direction); err != nil {
		return err
	}
	return validator.ValidateReferenceRequired(req.TransactionType, req.Reference)
}

// CalculateQuantityChange calculates the quantity change based on transaction type
func CalculateQuantityChange(
	req model.ManageInventoryRequest,
	currentQuantity int,
	isNewInventory bool,
) (int, error) {
	switch req.TransactionType {
	case entity.TXN_ADJUSTMENT:
		return calculateAdjustmentChange(req, isNewInventory)
	case entity.TXN_PURCHASE, entity.TXN_RETURN, entity.TXN_TRANSFER_IN:
		return req.Quantity, nil
	case entity.TXN_SALE, entity.TXN_TRANSFER_OUT, entity.TXN_DAMAGE:
		return -req.Quantity, nil
	case entity.TXN_RESERVED:
		return req.Quantity, nil
	case entity.TXN_RELEASED:
		return -req.Quantity, nil
	case entity.TXN_REFRESH:
		return req.Quantity - currentQuantity, nil
	default:
		return 0, invErrors.ErrInvalidTransactionType
	}
}

// calculateAdjustmentChange handles adjustment type quantity calculation
func calculateAdjustmentChange(
	req model.ManageInventoryRequest,
	isNewInventory bool,
) (int, error) {
	if isNewInventory {
		return req.Quantity, nil
	}
	if req.Direction == nil {
		return 0, invErrors.ErrDirectionRequired
	}
	if *req.Direction == entity.ADJ_ADD {
		return req.Quantity, nil
	}
	return -req.Quantity, nil
}

// ApplyInventoryChanges applies quantity changes to inventory based on transaction type
func ApplyInventoryChanges(
	inventory *entity.Inventory,
	req model.ManageInventoryRequest,
	quantityChange int,
) error {
	updatesReserved := req.TransactionType.UpdatesReservedQuantity()

	if updatesReserved {
		return applyReservedQuantityChange(inventory, quantityChange)
	}
	return applyQuantityChange(inventory, req, quantityChange)
}

// applyReservedQuantityChange applies changes to reserved quantity
func applyReservedQuantityChange(
	inventory *entity.Inventory,
	quantityChange int,
) error {
	if err := validator.ValidateReservedQuantityForOperation(
		inventory.ReservedQuantity, quantityChange,
	); err != nil {
		return err
	}
	inventory.ReservedQuantity += quantityChange
	return nil
}

// applyQuantityChange applies changes to regular quantity
func applyQuantityChange(
	inventory *entity.Inventory,
	req model.ManageInventoryRequest,
	quantityChange int,
) error {
	if req.TransactionType != entity.TXN_REFRESH {
		if err := validator.ValidateQuantityForOperation(
			inventory.Quantity, quantityChange, inventory.Threshold, req.TransactionType,
		); err != nil {
			return err
		}
	}
	inventory.Quantity += quantityChange
	return nil
}

// UpdateThreshold updates inventory threshold if provided in request
func UpdateThreshold(inventory *entity.Inventory, threshold *int) {
	if threshold != nil {
		inventory.Threshold = *threshold
	}
}

// BuildManageResponse builds the response for manage inventory operation
func BuildManageResponse(
	inventory *entity.Inventory,
	previousQuantity int,
	quantityChange int,
	transactionID uint,
) *model.ManageInventoryResponse {
	return &model.ManageInventoryResponse{
		InventoryID:       inventory.ID,
		PreviousQuantity:  previousQuantity,
		NewQuantity:       inventory.Quantity,
		QuantityChanged:   quantityChange,
		AvailableQuantity: inventory.Quantity - inventory.ReservedQuantity,
		Threshold:         inventory.Threshold,
		TransactionID:     transactionID,
	}
}

// DetermineReferenceType determines the reference type based on transaction type
func DetermineReferenceType(txnType entity.TransactionType) string {
	switch txnType {
	case entity.TXN_ADJUSTMENT, entity.TXN_DAMAGE, entity.TXN_REFRESH:
		return "MANUAL_ADJUSTMENT"
	case entity.TXN_RESERVED, entity.TXN_RELEASED:
		return "ORDER"
	case entity.TXN_PURCHASE:
		return "PURCHASE_ORDER"
	case entity.TXN_SALE:
		return "SALES_ORDER"
	case entity.TXN_RETURN:
		return "RETURN"
	case entity.TXN_TRANSFER_IN, entity.TXN_TRANSFER_OUT:
		return "STOCK_TRANSFER"
	default:
		return "SYSTEM"
	}
}
