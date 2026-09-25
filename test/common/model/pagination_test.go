package model_test

import (
	"testing"

	commonModel "ecommerce-be/common/model"

	"github.com/stretchr/testify/assert"
)

// ─── SetDefaults ────────────────────────────────────────────────────────────
func TestBaseListParamsSetDefaults(t *testing.T) {
	// Zero value gets defaults.
	p := &commonModel.BaseListParams{}
	p.SetDefaults()
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PageSize)
	assert.Equal(t, "created_at", p.SortBy)
	assert.Equal(t, "desc", p.SortOrder)

	// Provided values are preserved; pageSize capped at 100.
	p = &commonModel.BaseListParams{Page: 5, PageSize: 500, SortBy: "price", SortOrder: "asc"}
	p.SetDefaults()
	assert.Equal(t, 5, p.Page)
	assert.Equal(t, 100, p.PageSize)
	assert.Equal(t, "price", p.SortBy)
	assert.Equal(t, "asc", p.SortOrder)
}

// ─── NewPaginationResponse ──────────────────────────────────────────────────
func TestNewPaginationResponse(t *testing.T) {
	// Exact page division.
	r := commonModel.NewPaginationResponse(1, 20, 100)
	assert.Equal(t, 5, r.TotalPages)
	assert.Equal(t, 100, r.TotalItems)
	assert.True(t, r.HasNext)
	assert.False(t, r.HasPrev)

	// Partial last page rounds up.
	r = commonModel.NewPaginationResponse(1, 20, 105)
	assert.Equal(t, 6, r.TotalPages)

	// Middle page: has prev + next.
	r = commonModel.NewPaginationResponse(3, 20, 100)
	assert.True(t, r.HasPrev)
	assert.True(t, r.HasNext)

	// Last page: no next.
	r = commonModel.NewPaginationResponse(5, 20, 100)
	assert.False(t, r.HasNext)
	assert.True(t, r.HasPrev)
}
