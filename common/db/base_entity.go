package db

import (
	"encoding/json"
	"time"
)

// BaseEntity contains common fields for all entities
type BaseEntity struct {
	ID        uint      `json:"id"        gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// MarshalJSON customizes JSON serialization to always return timestamps in UTC
func (b BaseEntity) MarshalJSON() ([]byte, error) {
	type Alias BaseEntity
	return json.Marshal(&struct {
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
		*Alias
	}{
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
		Alias:     (*Alias)(&b),
	})
}

// BaseEntityWithoutID for entities that don't need auto-generated ID
type BaseEntityWithoutID struct {
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// MarshalJSON customizes JSON serialization to always return timestamps in UTC
func (b BaseEntityWithoutID) MarshalJSON() ([]byte, error) {
	type Alias BaseEntityWithoutID
	return json.Marshal(&struct {
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
		*Alias
	}{
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
		Alias:     (*Alias)(&b),
	})
}
