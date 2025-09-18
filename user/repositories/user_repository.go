package repositories

import (
	"errors"

	"ecommerce-be/user/entity"
	"ecommerce-be/user/utils"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User CRUD operations
	Create(user *entity.User) error
	FindByID(id uint) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	FindByEmailWithRole(email string) (*entity.User, *entity.Role, error)
	FindByIDWithRole(id uint) (*entity.User, *entity.Role, error)
	Update(user *entity.User) error
	Delete(id uint) error

	// Role operations
	FindRoleByName(name string) (*entity.Role, error)
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
	result := r.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.UserNotFoundMsg)
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
			return nil, errors.New(utils.UserNotFoundMsg)
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

// FindByEmailWithRole finds a user by email and includes role information
func (r *UserRepositoryImpl) FindByEmailWithRole(email string) (*entity.User, *entity.Role, error) {
	var user entity.User
	var role entity.Role

	// First find the user
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(utils.UserNotFoundMsg)
		}
		return nil, nil, result.Error
	}

	// Then find the role
	roleResult := r.db.First(&role, user.RoleID)
	if roleResult.Error != nil {
		if errors.Is(roleResult.Error, gorm.ErrRecordNotFound) {
			return &user, nil, errors.New("role not found")
		}
		return &user, nil, roleResult.Error
	}

	return &user, &role, nil
}

// FindByIDWithRole finds a user by ID and includes role information
func (r *UserRepositoryImpl) FindByIDWithRole(id uint) (*entity.User, *entity.Role, error) {
	var user entity.User
	var role entity.Role

	// First find the user
	result := r.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(utils.UserNotFoundMsg)
		}
		return nil, nil, result.Error
	}

	// Then find the role
	roleResult := r.db.First(&role, user.RoleID)
	if roleResult.Error != nil {
		if errors.Is(roleResult.Error, gorm.ErrRecordNotFound) {
			return &user, nil, errors.New("role not found")
		}
		return &user, nil, roleResult.Error
	}

	return &user, &role, nil
}

// FindRoleByName finds a role by its name
func (r *UserRepositoryImpl) FindRoleByName(name string) (*entity.Role, error) {
	var role entity.Role
	result := r.db.Where("name = ?", name).First(&role)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, result.Error
	}
	return &role, nil
}
