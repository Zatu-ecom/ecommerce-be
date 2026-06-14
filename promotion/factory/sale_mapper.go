package factory

import (
	"time"

	"ecommerce-be/common/db"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

func SaleRequestToEntity(req model.CreateSaleRequest, sellerID uint) (*entity.Sale, error) {
	startAt, endAt, err := parseSaleDateRange(req.StartAt, req.EndAt)
	if err != nil {
		return nil, err
	}

	status := entity.StatusDraft
	if req.Status != "" {
		status = req.Status
	}

	sale := &entity.Sale{
		SellerID:     sellerID,
		Name:         req.Name,
		Description:  req.Description,
		BannerImages: db.StringArray(req.BannerImages),
		Status:       status,
		StartAt:      startAt,
		EndAt:        endAt,
	}

	if req.Slug != nil && *req.Slug != "" {
		sale.Slug = *req.Slug
	}

	return sale, nil
}

func ApplyUpdateSaleRequest(sale *entity.Sale, req model.UpdateSaleRequest) (*entity.Sale, error) {
	startAt, endAt, err := parseSaleDateRange(req.StartAt, req.EndAt)
	if err != nil {
		return nil, err
	}

	sale.Name = req.Name
	sale.Description = req.Description
	sale.BannerImages = db.StringArray(req.BannerImages)
	sale.StartAt = startAt
	sale.EndAt = endAt
	sale.UpdatedAt = time.Now().UTC()

	if req.Slug != nil && *req.Slug != "" {
		sale.Slug = *req.Slug
	}
	if req.Status != "" {
		sale.Status = req.Status
	}

	return sale, nil
}

func SaleEntityToResponse(sale *entity.Sale) *model.SaleResponse {
	banners := []string(sale.BannerImages)
	if banners == nil {
		banners = []string{}
	}

	return &model.SaleResponse{
		ID:           sale.ID,
		SellerID:     sale.SellerID,
		Name:         sale.Name,
		Description:  sale.Description,
		Slug:         sale.Slug,
		BannerImages: banners,
		Status:       sale.Status,
		StartAt:      sale.StartAt.UTC().Format(time.RFC3339),
		EndAt:        sale.EndAt.UTC().Format(time.RFC3339),
		CreatedAt:    sale.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    sale.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func parseSaleDateRange(startAt, endAt string) (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, startAt)
	if err != nil {
		return time.Time{}, time.Time{}, promoErrors.ErrInvalidSaleDateRange
	}
	end, err := time.Parse(time.RFC3339, endAt)
	if err != nil {
		return time.Time{}, time.Time{}, promoErrors.ErrInvalidSaleDateRange
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, promoErrors.ErrInvalidSaleDateRange.WithMessage(
			"endAt must be after startAt",
		)
	}
	return start.UTC(), end.UTC(), nil
}
