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
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Brand string `json:"brand"`
}

// VariantDetailResponse represents detailed variant information
type VariantDetailResponse struct {
	ID              uint                     `json:"id"`
	ProductID       uint                     `json:"productId"`
	Product         ProductBasicInfo         `json:"product,omitempty"`
	SKU             string                   `json:"sku"`
	Price           float64                  `json:"price"`
	Images          []string                 `json:"images"`
	InStock         bool                     `json:"inStock"`
	IsPopular       bool                     `json:"isPopular"`
	Stock           int                      `json:"stock"`
	IsDefault       bool                     `json:"isDefault"`
	SelectedOptions []VariantOptionResponse  `json:"selectedOptions"`
	CreatedAt       string                   `json:"createdAt"`
	UpdatedAt       string                   `json:"updatedAt"`
}

// VariantResponse represents simplified variant information
type VariantResponse struct {
	ID              uint                     `json:"id"`
	SKU             string                   `json:"sku"`
	Price           float64                  `json:"price"`
	Images          []string                 `json:"images"`
	InStock         bool                     `json:"inStock"`
	IsPopular       bool                     `json:"isPopular"`
	Stock           int                      `json:"stock"`
	IsDefault       bool                     `json:"isDefault"`
	SelectedOptions []VariantOptionResponse  `json:"selectedOptions"`
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
	ValueID          uint   `json:"valueId"`
	Value            string `json:"value"`
	ValueDisplayName string `json:"valueDisplayName"`
	ColorCode        string `json:"colorCode,omitempty"`
	VariantCount     int    `json:"variantCount"`
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
	Value      string `json:"value" binding:"required"`
}

// CreateVariantRequest represents the request to create a new variant
type CreateVariantRequest struct {
	SKU       string               `json:"sku" binding:"required"`
	Price     float64              `json:"price" binding:"required,gt=0"`
	Stock     int                  `json:"stock" binding:"required,gte=0"`
	Images    []string             `json:"images"`
	InStock   *bool                `json:"inStock"`
	IsPopular *bool                `json:"isPopular"`
	IsDefault *bool                `json:"isDefault"`
	Options   []VariantOptionInput `json:"options" binding:"required,min=1,dive"`
}

// UpdateVariantRequest represents the request to update an existing variant
type UpdateVariantRequest struct {
	SKU       *string  `json:"sku"`
	Price     *float64 `json:"price" binding:"omitempty,gt=0"`
	Stock     *int     `json:"stock" binding:"omitempty,gte=0"`
	Images    []string `json:"images"`
	InStock   *bool    `json:"inStock"`
	IsPopular *bool    `json:"isPopular"`
	IsDefault *bool    `json:"isDefault"`
}

// UpdateVariantStockRequest represents the request to update variant stock
type UpdateVariantStockRequest struct {
	Stock     int    `json:"stock" binding:"required,gte=0"`
	Operation string `json:"operation" binding:"required,oneof=set add subtract"`
}

// UpdateVariantStockResponse represents the response for stock update
type UpdateVariantStockResponse struct {
	VariantID uint   `json:"variantId"`
	SKU       string `json:"sku"`
	Stock     int    `json:"stock"`
	InStock   bool   `json:"inStock"`
}

// BulkUpdateVariantItem represents a single variant update in bulk operation
type BulkUpdateVariantItem struct {
	ID        uint     `json:"id" binding:"required"`
	SKU       *string  `json:"sku,omitempty"`
	Price     *float64 `json:"price,omitempty" binding:"omitempty,gt=0"`
	Stock     *int     `json:"stock,omitempty" binding:"omitempty,gte=0"`
	Images    []string `json:"images,omitempty"`
	InStock   *bool    `json:"inStock,omitempty"`
	IsPopular *bool    `json:"isPopular,omitempty"`
	IsDefault *bool    `json:"isDefault,omitempty"`
}

// BulkUpdateVariantsRequest represents the request to bulk update variants
type BulkUpdateVariantsRequest struct {
	Variants []BulkUpdateVariantItem `json:"variants" binding:"required,min=1,dive"`
}

// BulkUpdateVariantSummary represents a single variant summary in response
type BulkUpdateVariantSummary struct {
	ID      uint    `json:"id"`
	SKU     string  `json:"sku"`
	Price   float64 `json:"price"`
	Stock   int     `json:"stock"`
	InStock bool    `json:"inStock"`
}

// BulkUpdateVariantsResponse represents the response for bulk update
type BulkUpdateVariantsResponse struct {
	UpdatedCount int                        `json:"updatedCount"`
	Variants     []BulkUpdateVariantSummary `json:"variants"`
}
