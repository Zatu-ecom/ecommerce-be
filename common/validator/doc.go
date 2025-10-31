// Package validator provides generic validation utilities for request structs
//
// USAGE EXAMPLES:
//
// 1. For UPDATE requests with pointer fields (RECOMMENDED):
//
//	type ProductOptionValueUpdateRequest struct {
//	    DisplayName *string `json:"displayName" binding:"omitempty,min=1,max=100"`
//	    ColorCode   *string `json:"colorCode"   binding:"omitempty,len=7"`
//	    Position    *int    `json:"position"    binding:"omitempty"`
//	}
//
//	func (r *ProductOptionValueUpdateRequest) Validate() error {
//	    return validator.RequireAtLeastOneNonNilPointer(r)
//	}
//
//	// In handler:
//	var req model.ProductOptionValueUpdateRequest
//	if err := h.BindJSON(c, &req); err != nil {
//	    h.HandleValidationError(c, err)
//	    return
//	}
//	if err := req.Validate(); err != nil {
//	    h.HandleValidationError(c, err)
//	    return
//	}
//
// 2. For mixed type requests:
//
//	type SearchRequest struct {
//	    Query    string
//	    Category *string
//	    MinPrice *float64
//	}
//
//	func (r *SearchRequest) Validate() error {
//	    return validator.RequireAtLeastOneField(r)
//	}
//
// 3. For requests with specific updateable fields:
//
//	type UserUpdateRequest struct {
//	    Name     *string `updateable:"true"`
//	    Email    *string `updateable:"true"`
//	    Internal string  `updateable:"false"` // Won't be checked
//	}
//
//	func (r *UserUpdateRequest) Validate() error {
//	    return validator.RequireAtLeastOneWithTag(r, "updateable", "true")
//	}
//
// KEY DIFFERENCES:
//
// RequireAtLeastOneField():
//   - Checks if ANY field is non-zero (not nil, not empty string, not 0, etc.)
//   - Works with mixed types (pointers, strings, ints, bools, etc.)
//   - Use when you have non-pointer fields that can have meaningful zero values
//
// RequireAtLeastOneNonNilPointer():
//   - ONLY checks pointer fields
//   - Returns success if ANY pointer is non-nil (even if pointing to empty/zero value)
//   - Use for update requests where all fields are pointers
//   - Best for distinguishing "not provided" vs "provided but empty"
//
// RequireAtLeastOneWithTag():
//   - Only checks fields with a specific struct tag
//   - Useful when you want to exclude certain fields from validation
//   - Most flexible but requires adding tags to your struct
package validator
