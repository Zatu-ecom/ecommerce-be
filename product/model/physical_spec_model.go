package model

// ─── Shippable physical specs (013 physical-specs.md) ───────────────────────
// Served to the seller dashboard for the shipping-specs block (numeric
// input + unit dropdown) and to fulfillment for base-unit normalization.

// SpecUnitOption is one selectable unit for a measurement parameter.
type SpecUnitOption struct {
	Key  string         `json:"key"`
	Unit SpecUnitLabels `json:"unit"`
}

// SpecUnitLabels carries short + full unit names for display.
type SpecUnitLabels struct {
	Short string `json:"short"`
	Full  string `json:"full"`
}

// SpecParameterGroup groups unit options by measurement (weight, length…).
type SpecParameterGroup struct {
	Parameter string           `json:"parameter"`
	Name      string           `json:"name"`
	Options   []SpecUnitOption `json:"options"`
}

// SpecCatalogResponse is the definitions-catalog envelope.
type SpecCatalogResponse struct {
	Parameters []SpecParameterGroup `json:"parameters"`
}

// PresentSpec is one attached shippable attribute on a product.
type PresentSpec struct {
	Parameter string `json:"parameter"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Unit      string `json:"unitShort"`
}

// ShippingSpecsResponse powers the "specs missing" badge: present specs
// plus the parameters still absent.
type ShippingSpecsResponse struct {
	Present []PresentSpec `json:"present"`
	Missing []string      `json:"missing"`
}
