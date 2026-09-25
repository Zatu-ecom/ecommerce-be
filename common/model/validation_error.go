package model

// ValidationError represents a single field error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
