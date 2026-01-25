package factory

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
)

// BuildWishlistResponse builds WishlistResponse from entity
func BuildWishlistResponse(wishlist *entity.Wishlist, itemCount int) *model.WishlistResponse {
	return &model.WishlistResponse{
		ID:        wishlist.ID,
		Name:      wishlist.Name,
		IsDefault: wishlist.IsDefault,
		ItemCount: itemCount,
		CreatedAt: wishlist.CreatedAt,
		UpdatedAt: wishlist.UpdatedAt,
	}
}

// BuildWishlistResponseFromMapper builds WishlistResponse from mapper
func BuildWishlistResponseFromMapper(w *mapper.WishlistWithItemCount) *model.WishlistResponse {
	return &model.WishlistResponse{
		ID:        w.ID,
		Name:      w.Name,
		IsDefault: w.IsDefault,
		ItemCount: w.ItemCount,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

// BuildWishlistsResponse builds WishlistsResponse from mapper slice
func BuildWishlistsResponse(wishlists []mapper.WishlistWithItemCount) *model.WishlistsResponse {
	response := &model.WishlistsResponse{
		Wishlists: make([]model.WishlistResponse, 0, len(wishlists)),
	}

	for _, w := range wishlists {
		response.Wishlists = append(response.Wishlists, *BuildWishlistResponseFromMapper(&w))
	}

	return response
}

// BuildWishlistEntity builds Wishlist entity from create request
func BuildWishlistEntity(userID uint, name string, isDefault bool) *entity.Wishlist {
	return &entity.Wishlist{
		UserID:    userID,
		Name:      name,
		IsDefault: isDefault,
	}
}

// BuildWishlistDetailResponse builds WishlistDetailResponse from entity with items
func BuildWishlistDetailResponse(wishlist *entity.Wishlist) *model.WishlistDetailResponse {
	items := make([]model.WishlistItemResponse, 0, len(wishlist.Items))
	for _, item := range wishlist.Items {
		items = append(items, model.WishlistItemResponse{
			ID:        item.ID,
			VariantID: item.VariantID,
			CreatedAt: item.CreatedAt,
		})
	}

	return &model.WishlistDetailResponse{
		ID:        wishlist.ID,
		Name:      wishlist.Name,
		IsDefault: wishlist.IsDefault,
		ItemCount: len(wishlist.Items),
		Items:     items,
		CreatedAt: wishlist.CreatedAt,
		UpdatedAt: wishlist.UpdatedAt,
	}
}
