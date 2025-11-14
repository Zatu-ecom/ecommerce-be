# Test Scenarios: GetProductByID API

## Feature Information

- **Endpoint**: `GET /api/products/:productId`
- **Method**: GET
- **Authentication**: Public API (JWT token optional, but requires X-Seller-ID header if no token)
- **User Roles**: Public (with X-Seller-ID), Customer, Seller, Admin
- **Description**: Retrieves detailed information about a specific product including variants, options, attributes, and package options

---

## API Behavior Overview

### Multi-Tenant Isolation Rules:

1. **Public Access (No Token)**: Requires `X-Seller-ID` header. Only returns products belonging to that seller.
2. **Customer Access**: Can access all products from any seller.
3. **Seller Access**: Can only access their own products. Returns 404 for products belonging to other sellers.
4. **Admin Access**: Can access any product from any seller.

### Response Includes:

- Basic product information (name, brand, SKU, descriptions, tags)
- Category hierarchy (parent category information)
- Product attributes (specifications)
- Package options
- Complete variant information with options
- Price range from variants
- Stock availability from variants

---

## Test Scenarios

### [Happy Path] - HP-01: Public User Gets Product Successfully with Valid Seller ID

**Scenario ID**: HP-01  
**Given**:

- Product with ID 101 exists for seller ID 5
- No JWT token is provided
- Valid `X-Seller-ID: 5` header is provided

**When**: GET request to `/api/products/101` with `X-Seller-ID: 5` header

**Then**: Product details are returned successfully with all nested data

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Response contains product ID, name, brand, SKU
- [ ] Category information includes parent category (if exists)
- [ ] Product attributes array is present (may be empty)
- [ ] Package options array is present (may be empty)
- [ ] Variants array contains at least one variant with complete option details
- [ ] Options array shows all available product options with values
- [ ] Price range (min/max) is calculated from variants
- [ ] `hasVariants` is true
- [ ] `allowPurchase` reflects if any variant is purchasable
- [ ] Timestamps (createdAt, updatedAt) are in RFC3339 format
- [ ] Response structure matches `ProductResponse` model

---

### [Happy Path] - HP-02: Customer Gets Product Successfully

**Scenario ID**: HP-02  
**Given**:

- Product with ID 101 exists
- Valid customer JWT token is provided
- Customer role is authenticated

**When**: GET request to `/api/products/101` with Authorization header

**Then**: Product details are returned successfully

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Product details are complete regardless of seller
- [ ] All nested data (variants, options, attributes) is included
- [ ] Customer can view products from any seller
- [ ] Response includes variant details with images
- [ ] No sensitive seller information is exposed

---

### [Happy Path] - HP-03: Seller Gets Their Own Product

**Scenario ID**: HP-03  
**Given**:

- Product with ID 101 exists and belongs to seller ID 5
- Valid seller JWT token for seller ID 5
- Seller is authenticated

**When**: GET request to `/api/products/101` with seller Authorization header

**Then**: Product details are returned successfully

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Seller can view complete details of their own product
- [ ] All variant information is accessible
- [ ] Product attributes and options are fully detailed
- [ ] SellerID in response matches authenticated seller ID

---

### [Happy Path] - HP-04: Admin Gets Any Product

**Scenario ID**: HP-04  
**Given**:

- Product with ID 101 exists for any seller
- Valid admin JWT token is provided
- Admin role is authenticated

**When**: GET request to `/api/products/101` with admin Authorization header

**Then**: Product details are returned successfully

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Admin can view product from any seller
- [ ] Complete product information is returned
- [ ] No multi-tenant restrictions apply
- [ ] All nested data is accessible

---

### [Happy Path] - HP-05: Product with Multiple Variants and Options Retrieved

**Scenario ID**: HP-05  
**Given**:

- Product with ID 101 has multiple options (Color: Red, Blue; Size: S, M, L)
- Product has 6 variants (combinations of options)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: All variants and options are returned with complete details

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Options array contains 2 options (Color, Size)
- [ ] Each option has correct values with display names
- [ ] Variants array contains all 6 variants
- [ ] Each variant has selectedOptions array showing its option combination
- [ ] Each variant has unique SKU, price, images
- [ ] Price range shows min and max across all variants
- [ ] Variant preview shows total count and available option values
- [ ] Default variant is marked with `isDefault: true`

---

