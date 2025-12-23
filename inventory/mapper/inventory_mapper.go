package mapper

// LocationInventorySummaryAggregate represents aggregated inventory statistics for a location
// Used by GetLocationInventorySummary repository method
type LocationInventorySummaryAggregate struct {
	VariantCount    uint
	TotalStock      uint
	TotalReserved   uint
	LowStockCount   uint
	OutOfStockCount uint
	VariantIDs      []uint
}

// VariantInventoryRow represents a single variant's inventory at a location
// Used by GetVariantInventoriesAtLocation repository method
type VariantInventoryRow struct {
	VariantID        uint
	Quantity         int
	ReservedQuantity int
	Threshold        int
	BinLocation      string
}
