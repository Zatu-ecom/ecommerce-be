package service

import (
	"ecommerce-be/common"
	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
	"ecommerce-be/user/repository"
)

// UserQueryService defines the interface for user query operations
type UserQueryService interface {
	// ListUsers returns paginated list of users based on filters
	// Applies seller isolation based on callerSellerID presence
	ListUsers(
		filter model.ListUsersFilter,
		callerSellerID *uint,
	) (*model.ListUsersResponse, error)
}

// UserQueryServiceImpl implements the UserQueryService interface
type UserQueryServiceImpl struct {
	userRepo repository.UserRepository
}

// NewUserQueryService creates a new instance of UserQueryService
func NewUserQueryService(userRepo repository.UserRepository) UserQueryService {
	return &UserQueryServiceImpl{
		userRepo: userRepo,
	}
}

// ============================================================================
// List Users
// ============================================================================

// ListUsers returns paginated list of users based on filters
func (s *UserQueryServiceImpl) ListUsers(
	filter model.ListUsersFilter,
	callerSellerID *uint,
) (*model.ListUsersResponse, error) {
	// Apply seller isolation based on sellerID presence
	s.applySellerIsolation(&filter, callerSellerID)

	// Set defaults for pagination and sorting
	filter.SetDefaults()

	// Fetch users from repository
	users, total, err := s.userRepo.FindByFilter(filter)
	if err != nil {
		return nil, err
	}

	// Fetch roles for all users
	roleMap, err := s.fetchRolesForUsers(users)
	if err != nil {
		return nil, err
	}

	// Build response
	return s.buildListResponse(users, roleMap, total, filter), nil
}

// applySellerIsolation applies seller filtering based on caller's sellerID
// If sellerID is present, filter by it; otherwise no filtering (admin access)
func (s *UserQueryServiceImpl) applySellerIsolation(
	filter *model.ListUsersFilter,
	callerSellerID *uint,
) {
	// If caller has a sellerID, restrict to their seller's users
	if callerSellerID != nil {
		filter.SellerIDs = []uint{*callerSellerID}
	}
	// No sellerID means admin-level access - no filtering needed
}

// fetchRolesForUsers fetches roles for a list of users
func (s *UserQueryServiceImpl) fetchRolesForUsers(
	users []entity.User,
) (map[uint]*entity.Role, error) {
	if len(users) == 0 {
		return make(map[uint]*entity.Role), nil
	}

	// Extract unique role IDs
	roleIDMap := make(map[uint]bool)
	for _, user := range users {
		roleIDMap[user.RoleID] = true
	}

	roleIDs := make([]uint, 0, len(roleIDMap))
	for id := range roleIDMap {
		roleIDs = append(roleIDs, id)
	}

	// Fetch roles
	roles, err := s.userRepo.FindRolesByIDs(roleIDs)
	if err != nil {
		return nil, err
	}

	// Build role map
	roleMap := make(map[uint]*entity.Role)
	for i := range roles {
		roleMap[roles[i].ID] = &roles[i]
	}

	return roleMap, nil
}

// buildListResponse builds the list users response
func (s *UserQueryServiceImpl) buildListResponse(
	users []entity.User,
	roleMap map[uint]*entity.Role,
	total int64,
	filter model.ListUsersFilter,
) *model.ListUsersResponse {
	userResponses := make([]model.UserListResponse, len(users))

	for i, user := range users {
		role := roleMap[user.RoleID]
		userResponses[i] = s.buildUserListResponse(user, role)
	}

	return &model.ListUsersResponse{
		Users:      userResponses,
		Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}
}

// buildUserListResponse builds a single user response
func (s *UserQueryServiceImpl) buildUserListResponse(
	user entity.User,
	role *entity.Role,
) model.UserListResponse {
	response := model.UserListResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Name:      user.FirstName + " " + user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		SellerID:  user.SellerID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if role != nil {
		response.Role = model.RoleResponse{
			ID:   role.ID,
			Name: string(role.Name),
		}
	}

	return response
}