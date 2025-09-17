package entity

import (
	"ecommerce-be/common/db"
)

// Plan represents subscription plans available for sellers
type Plan struct {
	db.BaseEntity
	Name         string  `json:"name"         gorm:"unique;not null;size:100"` // e.g., "Basic", "Pro", "Enterprise", "Free Trial"
	Description  string  `json:"description"  gorm:"type:text"`                // Detailed plan description
	Price        float64 `json:"price"        gorm:"not null;default:0"`       // Monthly price (0 for free plans)
	Currency     string  `json:"currency"     gorm:"size:3;default:USD"`       // Currency code (USD, EUR, etc.)
	BillingCycle string  `json:"billingCycle" gorm:"size:20;default:monthly"`  // monthly, yearly, lifetime
	IsPopular    bool    `json:"isPopular"    gorm:"default:false"`            // Featured/popular plan flag
	SortOrder    int     `json:"sortOrder"    gorm:"default:0"`                // Display order
	TrialDays    int     `json:"trialDays"    gorm:"default:0"`                // Free trial days (0 = no trial)
}
