package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	commonModel "ecommerce-be/common/model"
	productRepo "ecommerce-be/product/repository"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

type DiscountCodeVariantScopeService interface {
	AddVariants(ctx context.Context, req model.AddDiscountCodeVariantRequest, sellerID uint) error
	RemoveVariants(ctx context.Context, req model.RemoveDiscountCodeVariantRequest, sellerID uint) error
	RemoveAllVariants(ctx context.Context, discountCodeID, sellerID uint) error
	GetVariants(
		ctx context.Context,
		req model.GetDiscountCodeVariantsRequest,
		sellerID uint,
	) (*model.GetDiscountCodeVariantsResponse, error)
}

type DiscountCodeVariantScopeServiceImpl struct {
	repo             repository.DiscountCodeVariantScopeRepository
	discountCodeRepo repository.DiscountCodeRepository
	variantRepo      productRepo.VariantRepository
}

func NewDiscountCodeVariantScopeService(
	repo repository.DiscountCodeVariantScopeRepository,
	discountCodeRepo repository.DiscountCodeRepository,
	variantRepo productRepo.VariantRepository,
) DiscountCodeVariantScopeService {
	return &DiscountCodeVariantScopeServiceImpl{
		repo:             repo,
		discountCodeRepo: discountCodeRepo,
		variantRepo:      variantRepo,
	}
}

func (s *DiscountCodeVariantScopeServiceImpl) AddVariants(
	ctx context.Context,
	req model.AddDiscountCodeVariantRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecficVariant,
	); err != nil {
		return err
	}

	variants, err := s.variantRepo.FindVariantsByIDs(ctx, req.VariantIDs)
	if err != nil {
		return err
	}
	byID := make(map[uint]uint, len(variants))
	for _, v := range variants {
		byID[v.ID] = v.ProductID
	}

	rows := make([]entity.DiscountCodeProduct, 0, len(req.VariantIDs))
	for _, vid := range req.VariantIDs {
		productID, ok := byID[vid]
		if !ok {
			return promoErrors.ErrInvalidDiscountCodeScope.WithMessage(
				fmt.Sprintf("variant %d not found", vid),
			)
		}
		vidCopy := vid
		rows = append(rows, entity.DiscountCodeProduct{
			DiscountCodeID: req.DiscountCodeID,
			ProductID:      productID,
			VariantID:      &vidCopy,
		})
	}

	if err := s.repo.AddVariants(ctx, rows); err != nil {
		log.ErrorWithContext(ctx, "Failed to add discount code variants", err)
		return err
	}
	return nil
}

func (s *DiscountCodeVariantScopeServiceImpl) RemoveVariants(
	ctx context.Context,
	req model.RemoveDiscountCodeVariantRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecficVariant,
	); err != nil {
		return err
	}
	return s.repo.DeleteVariants(ctx, req.DiscountCodeID, req.VariantIDs)
}

func (s *DiscountCodeVariantScopeServiceImpl) RemoveAllVariants(
	ctx context.Context,
	discountCodeID, sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, discountCodeID, sellerID, entity.ScopeSpecficVariant,
	); err != nil {
		return err
	}
	return s.repo.DeleteAllVariants(ctx, discountCodeID)
}

func (s *DiscountCodeVariantScopeServiceImpl) GetVariants(
	ctx context.Context,
	req model.GetDiscountCodeVariantsRequest,
	sellerID uint,
) (*model.GetDiscountCodeVariantsResponse, error) {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecficVariant,
	); err != nil {
		return nil, err
	}

	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	rows, total, err := s.repo.GetVariants(
		ctx, req.DiscountCodeID, req.VariantIDs, offset, req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	variantIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.VariantID != nil {
			variantIDs = append(variantIDs, *row.VariantID)
		}
	}

	details := map[uint]struct {
		productID uint
		sku       string
		price     string
	}{}
	if len(variantIDs) > 0 {
		variants, findErr := s.variantRepo.FindVariantsByIDs(ctx, variantIDs)
		if findErr == nil {
			for _, v := range variants {
				details[v.ID] = struct {
					productID uint
					sku       string
					price     string
				}{
					productID: v.ProductID,
					sku:       v.SKU,
					price:     fmt.Sprintf("%.2f", float64(v.PriceCents)/100.0),
				}
			}
		}
	}

	response := &model.GetDiscountCodeVariantsResponse{
		BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
			DiscountCodeID: req.DiscountCodeID,
		},
		Variants:   make([]model.DiscountCodeVariantResponse, len(rows)),
		Pagination: commonModel.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, row := range rows {
		vid := uint(0)
		if row.VariantID != nil {
			vid = *row.VariantID
		}
		d := details[vid]
		productID := row.ProductID
		if d.productID != 0 {
			productID = d.productID
		}
		response.Variants[i] = model.DiscountCodeVariantResponse{
			BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
				DiscountCodeID: req.DiscountCodeID,
			},
			VariantID: vid,
			ProductID: productID,
			SKU:       d.sku,
			Price:     d.price,
		}
	}

	return response, nil
}
