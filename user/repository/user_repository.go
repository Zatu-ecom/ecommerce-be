package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
	"ecommerce-be/user/utils/constant"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User CRUD operations
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByEmailWithRole(ctx context.Context, email string) (*entity.User, *entity.Role, error)
	FindByIDWithRole(ctx context.Context, id uint) (*entity.User, *entity.Role, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error

	// Role operations
	FindRoleByName(ctx context.Context, name string) (*entity.Role, error)
	FindRolesByIDs(ctx context.Context, ids []uint) ([]entity.Role, error)

	// List operations
	FindByFilter(ctx context.Context, filter model.ListUsersFilter) ([]entity.User, int64, error)
	FindByIDs(ctx context.Context, ids []uint) ([]entity.User, error)
}

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct{}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

// Create creates a new user in the database
func (r *UserRepositoryImpl) Create(ctx context.Context, user *entity.User) error {
	return db.DB(ctx).Create(user).Error
}

// FindByID finds a user by ID
func (r *UserRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	result := db.DB(ctx).First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(constant.USER_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	result := db.DB(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(constant.USER_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &user, nil
}

// Update updates an existing user
func (r *UserRepositoryImpl) Update(ctx context.Context, user *entity.User) error {
	return db.DB(ctx).Save(user).Error
}

// Delete deletes a user by ID
func (r *UserRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.User{}, id).Error
}

// FindByEmailWithRole finds a user by email and includes role information
func (r *UserRepositoryImpl) FindByEmailWithRole(
	ctx context.Context,
	email string,
) (*entity.User, *entity.Role, error) {
	var user entity.User
	var role entity.Role

	// First find the user
	result := db.DB(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(constant.USER_NOT_FOUND_MSG)
		}
		return nil, nil, result.Error
	}

	// Then find the role
	roleResult := db.DB(ctx).First(&role, user.RoleID)
	if roleResult.Error != nil {
		if errors.Is(roleResult.Error, gorm.ErrRecordNotFound) {
			return &user, nil, errors.New("role not found")
		}
		return &user, nil, roleResult.Error
	}

	return &user, &role, nil
}

// FindByIDWithRole finds a user by ID and includes role information
func (r *UserRepositoryImpl) FindByIDWithRole(
	ctx context.Context,
	id uint,
) (*entity.User, *entity.Role, error) {
	var user entity.User
	var role entity.Role

	// First find the user
	result := db.DB(ctx).First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(constant.USER_NOT_FOUND_MSG)
		}
		return nil, nil, result.Error
	}

	// Then find the role
	roleResult := db.DB(ctx).First(&role, user.RoleID)
	if roleResult.Error != nil {
		if errors.Is(roleResult.Error, gorm.ErrRecordNotFound) {
			return &user, nil, errors.New("role not found")
		}
		return &user, nil, roleResult.Error
	}

	return &user, &role, nil
}

// FindRoleByName finds a role by its name
func (r *UserRepositoryImpl) FindRoleByName(
	ctx context.Context,
	name string,
) (*entity.Role, error) {
	var role entity.Role
	result := db.DB(ctx).Where("name = ?", name).First(&role)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, result.Error
	}
	return &role, nil
}

// FindRolesByIDs finds roles by multiple IDs
func (r *UserRepositoryImpl) FindRolesByIDs(
	ctx context.Context,
	ids []uint,
) ([]entity.Role, error) {
	if len(ids) == 0 {
		return []entity.Role{}, nil
	}

	var roles []entity.Role
	if err := db.DB(ctx).Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// FindByFilter finds users based on filter criteria with pagination
func (r *UserRepositoryImpl) FindByFilter(
	ctx context.Context,
	filter model.ListUsersFilter,
) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	query := db.DB(ctx).Model(&entity.User{})

	// Apply filters
	query = r.applyUserFilters(query, filter)

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortColumn := filter.SortBy
	if sortColumn == "createdAt" {
		sortColumn = "created_at"
	}
	query = query.Order(sortColumn + " " + filter.SortOrder)

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Execute query
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// applyUserFilters applies all filter conditions to the query
func (r *UserRepositoryImpl) applyUserFilters(
	query *gorm.DB,
	filter model.ListUsersFilter,
) *gorm.DB {
	// Filter by IDs
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}

	// Filter by emails
	if len(filter.Emails) > 0 {
		query = query.Where("email IN ?", filter.Emails)
	}

	// Filter by phones
	if len(filter.Phones) > 0 {
		query = query.Where("phone IN ?", filter.Phones)
	}

	// Filter by role IDs
	if len(filter.RoleIDs) > 0 {
		query = query.Where("role_id IN ?", filter.RoleIDs)
	}

	// Filter by role names (requires join with roles table)
	if len(filter.RoleNames) > 0 {
		query = query.Joins("JOIN role ON role.id = \"user\".role_id").
			Where("role.name IN ?", filter.RoleNames)
	}

	// Filter by seller IDs
	if len(filter.SellerIDs) > 0 {
		query = query.Where("seller_id IN ?", filter.SellerIDs)
	}

	// Search by name (partial match in firstName or lastName)
	if filter.Name != nil && *filter.Name != "" {
		searchTerm := "%" + *filter.Name + "%"
		query = query.Where(
			"first_name ILIKE ? OR last_name ILIKE ? OR CONCAT(first_name, ' ', last_name) ILIKE ?",
			searchTerm, searchTerm, searchTerm,
		)
	}

	// Filter by active status
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Filter by date range
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}

	return query
}

// FindByIDs finds users by multiple IDs
func (r *UserRepositoryImpl) FindByIDs(ctx context.Context, ids []uint) ([]entity.User, error) {
	if len(ids) == 0 {
		return []entity.User{}, nil
	}

	var users []entity.User
	if err := db.DB(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
