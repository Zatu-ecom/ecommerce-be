package factory

import (
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/mapper"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/utils/helper"
	productModel "ecommerce-be/product/model"
)

// ============================================================================
// InventoryResponse Builder
// ============================================================================

// BuildInventoryResponse creates an InventoryResponse from inventory data
func BuildInventoryResponse(
	variantID uint,
	locationID uint,
	quantity int,
	reservedQuantity int,
	threshold int,
	binLocation string,
) model.InventoryResponse {
	return model.InventoryResponse{
		VariantID:         variantID,
		LocationID:        locationID,
		Quantity:          quantity,
		ReservedQuantity:  reservedQuantity,
		Threshold:         threshold,
		AvailableQuantity: quantity - reservedQuantity,
		StockStatus:       model.CalculateStockStatus(quantity, threshold),
		BinLocation:       binLocation,
	}
}

// BuildInventoryResponseFromRow creates an InventoryResponse from a VariantInventoryRow
func BuildInventoryResponseFromRow(
	row mapper.VariantInventoryRow,
	locationID uint,
) model.InventoryResponse {
	return BuildInventoryResponse(
		row.VariantID,
		locationID,
		row.Quantity,
		row.ReservedQuantity,
		row.Threshold,
		row.BinLocation,
	)
}

func BuildInventoryResponseFromEntity(
	inv entity.Inventory,
) model.InventoryResponse {
	return BuildInventoryResponse(
		inv.VariantID,
		inv.LocationID,
		inv.Quantity,
		inv.ReservedQuantity,
		inv.Threshold,
		inv.BinLocation,
	)
}

// ============================================================================
// VariantWithInventory Builder
// ============================================================================

// BuildVariantWithInventory creates a VariantWithInventory from variant and inventory data
func BuildVariantWithInventory(
	variant productModel.VariantDetailResponse,
	inv mapper.VariantInventoryRow,
	locationID uint,
) model.VariantWithInventory {
	return model.VariantWithInventory{
		VariantID:   variant.ID,
		SKU:         variant.SKU,
		VariantName: BuildVariantName(variant.SelectedOptions),
		Options:     BuildVariantOptions(variant.SelectedOptions),
		Inventory:   BuildInventoryResponseFromRow(inv, locationID),
	}
}

// BuildVariantOptions converts product model options to inventory model options
func BuildVariantOptions(options []productModel.VariantOptionResponse) []model.VariantOption {
	result := make([]model.VariantOption, 0, len(options))
	for _, opt := range options {
		result = append(result, model.VariantOption{
			Name:  opt.OptionDisplayName,
			Value: opt.ValueDisplayName,
		})
	}
	return result
}

// BuildVariantName creates a display name from variant options (e.g., "Black - 256GB")
func BuildVariantName(options []productModel.VariantOptionResponse) string {
	if len(options) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(options[0].ValueDisplayName)
	for i := 1; i < len(options); i++ {
		builder.WriteString(" - ")
		builder.WriteString(options[i].ValueDisplayName)
	}
	return builder.String()
}

// ============================================================================
// VariantInventorySummary Builder
// ============================================================================

// VariantSummaryAccumulator accumulates summary data while iterating variants
type VariantSummaryAccumulator struct {
	TotalVariants      uint
	TotalStock         int
	TotalReserved      int
	TotalAvailable     int
	LowStockVariants   uint
	OutOfStockVariants uint
}

// AddInventory adds inventory data to the accumulator
func (a *VariantSummaryAccumulator) AddInventory(inv mapper.VariantInventoryRow) {
	a.TotalVariants++
	a.TotalStock += inv.Quantity
	a.TotalReserved += inv.ReservedQuantity
	a.TotalAvailable += inv.Quantity - inv.ReservedQuantity

	if inv.Quantity <= 0 {
		a.OutOfStockVariants++
	} else if inv.Quantity <= inv.Threshold {
		a.LowStockVariants++
	}
}

