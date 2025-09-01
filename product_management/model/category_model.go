package model

// CategoryCreateRequest represents the request body for creating a category
type CategoryCreateRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=100"`
	ParentID    *uint  `json:"parentId"`
	Description string `json:"description" binding:"max=500"`
}

// CategoryUpdateRequest represents the request body for updating a category
type CategoryUpdateRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=100"`
	ParentID    *uint  `json:"parentId"`
	Description string `json:"description" binding:"max=500"`
	IsActive    bool   `json:"isActive"`
}

// CategoryResponse represents the category data returned in API responses
type CategoryResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ParentID    *uint  `json:"parentId"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CategoryHierarchyResponse represents the hierarchical category structure
type CategoryHierarchyResponse struct {
	ID          uint                        `json:"id"`
	Name        string                      `json:"name"`
	ParentID    *uint                       `json:"parentId"`
	Description string                      `json:"description"`
	IsActive    bool                        `json:"isActive"`
	Children    []CategoryHierarchyResponse `json:"children"`
}

// CategoriesResponse represents the response for getting all categories
type CategoriesResponse struct {
	Categories []CategoryHierarchyResponse `json:"categories"`
}

// CategoryListResponse represents the response for getting categories with pagination
type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Pagination PaginationResponse `json:"pagination"`
}
