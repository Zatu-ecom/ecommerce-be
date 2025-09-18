package user

import (
	"ecommerce-be/user/entity"
)

type UserAutoMigrate struct{}

func (u *UserAutoMigrate) AutoMigrate() []interface{} {
	return []interface{}{
		&entity.User{},
		&entity.Address{},
		&entity.Plan{},
		&entity.Subscription{},
		&entity.Role{},
		&entity.SellerProfile{},
	}
}

func NewUserAutoMigrate() *UserAutoMigrate {
	return &UserAutoMigrate{}
}