### [Happy Path] - HP-06: Product with Attributes and Package Options

**Scenario ID**: HP-06  
**Given**:

- Product with ID 101 has product attributes (Weight: 500g, Dimensions: 10x20x5)
- Product has package options (Gift Wrap, Express Shipping)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product returned with attributes and package options

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Attributes array contains all product attributes
- [ ] Each attribute has key, name, value, unit, sortOrder
- [ ] Package options array contains all options
- [ ] Each package option has name, description, price, quantity
- [ ] Attributes are sorted by sortOrder
- [ ] Package option prices are positive numbers

---

### [Happy Path] - HP-07: Product with Category Hierarchy

**Scenario ID**: HP-07  
**Given**:

- Product with ID 101 belongs to category "Smartphones" (ID: 20)
- Category "Smartphones" has parent category "Electronics" (ID: 10)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product returned with complete category hierarchy

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Category object contains ID and name
- [ ] Category.parent object is present
- [ ] Parent category ID and name are correct
- [ ] Category hierarchy is properly nested
- [ ] CategoryID field matches category.id

---

### [Negative] - NEG-01: Product ID Does Not Exist

**Scenario ID**: NEG-01  
**Given**:

- Product with ID 99999 does not exist in database
- Valid authentication context with seller ID 5

**When**: GET request to `/api/products/99999`

**Then**: 404 Not Found error is returned

**Expected Status Code**: 404 Not Found

**Validation Points**:

- [ ] Error message states "Product not found"
- [ ] Error code is `PRODUCT_NOT_FOUND`
- [ ] No product data is returned
- [ ] Response follows standard error format
- [ ] No sensitive database information is leaked

---

### [Negative] - NEG-02: Invalid Product ID Format (Non-Numeric)

**Scenario ID**: NEG-02  
**Given**:

- Product ID parameter is non-numeric string "abc"
- Valid authentication context

**When**: GET request to `/api/products/abc`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] Error message indicates "Invalid product ID"
- [ ] Clear validation error is returned
- [ ] No database query is attempted
- [ ] Response follows standard error format

---

### [Negative] - NEG-03: Invalid Product ID (Negative Number)

**Scenario ID**: NEG-03  
**Given**:

- Product ID parameter is negative number "-5"
- Valid authentication context

**When**: GET request to `/api/products/-5`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] Error message indicates invalid product ID
- [ ] Negative numbers are rejected
- [ ] No database query is attempted

---

### [Negative] - NEG-04: Invalid Product ID (Zero)

**Scenario ID**: NEG-04  
**Given**:

- Product ID parameter is "0"
- Valid authentication context

**When**: GET request to `/api/products/0`

**Then**: 400 Bad Request or 404 Not Found error is returned

**Expected Status Code**: 400 Bad Request or 404 Not Found

**Validation Points**:

- [ ] Zero ID is rejected or not found
- [ ] Appropriate error message is returned
- [ ] No internal server error occurs

---

### [Negative] - NEG-05: Product ID Exceeds Maximum Integer Value

**Scenario ID**: NEG-05  
**Given**:

- Product ID parameter is extremely large number "999999999999999999999"
- Valid authentication context

**When**: GET request to `/api/products/999999999999999999999`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] Overflow is handled gracefully
- [ ] Error message indicates invalid product ID
- [ ] No internal server error or panic occurs
- [ ] System remains stable

---

### [Negative] - NEG-06: Missing Product ID Parameter

**Scenario ID**: NEG-06  
**Given**:

- No product ID provided in URL path
- Valid authentication context

**When**: GET request to `/api/products/` (trailing slash, no ID)

**Then**: 404 Not Found error (route not matched) or 400 Bad Request

**Expected Status Code**: 404 Not Found or 400 Bad Request

**Validation Points**:

- [ ] Missing parameter is caught by router or handler
- [ ] Appropriate error message is returned
- [ ] No internal server error occurs

---

### [Edge Case] - EDGE-01: Product ID with Special Characters

**Scenario ID**: EDGE-01  
**Given**:

- Product ID parameter contains special characters "101<script>alert('xss')</script>"
- Valid authentication context

**When**: GET request to `/api/products/101<script>alert('xss')</script>`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] Special characters are rejected
- [ ] XSS payload is not executed or stored
- [ ] Input is properly sanitized
- [ ] Error response does not echo malicious input