// Build creates the final VariantInventorySummary
func (a *VariantSummaryAccumulator) Build() model.VariantInventorySummary {
	return model.VariantInventorySummary{
		TotalVariants:      a.TotalVariants,
		TotalStock:         a.TotalStock,
		TotalReserved:      a.TotalReserved,
		TotalAvailable:     a.TotalAvailable,
		LowStockVariants:   a.LowStockVariants,
		OutOfStockVariants: a.OutOfStockVariants,
		StockStatus: helper.CalculateOverallStockStatus(
			a.OutOfStockVariants,
			a.LowStockVariants,
			a.TotalVariants,
		),
	}
}

// ============================================================================
// ProductInventorySummary Builder
// ============================================================================

// ProductSummaryAccumulator accumulates summary data for a product
type ProductSummaryAccumulator struct {
	ProductID          uint
	ProductName        string
	CategoryID         uint
	BaseSKU            string
	VariantCount       uint
	TotalStock         int
	TotalReserved      int
	LowStockVariants   uint
	OutOfStockVariants uint
}

// AddInventory adds inventory data to the accumulator
func (a *ProductSummaryAccumulator) AddInventory(inv mapper.VariantInventoryRow) {
	a.VariantCount++
	a.TotalStock += inv.Quantity
	a.TotalReserved += inv.ReservedQuantity

	available := inv.Quantity - inv.ReservedQuantity
	if inv.Quantity == 0 || available <= 0 {
		a.OutOfStockVariants++
	} else if inv.Quantity <= inv.Threshold {
		a.LowStockVariants++
	}
}

// Build creates the final ProductInventorySummary
func (a *ProductSummaryAccumulator) Build() model.ProductInventorySummary {
	return model.ProductInventorySummary{
		ProductID:          a.ProductID,
		ProductName:        a.ProductName,
		CategoryID:         a.CategoryID,
		BaseSKU:            a.BaseSKU,
		VariantCount:       a.VariantCount,
		TotalStock:         a.TotalStock,
		TotalReserved:      a.TotalReserved,
		TotalAvailable:     a.TotalStock - a.TotalReserved,
		LowStockVariants:   a.LowStockVariants,
		OutOfStockVariants: a.OutOfStockVariants,
		StockStatus: helper.CalculateOverallStockStatus(
			a.OutOfStockVariants,
			a.LowStockVariants,
			a.VariantCount,
		),
	}
}

// ============================================================================
// VariantInventoryResponse Builder
// ============================================================================

// BuildVariantInventoryResponse creates the final API 3 response
func BuildVariantInventoryResponse(
	productID uint,
	productName string,
	categoryID uint,
	baseSKU string,
	locationID uint,
	locationName string,
	variants []model.VariantWithInventory,
	summary model.VariantInventorySummary,
	filter model.VariantInventoryFilter,
) *model.VariantInventoryResponse {
	return &model.VariantInventoryResponse{
		ProductID:    productID,
		ProductName:  productName,
		CategoryID:   categoryID,
		BaseSKU:      baseSKU,
		LocationID:   locationID,
		LocationName: locationName,
		Variants:     variants,
		Summary:      summary,
		Filters:      filter,
	}
}

// ============================================================================
// ProductsAtLocationResponse Builder
// ============================================================================

// BuildProductsAtLocationResponse creates the API 2 response
func BuildProductsAtLocationResponse(
	locationID uint,
	locationName string,
	products []model.ProductInventorySummary,
	page, pageSize int,
	totalCount int64,
) *model.ProductsAtLocationResponse {
	return &model.ProductsAtLocationResponse{
		LocationID:   locationID,
		LocationName: locationName,
		Products:     products,
		Pagination:   common.NewPaginationResponse(page, pageSize, totalCount),
	}
}

// BuildEmptyProductsAtLocationResponse creates an empty response for API 2
func BuildEmptyProductsAtLocationResponse(
	locationID uint,
	locationName string,
	page, pageSize int,
) *model.ProductsAtLocationResponse {
	return BuildProductsAtLocationResponse(
		locationID,
		locationName,
		[]model.ProductInventorySummary{},
		page,
		pageSize,
		0,
	)
}
