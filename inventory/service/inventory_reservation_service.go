package service

import (
	"context"
	"strings"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/validator"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/service"
)

type InventoryReservationService interface {
	CreateReservation(
		ctx context.Context,
		sellerId uint,
		req model.ReservationRequest,
	) (*model.ReservationResponse, error)

	ExpireScheduleReservation(
		ctx context.Context,
		sellerId uint,
		reservationExpiry model.ReservationExpiryPayload,
	) error
}

type InventoryReservationServiceImpl struct {
	reservationRepo        repository.InventoryReservationRepository
	inventoryQueryService  InventoryQueryService
	variantService         service.VariantQueryService
	schedulerService       ReservationSchedulerService
	inventoryManageService InventoryManageService
}

func NewInventoryReservationService(
	reservationRepo repository.InventoryReservationRepository,
	inventoryQueryService InventoryQueryService,
	variantService service.VariantQueryService,
	schedulerService ReservationSchedulerService,
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
	if err = validator.ValidateVariantIds(variantIds, sellerVariantIdSet); err != nil {
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

	if err = s.reservationRepo.CreateReservations(reservationEntities); err != nil {
		return nil, err
	}

	// Schedule Redis-based expiry for automatic stock release
	if err = s.scheduleReservationExpiry(ctx, sellerId, reservationEntities, expiresAt, req.ReferenceId); err != nil {
		return nil, err
	}

	// Reserve inventory quantities immediately
	if err = s.manageInventoryQuantity(ctx, sellerId, entity.TXN_RESERVED, reservationEntities, inventories); err != nil {
		return nil, err
	}

	return s.buildReservationResponse(
		req.ReferenceId,
		expiresAt,
		reservationEntities,
		inventories,
	), nil
}

func (s *InventoryReservationServiceImpl) ExpireScheduleReservation(
	ctx context.Context,
	sellerId uint,
	reservationExpiry model.ReservationExpiryPayload,
) error {
	if err := s.reservationRepo.UpdateStatusByIDs(reservationExpiry.ReservationIDs, entity.ResExpired); err != nil {
		return err
	}

	inventoryReservations, err := s.reservationRepo.FindByIDs(reservationExpiry.ReservationIDs)
	if err != nil {
		return err
	}

	inventoryMap, err := helper.BatchFetch(
		ctx,
		s.extractInventoryIDs(inventoryReservations),
		100,
		func(inventoryIDs []uint) (map[uint]*model.InventoryResponse, error) {
			return s.callGetInventories(ctx, &sellerId, inventoryIDs)
		},
	)
	if err != nil {
		return err
	}

	// Convert map values to slice for manageInventoryQuantity
	inventories := make([]model.InventoryResponse, 0, len(inventoryMap))
	for _, inv := range inventoryMap {
		if inv != nil {
			inventories = append(inventories, *inv)
		}
	}

	return s.manageInventoryQuantity(
		ctx,
		sellerId,
		entity.TXN_RELEASED,
		inventoryReservations,
		inventories,
	)
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
	expiresAt time.Time,
	referenceID uint,
) error {
	reservationIDs := make([]uint, 0, len(reservationEntities))
	for _, reservation := range reservationEntities {
		reservationIDs = append(reservationIDs, reservation.ID)
	}

	jobID, err := s.schedulerService.ScheduleBulkReservationExpiry(
		ctx,
		sellerID,
		referenceID,
		reservationIDs,
		expiresAt,
	)
	if err != nil {
		return err
	}

	log.InfoWithContext(ctx,
		"Scheduled reservation expiry job with ID: "+jobID.String(),
	)
	return nil
}

func (s *InventoryReservationServiceImpl) manageInventoryQuantity(
	ctx context.Context,
	sellerId uint,
	transactionType entity.TransactionType,
	reservationEntities []*entity.InventoryReservation,
	inventories []model.InventoryResponse,
) error {
	mapInventory := make(map[uint]model.InventoryResponse)
	for _, inv := range inventories {
		mapInventory[inv.ID] = inv
	}

	reason := "Inventory " + strings.ToLower(string(transactionType)) + " for reservation ID "
	userId := ctx.Value(constants.USER_ID_KEY).(uint)
	var manageInventoryRequests []model.ManageInventoryRequest
	for _, reservation := range reservationEntities {
		inv := mapInventory[reservation.InventoryID]
		reference := string(reservation.ID)
		manageInventoryRequests = append(manageInventoryRequests, model.ManageInventoryRequest{
			VariantID:       inv.VariantID,
			LocationID:      inv.LocationID,
			Quantity:        int(reservation.Quantity),
			TransactionType: transactionType,
			Reference:       &reference,
			Reason:          reason + reference,
		})
	}
	req := model.BulkManageInventoryRequest{
		Items: manageInventoryRequests,
	}
	_, err := s.inventoryManageService.BulkManageInventory(ctx, req, sellerId, userId)
	return err
}

func (s *InventoryReservationServiceImpl) callGetInventories(
	ctx context.Context,
	sellerId *uint,
	inventoryIDs []uint,
) (map[uint]*model.InventoryResponse, error) {
	response, err := s.inventoryQueryService.GetInventories(
		ctx,
		sellerId,
		model.GetInventoriesFilter{
			GetInventoriesBase: model.GetInventoriesBase{
				BaseListParams: common.BaseListParams{
					Page:     1,
					PageSize: len(inventoryIDs),
				},
			},
			IDs: inventoryIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	inventoryMap := make(map[uint]*model.InventoryResponse)
	for _, inv := range response.Inventories {
		invCopy := inv
		inventoryMap[inv.ID] = &invCopy
	}
	return inventoryMap, nil
}

func (s *InventoryReservationServiceImpl) extractInventoryIDs(
	reservations []*entity.InventoryReservation,
) []uint {
	inventoryIDs := make([]uint, 0, len(reservations))
	for _, res := range reservations {
		inventoryIDs = append(inventoryIDs, res.InventoryID)
	}
	return inventoryIDs
}
