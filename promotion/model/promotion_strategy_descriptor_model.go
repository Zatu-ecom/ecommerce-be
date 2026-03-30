package model

import "ecommerce-be/promotion/entity"

// PromotionConfigFieldDescriptor describes a single config field for a promotion type.
type PromotionConfigFieldDescriptor struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Required      bool        `json:"required"`
	Description   string      `json:"description"`
	DefaultValue  interface{} `json:"defaultValue,omitempty"`
	AllowedValues []string    `json:"allowedValues,omitempty"`
}

// PromotionStrategyDescriptor describes the supported config contract and guidance for a promotion type.
type PromotionStrategyDescriptor struct {
	PromotionType  entity.PromotionType           `json:"promotionType"`
	Name           string                         `json:"name"`
	Description    string                         `json:"description"`
	Fields         []PromotionConfigFieldDescriptor `json:"fields"`
	BestPractices  []string                       `json:"bestPractices,omitempty"`
}
