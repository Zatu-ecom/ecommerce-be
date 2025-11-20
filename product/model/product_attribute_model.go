package model

// AddProductAttributeRequest represents the request to add an attribute to a product
type AddProductAttributeRequest struct {
	AttributeDefinitionID uint   `json:"attributeDefinitionId" binding:"required"`
	Value                 string `json:"value"                 binding:"required,min=1,max=500"`
	SortOrder             uint   `json:"sortOrder"`
}

// UpdateProductAttributeRequest represents the request to update a product attribute
type UpdateProductAttributeRequest struct {
	Value     string `json:"value"     binding:"required,min=1,max=500"`
	SortOrder uint   `json:"sortOrder"`
}

// BulkUpdateAttributeItem represents a single attribute update in bulk operation
type BulkUpdateAttributeItem struct {
	AttributeID uint   `json:"attributeId" binding:"required"`
	Value       string `json:"value"       binding:"required,min=1,max=500"`
	SortOrder   uint   `json:"sortOrder"`
}

// BulkUpdateProductAttributesRequest represents the request to update multiple attributes
type BulkUpdateProductAttributesRequest struct {
	Attributes []BulkUpdateAttributeItem `json:"attributes" binding:"required,min=1,dive"`
}

// BulkUpdateProductAttributesResponse represents the response for bulk update
type BulkUpdateProductAttributesResponse struct {
	UpdatedCount int                              `json:"updatedCount"`
	Attributes   []ProductAttributeDetailResponse `json:"attributes"`
}

// ProductAttributeDetailResponse represents detailed product attribute information
type ProductAttributeDetailResponse struct {
	ID                    uint   `json:"id"`
	ProductID             uint   `json:"productId"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId"`
	AttributeKey          string `json:"attributeKey"`
	AttributeName         string `json:"attributeName"`
	Value                 string `json:"value"`
	Unit                  string `json:"unit,omitempty"`
	SortOrder             uint   `json:"sortOrder"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
}

// ProductAttributesListResponse represents a list of product attributes
type ProductAttributesListResponse struct {
	ProductID  uint                             `json:"productId"`
	Attributes []ProductAttributeDetailResponse `json:"attributes"`
	Total      int                              `json:"total"`
}
