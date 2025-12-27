package db

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// BaseEntity contains common fields for all entities.
// GORM automatically handles CreatedAt and UpdatedAt:
//   - CreatedAt: Set to current time on record creation
//   - UpdatedAt: Set to current time on creation AND every update
type BaseEntity struct {
	ID        uint      `json:"id"        gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// BeforeCreate hook ensures timestamps are set before inserting
func (b *BaseEntity) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook ensures UpdatedAt is refreshed on every update
func (b *BaseEntity) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now().UTC()
	return nil
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
// (e.g., junction tables with composite primary keys)
type BaseEntityWithoutID struct {
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// BeforeCreate hook ensures timestamps are set before inserting
func (b *BaseEntityWithoutID) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook ensures UpdatedAt is refreshed on every update
func (b *BaseEntityWithoutID) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now().UTC()
	return nil
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
