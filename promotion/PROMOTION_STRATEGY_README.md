# Promotion Strategy Pattern

## 📁 Structure

```
promotion/
├── promotionStrategy/           # Strategy pattern implementations
│   ├── promotion_strategy.go    # Interface definition
│   ├── strategy_factory.go      # Factory to get strategy by type
│   ├── percentage_strategy.go   # Percentage discount strategy
│   ├── fixed_amount_strategy.go # Fixed amount strategy
│   ├── free_shipping_strategy.go# Free shipping strategy
│   ├── buy_x_get_y_strategy.go  # Buy X Get Y strategy
│   ├── bundle_strategy.go       # Bundle discount strategy
│   ├── tiered_strategy.go       # Tiered pricing strategy
│   └── flash_sale_strategy.go   # Flash sale strategy
```

## 🎯 Interface

```go
type PromotionStrategy interface {
    // Validates the discount config structure
    ValidateConfig(config map[string]interface{}) error
    
    // Validates if promotion can be applied to cart
    ValidateCart(
        ctx context.Context,
        promotion *entity.Promotion,
        cart *model.CartValidationRequest,
    ) (*model.PromotionValidationResult, error)
    
    // Calculates discount amount
    CalculateDiscount(
        ctx context.Context,
        promotion *entity.Promotion,
        cart *model.CartValidationRequest,
    ) (int64, error)
}
```

## 🏭 Factory Usage

```go
import "ecommerce-be/promotion/promotionStrategy"

// Get strategy for a promotion type
strategy := promotionStrategy.GetPromotionStrategy(entity.PromoTypePercentage)

// Validate config
err := strategy.ValidateConfig(discountConfig)

// Validate cart
result, err := strategy.ValidateCart(ctx, promotion, cart)

// Calculate discount
discount, err := strategy.CalculateDiscount(ctx, promotion, cart)
```

## 📝 Implementation Details

### Each Strategy Implements:

1. **ValidateConfig** - Validates discount_config structure
   - Unmarshals to typed config model
   - Validates required fields
   - Validates field values

2. **ValidateCart** - Checks if promotion applies to cart
   - Checks minimum purchase amount
   - Checks promotion-specific conditions
   - Calculates discount amount
   - Returns validation result

3. **CalculateDiscount** - Returns discount amount
   - Calls ValidateCart internally
   - Returns 0 if not valid
   - Returns discount amount if valid

## 🔧 Adding New Promotion Type

1. **Define config model** in `model/discount_config_model.go`:
```go
type NewTypeConfig struct {
    Field1 string `json:"field1" binding:"required"`
    Field2 int    `json:"field2" binding:"required,min=1"`
}
```

2. **Create strategy** in `promotionStrategy/new_type_strategy.go`:
```go
type NewTypeStrategy struct{}

func NewNewTypeStrategy() PromotionStrategy {
    return &NewTypeStrategy{}
}

func (s *NewTypeStrategy) ValidateConfig(config map[string]interface{}) error {
    // Implementation
}

func (s *NewTypeStrategy) ValidateCart(...) (*model.PromotionValidationResult, error) {
    // Implementation
}

func (s *NewTypeStrategy) CalculateDiscount(...) (int64, error) {
    // Implementation
}
```

3. **Register in factory** in `strategy_factory.go`:
```go
case entity.PromoTypeNewType:
    return NewNewTypeStrategy()
```

## ✅ Benefits

- **Single Responsibility** - Each strategy handles one promotion type
- **Open/Closed** - Add new types without modifying existing code
- **Type Safety** - Strongly-typed config models
- **Testable** - Each strategy can be tested independently
- **Maintainable** - Clear separation of concerns

## 🚀 Usage in Service

The service layer uses the factory to get the appropriate strategy:

```go
// In CreatePromotion
strategy := promotionStrategy.GetPromotionStrategy(req.PromotionType)
if err := strategy.ValidateConfig(req.DiscountConfig); err != nil {
    return nil, err
}

// In ValidatePromotionForCart
strategy := promotionStrategy.GetPromotionStrategy(promotion.PromotionType)
return strategy.ValidateCart(ctx, promotion, cart)
```

## 📊 Strategy Implementations

| Strategy | Config Model | Key Validations |
|----------|-------------|-----------------|
| Percentage | `PercentageDiscountConfig` | 0.01 ≤ percentage ≤ 100 |
| Fixed Amount | `FixedAmountConfig` | amount_cents > 0 |
| Free Shipping | `FreeShippingConfig` | Optional min_order_cents |
| Buy X Get Y | `BuyXGetYConfig` | buy_quantity, get_quantity > 0 |
| Bundle | `BundleConfig` | bundle_items not empty |
| Tiered | `TieredConfig` | tiers not empty, valid tier_type |
| Flash Sale | `FlashSaleConfig` | discount_value > 0, stock limits |

## 🎯 Build Status

✅ All strategies compile successfully
✅ Service layer integrated
✅ Ready for production use
