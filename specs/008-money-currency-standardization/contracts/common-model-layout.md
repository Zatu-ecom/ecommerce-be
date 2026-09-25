# Contract: common/model layout (encapsulation)

**Package**: `ecommerce-be/common/model`

## Files (type + methods co-located)

| File | Types | Methods / funcs in same file |
| ---- | ----- | ---------------------------- |
| `api_response.go` | `Response`, `ErrorResponse` | `SuccessResponse`, `ErrorWithValidation`, `ErrorWithCode`, `ErrorResp` |
| `pagination.go` | `BaseListParams`, `PaginationResponse` | `(*BaseListParams).SetDefaults`, `NewPaginationResponse` |
| `validation_error.go` | `ValidationError` | helpers if any |
| `currency.go` | `CurrencyInfo` | `Factor`, digit helpers |
| `money.go` | `Money` | `NewMoney`, `FromCents`, `ToCents`/`ValidateMajorAmount`, `Format`, errors |

## Forbidden

- `common/money` package
- Split files: `money_convert.go`, `money_format.go`, `money_validate.go`, `money_errors.go`, `api_response_write.go`
- `common/model` importing `user`, `product`, `order`, `promotion`, `report`, `payment`

## Import migration

```go
// before
import "ecommerce-be/common"
common.PaginationResponse

// after
import commonModel "ecommerce-be/common/model"
commonModel.PaginationResponse
commonModel.Money
```

Optional temporary aliases in `common/response.go` for one PR series, then delete.
