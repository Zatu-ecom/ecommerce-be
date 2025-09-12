package entity

import "time"

// BaseEntity contains common fields for all entities
type BaseEntity struct {
	ID        uint      `json:"id"        gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

// BaseEntityWithoutID for entities that don't need auto-generated ID
type BaseEntityWithoutID struct {
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}
