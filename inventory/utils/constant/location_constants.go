package constant

// Location types
const (
	LOCATION_TYPE_WAREHOUSE     = "WAREHOUSE"
	LOCATION_TYPE_STORE         = "STORE"
	LOCATION_TYPE_RETURN_CENTER = "RETURN_CENTER"
)

// Location success messages
const (
	LOCATION_CREATED_MSG    = "Location created successfully"
	LOCATION_UPDATED_MSG    = "Location updated successfully"
	LOCATION_DELETED_MSG    = "Location deleted successfully"
	LOCATIONS_RETRIEVED_MSG = "Locations retrieved successfully"
	LOCATION_RETRIEVED_MSG  = "Location retrieved successfully"
)

// Location error messages
const (
	LOCATION_NOT_FOUND_MSG           = "Location not found"
	DUPLICATE_LOCATION_NAME_MSG      = "A location with this name already exists"
	INVALID_LOCATION_TYPE_MSG        = "Invalid location type. Must be WAREHOUSE, STORE, or RETURN_CENTER"
	LOCATION_INACTIVE_MSG            = "Location is not active"
	UNAUTHORIZED_LOCATION_ACCESS_MSG = "You do not have permission to access this location"
)

// Location operation failure messages
const (
	FAILED_TO_CREATE_LOCATION_MSG = "Failed to create location"
	FAILED_TO_UPDATE_LOCATION_MSG = "Failed to update location"
	FAILED_TO_DELETE_LOCATION_MSG = "Failed to delete location"
	FAILED_TO_GET_LOCATIONS_MSG   = "Failed to get locations"
	FAILED_TO_GET_LOCATION_MSG    = "Failed to get location"
)

// Location validation messages
const (
	LOCATION_NAME_REQUIRED_MSG    = "Location name is required"
	LOCATION_NAME_LENGTH_MSG      = "Location name must be between 3 and 255 characters"
	LOCATION_TYPE_REQUIRED_MSG    = "Location type is required"
	LOCATION_ADDRESS_REQUIRED_MSG = "Location address is required"
	ADDRESS_STREET_REQUIRED_MSG   = "Address street is required"
	ADDRESS_STREET_LENGTH_MSG     = "Address street must be at least 5 characters"
	ADDRESS_CITY_REQUIRED_MSG     = "Address city is required"
	ADDRESS_CITY_LENGTH_MSG       = "Address city must be at least 2 characters"
	ADDRESS_STATE_REQUIRED_MSG    = "Address state is required"
	ADDRESS_STATE_LENGTH_MSG      = "Address state must be at least 2 characters"
	ADDRESS_ZIPCODE_REQUIRED_MSG  = "Address zip code is required"
	ADDRESS_COUNTRY_REQUIRED_MSG  = "Address country is required"
	ADDRESS_COUNTRY_LENGTH_MSG    = "Address country must be at least 2 characters"
)

// Location field names
const (
	LOCATION_FIELD_NAME  = "location"
	LOCATIONS_FIELD_NAME = "locations"
)
