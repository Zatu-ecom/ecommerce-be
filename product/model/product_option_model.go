package model

// ProductOptionCreateRequest represents the request body for creating a product option
type ProductOptionCreateRequest struct {
	Name        string                      `json:"name"        binding:"required,min=2,max=50"`
	DisplayName string                      `json:"displayName" binding:"required,min=3,max=100"`
	Position    int                         `json:"position"`
	Values      []ProductOptionValueRequest `json:"values"`
}

// ProductOptionUpdateRequest represents the request body for updating a product option
type ProductOptionUpdateRequest struct {
	DisplayName string `json:"displayName" binding:"min=3,max=100"`
	Position    int    `json:"position"`
}

// ProductOptionValueRequest represents a product option value in requests
type ProductOptionValueRequest struct {
	Value       string `json:"value"       binding:"required,min=1,max=100"`
	DisplayName string `json:"displayName" binding:"required,min=1,max=100"`
	ColorCode   string `json:"colorCode"   binding:"omitempty,len=7"`
	Position    int    `json:"position"`
}

// ProductOptionValueUpdateRequest represents the request body for updating a product option value
type ProductOptionValueUpdateRequest struct {
	DisplayName string `json:"displayName" binding:"omitempty,min=1,max=100"`
	ColorCode   string `json:"colorCode"   binding:"omitempty,len=7"`
	Position    int    `json:"position"`
}

// ProductOptionValueBulkAddRequest represents the request body for bulk adding option values
type ProductOptionValueBulkAddRequest struct {
	Values []ProductOptionValueRequest `json:"values" binding:"required,min=1,dive"`
}

// ProductOptionResponse represents a product option in responses
type ProductOptionResponse struct {
	ID          uint                         `json:"id"`
	ProductID   uint                         `json:"productId"`
	Name        string                       `json:"name"`
	DisplayName string                       `json:"displayName"`
	Position    int                          `json:"position"`
	Values      []ProductOptionValueResponse `json:"values,omitempty"`
	CreatedAt   string                       `json:"createdAt"`
	UpdatedAt   string                       `json:"updatedAt"`
}

// ProductOptionValueResponse represents a product option value in responses
type ProductOptionValueResponse struct {
	ID          uint   `json:"id"`
	OptionID    uint   `json:"optionId"`
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
	ColorCode   string `json:"colorCode,omitempty"`
	Position    int    `json:"position"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ProductOptionListResponse represents the response for listing product options
type ProductOptionListResponse struct {
	Options []ProductOptionResponse `json:"options"`
}
