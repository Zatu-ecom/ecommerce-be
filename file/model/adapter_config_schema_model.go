package model

import "ecommerce-be/file/entity"

// Field types for storage adapter config forms (UI hints).
const (
	FieldTypeString   = "string"
	FieldTypeText     = "text"     // multiline (e.g. service_account_json)
	FieldTypePassword = "password" // masked in UI
	FieldTypeBoolean  = "boolean"
)

// FieldDescriptor describes a single input field for a storage provider config.
// Returned by GET /storage-config/schema for dynamic forms.
type FieldDescriptor struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"` // FieldType* constants
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"` // stored in AES-encrypted config_data
	Description string `json:"description"`
	Placeholder string `json:"placeholder,omitempty"`
}

// AdapterConfigSchema describes all fields needed to configure one adapter type.
type AdapterConfigSchema struct {
	AdapterType entity.AdapterType `json:"adapterType"`
	Fields      []FieldDescriptor  `json:"fields"`
}
