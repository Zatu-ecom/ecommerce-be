package validator

import (
	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
)

// ValidateLocationType validates that the location type is one of the allowed values
// Uses entity.LocationType.IsValid() for single source of truth
func ValidateLocationType(locationType entity.LocationType) error {
	if !locationType.IsValid() {
		return invErrors.ErrInvalidLocationType
	}
	return nil
}

// ValidateUniqueName validates that the location name is unique for the seller
func ValidateUniqueName(
	name string,
	sellerID uint,
	existingLocation *entity.Location,
	excludeID *uint,
) error {
	if existingLocation != nil {
		// If we're updating, check if the existing location is the same one
		if excludeID != nil && existingLocation.ID == *excludeID {
			return nil
		}
		// If name exists for this seller and it's not the same location, it's a duplicate
		if existingLocation.SellerID == sellerID {
			return invErrors.ErrDuplicateLocationName
		}
	}
	return nil
}

// ValidateLocationActive validates that the location is active
func ValidateLocationActive(location *entity.Location) error {
	if location != nil && !location.IsActive {
		return invErrors.ErrLocationInactive
	}
	return nil
}
