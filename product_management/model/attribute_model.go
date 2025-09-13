package model

// AttributeDefinitionCreateRequest represents the request body for creating an attribute definition
type AttributeDefinitionCreateRequest struct {
	Key           string   `json:"key"           binding:"required,min=3,max=50"`
	Name          string   `json:"name"          binding:"required,min=3,max=100"`
	Unit          string   `json:"unit"          binding:"max=20"`
	Description   string   `json:"description"   binding:"max=500"`
	AllowedValues []string `json:"allowedValues"`
}

// AttributeDefinitionUpdateRequest represents the request body for updating an attribute definition
type AttributeDefinitionUpdateRequest struct {
	Name          string   `json:"name"          binding:"required,min=3,max=100"`
	Unit          string   `json:"unit"          binding:"max=20"`
	Description   string   `json:"description"   binding:"max=500"`
	AllowedValues []string `json:"allowedValues"`
}

// AttributeDefinitionResponse represents the attribute definition data returned in API responses
type AttributeDefinitionResponse struct {
	ID            uint     `json:"id"`
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Unit          string   `json:"unit"`
	Description   string   `json:"description"`
	AllowedValues []string `json:"allowedValues"`
	CreatedAt     string   `json:"createdAt"`
}

// AttributeDefinitionsResponse represents the response for getting all attribute definitions
type AttributeDefinitionsResponse struct {
	Attributes []AttributeDefinitionResponse `json:"attributes"`
}

// CategoryAttributeConfig represents the configuration of an attribute for a category
type CategoryAttributeConfig struct {
	AttributeDefinitionID uint   `json:"attributeDefinitionId" binding:"required"`
	IsRequired            bool   `json:"isRequired"`
	IsSearchable          bool   `json:"isSearchable"`
	IsFilterable          bool   `json:"isFilterable"`
	DefaultValue          string `json:"defaultValue"`
}

// ConfigureCategoryAttributesRequest represents the request body for configuring category attributes
type ConfigureCategoryAttributesRequest struct {
	Attributes []CategoryAttributeConfig `json:"attributes" binding:"required"`
}

// CategoryAttributeResponse represents a category attribute in API responses
type CategoryAttributeResponse struct {
	ID                  uint                        `json:"id"`
	AttributeDefinition AttributeDefinitionResponse `json:"attributeDefinition"`
	IsRequired          bool                        `json:"isRequired"`
	IsSearchable        bool                        `json:"isSearchable"`
	IsFilterable        bool                        `json:"isFilterable"`
	DefaultValue        string                      `json:"defaultValue"`
}

// CategoryAttributesResponse represents the response for getting category attributes
type CategoryAttributesResponse struct {
	CategoryID   uint                        `json:"categoryId"`
	CategoryName string                      `json:"categoryName"`
	Attributes   []CategoryAttributeResponse `json:"attributes"`
}

// ConfigureCategoryAttributesResponse represents the response for configuring category attributes
type ConfigureCategoryAttributesResponse struct {
	CategoryID           uint `json:"categoryId"`
	ConfiguredAttributes int  `json:"configuredAttributes"`
}
