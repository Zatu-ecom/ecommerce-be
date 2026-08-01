package service

import (
	"context"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/log"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

// DiscountCodeService defines seller-facing discount-code CRUD operations.
type DiscountCodeService interface {
	CreateDiscountCode(
		ctx context.Context,
		req model.CreateDiscountCodeRequest,
		sellerID uint,
	) (*model.DiscountCodeResponse, error)
	UpdateDiscountCode(
		ctx context.Context,
		id uint,
		req model.UpdateDiscountCodeRequest,
		sellerID uint,
	) (*model.DiscountCodeResponse, error)
	DeleteDiscountCode(ctx context.Context, id uint, sellerID uint) error
	GetDiscountCodeByID(
		ctx context.Context,
		id uint,
		sellerID uint,
	) (*model.DiscountCodeResponse, error)
	ListDiscountCodes(
		ctx context.Context,
		req model.ListDiscountCodesRequest,
	) (*model.ListDiscountCodesResponse, error)
	UpdateStatus(
		ctx context.Context,
		id uint,
		req model.UpdateDiscountCodeStatusRequest,
		sellerID uint,
	) (*model.DiscountCodeResponse, error)
}

// DiscountCodeServiceImpl implements DiscountCodeService.
type DiscountCodeServiceImpl struct {
	discountCodeRepo repository.DiscountCodeRepository
	usageRepo        repository.DiscountCodeUsageRepository
}

// NewDiscountCodeService creates a new DiscountCodeService.
func NewDiscountCodeService(
	discountCodeRepo repository.DiscountCodeRepository,
	usageRepo repository.DiscountCodeUsageRepository,
) DiscountCodeService {
	return &DiscountCodeServiceImpl{
		discountCodeRepo: discountCodeRepo,
		usageRepo:        usageRepo,
	}
}

func (s *DiscountCodeServiceImpl) CreateDiscountCode(
	ctx context.Context,
	req model.CreateDiscountCodeRequest,
	sellerID uint,
) (*model.DiscountCodeResponse, error) {
	log.InfoWithContext(ctx, "Creating discount code")

	normalized := factory.NormalizeDiscountCode(req.Code)
	if normalized == "" {
		return nil, promoErrors.ErrInvalidDiscountCodeValue.WithMessage("code is required")
	}

	existing, err := s.discountCodeRepo.FindByCode(ctx, sellerID, normalized)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, promoErrors.ErrDiscountCodeExists
	}

	code, err := factory.DiscountCodeRequestToEntity(req, sellerID)
	if err != nil {
		return nil, err
	}
	code.Code = normalized

	if err := s.discountCodeRepo.Create(ctx, code); err != nil {
		if isDiscountCodeUniqueViolation(err) {
			return nil, promoErrors.ErrDiscountCodeExists
		}
		log.ErrorWithContext(ctx, "Failed to create discount code", err)
		return nil, err
	}

	return factory.DiscountCodeEntityToResponse(code), nil
}

func (s *DiscountCodeServiceImpl) UpdateDiscountCode(
	ctx context.Context,
	id uint,
	req model.UpdateDiscountCodeRequest,
	sellerID uint,
) (*model.DiscountCodeResponse, error) {
	log.InfoWithContext(ctx, "Updating discount code")

	code, err := s.findOwnedOrNotFound(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}

	code, err = factory.ApplyUpdateDiscountCodeRequest(code, req)
	if err != nil {
		return nil, err
	}

	if err := s.discountCodeRepo.Update(ctx, code); err != nil {
		log.ErrorWithContext(ctx, "Failed to update discount code", err)
		return nil, err
	}

	return factory.DiscountCodeEntityToResponse(code), nil
}

func (s *DiscountCodeServiceImpl) DeleteDiscountCode(
	ctx context.Context,
	id uint,
	sellerID uint,
) error {
	log.InfoWithContext(ctx, "Deleting discount code")

	if _, err := s.findOwnedOrNotFound(ctx, id, sellerID); err != nil {
		return err
	}

	usageCount, err := s.discountCodeRepo.CountUsage(ctx, id)
	if err != nil {
		return err
	}
	if usageCount > 0 {
		return promoErrors.ErrDiscountCodeHasUsage
	}

	if err := s.discountCodeRepo.Delete(ctx, id); err != nil {
		log.ErrorWithContext(ctx, "Failed to delete discount code", err)
		return err
	}
	return nil
}

func (s *DiscountCodeServiceImpl) GetDiscountCodeByID(
	ctx context.Context,
	id uint,
	sellerID uint,
) (*model.DiscountCodeResponse, error) {
	code, err := s.findOwnedOrNotFound(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}
	return factory.DiscountCodeEntityToResponse(code), nil
}

func (s *DiscountCodeServiceImpl) ListDiscountCodes(
	ctx context.Context,
	req model.ListDiscountCodesRequest,
) (*model.ListDiscountCodesResponse, error) {
	req.SetDefaults()

	codes, total, err := s.discountCodeRepo.List(ctx, repository.ListDiscountCodeFilter{
		SellerID:     req.SellerID,
		IsActive:     req.IsActive,
		DiscountType: req.DiscountType,
		AppliesTo:    req.AppliesTo,
		Page:         req.Page,
		Limit:        req.PageSize,
	})
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to list discount codes", err)
		return nil, err
	}

	responses := make([]model.DiscountCodeResponse, 0, len(codes))
	for _, code := range codes {
		responses = append(responses, *factory.DiscountCodeEntityToResponse(code))
	}

	return &model.ListDiscountCodesResponse{
		DiscountCodes: responses,
		Pagination:    common.NewPaginationResponse(req.Page, req.PageSize, total),
	}, nil
}

func (s *DiscountCodeServiceImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	req model.UpdateDiscountCodeStatusRequest,
	sellerID uint,
) (*model.DiscountCodeResponse, error) {
	log.InfoWithContext(ctx, "Updating discount code status")

	code, err := s.findOwnedOrNotFound(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}

	if err := s.discountCodeRepo.UpdateActive(ctx, id, req.IsActive); err != nil {
		log.ErrorWithContext(ctx, "Failed to update discount code status", err)
		return nil, err
	}

	isActive := req.IsActive
	code.IsActive = &isActive
	return factory.DiscountCodeEntityToResponse(code), nil
}

func (s *DiscountCodeServiceImpl) findOwnedOrNotFound(
	ctx context.Context,
	id uint,
	sellerID uint,
) (*entity.DiscountCode, error) {
	code, err := s.discountCodeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if code.SellerID != sellerID {
		return nil, promoErrors.ErrDiscountCodeNotFound
	}
	return code, nil
}

func isDiscountCodeUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique_seller_discount_code") ||
		strings.Contains(msg, "duplicate key")
}
