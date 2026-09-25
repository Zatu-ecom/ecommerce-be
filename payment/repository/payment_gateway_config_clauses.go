package repository

import (
	"gorm.io/gorm/clause"
)

// gormOnConflictConfig returns the ON CONFLICT clause used when upserting a seller's
// gateway configuration, keyed on the table's natural unique constraint.
func gormOnConflictConfig() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "seller_id"}, {Name: "gateway_id"}, {Name: "environment"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"credentials",
			"is_active",
			"priority",
			"updated_at",
		}),
	}
}
