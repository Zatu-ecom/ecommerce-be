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
	DisplayName  string `json:"DisplayName"`
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
	Images        []string             `json:"images"`
	AllowPurchase *bool                `json:"allowPurchase"`
	IsPopular     *bool                `json:"isPopular"`
	IsDefault     *bool                `json:"isDefault"`
	Options       []VariantOptionInput `json:"options"       binding:"required,min=1,dive"`
}

// UpdateVariantRequest represents the request to update an existing variant
type UpdateVariantRequest struct {
	SKU           *string  `json:"sku"`
	Price         *float64 `json:"price"         binding:"omitempty,gt=0"`
	Images        []string `json:"images"`
	AllowPurchase *bool    `json:"allowPurchase"`
	IsPopular     *bool    `json:"isPopular"`
	IsDefault     *bool    `json:"isDefault"`
}

// BulkUpdateVariantItem represents a single variant update in bulk operation
type BulkUpdateVariantItem struct {
	ID            uint     `json:"id"                  binding:"required"`
	SKU           *string  `json:"sku,omitempty"`
	Price         *float64 `json:"price,omitempty"     binding:"omitempty,gt=0"`
	Images        []string `json:"images,omitempty"`
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
