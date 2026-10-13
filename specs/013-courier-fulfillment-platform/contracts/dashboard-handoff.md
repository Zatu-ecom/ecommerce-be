# Contract: dashboard handoff — shipping specs (US6)

> Backend surface for the seller-dashboard "Shipping specs" block and the
> "specs missing" badge. Catalog data lives in `physical-specs.md`; write
> rules in `physical-specs.md` §5; read path in `physical-specs.md` §6.

## Endpoints

### `GET /api/product/attribute/definitions?scope=fulfillment`

Returns the active unit catalog grouped by measurement parameter. The
frontend renders one numeric input + unit dropdown per group — no
hardcoded unit lists.

- `scope` is required and must equal `fulfillment`; anything else is
  `400 INVALID_SCOPE`. (The path leaves room for future scopes.)
- Inactive units (`is_active = false`) are hidden, never deleted.
- Shape: `{ "parameters": [{ "parameter", "name", "options":
  [{ "key", "unit": { "short", "full" } }] }] }`.
- Adding a unit later (e.g. `mm`, `lb`) is one `attribute_definition`
  row + one `physical_spec_unit` row; the UI picks it up with no changes.

### `GET /api/product/:productId/attribute/shipping-specs` (seller)

Badge source. Shape: `{ "present": [{ "parameter", "key", "value",
"unit" }], "missing": ["weight", ...] }`.

- `missing` empty ⇒ specs complete; non-empty ⇒ show the badge, not a
  blocker. Products ship without specs (planner leaves weight NULL for
  manual PATCH).
- Ownership enforced: only the owning seller (or above) may view.

## Write rules (already on the product-attribute routes)

- Shippable keys accept numerics `> 0` only (`400 INVALID_ATTRIBUTE_VALUE`).
- At most one key per family per product (`400
  PHYSICAL_SPEC_FAMILY_CONFLICT`): adding `weight_kg` while `weight_g`
  is attached is rejected — remove the old key first.
- `value` stores the number only; the unit is implied by the chosen
  definition key. Fulfillment multiplies by the catalog `factor_to_base`
  (grams/cm) at read time.

## Planner / book behavior the dashboard should reflect

- Drafts planned from specced products carry `weightGrams` (+ dims via
  max-L / max-B / stacked-H); unspecced drafts carry NULL weight.
- Booking a NULL-weight draft resolves `default_weight_grams` from the
  provider config when set, else `400 FULFILLMENT_WEIGHT_REQUIRED` — the
  dashboard should surface the manual weight PATCH there, not at plan time.
