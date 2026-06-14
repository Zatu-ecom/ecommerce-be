package service

import (
	"context"
	"strings"

	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

// SaleService defines the interface for sale-related business logic
type SaleService interface {
	CreateSale(
		ctx context.Context,
		req model.CreateSaleRequest,
		sellerID uint,
	) (*model.SaleResponse, error)
	UpdateSale(
		ctx context.Context,
		id uint,
		req model.UpdateSaleRequest,
		sellerID uint,
	) (*model.SaleResponse, error)
	DeleteSale(ctx context.Context, id uint, sellerID uint) error
	GetSaleByID(ctx context.Context, id uint, sellerID uint) (*model.SaleResponse, error)
	ListSales(ctx context.Context, sellerID uint) (*model.SalesResponse, error)
	UpdateStatus(
		ctx context.Context,
		id uint,
		req model.UpdateSaleStatusRequest,
		sellerID uint,
	) (*model.SaleResponse, error)
}

// SaleServiceImpl implements SaleService
type SaleServiceImpl struct {
	saleRepo repository.SaleRepository
}

// NewSaleService creates a new SaleService
func NewSaleService(saleRepo repository.SaleRepository) SaleService {
	return &SaleServiceImpl{saleRepo: saleRepo}
}

func (s *SaleServiceImpl) CreateSale(
	ctx context.Context,
	req model.CreateSaleRequest,
	sellerID uint,
) (*model.SaleResponse, error) {
	log.InfoWithContext(ctx, "Creating sale")

	if req.Slug != nil && *req.Slug != "" {
		existing, err := s.saleRepo.FindBySlugAndSeller(ctx, *req.Slug, sellerID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, promoErrors.ErrSaleSlugExists
		}
	}

	sale, err := factory.SaleRequestToEntity(req, sellerID)
	if err != nil {
		return nil, err
	}

	if err := s.saleRepo.Create(ctx, sale); err != nil {
		if isSaleUniqueViolation(err) {
			return nil, promoErrors.ErrSaleSlugExists
		}
		log.ErrorWithContext(ctx, "Failed to create sale", err)
		return nil, err
	}

	return factory.SaleEntityToResponse(sale), nil
}

func (s *SaleServiceImpl) UpdateSale(
	ctx context.Context,
	id uint,
	req model.UpdateSaleRequest,
	sellerID uint,
) (*model.SaleResponse, error) {
	log.InfoWithContext(ctx, "Updating sale")

	sale, err := s.saleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateSaleOwnership(sale, sellerID); err != nil {
		return nil, err
	}

	if req.Slug != nil && *req.Slug != "" && *req.Slug != sale.Slug {
		existing, err := s.saleRepo.FindBySlugAndSeller(ctx, *req.Slug, sellerID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, promoErrors.ErrSaleSlugExists
		}
	}

	sale, err = factory.ApplyUpdateSaleRequest(sale, req)
	if err != nil {
		return nil, err
	}

	if err := s.saleRepo.Update(ctx, sale); err != nil {
		if isSaleUniqueViolation(err) {
			return nil, promoErrors.ErrSaleSlugExists
		}
		log.ErrorWithContext(ctx, "Failed to update sale", err)
		return nil, err
	}

	return factory.SaleEntityToResponse(sale), nil
}

func (s *SaleServiceImpl) DeleteSale(ctx context.Context, id uint, sellerID uint) error {
	log.InfoWithContext(ctx, "Deleting sale")

	sale, err := s.saleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := validateSaleOwnership(sale, sellerID); err != nil {
		return err
	}

	if err := s.saleRepo.Delete(ctx, id); err != nil {
		log.ErrorWithContext(ctx, "Failed to delete sale", err)
		return err
	}

	return nil
}

func (s *SaleServiceImpl) GetSaleByID(
	ctx context.Context,
	id uint,
	sellerID uint,
) (*model.SaleResponse, error) {
	sale, err := s.saleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateSaleOwnership(sale, sellerID); err != nil {
		return nil, err
	}
	return factory.SaleEntityToResponse(sale), nil
}

func (s *SaleServiceImpl) ListSales(
	ctx context.Context,
	sellerID uint,
) (*model.SalesResponse, error) {
	sales, err := s.saleRepo.FindAllBySellerID(ctx, sellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to list sales", err)
		return nil, err
	}

	responses := make([]model.SaleResponse, 0, len(sales))
	for i := range sales {
		responses = append(responses, *factory.SaleEntityToResponse(&sales[i]))
	}

	return &model.SalesResponse{Sales: responses}, nil
}

func (s *SaleServiceImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	req model.UpdateSaleStatusRequest,
	sellerID uint,
) (*model.SaleResponse, error) {
	log.InfoWithContext(ctx, "Updating sale status")

	sale, err := s.saleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateSaleOwnership(sale, sellerID); err != nil {
		return nil, err
	}

	if err := validateStatusTransition(sale.Status, req.Status); err != nil {
		return nil, err
	}

	if err := s.saleRepo.UpdateStatus(ctx, id, req.Status); err != nil {
		log.ErrorWithContext(ctx, "Failed to update sale status", err)
		return nil, err
	}

	sale.Status = req.Status
	return factory.SaleEntityToResponse(sale), nil
}

func validateSaleOwnership(sale *entity.Sale, sellerID uint) error {
	if sale.SellerID != sellerID {
		return promoErrors.ErrUnauthorizedSaleAccess
	}
	return nil
}

func isSaleUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique_seller_sale_slug") || strings.Contains(msg, "duplicate key")
}

// ValidateSaleForPromotion ensures a sale exists and belongs to the seller when linking to a promotion
func ValidateSaleForPromotion(ctx context.Context, saleRepo repository.SaleRepository, saleID *uint, sellerID uint) error {
	if saleID == nil || *saleID == 0 {
		return nil
	}

	sale, err := saleRepo.FindByID(ctx, *saleID)
	if err != nil {
		return promoErrors.ErrInvalidSaleForPromotion
	}
	if sale.SellerID != sellerID {
		return promoErrors.ErrInvalidSaleForPromotion
	}
	return nil
}
