package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	productService "ecommerce-be/product/service"
	productModel "ecommerce-be/product/model"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
	productRepo "ecommerce-be/product/repository"
)

type PromotionProductScopeService interface {
	AddProducts(ctx context.Context, req model.AddPromotionProductRequest) error
	RemoveProducts(ctx context.Context, req model.RemovePromotionProductRequest) error
	RemoveAllProducts(ctx context.Context, promotionID uint) error
	GetProducts(
		ctx context.Context,
		req model.GetPromotionProductsRequest,
	) (*model.GetPromotionProductsResponse, error)
}

type PromotionProductScopeServiceImpl struct {
	repo              repository.PromotionProductScopeRepository
	promotionRepo     repository.PromotionRepository
	productRepo       productRepo.ProductRepository
	productMediaSvc   productService.ProductMediaService
}

func NewPromotionProductScopeServiceImpl(
	repo repository.PromotionProductScopeRepository,
	promotionRepo repository.PromotionRepository,
	productRepo productRepo.ProductRepository,
	productMediaSvc productService.ProductMediaService,
) *PromotionProductScopeServiceImpl {
	return &PromotionProductScopeServiceImpl{
		repo:            repo,
		promotionRepo:   promotionRepo,
		productRepo:     productRepo,
		productMediaSvc: productMediaSvc,
	}
}

func (s *PromotionProductScopeServiceImpl) AddProducts(
	ctx context.Context,
	req model.AddPromotionProductRequest,
) error {
	log.InfoWithContext(ctx, "Adding products to promotion scope")

	var entities []entity.PromotionProduct
	for _, pid := range req.ProductIDs {
		entities = append(entities, entity.PromotionProduct{
			PromotionID: req.PromotionID,
			ProductID:   pid,
		})
	}

	if err := s.repo.AddPromotionProducts(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add promotion products", err)
		return err
	}

	return nil
}

func (s *PromotionProductScopeServiceImpl) RemoveProducts(
	ctx context.Context,
	req model.RemovePromotionProductRequest,
) error {
	log.InfoWithContext(ctx, "Removing products from promotion scope")
	return s.repo.DeletePromotionProducts(ctx, req.PromotionID, req.ProductIDs)
}

func (s *PromotionProductScopeServiceImpl) RemoveAllProducts(
	ctx context.Context,
	promotionID uint,
) error {
	log.InfoWithContext(ctx, "Removing all products from promotion scope")
	return s.repo.DeletePromotionProductByPromotionID(ctx, promotionID)
}

func (s *PromotionProductScopeServiceImpl) GetProducts(
	ctx context.Context,
	req model.GetPromotionProductsRequest,
) (*model.GetPromotionProductsResponse, error) {
	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	products, total, err := s.repo.GetPromotionProducts(
		ctx,
		req.PromotionID,
		req.ProductIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get promotion products", err)
		return nil, err
	}

	promotion, err := s.promotionRepo.FindByID(ctx, req.PromotionID)
	if err != nil {
		return nil, err
	}

	productIDs := make([]uint, len(products))
	for i, p := range products {
		productIDs[i] = p.ProductID
	}

	productDetails := map[uint]struct {
		name string
		slug string
	}{}
	if len(productIDs) > 0 {
		rows, findErr := s.productRepo.FindByIDs(ctx, productIDs)
		if findErr == nil {
			for _, row := range rows {
				productDetails[row.ID] = struct {
					name string
					slug string
				}{name: row.Name}
			}
		}
	}

	sellerID := promotion.SellerID
	mediaByProduct, _ := s.productMediaSvc.GetMediaForProducts(ctx, productIDs, &sellerID)

	response := &model.GetPromotionProductsResponse{
		BasePromotionScopeResponse: model.BasePromotionScopeResponse{PromotionID: req.PromotionID},
		Products:                   make([]model.PromotionProductResponse, len(products)),
		Pagination:                 common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, p := range products {
		details := productDetails[p.ProductID]
		imageURL := primaryProductMediaURL(mediaByProduct[p.ProductID])
		response.Products[i] = model.PromotionProductResponse{
			BasePromotionScopeResponse: model.BasePromotionScopeResponse{
				PromotionID: req.PromotionID,
			},
			ProductID:   p.ProductID,
			ProductName: details.name,
			ProductSlug: details.slug,
			ImageURL:    imageURL,
		}
	}

	return response, nil
}

func primaryProductMediaURL(media []productModel.ProductMediaResponse) string {
	for _, m := range media {
		if m.IsPrimary && m.URL != "" {
			return m.URL
		}
	}
	for _, m := range media {
		if m.URL != "" {
			return m.URL
		}
	}
	return ""
}

func (s *PromotionProductScopeServiceImpl) IsCartEligible(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) (bool, []string) {
	cartProductIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		cartProductIDs[i] = item.ProductID
	}

	resp, err := s.GetProducts(ctx, model.GetPromotionProductsRequest{
		GetPromotionScopeRequest: model.GetPromotionScopeRequest{
			BasePromotionScopeRequest: model.BasePromotionScopeRequest{
				PromotionID: promotionID,
			},
		},
		ProductIDs: cartProductIDs,
	})
	if err != nil || resp == nil {
		return false, nil
	}

	eligibleProductIDs := make(map[uint]bool)
	for _, product := range resp.Products {
		eligibleProductIDs[product.ProductID] = true
	}

	eligibleLineItems := []string{}

	for _, item := range cart.Items {
		if eligibleProductIDs[item.ProductID] {
			eligibleLineItems = append(eligibleLineItems, item.ItemID)
		}
	}

	return len(resp.Products) > 0, eligibleLineItems
}
