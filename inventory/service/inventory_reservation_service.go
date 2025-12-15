package service

import (
	"context"
	"time"

	"ecommerce-be/common/constants"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/validator"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/service"
)

type InventoryReservationService interface{}

type InventoryReservationServiceImpl struct {
	reservationRepo        repository.InventoryReservationRepository
	inventoryQueryService  InventoryQueryService
	variantService         service.VariantQueryService
	schedulerService       ReservationShedulerService
	inventoryManageService InventoryManageService
}

func NewInventoryReservationService(
	reservationRepo repository.InventoryReservationRepository,
	inventoryQueryService InventoryQueryService,
	variantService service.VariantQueryService,
	schedulerService ReservationShedulerService,
	inventoryManageService InventoryManageService,
) *InventoryReservationServiceImpl {
	return &InventoryReservationServiceImpl{
		reservationRepo:        reservationRepo,
		inventoryQueryService:  inventoryQueryService,
		variantService:         variantService,
		schedulerService:       schedulerService,
		inventoryManageService: inventoryManageService,
	}
}

func (s *InventoryReservationServiceImpl) CreateReservation(
	ctx context.Context,
	sellerId uint,
	req model.ReservationRequest,
) (*model.ReservationResponse, error) {
	variantIds := s.extractReqVariantIds(req.Items)

	variantInfo, err := s.variantService.GetProductBasicInfoByVariantIDs(variantIds, &sellerId)
	if err != nil {
		return nil, err
	}

	sellerVariantIdSet := s.extractSellerVariantIds(variantInfo)
	err = validator.ValidateVariantIds(variantIds, sellerVariantIdSet)
	if err != nil {
		return nil, err
	}

	inventories, err := s.inventoryQueryService.GetInventoryByVariantAndLocationPriority(
		ctx,
		req.Items,
		sellerId,
	)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiresInMinutes) * time.Minute)
	reservationEntities := s.buildreservationEntities(req, inventories, expiresAt)

	err = s.reservationRepo.CreateReservations(reservationEntities)
	if err != nil {
		return nil, err
	}

	// Schedule Redis-based expiry for automatic stock release
	if err = s.scheduleReservationExpiry(ctx, sellerId, reservationEntities); err != nil {
		return nil, err
	}

	// Reserve inventory quantities immediately
	s.reserveInventoryQuantity(ctx, sellerId, reservationEntities, inventories)

	return s.buildReservationResponse(
		req.ReferenceId,
		expiresAt,
		reservationEntities,
		inventories,
	), nil
}

func (s *InventoryReservationServiceImpl) extractReqVariantIds(
	Items []model.ReservationItem,
) []uint {
	variantIds := make([]uint, 0, len(Items))
	for _, item := range Items {
		variantIds = append(variantIds, item.VariantID)
	}
	return variantIds
}

func (s *InventoryReservationServiceImpl) extractSellerVariantIds(
	variantInfo []mapper.VariantBasicInfoRow,
) map[uint]bool {
	sellerVariantIds := make(map[uint]bool)
	for _, info := range variantInfo {
		sellerVariantIds[info.VariantID] = true
	}
	return sellerVariantIds
}

func (s *InventoryReservationServiceImpl) buildreservationEntities(
	req model.ReservationRequest,
	inventories []model.InventoryResponse,
	expiresAt time.Time,
) []*entity.InventoryReservation {
	inventoryMap := make(map[uint][]model.InventoryResponse)
	for _, inv := range inventories {
		inventoryMap[inv.VariantID] = append(inventoryMap[inv.VariantID], inv)
	}

	var reservations []*entity.InventoryReservation

	for _, item := range req.Items {
		quentity := item.ReservedQuantity
		inventoryResponses := inventoryMap[item.VariantID]

		for _, inv := range inventoryResponses {
			if quentity <= 0 {
				break
			}

			var reservationQuantity uint
			if inv.AvailableQuantity >= int(quentity) {
				reservationQuantity = quentity
				quentity = 0
			} else {
				reservationQuantity = uint(inv.AvailableQuantity)
				quentity = quentity - reservationQuantity
			}
			reservations = append(reservations, &entity.InventoryReservation{
				InventoryID: inv.ID,
				ReferenceID: req.ReferenceId,
				Quantity:    reservationQuantity,
				Status:      entity.ResPending,
				ExpiresAt:   expiresAt,
			})
		}
	}
	return reservations
}

func (s *InventoryReservationServiceImpl) buildReservationResponse(
	referenceID uint,
	expiresAt time.Time,
	reservations []*entity.InventoryReservation,
	inventories []model.InventoryResponse,
) *model.ReservationResponse {
	// Build inventory map for quick lookup of available quantity
	inventoryMap := make(map[uint]model.InventoryResponse)
	for _, inv := range inventories {
		inventoryMap[inv.ID] = inv
	}

	reservationItems := make([]model.Resevation, 0, len(reservations))
	for _, res := range reservations {
		inv := inventoryMap[res.InventoryID]
		reservationItems = append(reservationItems, model.Resevation{
			Id:                         res.ID,
			InventoryId:                res.InventoryID,
			Quantity:                   res.Quantity,
			Status:                     res.Status,
			TotalAvailableAfterReserve: inv.AvailableQuantity - int(res.Quantity),
		})
	}

	return &model.ReservationResponse{
		ReferenceId: referenceID,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		Resevations: reservationItems,
	}
}

func (s *InventoryReservationServiceImpl) scheduleReservationExpiry(
	ctx context.Context,
	sellerID uint,
	reservationEntities []*entity.InventoryReservation,
) error {
	var ReservationExpiryItems []ReservationExpiryItem
	for _, reservation := range reservationEntities {
		ReservationExpiryItems = append(ReservationExpiryItems, ReservationExpiryItem{
			ReservationID: reservation.ID,
			ExpiresAt:     reservation.ExpiresAt,
		})
	}
	return s.schedulerService.ScheduleBulkReservationExpiry(ctx, sellerID, ReservationExpiryItems)
}

func (s *InventoryReservationServiceImpl) reserveInventoryQuantity(
	ctx context.Context,
	sellerId uint,
	reservationEntities []*entity.InventoryReservation,
	inventories []model.InventoryResponse,
) {
	mapInventory := make(map[uint]model.InventoryResponse)
	for _, inv := range inventories {
		mapInventory[inv.ID] = inv
	}

	userId := ctx.Value(constants.USER_ID_KEY).(uint)
	var manageInventoryRequests []model.ManageInventoryRequest
	for _, reservation := range reservationEntities {
		inv := mapInventory[reservation.InventoryID]
		reference := string(rune(reservation.ID))
		manageInventoryRequests = append(manageInventoryRequests, model.ManageInventoryRequest{
			VariantID:       inv.VariantID,
			LocationID:      inv.LocationID,
			Quantity:        int(reservation.Quantity),
			TransactionType: entity.TXN_RESERVED,
			Reference:       &reference,
			Reason:          "Inventory reserved for reservation ID " + reference,
		})
	}
	req := model.BulkManageInventoryRequest{
		Items: manageInventoryRequests,
	}
	s.inventoryManageService.BulkManageInventory(ctx, req, sellerId, userId)
}