---

### [Edge Case] - EDGE-02: Product ID with SQL Injection Attempt

**Scenario ID**: EDGE-02  
**Given**:

- Product ID parameter is SQL injection payload "101' OR '1'='1"
- Valid authentication context

**When**: GET request to `/api/products/101' OR '1'='1`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] SQL injection is prevented
- [ ] Parameterized queries protect against SQL injection
- [ ] Invalid format is rejected at parsing stage
- [ ] No unauthorized data access occurs
- [ ] Database remains secure

---

### [Edge Case] - EDGE-04: Product with Unicode Characters in Data

**Scenario ID**: EDGE-04  
**Given**:

- Product with ID 101 has Unicode characters in name (中文, العربية, 😀)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product is returned with Unicode preserved

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Unicode characters in product name are preserved
- [ ] UTF-8 encoding is correctly handled
- [ ] Special characters in descriptions are intact
- [ ] Emojis in tags are displayed correctly
- [ ] Response content-type is UTF-8

---

### [Edge Case] - EDGE-05: Product with Empty Optional Fields

**Scenario ID**: EDGE-05  
**Given**:

- Product with ID 101 has empty brand, tags, descriptions
- Product has minimal required fields only
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product is returned with empty strings for optional fields

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Brand is empty string ""
- [ ] Tags array is empty []
- [ ] Short and long descriptions are empty strings
- [ ] Required fields (name, categoryId) are present
- [ ] Response structure is valid
- [ ] No null pointer errors occur

---

### [Edge Case] - EDGE-06: Product with Maximum Field Lengths

**Scenario ID**: EDGE-06  
**Given**:

- Product with ID 101 has maximum length values:
  - Name: 200 characters
  - Brand: 100 characters
  - ShortDescription: 500 characters
  - LongDescription: 5000 characters
  - Tags: 20 items
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product is returned with full-length fields

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] All maximum-length fields are returned completely
- [ ] No truncation occurs in response
- [ ] Response JSON is valid
- [ ] No performance degradation
- [ ] Character encoding is preserved

---

### [Security] - SEC-01: Seller Cannot Access Another Seller's Product

**Scenario ID**: SEC-01  
**Given**:

- Product with ID 101 belongs to seller ID 5
- Valid JWT token for seller ID 10 (different seller)
- Seller 10 is authenticated

**When**: GET request to `/api/products/101` with seller 10's token

**Then**: 404 Not Found error is returned (product hidden from other sellers)

**Expected Status Code**: 404 Not Found

**Validation Points**:

- [ ] Multi-tenant isolation is enforced
- [ ] Seller cannot discover products from other sellers
- [ ] Error message does not reveal product exists
- [ ] Same error as non-existent product
- [ ] Authorization check is performed in service layer

---

### [Security] - SEC-02: Public Access Without Seller ID Header

**Scenario ID**: SEC-02  
**Given**:

- Product with ID 101 exists
- No JWT token provided
- No `X-Seller-ID` header provided

**When**: GET request to `/api/products/101` without authentication headers

**Then**: 400 Bad Request or 401 Unauthorized error is returned

**Expected Status Code**: 400 Bad Request or 401 Unauthorized

**Validation Points**:

- [ ] Request is rejected without seller context
- [ ] Error message requires X-Seller-ID header
- [ ] Public API middleware enforces seller ID requirement
- [ ] No product data is returned
- [ ] Multi-tenant isolation is maintained

---

### [Security] - SEC-03: Invalid Seller ID in Header

**Scenario ID**: SEC-03  
**Given**:

- Product with ID 101 exists
- No JWT token provided
- Invalid `X-Seller-ID: abc` header (non-numeric)

**When**: GET request to `/api/products/101` with invalid X-Seller-ID

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] Invalid seller ID format is rejected
- [ ] Validation error is clear
- [ ] No product data is returned
- [ ] Middleware validates seller ID format

---

### [Security] - SEC-04: Non-Existent Seller ID in Header

**Scenario ID**: SEC-04  
**Given**:

- Product with ID 101 exists for seller 5
- No JWT token provided
- Valid format but non-existent `X-Seller-ID: 99999` header

**When**: GET request to `/api/products/101` with non-existent seller ID

**Then**: 404 Not Found or 403 Forbidden error is returned

**Expected Status Code**: 404 Not Found or 403 Forbidden

