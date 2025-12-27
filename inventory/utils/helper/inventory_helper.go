package helper

import (
	"sort"
	"strings"

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
	case entity.TXN_OUTBOUND, entity.TXN_TRANSFER_OUT, entity.TXN_DAMAGE:
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
	txnType := req.TransactionType

	// OUTBOUND (SALE) updates BOTH quantities:
	// - Decreases reserved_quantity (release the reservation)
	// - Decreases quantity (ship the actual stock)
	if txnType.UpdatesBothQuantities() {
		// First release the reserved quantity
		if err := applyReservedQuantityChange(inventory, quantityChange); err != nil {
			return err
		}
		// Then decrease the actual quantity
		return applyQuantityChange(inventory, req, quantityChange)
	}

	// RESERVED and RELEASED only update reserved_quantity
	if txnType.UpdatesReservedQuantity() {
		return applyReservedQuantityChange(inventory, quantityChange)
	}

	// All other types update only regular quantity
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
	case entity.TXN_OUTBOUND:
		return "SALES_ORDER"
	case entity.TXN_RETURN:
		return "RETURN"
	case entity.TXN_TRANSFER_IN, entity.TXN_TRANSFER_OUT:
		return "STOCK_TRANSFER"
	default:
		return "SYSTEM"
	}
}

// ============================================================================
// String Utility Functions
// ============================================================================

// ContainsIgnoreCase checks if s contains substr (case-insensitive)
func ContainsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ============================================================================
// Stock Status Calculation Functions
// ============================================================================

// CalculateOverallStockStatus determines overall stock status based on variant counts
func CalculateOverallStockStatus(outOfStock, lowStock, total uint) model.StockStatus {
	if total == 0 || outOfStock == total {
		return model.StockStatusOutOfStock
	}
	if outOfStock > 0 || lowStock > 0 {
		return model.StockStatusLowStock
	}
	return model.StockStatusInStock
}

// ============================================================================
// Sorting Functions
// ============================================================================

// SortVariantsWithInventory sorts variants using Go's sort.Slice
func SortVariantsWithInventory(variants []model.VariantWithInventory, sortBy, sortOrder string) {
	if len(variants) <= 1 {
		return
	}

	isAsc := sortOrder == "asc"

	sort.Slice(variants, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "sku":
			cmp = strings.Compare(variants[i].SKU, variants[j].SKU)
		case "variantName":
			cmp = strings.Compare(variants[i].VariantName, variants[j].VariantName)
		case "quantity":
			cmp = variants[i].Inventory.Quantity - variants[j].Inventory.Quantity
		case "reservedQuantity":
			cmp = variants[i].Inventory.ReservedQuantity - variants[j].Inventory.ReservedQuantity
		case "availableQuantity":
			cmp = variants[i].Inventory.AvailableQuantity - variants[j].Inventory.AvailableQuantity
		case "threshold":
			cmp = variants[i].Inventory.Threshold - variants[j].Inventory.Threshold
		default:
			cmp = strings.Compare(variants[i].SKU, variants[j].SKU)
		}

		if isAsc {
			return cmp < 0
		}
		return cmp > 0
	})
}

// SortProductSummaries sorts product summaries using Go's sort.Slice
func SortProductSummaries(summaries []model.ProductInventorySummary, sortBy, sortOrder string) {
	if len(summaries) <= 1 {
		return
	}

	isAsc := sortOrder == "asc"

	sort.Slice(summaries, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "totalStock":
			cmp = summaries[i].TotalStock - summaries[j].TotalStock
		case "totalAvailable":
			cmp = summaries[i].TotalAvailable - summaries[j].TotalAvailable
		case "lowStockVariants":
			cmp = int(summaries[i].LowStockVariants) - int(summaries[j].LowStockVariants)
		case "outOfStockVariants":
			cmp = int(summaries[i].OutOfStockVariants) - int(summaries[j].OutOfStockVariants)
		case "variantCount":
			cmp = int(summaries[i].VariantCount) - int(summaries[j].VariantCount)
		case "productName":
			cmp = strings.Compare(summaries[i].ProductName, summaries[j].ProductName)
		case "baseSku":
			cmp = strings.Compare(summaries[i].BaseSKU, summaries[j].BaseSKU)
		default:
			cmp = strings.Compare(summaries[i].ProductName, summaries[j].ProductName)
		}

		if isAsc {
			return cmp < 0
		}
		return cmp > 0
	})
}

// ============================================================================
// Filter Functions
// ============================================================================

// FilterVariantsByStockStatus filters variants by stock status
func FilterVariantsByStockStatus(variants []model.VariantWithInventory, stockStatus string) []model.VariantWithInventory {
	if stockStatus == "all" || stockStatus == "" {
		return variants
	}

	filtered := make([]model.VariantWithInventory, 0, len(variants))
	for _, v := range variants {
		if string(v.Inventory.StockStatus) == stockStatus {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// FilterVariantsBySearch filters variants by search term (SKU or variant name)
func FilterVariantsBySearch(variants []model.VariantWithInventory, search string) []model.VariantWithInventory {
	if search == "" {
		return variants
	}

	filtered := make([]model.VariantWithInventory, 0, len(variants))
	for _, v := range variants {
		if ContainsIgnoreCase(v.SKU, search) || ContainsIgnoreCase(v.VariantName, search) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// ApplyVariantFilters applies all filters to variants
func ApplyVariantFilters(variants []model.VariantWithInventory, filter model.VariantInventoryFilter) []model.VariantWithInventory {
	result := FilterVariantsByStockStatus(variants, filter.StockStatus)
	result = FilterVariantsBySearch(result, filter.Search)
	return result
}
