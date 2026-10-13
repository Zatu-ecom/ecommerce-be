package service

import (
	"context"

	"ecommerce-be/fulfillment/entity"
	"ecommerce-be/fulfillment/model"
)

// draftMeasure accumulates catalog specs for one draft box: weights sum,
// dims follow the heuristic (max L, max B, stacked H).
type draftMeasure struct {
	weight                                      int
	length, breadth, height                     float64
	hasWeight, hasLength, hasBreadth, hasHeight bool
}

// draftMeasures resolves catalog specs per draft box (keyed by location).
// Variants without specs contribute nothing — the box stays NULL-weighted
// for manual PATCH. Corrupt values fail loudly via the reader: planning
// aborts rather than shipping guessed numbers.
func (p *ShipmentPlannerImpl) draftMeasures(
	ctx context.Context,
	allocations []allocation,
) (map[uint]*draftMeasure, error) {
	measures := map[uint]*draftMeasure{}
	if p.productHooks == nil {
		return measures, nil
	}
	variantIDs := uniqueAllocationVariants(allocations)
	if len(variantIDs) == 0 {
		return measures, nil
	}
	specs, err := p.productHooks.GetPhysicalSpecs(ctx, variantIDs)
	if err != nil {
		return nil, err
	}
	byVariant := map[uint]model.PhysicalSpec{}
	for _, spec := range specs {
		byVariant[spec.VariantID] = spec
	}
	for _, a := range allocations {
		if a.variantID == nil {
			continue
		}
		if spec, ok := byVariant[*a.variantID]; ok {
			measureFor(measures, a.locationID).accumulate(spec, a.quantity)
		}
	}
	return measures, nil
}

// uniqueAllocationVariants dedupes variant ids across allocations.
func uniqueAllocationVariants(allocations []allocation) []uint {
	seen := map[uint]bool{}
	var variantIDs []uint
	for _, a := range allocations {
		if a.variantID == nil || seen[*a.variantID] {
			continue
		}
		seen[*a.variantID] = true
		variantIDs = append(variantIDs, *a.variantID)
	}
	return variantIDs
}

// measureFor returns the accumulator for a draft box, creating it on demand.
func measureFor(measures map[uint]*draftMeasure, locationID uint) *draftMeasure {
	measure := measures[locationID]
	if measure == nil {
		measure = &draftMeasure{}
		measures[locationID] = measure
	}
	return measure
}

// accumulate folds one line's specs into the box: weight sums, L/B take
// the max, H stacks per unit.
func (m *draftMeasure) accumulate(spec model.PhysicalSpec, quantity int) {
	if spec.WeightGrams != nil {
		m.weight += *spec.WeightGrams * quantity
		m.hasWeight = true
	}
	if spec.LengthCm != nil {
		if !m.hasLength || *spec.LengthCm > m.length {
			m.length = *spec.LengthCm
		}
		m.hasLength = true
	}
	if spec.BreadthCm != nil {
		if !m.hasBreadth || *spec.BreadthCm > m.breadth {
			m.breadth = *spec.BreadthCm
		}
		m.hasBreadth = true
	}
	if spec.HeightCm != nil {
		m.height += *spec.HeightCm * float64(quantity)
		m.hasHeight = true
	}
}

// stamp writes accumulated measures onto a draft; absent specs leave the
// columns NULL for manual PATCH.
func (m *draftMeasure) stamp(shipment *entity.FulfillmentShipment) {
	if m == nil {
		return
	}
	if m.hasWeight {
		shipment.WeightGrams = intPtr(m.weight)
	}
	if m.hasLength {
		shipment.LengthCm = &m.length
	}
	if m.hasBreadth {
		shipment.BreadthCm = &m.breadth
	}
	if m.hasHeight {
		shipment.HeightCm = &m.height
	}
}
