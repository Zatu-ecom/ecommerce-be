package entity

import (
	"fmt"

	"ecommerce-be/common/db"
)

// Role defines the user roles in the system.
type Role struct {
	db.BaseEntity
	Name        RoleName  `json:"name"        gorm:"unique;not null;size:50"` // e.g., "ADMIN", "SELLER", "CUSTOMER"
	Description string    `json:"description"`                                // A brief description of the role's purpose.
	Level       RoleLevel `json:"level"       gorm:"not null"`                // 1=ADMIN, 2=SELLER, 3=CUSTOMER
}

/*********** RoleLevel ***********/

type RoleLevel uint

const (
	ADMIN_ROLE_LEVEL    RoleLevel = 1
	SELLER_ROLE_LEVEL   RoleLevel = 2
	CUSTOMER_ROLE_LEVEL RoleLevel = 3
)

func (r RoleLevel) ToUint() uint {
	return uint(r)
}

// IsValid checks if the RoleLevel is a valid enum value
func (r RoleLevel) IsValid() bool {
	switch r {
	case ADMIN_ROLE_LEVEL, SELLER_ROLE_LEVEL, CUSTOMER_ROLE_LEVEL:
		return true
	default:
		return false
	}
}

// ParseRoleLevel converts uint to RoleLevel with validation
func ParseRoleLevel(level uint) (RoleLevel, error) {
	roleLevel := RoleLevel(level)
	if !roleLevel.IsValid() {
		return 0, fmt.Errorf(
			"invalid role level: %d, valid levels are 1(ADMIN), 2(SELLER), 3(CUSTOMER)",
			level,
		)
	}
	return roleLevel, nil
}

/*********** RoleName ***********/
type RoleName string

const (
	ADMIN_ROLE    RoleName = "ADMIN"
	SELLER_ROLE   RoleName = "SELLER"
	CUSTOMER_ROLE RoleName = "CUSTOMER"
)

func (r RoleName) ToString() string {
	return string(r)
}

// IsValid checks if the RoleName is a valid enum value
func (r RoleName) IsValid() bool {
	switch r {
	case ADMIN_ROLE, SELLER_ROLE, CUSTOMER_ROLE:
		return true
	default:
		return false
	}
}

// ParseRoleName converts string to RoleName with validation
func ParseRoleName(name string) (RoleName, error) {
	roleName := RoleName(name)
	if !roleName.IsValid() {
		return "", fmt.Errorf(
			"invalid role name: %s, valid names are ADMIN, SELLER, CUSTOMER",
			name,
		)
	}
	return roleName, nil
}