**Validation Points**:

- [ ] Non-existent seller is validated
- [ ] Product not found for that seller context
- [ ] Middleware validates seller existence
- [ ] Clear error message is returned

---

### [Security] - SEC-05: Expired JWT Token

**Scenario ID**: SEC-05  
**Given**:

- Product with ID 101 exists
- Expired JWT token is provided
- Token was previously valid

**When**: GET request to `/api/products/101` with expired token

**Then**: 401 Unauthorized error is returned

**Expected Status Code**: 401 Unauthorized

**Validation Points**:

- [ ] Expired token is rejected
- [ ] Error message indicates token expiration
- [ ] No product data is returned
- [ ] User is prompted to re-authenticate
- [ ] Token expiration is checked by auth middleware

---

### [Security] - SEC-06: Malformed JWT Token

**Scenario ID**: SEC-06  
**Given**:

- Product with ID 101 exists
- Malformed JWT token "Bearer invalid.token.format"

**When**: GET request to `/api/products/101` with malformed token

**Then**: 401 Unauthorized error is returned

**Expected Status Code**: 401 Unauthorized

**Validation Points**:

- [ ] Malformed token is rejected
- [ ] JWT validation catches format errors
- [ ] Error message indicates invalid token
- [ ] No product data is returned
- [ ] System handles JWT parsing errors gracefully

---

### [Security] - SEC-07: SQL Injection in Product ID

**Scenario ID**: SEC-07  
**Given**:

- Product ID parameter is "101 UNION SELECT \* FROM users--"
- Valid authentication context

**When**: GET request to `/api/products/101 UNION SELECT * FROM users--`

**Then**: 400 Bad Request error is returned

**Expected Status Code**: 400 Bad Request

**Validation Points**:

- [ ] SQL injection attempt is prevented
- [ ] GORM parameterized queries protect database
- [ ] Invalid ID format is rejected at parsing
- [ ] No unauthorized data is returned
- [ ] Attack is logged for security monitoring

---

### [Security] - SEC-08: Authorization Header Injection

**Scenario ID**: SEC-08  
**Given**:

- Product with ID 101 exists
- Malicious Authorization header with injection payload

**When**: GET request with header `Authorization: Bearer token\r\nX-Admin: true`

**Then**: 401 Unauthorized error is returned

**Expected Status Code**: 401 Unauthorized

**Validation Points**:

- [ ] Header injection is prevented
- [ ] HTTP header parser handles newlines safely
- [ ] JWT validation rejects malformed headers
- [ ] No privilege escalation occurs
- [ ] Framework security prevents header manipulation

---

### [Security] - SEC-09: Cross-Seller Data Access Attempt via Admin Impersonation

**Scenario ID**: SEC-09  
**Given**:

- Product with ID 101 belongs to seller 5
- Valid seller 10 token with manipulated admin claim
- Token signature is invalid

**When**: GET request to `/api/products/101` with tampered token

**Then**: 401 Unauthorized error is returned

**Expected Status Code**: 401 Unauthorized

**Validation Points**:

- [ ] Token signature validation prevents tampering
- [ ] JWT secret key properly verifies tokens
- [ ] Claims cannot be modified without detection
- [ ] Role escalation is prevented
- [ ] Multi-tenant isolation remains secure

---

### [Security] - SEC-10: Rate Limiting on Product Retrieval

**Scenario ID**: SEC-10  
**Given**:

- Product with ID 101 exists
- 1000 rapid requests from same IP/user
- Valid authentication context

**When**: 1000 GET requests to `/api/products/101` in short time

**Then**: Rate limit is enforced (if implemented)

**Expected Status Code**: 429 Too Many Requests (if rate limiting enabled)

**Validation Points**:

- [ ] Rate limiting protects against abuse
- [ ] Excessive requests are throttled
- [ ] Error indicates rate limit exceeded
- [ ] Retry-After header is provided
- [ ] Legitimate users are not impacted

---

### [Business Logic] - BL-01: Product Price Range Calculation

**Scenario ID**: BL-01  
**Given**:

- Product with ID 101 has 5 variants
- Variant prices: $10.00, $15.50, $20.00, $25.99, $12.00
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Price range reflects correct min and max

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] priceRange.min is $10.00
- [ ] priceRange.max is $25.99
- [ ] Price calculation includes all variants
- [ ] Prices are formatted with 2 decimal places
- [ ] Currency handling is consistent

