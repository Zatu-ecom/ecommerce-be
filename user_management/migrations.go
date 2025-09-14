package userManagement

import (
	"ecommerce-be/user_management/entity"
)

type UserAutoMigrate struct{}

func (u *UserAutoMigrate) AutoMigrate() []interface{} {
	return []interface{}{
		&entity.User{},
		&entity.Address{},
	}
}

func NewUserAutoMigrate() *UserAutoMigrate {
	return &UserAutoMigrate{}
}
