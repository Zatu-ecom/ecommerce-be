# API Contract: Shared Money & Currency

**Feature**: `008-money-currency-standardization`  
**Breaking**: Yes — coordinated client release required

## CurrencyInfo

```json
{
  "code": "INR",
  "symbol": "₹",
  "decimalDigits": 2
}
```

## Money

```json
{
  "amount": 99.99,
  "amountCents": 9999,
  "formatted": "₹99.99"
}
```

### Rules

| Rule | Detail |
| ---- | ------ |
| Write | Clients send major-unit numbers only (e.g. `"price": 99.99`), never required cents |
| Currency on write | Implied by seller base currency; do not send per-field currency code |
| Read | Every monetary field is a Money object |
| Parent | Include `currency` once on the resource |
| Percentage | Plain number (e.g. `15`), not Money |
| Precision | Fractional digits must be ≤ `decimalDigits` or `400` validation error |

---

## Endpoint impact (money-bearing)

### Product / variant / package

| Direction | Change |
| --------- | ------ |
| Request | `price` remains major units; interpreted in seller currency |
| Response | `price` becomes Money; add/ensure `currency` on parent |
| Query | `minPrice` / `maxPrice` major units → server converts to cents |

### Cart (auth + guest)

| Direction | Change |
| --------- | ------ |
| Request add-item | No money fields (unchanged) |
| Response | All former unlabeled cent ints become Money; keep top-level `currency` as CurrencyInfo |
| Available coupons | Fixed `value`, thresholds, potential discount → Money; rename away from `*Cents` request-era names |

### Order

| Direction | Change |
| --------- | ------ |
| Create request | No money fields (unchanged) |
| Detail/list/create response | Replace bare `*Cents` scalars with Money fields (`subtotal`, `total`, item `unitPrice`, …) + `currency` |

### Promotion / discount-code

| Direction | Change |
| --------- | ------ |
| Request | `minPurchaseAmount`, `maxDiscountAmount`; config major keys (`amount`, …) |
| Response | Monetary fields as Money; percentages unchanged |

### Report

| Direction | Change |
| --------- | ------ |
| Response | Revenue/discount metrics as Money (or identical shape) in seller currency |

---

## Example: cart item excerpt

```json
{
  "currency": { "code": "INR", "symbol": "₹", "decimalDigits": 2 },
  "unitPrice": {
    "amount": 1299.5,
    "amountCents": 129950,
    "formatted": "₹1299.50"
  },
  "lineTotal": {
    "amount": 2599.0,
    "amountCents": 259900,
    "formatted": "₹2599.00"
  }
}
```

## Example: order totals excerpt

```json
{
  "currency": { "code": "INR", "symbol": "₹", "decimalDigits": 2 },
  "subtotal": {
    "amount": 2599.0,
    "amountCents": 259900,
    "formatted": "₹2599.00"
  },
  "total": {
    "amount": 2289.1,
    "amountCents": 228910,
    "formatted": "₹2289.10"
  }
}
```

Full cart/order samples: see pre-spec Appendix A (`plans/009-money-currency-standardization/pre-spec.md`).
