package service

import (
	"context"
	"math"
	"strconv"
	"strings"

	fulfillmentmodel "ecommerce-be/fulfillment/model"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
)

// PhysicalSpecService owns the shippable spec catalog (read) and the
// per-product badge, plus the write-path guard shared by product-attribute
// mutations: numeric values and one key per family.
type PhysicalSpecService interface {
	// ListFulfillmentSpecs returns active units grouped by parameter.
	ListFulfillmentSpecs(ctx context.Context) ([]model.SpecParameterGroup, error)
	// ProductShippingSpecs returns present specs + missing parameters.
	ProductShippingSpecs(ctx context.Context, sellerID uint, productID uint) (*model.ShippingSpecsResponse, error)
	// ValidateShippableValue enforces numeric > 0 values for shippable keys.
	// Non-shippable keys pass through untouched (nil, nil).
	ValidateShippableValue(ctx context.Context, definitionKey, value string) error
	// CheckFamilyConflict rejects a second key from an already-represented
	// family on the product (weight_g + weight_kg). excludeDefinitionID
	// skips the row being updated. Non-shippable keys pass untouched.
	CheckFamilyConflict(ctx context.Context, productID uint, definitionKey string, excludeDefinitionID uint) error
	// GetPhysicalSpecs implements fulfillment's FulfillmentProductHooks:
	// normalized specs per variant in base units (grams, cm). Products
	// without specs yield zero-valued fields; unparseable values are loud
	// errors, never guessed conversions or silent zeros.
	GetPhysicalSpecs(ctx context.Context, variantIDs []uint) ([]fulfillmentmodel.PhysicalSpec, error)
	// ValidateShippableBulk enforces the shippable-spec rules over a bulk
	// payload: numeric > 0 values, no second key from an already-present
	// family on the product, and no two keys from one family inside the
	// payload itself.
	ValidateShippableBulk(ctx context.Context, productID uint, requests []model.ProductAttributeRequest) error
}

// PhysicalSpecServiceImpl implements PhysicalSpecService.
type PhysicalSpecServiceImpl struct {
	specRepo         repository.PhysicalSpecRepository
	attributeRepo    repository.AttributeDefinitionRepository
	productAttrRepo  repository.ProductAttributeRepository
	variantRepo      repository.VariantRepository
	validatorService ProductValidatorService
}

// NewPhysicalSpecService builds the spec service.
func NewPhysicalSpecService(
	specRepo repository.PhysicalSpecRepository,
	attributeRepo repository.AttributeDefinitionRepository,
	productAttrRepo repository.ProductAttributeRepository,
	variantRepo repository.VariantRepository,
	validatorService ProductValidatorService,
) PhysicalSpecService {
	return &PhysicalSpecServiceImpl{
		specRepo:         specRepo,
		attributeRepo:    attributeRepo,
		productAttrRepo:  productAttrRepo,
		variantRepo:      variantRepo,
		validatorService: validatorService,
	}
}

// ListFulfillmentSpecs groups active catalog rows by parameter.
func (s *PhysicalSpecServiceImpl) ListFulfillmentSpecs(
	ctx context.Context,
) ([]model.SpecParameterGroup, error) {
	units, err := s.specRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	groups := []model.SpecParameterGroup{}
	index := map[string]int{}
	for _, unit := range units {
		i, ok := index[unit.Parameter]
		if !ok {
			groups = append(groups, model.SpecParameterGroup{
				Parameter: unit.Parameter,
				Name:      parameterDisplayName(unit.Parameter),
			})
			i = len(groups) - 1
			index[unit.Parameter] = i
		}
		groups[i].Options = append(groups[i].Options, model.SpecUnitOption{
			Key:  unit.Key,
			Unit: model.SpecUnitLabels{Short: unit.UnitShort, Full: unit.UnitFull},
		})
	}
	return groups, nil
}

// parameterDisplayName renders family names for the dashboard.
func parameterDisplayName(parameter string) string {
	if parameter == "" {
		return parameter
	}
	return strings.ToUpper(parameter[:1]) + parameter[1:]
}

