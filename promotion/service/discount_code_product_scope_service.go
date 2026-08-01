package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	productRepo "ecommerce-be/product/repository"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

type DiscountCodeProductScopeService interface {
	AddProducts(ctx context.Context, req model.AddDiscountCodeProductRequest, sellerID uint) error
	RemoveProducts(ctx context.Context, req model.RemoveDiscountCodeProductRequest, sellerID uint) error
	RemoveAllProducts(ctx context.Context, discountCodeID, sellerID uint) error
	GetProducts(
		ctx context.Context,
		req model.GetDiscountCodeProductsRequest,
		sellerID uint,
	) (*model.GetDiscountCodeProductsResponse, error)
}

type DiscountCodeProductScopeServiceImpl struct {
	repo             repository.DiscountCodeProductScopeRepository
	discountCodeRepo repository.DiscountCodeRepository
	productRepo      productRepo.ProductRepository
}

func NewDiscountCodeProductScopeService(
	repo repository.DiscountCodeProductScopeRepository,
	discountCodeRepo repository.DiscountCodeRepository,
	productRepo productRepo.ProductRepository,
) DiscountCodeProductScopeService {
	return &DiscountCodeProductScopeServiceImpl{
		repo:             repo,
		discountCodeRepo: discountCodeRepo,
		productRepo:      productRepo,
	}
}

func (s *DiscountCodeProductScopeServiceImpl) AddProducts(
	ctx context.Context,
	req model.AddDiscountCodeProductRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificProducts,
	); err != nil {
		return err
	}

	entities := make([]entity.DiscountCodeProduct, 0, len(req.ProductIDs))
	for _, pid := range req.ProductIDs {
		entities = append(entities, entity.DiscountCodeProduct{
			DiscountCodeID: req.DiscountCodeID,
			ProductID:      pid,
		})
	}

	if err := s.repo.AddProducts(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add discount code products", err)
		return err
	}
	return nil
}

func (s *DiscountCodeProductScopeServiceImpl) RemoveProducts(
	ctx context.Context,
	req model.RemoveDiscountCodeProductRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificProducts,
	); err != nil {
		return err
	}
	return s.repo.DeleteProducts(ctx, req.DiscountCodeID, req.ProductIDs)
}

func (s *DiscountCodeProductScopeServiceImpl) RemoveAllProducts(
	ctx context.Context,
	discountCodeID, sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, discountCodeID, sellerID, entity.ScopeSpecificProducts,
	); err != nil {
		return err
	}
	return s.repo.DeleteAllProducts(ctx, discountCodeID)
}

func (s *DiscountCodeProductScopeServiceImpl) GetProducts(
	ctx context.Context,
	req model.GetDiscountCodeProductsRequest,
	sellerID uint,
) (*model.GetDiscountCodeProductsResponse, error) {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificProducts,
	); err != nil {
		return nil, err
	}

	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	products, total, err := s.repo.GetProducts(
		ctx, req.DiscountCodeID, req.ProductIDs, offset, req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	productIDs := make([]uint, len(products))
	for i, p := range products {
		productIDs[i] = p.ProductID
	}

	names := map[uint]string{}
	if len(productIDs) > 0 {
		rows, findErr := s.productRepo.FindByIDs(ctx, productIDs)
		if findErr == nil {
			for _, row := range rows {
				names[row.ID] = row.Name
			}
		}
	}

	response := &model.GetDiscountCodeProductsResponse{
		BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
			DiscountCodeID: req.DiscountCodeID,
		},
		Products:   make([]model.DiscountCodeProductResponse, len(products)),
		Pagination: common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, p := range products {
		response.Products[i] = model.DiscountCodeProductResponse{
			BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
				DiscountCodeID: req.DiscountCodeID,
			},
			ProductID:   p.ProductID,
			ProductName: names[p.ProductID],
		}
	}

	return response, nil
}
