package repositories

import (
	"errors"

	"datun.com/be/user_management/entity"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User CRUD operations
	Create(user *entity.User) error
	FindByID(id uint) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	Update(user *entity.User) error
	Delete(id uint) error
}

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{
		db: db,
	}
}

// Create creates a new user in the database
func (r *UserRepositoryImpl) Create(user *entity.User) error {
	return r.db.Create(user).Error
}

// FindByID finds a user by ID
func (r *UserRepositoryImpl) FindByID(id uint) (*entity.User, error) {
	var user entity.User
	result := r.db.Preload("Addresses").First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepositoryImpl) FindByEmail(email string) (*entity.User, error) {
	var user entity.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

// Update updates an existing user
func (r *UserRepositoryImpl) Update(user *entity.User) error {
	return r.db.Save(user).Error
}

// Delete deletes a user by ID
func (r *UserRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&entity.User{}, id).Error
}