// ProductShippingSpecs lists attached shippable specs plus missing families.
// Ownership is enforced: only the owning seller (or above) may view the badge.
func (s *PhysicalSpecServiceImpl) ProductShippingSpecs(
	ctx context.Context,
	sellerID uint,
	productID uint,
) (*model.ShippingSpecsResponse, error) {
	if _, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID); err != nil {
		return nil, err
	}
	units, err := s.specRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	attrs, err := s.productAttrRepo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	defUnit := s.specUnitByDefinition(ctx, units)
	present, seen := presentSpecs(attrs, defUnit)
	return &model.ShippingSpecsResponse{
		Present: present,
		Missing: missingParameters(units, seen),
	}, nil
}

// specUnitByDefinition indexes catalog rows by attribute definition id.
func (s *PhysicalSpecServiceImpl) specUnitByDefinition(
	ctx context.Context,
	units []entity.PhysicalSpecUnit,
) map[uint]entity.PhysicalSpecUnit {
	defUnit := map[uint]entity.PhysicalSpecUnit{}
	for _, unit := range units {
		def, err := s.attributeRepo.FindByID(ctx, unit.AttributeDefinitionID)
		if err != nil {
			continue
		}
		defUnit[def.ID] = unit
	}
	return defUnit
}

// presentSpecs maps attached attributes to catalog specs plus seen families.
func presentSpecs(
	attrs []entity.ProductAttribute,
	defUnit map[uint]entity.PhysicalSpecUnit,
) ([]model.PresentSpec, map[string]bool) {
	present := []model.PresentSpec{}
	seen := map[string]bool{}
	for _, attr := range attrs {
		unit, ok := defUnit[attr.AttributeDefinitionID]
		if !ok {
			continue
		}
		seen[unit.Parameter] = true
		present = append(present, model.PresentSpec{
			Parameter: unit.Parameter,
			Key:       unit.Key,
			Value:     attr.Value,
			Unit:      unit.UnitShort,
		})
	}
	return present, seen
}

