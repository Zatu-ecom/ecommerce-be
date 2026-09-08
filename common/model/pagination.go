package model

// BaseListParams contains common pagination and sorting parameters
// Embed this in query params and filter structs for list APIs
type BaseListParams struct {
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortBy    string `form:"sortBy" json:"sortBy"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

// SetDefaults sets default values for pagination and sorting
func (b *BaseListParams) SetDefaults() {
	if b.Page <= 0 {
		b.Page = 1
	}
	if b.PageSize <= 0 {
		b.PageSize = 20
	}
	if b.PageSize > 100 {
		b.PageSize = 100
	}
	if b.SortBy == "" {
		b.SortBy = "created_at"
	}
	if b.SortOrder == "" {
		b.SortOrder = "desc"
	}
}

// PaginationResponse represents pagination information in API responses
type PaginationResponse struct {
	CurrentPage  int  `json:"currentPage"`
	TotalPages   int  `json:"totalPages"`
	TotalItems   int  `json:"totalItems"`
	ItemsPerPage int  `json:"itemsPerPage"`
	HasNext      bool `json:"hasNext"`
	HasPrev      bool `json:"hasPrev"`
}

// NewPaginationResponse creates a pagination response from params and total count
func NewPaginationResponse(page, pageSize int, total int64) PaginationResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: pageSize,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}
}