---

### [Business Logic] - BL-02: Product Allow Purchase Status

**Scenario ID**: BL-02  
**Given**:

- Product with ID 101 has 3 variants
- Variant 1: allowPurchase = true
- Variant 2: allowPurchase = false
- Variant 3: allowPurchase = false
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product-level allowPurchase is true

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Product allowPurchase is true (at least one variant available)
- [ ] Aggregation logic is correct
- [ ] Individual variant allowPurchase flags are accurate
- [ ] Out-of-stock variants show allowPurchase = false

---

### [Business Logic] - BL-03: Product with All Variants Unavailable

**Scenario ID**: BL-03  
**Given**:

- Product with ID 101 has 3 variants
- All variants have allowPurchase = false (out of stock)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product-level allowPurchase is false

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Product allowPurchase is false
- [ ] Product details are still returned
- [ ] Variants array shows all variants unavailable
- [ ] Customer can view product but cannot purchase
- [ ] Frontend can show "out of stock" message

---

### [Business Logic] - BL-04: Default Variant Identification

**Scenario ID**: BL-04  
**Given**:

- Product with ID 101 has 5 variants
- One variant is marked as default (isDefault = true)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Default variant is correctly identified

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Exactly one variant has isDefault = true
- [ ] Default variant is prominently displayed
- [ ] Frontend can show default variant first
- [ ] Default variant selection logic is correct

---

### [Business Logic] - BL-05: Popular Product Flag

**Scenario ID**: BL-05  
**Given**:

- Product with ID 101 has at least one variant with isPopular = true
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Popular variants are identified

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Variants with isPopular flag are marked
- [ ] Popular flag is used for recommendations
- [ ] Multiple variants can be popular
- [ ] Frontend can highlight popular choices

---

### [Business Logic] - BL-06: Category Inheritance and Attributes

**Scenario ID**: BL-06  
**Given**:

- Product with ID 101 in category "Smartphones"
- Category has specific attribute definitions
- Product has category-required attributes filled
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product attributes match category requirements

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Required category attributes are present
- [ ] Attribute values are valid for category
- [ ] Category-specific validations are enforced
- [ ] Product fits category schema

---

### [Performance] - PERF-01: Product with Large Number of Variants

**Scenario ID**: PERF-01  
**Given**:

- Product with ID 101 has 100 variants (10 options x 10 values)
- Each variant has images and option combinations
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product is returned within acceptable time

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Response time is under 500ms
- [ ] All 100 variants are returned
- [ ] Response JSON is properly formatted
- [ ] No timeout errors occur
- [ ] Database queries are optimized
- [ ] Memory usage is acceptable

---

### [Performance] - PERF-02: Product with Large Product Attributes

**Scenario ID**: PERF-02  
**Given**:

- Product with ID 101 has 50 product attributes
- Each attribute has name, value, key, unit, sortOrder
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Product with all attributes returned quickly

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] All 50 attributes are returned
- [ ] Response time is under 500ms
- [ ] Attributes are sorted correctly
- [ ] No pagination needed for attributes
- [ ] Query performance is optimal

---

### [Performance] - PERF-03: Concurrent Product Requests

**Scenario ID**: PERF-03  
**Given**:

- Product with ID 101 exists
- 100 concurrent requests from different users
- Valid authentication context for each

**When**: 100 simultaneous GET requests to `/api/products/101`

**Then**: All requests complete successfully

**Expected Status Code**: 200 OK for all requests

**Validation Points**:

- [ ] No requests fail due to concurrency
- [ ] Database connections are managed properly
- [ ] Response times remain consistent
- [ ] No deadlocks or race conditions
- [ ] Cache (if enabled) improves performance
- [ ] System handles load gracefully

---

### [Performance] - PERF-04: Product Retrieval with Database Connection Issues

**Scenario ID**: PERF-04  
**Given**:

- Product with ID 101 exists
- Database connection pool is exhausted
- Valid authentication context

**When**: GET request to `/api/products/101` during high load

**Then**: Request waits for connection or times out gracefully

**Expected Status Code**: 503 Service Unavailable or 500 Internal Server Error

**Validation Points**:

- [ ] Timeout is handled gracefully
- [ ] Error message indicates service unavailable
- [ ] No connection leaks occur
- [ ] System recovers after load decreases
- [ ] Connection pooling is properly configured