// missingParameters lists catalog families with no attached spec.
func missingParameters(units []entity.PhysicalSpecUnit, seen map[string]bool) []string {
	missing := []string{}
	for _, unit := range units {
		if !seen[unit.Parameter] && !containsString(missing, unit.Parameter) {
			missing = append(missing, unit.Parameter)
		}
	}
	return missing
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// ValidateShippableValue enforces numeric > 0 for shippable keys.
func (s *PhysicalSpecServiceImpl) ValidateShippableValue(
	ctx context.Context,
	definitionKey, value string,
) error {
	unit, err := s.findSpecUnit(ctx, definitionKey)
	if err != nil || unit == nil {
		return err
	}
	trimmed := strings.TrimSpace(value)
	number, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || !(number > 0) {
		return prodErrors.ErrInvalidAttributeValue
	}
	return nil
}

// CheckFamilyConflict rejects attaching a second key from a family already
// present on the product (weight_g when weight_kg exists, etc.).
func (s *PhysicalSpecServiceImpl) CheckFamilyConflict(
	ctx context.Context,
	productID uint,
	definitionKey string,
	excludeDefinitionID uint,
) error {
	unit, err := s.findSpecUnit(ctx, definitionKey)
	if err != nil || unit == nil {
		return err
	}
	attrs, err := s.productAttrRepo.FindAllByProductID(ctx, productID)
	if err != nil {
		return err
	}
	for _, attr := range attrs {
		if attr.AttributeDefinitionID == excludeDefinitionID {
			continue
		}
		other, err := s.findSpecUnitByDefinitionID(ctx, attr.AttributeDefinitionID)
		if err != nil {
			return err
		}
		if other != nil && other.Parameter == unit.Parameter && other.Key != unit.Key {
			return prodErrors.ErrPhysicalSpecFamilyConflict
		}
	}
	return nil
}

// findSpecUnit resolves a definition key to its catalog row (nil when the
// key is not a shippable spec — unit keys double as definition keys).
func (s *PhysicalSpecServiceImpl) findSpecUnit(
	ctx context.Context,
	definitionKey string,
) (*entity.PhysicalSpecUnit, error) {
	units, err := s.specRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	for i := range units {
		if units[i].Key == definitionKey {
			unit := units[i]
			return &unit, nil
		}
	}
	return nil, nil
}

// findSpecUnitByDefinitionID resolves a definition id to its catalog row.
func (s *PhysicalSpecServiceImpl) findSpecUnitByDefinitionID(
	ctx context.Context,
	definitionID uint,
) (*entity.PhysicalSpecUnit, error) {
	units, err := s.specRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	for i := range units {
		if units[i].AttributeDefinitionID == definitionID {
			unit := units[i]
			return &unit, nil
		}
	}
	return nil, nil
}

// GetPhysicalSpecs returns normalized specs per variant in base units.
// Variant → product → product_attribute → physical_spec_unit (× factor).
// Products without specs yield absent fields; corrupt values fail loudly.
func (s *PhysicalSpecServiceImpl) GetPhysicalSpecs(
	ctx context.Context,
	variantIDs []uint,
) ([]fulfillmentmodel.PhysicalSpec, error) {
	units, err := s.specRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}
	byDefinitionID := map[uint]entity.PhysicalSpecUnit{}
	for _, unit := range units {
		byDefinitionID[unit.AttributeDefinitionID] = unit
	}

	specs := make([]fulfillmentmodel.PhysicalSpec, 0, len(variantIDs))
	for _, variantID := range variantIDs {
		variant, err := s.variantRepo.FindVariantByID(ctx, variantID)
		if err != nil {
			return nil, err
		}
		attrs, err := s.productAttrRepo.FindAllByProductID(ctx, variant.ProductID)
		if err != nil {
			return nil, err
		}
		spec := fulfillmentmodel.PhysicalSpec{VariantID: variantID}
		for _, attr := range attrs {
			unit, ok := byDefinitionID[attr.AttributeDefinitionID]
			if !ok {
				continue
			}
			if err := applySpecValue(&spec, unit, attr.Value); err != nil {
				return nil, err
			}
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// ValidateShippableBulk enforces the shippable-spec rules over a bulk
// payload (see the interface contract above).
func (s *PhysicalSpecServiceImpl) ValidateShippableBulk(
	ctx context.Context,
	productID uint,
	requests []model.ProductAttributeRequest,
) error {
	keyParameter, err := s.shippableKeyParameters(ctx)
	if err != nil {
		return err
	}
	seenParameters := map[string]string{}
	for _, req := range requests {
		if err := s.ValidateShippableValue(ctx, req.Key, req.Value); err != nil {
			return err
		}
		if err := s.CheckFamilyConflict(ctx, productID, req.Key, 0); err != nil {
			return err
		}
		if parameter, ok := keyParameter[req.Key]; ok {
			if other, seen := seenParameters[parameter]; seen && other != req.Key {
				return prodErrors.ErrPhysicalSpecFamilyConflict
			}
			seenParameters[parameter] = req.Key
		}
	}
	return nil
}

// shippableKeyParameters maps each shippable definition key to its family.
func (s *PhysicalSpecServiceImpl) shippableKeyParameters(
	ctx context.Context,
) (map[string]string, error) {
	groups, err := s.ListFulfillmentSpecs(ctx)
	if err != nil {
		return nil, err
	}
	keyParameter := map[string]string{}
	for _, group := range groups {
		for _, option := range group.Options {
			keyParameter[option.Key] = group.Parameter
		}
	}
	return keyParameter, nil
}

// applySpecValue normalizes one attached value into base units (grams, cm).
// Unparseable or non-positive values fail loudly — never guessed or zeroed.
func applySpecValue(
	spec *fulfillmentmodel.PhysicalSpec,
	unit entity.PhysicalSpecUnit,
	value string,
) error {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || !(number > 0) {
		return prodErrors.ErrInvalidAttributeValue
	}
	base := number * unit.FactorToBase
	switch unit.Parameter {
	case "weight":
		grams := int(math.Round(base))
		spec.WeightGrams = &grams
	case "length":
		spec.LengthCm = &base
	case "breadth":
		spec.BreadthCm = &base
	case "height":
		spec.HeightCm = &base
	}
	return nil
}
