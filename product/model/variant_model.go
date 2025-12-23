package model

// VariantOptionResponse represents the selected option for a variant
type VariantOptionResponse struct {
	OptionID          uint   `json:"optionId"`
	OptionName        string `json:"optionName"`
	OptionDisplayName string `json:"optionDisplayName"`
	ValueID           uint   `json:"valueId"`
	Value             string `json:"value"`
	ValueDisplayName  string `json:"valueDisplayName"`
	ColorCode         string `json:"colorCode,omitempty"`
}

// ProductBasicInfo represents basic product information in variant response
type ProductBasicInfo struct {
	ID    uint   `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Brand string `json:"brand,omitempty"`
}

// VariantDetailResponse represents detailed variant information
type VariantDetailResponse struct {
	ID              uint                    `json:"id"`
	ProductID       uint                    `json:"productId,omitempty"`
	Product         ProductBasicInfo        `json:"product,omitzero"`
	SKU             string                  `json:"sku"`
	Price           float64                 `json:"price"`
	Images          []string                `json:"images"`
	AllowPurchase   bool                    `json:"allowPurchase"`
	IsPopular       bool                    `json:"isPopular"`
	IsDefault       bool                    `json:"isDefault"`
	SelectedOptions []VariantOptionResponse `json:"selectedOptions"`
	CreatedAt       string                  `json:"createdAt,omitempty"`
	UpdatedAt       string                  `json:"updatedAt,omitempty"`
}

// VariantResponse represents simplified variant information
type VariantResponse struct {
	ID              uint                    `json:"id"`
	SKU             string                  `json:"sku"`
	Price           float64                 `json:"price"`
	Images          []string                `json:"images"`
	AllowPurchase   bool                    `json:"allowPurchase"`
	IsPopular       bool                    `json:"isPopular"`
	IsDefault       bool                    `json:"isDefault"`
	SelectedOptions []VariantOptionResponse `json:"selectedOptions"`
}

// FindVariantByOptionsRequest represents the request to find a variant by options
type FindVariantByOptionsRequest struct {
	Options map[string]string `json:"options"`
}

// VariantNotFoundErrorDetails provides details when a variant is not found
type VariantNotFoundErrorDetails struct {
	RequestedOptions map[string]string   `json:"requestedOptions"`
	AvailableOptions map[string][]string `json:"availableOptions"`
}

// OptionValueResponse represents an option value with variant count
type OptionValueResponse struct {
	ValueID      uint   `json:"valueId"`
	Value        string `json:"value"`
	DisplayName  string `json:"displayName"`
	ColorCode    string `json:"colorCode,omitempty"`
	VariantCount int    `json:"variantCount"`
	Position     int    `json:"position"`
}

// ProductOptionDetailResponse represents a product option with its values
type ProductOptionDetailResponse struct {
	OptionID          uint                  `json:"optionId"`
	OptionName        string                `json:"optionName"`
	OptionDisplayName string                `json:"optionDisplayName"`
	Position          int                   `json:"position"`
	Values            []OptionValueResponse `json:"values"`
}

// GetAvailableOptionsResponse represents the response for available options
type GetAvailableOptionsResponse struct {
	ProductID uint                          `json:"productId"`
	Options   []ProductOptionDetailResponse `json:"options"`
}

// VariantOptionInput represents an option selection for variant creation
type VariantOptionInput struct {
	OptionName string `json:"optionName" binding:"required"`
	Value      string `json:"value"      binding:"required"`
}

// CreateVariantRequest represents the request to create a new variant
type CreateVariantRequest struct {
	SKU           string               `json:"sku"`
	Price         float64              `json:"price"         binding:"required,gt=0"`
	Images        []string             `json:"images"        binding:"max=20"`
	AllowPurchase *bool                `json:"allowPurchase"`
	IsPopular     *bool                `json:"isPopular"`
	IsDefault     *bool                `json:"isDefault"`
	Options       []VariantOptionInput `json:"options"       binding:"required,min=1,dive"`
}

// UpdateVariantRequest represents the request to update an existing variant
type UpdateVariantRequest struct {
	SKU           *string  `json:"sku"`
	Price         *float64 `json:"price"         binding:"omitempty,gt=0"`
	Images        []string `json:"images"        binding:"max=20"`
	AllowPurchase *bool    `json:"allowPurchase"`
	IsPopular     *bool    `json:"isPopular"`
	IsDefault     *bool    `json:"isDefault"`
}

// BulkUpdateVariantItem represents a single variant update in bulk operation
type BulkUpdateVariantItem struct {
	ID            uint     `json:"id"                      binding:"required"`
	SKU           *string  `json:"sku,omitempty"`
	Price         *float64 `json:"price,omitempty"         binding:"omitempty,gt=0"`
	Images        []string `json:"images,omitempty"        binding:"omitempty,max=20"`
	AllowPurchase *bool    `json:"allowPurchase,omitempty"`
	IsPopular     *bool    `json:"isPopular,omitempty"`
	IsDefault     *bool    `json:"isDefault,omitempty"`
}

// BulkUpdateVariantsRequest represents the request to bulk update variants
type BulkUpdateVariantsRequest struct {
	Variants []BulkUpdateVariantItem `json:"variants" binding:"required,min=1,dive"`
}

// BulkUpdateVariantSummary represents a single variant summary in response
type BulkUpdateVariantSummary struct {
	ID            uint    `json:"id"`
	SKU           string  `json:"sku"`
	Price         float64 `json:"price"`
	AllowPurchase bool    `json:"allowPurchase"`
}

// BulkUpdateVariantsResponse represents the response for bulk update
type BulkUpdateVariantsResponse struct {
	UpdatedCount int                        `json:"updatedCount"`
	Variants     []BulkUpdateVariantSummary `json:"variants"`
}

// ListVariantsRequest represents the request to list/filter variants
type ListVariantsRequest struct {
	// Filter by variant IDs (for home page recommendations)
	IDs string `form:"ids"`

	// Filter by product IDs (optional - if you want variants across multiple products)
	// Supports format: ?productIds=1,2,3
	ProductIDs string `form:"productIds"`

	// Price range filters
	MinPrice *float64 `form:"minPrice" binding:"omitempty,gte=0"`
	MaxPrice *float64 `form:"maxPrice" binding:"omitempty,gte=0"`

	// Availability and status filters
	AllowPurchase *bool `form:"allowPurchase"`
	IsPopular     *bool `form:"isPopular"`
	IsDefault     *bool `form:"isDefault"`

	// SKU search (partial match)
	SKU string `form:"sku" binding:"omitempty,max=100"`

	// TODO: Stock filters - integrate with inventory service when ready
	// MinStock     *int  `form:"minStock" binding:"omitempty,gte=0"`
	// MaxStock     *int  `form:"maxStock" binding:"omitempty,gte=0"`
	// InStock      *bool `form:"inStock"` // Only variants with stock > 0
	// LowStock     *bool `form:"lowStock"` // Only variants below threshold
	// When inventory service is ready:
	// 1. Add these fields to struct
	// 2. Add filters to repository query (JOIN with inventory table)
	// 3. For microservices: Call inventory service API to get variant IDs, then filter

	// Option filters (e.g., ?color=red&size=M)
	// Handled separately via query params

	// Pagination
	Page     int `form:"page"     binding:"omitempty,gte=1"`
	PageSize int `form:"pageSize" binding:"omitempty,gte=1,lte=100"`

	// Sorting
	SortBy    string `form:"sortBy"    binding:"omitempty,oneof=price created_at updated_at"` // price, createdAt, updatedAt
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`                    // asc or desc
}

// ListVariantsResponse represents the response for listing variants
type ListVariantsResponse struct {
	Variants []VariantDetailResponse `json:"variants"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}