---

### [Integration] - INT-01: Product with Related Category Data

**Scenario ID**: INT-01  
**Given**:

- Product with ID 101 exists
- Category table has corresponding category record
- Category has parent category
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Category data is joined and returned correctly

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Category details are fetched via foreign key
- [ ] Parent category relationship is resolved
- [ ] JOIN queries are executed correctly
- [ ] Category data is not stale
- [ ] Relationship constraints are maintained

---

### [Integration] - INT-03: Product Variant Image URLs

**Scenario ID**: INT-03  
**Given**:

- Product with ID 101 has variants with image URLs
- Images are stored in external storage (S3, CDN)
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Variant image URLs are returned as-is

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Image URLs are complete and accessible
- [ ] URLs point to valid image locations
- [ ] No image processing is done in GET request
- [ ] Image URLs are returned as stored
- [ ] External storage integration works correctly

---

### [Integration] - INT-04: Product with Seller Validation Data

**Scenario ID**: INT-04  
**Given**:

- Product with ID 101 belongs to seller 5
- Seller 5 has active subscription and is validated
- Public API call with X-Seller-ID header
- Valid seller ID in header

**When**: GET request to `/api/products/101` with `X-Seller-ID: 5`

**Then**: Product is returned after seller validation

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Seller validation middleware executes
- [ ] Seller active status is checked
- [ ] Seller subscription is validated
- [ ] Product access is granted after validation
- [ ] Multi-tenant isolation is maintained

---

### [Integration] - INT-05: Product with Inactive Seller Account

**Scenario ID**: INT-05  
**Given**:

- Product with ID 101 belongs to seller 5
- Seller 5 account is inactive or suspended
- Public API call with X-Seller-ID header
- Valid seller ID in header

**When**: GET request to `/api/products/101` with `X-Seller-ID: 5`

**Then**: Request is rejected due to inactive seller

**Expected Status Code**: 403 Forbidden

**Validation Points**:

- [ ] Seller validation middleware rejects request
- [ ] Error message indicates seller account issue
- [ ] Product is not accessible via inactive seller
- [ ] Customer protection is enforced
- [ ] Seller must resolve account status

---

### [Edge Case] - EDGE-07: Variant Option Values with Special Characters

**Scenario ID**: EDGE-07  
**Given**:

- Product with ID 101 has variant options with special characters
- Option values: "Red & Blue", "Size: M/L", "Color #FF0000"
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Option values with special characters are returned correctly

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Special characters in option values are preserved
- [ ] JSON encoding handles special characters
- [ ] No escaping issues in response
- [ ] Option display names are readable
- [ ] URL encoding is not required in response

---

### [Edge Case] - EDGE-08: Product with Very Long SKU

**Scenario ID**: EDGE-08  
**Given**:

- Product with ID 101 has SKU at maximum length (50 characters)
- Variants have extended SKUs based on base SKU
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: SKUs are returned without truncation

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Base SKU is returned in full
- [ ] Variant SKUs are complete
- [ ] No truncation of SKU values
- [ ] SKU format is maintained

---

### [Edge Case] - EDGE-09: Product with Zero-Price Variant (Free Product)

**Scenario ID**: EDGE-09  
**Given**:

- Product with ID 101 has one variant with price = 0.00 (free sample)
- Other variants have positive prices
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Zero-price variant is included in results

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Variant with price 0.00 is returned
- [ ] Price range minimum is 0.00
- [ ] Free variant is purchasable (allowPurchase = true)
- [ ] Business logic allows free products
- [ ] No division-by-zero errors

---

### [Edge Case] - EDGE-10: Product Timestamps at Boundary Values

**Scenario ID**: EDGE-10  
**Given**:

- Product with ID 101 has createdAt and updatedAt timestamps
- Timestamps are at or near Unix epoch boundaries
- Valid authentication context

**When**: GET request to `/api/products/101`

**Then**: Timestamps are returned in RFC3339 format

**Expected Status Code**: 200 OK

**Validation Points**:

- [ ] Timestamps are formatted correctly
- [ ] RFC3339 format is consistent
- [ ] Timezone information is included
- [ ] No timestamp overflow errors
- [ ] Date parsing is accurate

---

## Summary Checklist

Before promoting to upper environments, verify:

