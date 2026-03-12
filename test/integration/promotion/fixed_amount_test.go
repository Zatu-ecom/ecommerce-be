// Functional Integration Test Scenarios for Fixed Amount Strategy
//
// 1. Scenario: Apply flat fixed amount off the cart total (₹500 off)
//    - Setup:
//      - Insert active fixed_amount promotion (amount_cents: 50000 -> ₹500).
//      - Set 'applies_to' as 'all_products'.
//      - Insert Product A (price: ₹1000).
//    - Execution:
//      - Add Product A to cart. Call ApplyPromotionsToCart.
//    - Assertion:
//      - Verify Cart Total is ₹500.
//      - Verify total discount applied is ₹500.
//
// 2. Scenario: Fixed amount discount where discount exceeds cart total
//    - Setup:
//      - Insert active fixed_amount promotion (amount_cents: 100000 -> ₹1000).
//      - Insert Product A (price: ₹600).
//    - Execution:
//      - Add Product A to cart. Call ApplyPromotionsToCart.
//    - Assertion:
//      - Verify Cart Total is ₹0 (It should NOT be negative -₹400).
//      - Verify total discount applied is scaled exactly to ₹600.
//
// 3. Scenario: Fixed amount discount on specific target products and cart has mixed items
//    - Setup:
//      - Insert active fixed_amount promotion (amount_cents: 20000 -> ₹200).
//      - Set 'applies_to' as 'specific_products'. Call POST /api/promotion/{id}/products to Link to Variant X.
//      - Add Variant X (₹500) and Variant Y (₹500) to DB.
//    - Execution:
//      - Add both items to cart. Call API.
//    - Assertion:
//      - Verify Total discount is ₹200.
//      - Verify ₹200 discount is mapped only to Variant X in the discount details breakdown.
//
// 4. Scenario: Fixed amount discount with Minimum Order Requirement
//    - Setup:
//      - Insert active fixed_amount promotion (₹100 off, min_order: ₹1000).
//      - Insert Product A (₹800) and Product B (₹300).
//    - Execution:
//      - Step 1: Add Product A to cart -> Call API. 
//      - Step 2: Add Product B to cart -> Call API. 
//    - Assertion:
//      - Step 1: Total is ₹800 (Discount ₹0 - min_order not met).
//      - Step 2: Total is ₹1100 -> ₹1000 (Discount ₹100 - min_order met).
//
// 5. Scenario: Fixed amount discount scoped to specific Categories/Collections
//    - Setup:
//      - Insert fixed discount (₹300 off). Set applies_to 'specific_categories'. Call POST /api/promotion/{id}/categories to Link to Category A.
//    - Execution:
//      - Add Product from Category A (₹1000) and Product from Category B (₹2000) to cart.
//    - Assertion:
//      - Verify Total is ₹2700. The ₹300 discount maps only to the Category A product.
//
// 6. Scenario: Fixed amount discount restricted to Returning Customers only
//    - Setup:
//      - Insert fixed discount (₹100 off). Set 'eligible_for': 'returning_customers'.
//    - Execution:
//      - Call API with New Customer cart.
//      - Call API with Returning Customer cart (has previous COMPLETED order).
//    - Assertion:
//      - New Customer cart total discount = ₹0.
//      - Returning Customer cart total discount = ₹100.
//
// 7. Scenario: Create valid fixed amount promotion via Admin API
//    - Setup: Admin Token.
//    - Execution:
//      - POST /admin/promotions payload: { type: "fixed_amount", discount_config: { amount_cents: 50000 } }
//    - Assertion:
//      - HTTP 201 Created. DB record created correctly with parsed JSON.
//
// 8. Scenario: Create invalid fixed amount promotion (Negative amount)
//    - Setup: Admin context.
//    - Execution: POST /admin/promotions payload with amount_cents = -100.
//    - Assertion: HTTP 400 Bad Request. Error asserts amount must be greater than 0.
//
// --- SECURITY & EDGE CASES ---
//
// 9. Scenario: Multi-Seller Data Leakage (Cross-Tenant Access)
//    - Setup:
//      - Seller A creates Promotion X (Fixed ₹100 off).
//      - Cart contains only items from Seller B.
//    - Execution: Call ApplyPromotionsToCart.
//    - Assertion: Promotion X is ignored. Discount is ₹0.
//
// 10. Scenario: Unauthorized Admin Creation (RBAC)
//    - Execution: Call POST /admin/promotions using a standard Customer token.
//    - Assertion: HTTP 403 Forbidden.
//
// 11. Scenario: Extreme Boundary Values (Integer Overflow)
//    - Execution: POST /admin/promotions with amount_cents = 999999999999999.
//    - Assertion: HTTP 400 Bad Request if max value validation exists, or safely handled without crashing.

package promotion_test

// Tests will be implemented here