### Functional Requirements

- [ ] Product details retrieval works for all user roles
- [ ] Multi-tenant isolation is enforced correctly
- [ ] All nested data (variants, options, attributes) is returned
- [ ] Category hierarchy is properly resolved
- [ ] Price range calculation is accurate
- [ ] Product availability is correctly aggregated from variants

### Authentication & Authorization

- [ ] Public access requires X-Seller-ID header
- [ ] JWT token authentication works correctly
- [ ] Expired/malformed tokens are rejected
- [ ] Customer can access all products
- [ ] Seller can only access own products
- [ ] Admin can access any product

### Input Validation

- [ ] Product ID validation catches invalid formats
- [ ] Negative and zero IDs are handled
- [ ] Special characters and SQL injection are prevented
- [ ] Large numbers are handled without overflow

### Error Handling

- [ ] Non-existent products return 404
- [ ] Authorization failures return appropriate errors
- [ ] Database errors are caught and logged
- [ ] Error messages don't leak sensitive information

### Security

- [ ] SQL injection is prevented
- [ ] XSS payloads are sanitized
- [ ] Multi-tenant data isolation is enforced
- [ ] Cross-seller access is blocked
- [ ] Token tampering is detected

### Performance

- [ ] Product retrieval completes in <500ms
- [ ] Large number of variants handled efficiently
- [ ] Concurrent requests are handled correctly
- [ ] Database queries are optimized
- [ ] Connection pooling works properly

### Data Integrity

- [ ] Foreign key relationships are maintained
- [ ] Unicode and special characters are preserved
- [ ] Timestamps are accurate
- [ ] Optional fields handle empty values

### Integration

- [ ] Category data is joined correctly
- [ ] Seller validation integration works
- [ ] Image URLs are accessible
- [ ] External storage integration functions

### Edge Cases

- [ ] Empty optional fields handled
- [ ] Maximum field lengths supported
- [ ] Zero-price variants allowed
- [ ] Special characters in data preserved
- [ ] Boundary values handled correctly

---

## Test Data Requirements

### Setup Needed:

1. **Test Products**: Create products with various configurations:

   - Product with single variant
   - Product with multiple variants (5-10)
   - Product with 100+ variants (performance testing)
   - Product with attributes and package options
   - Product with empty optional fields
   - Product with maximum field lengths
   - Product with Unicode/special characters
   - Product with zero-price variant

2. **Test Sellers**: Create multiple seller accounts:

   - Active seller with products
   - Inactive/suspended seller
   - Seller with no products

3. **Test Categories**: Create category hierarchy:

   - Root categories
   - Subcategories with parents
   - Categories with attribute definitions

4. **Test Users**: Create user accounts:

   - Customer role
   - Seller role (multiple sellers)
   - Admin role

5. **Test Tokens**: Generate JWT tokens:
   - Valid tokens for each role
   - Expired token
   - Malformed token
   - Token with tampered claims

### Database State:

- Ensure referential integrity
- Set up soft-delete scenarios
- Create data integrity violation scenario for testing

---

## Notes for QA Team

- **Priority**: This is a critical public-facing API. Test thoroughly.
- **Multi-Tenant Testing**: Pay special attention to seller isolation scenarios.
- **Performance Baseline**: Establish performance benchmarks for large datasets.
- **Security Focus**: Prioritize security test scenarios due to public access.
- **Data Validation**: Verify all field lengths and formats match validation rules.
- **Error Message Review**: Ensure error messages are helpful but don't leak sensitive data.
- **Postman Collection**: Use provided Postman collection for automated testing.
- **Monitor Logs**: Check application logs during testing for any warnings or errors.

---

## Automation Recommendations

### High Priority for Automation:

1. Happy path scenarios (HP-01 to HP-07)
2. Authentication/Authorization scenarios (SEC-01 to SEC-06)
3. Input validation scenarios (NEG-01 to NEG-06)
4. Multi-tenant isolation tests (SEC-01, INT-04)

### Manual Testing Recommended:

1. Performance scenarios with monitoring (PERF-01 to PERF-04)
2. Data integrity violation scenarios (INT-02)
3. Unicode and special character edge cases (EDGE-06, EDGE-09)

### Continuous Monitoring:

1. Response time metrics
2. Error rate tracking
3. Database query performance
4. Cache hit rates (if caching implemented)
