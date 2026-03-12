# GitHub Copilot Instructions for Generating Test Scenarios

## Purpose

When asked to generate test scenarios for any feature, API endpoint, or functionality in this e-commerce backend, you should create a comprehensive test scenario document that lists all test cases that QA engineers or developers need to verify before promoting code to upper environments (staging, production).

**You are NOT writing test code** - you are writing test scenario documentation/checklists.

## QA Mindset Guidelines

Think like a QA Engineer creating a test plan. Ask:

- "What could go wrong?"
- "How might users misuse this?"
- "What edge cases exist?"
- "What security vulnerabilities could be exploited?"
- "What happens under load or concurrent access?"

## Test Coverage Categories

Every feature should have scenarios covering:

### 1. **Happy Path Scenarios** (Successful Flows)

- Valid inputs with expected successful outcomes
- Typical user workflows
- Standard business operations

### 2. **Negative Scenarios** (Error Handling)

- Invalid inputs (wrong data types, formats)
- Missing required fields
- Out-of-range values
- Malformed requests

### 3. **Edge Case Scenarios** (Boundary Conditions)

- Empty values (empty strings, null, empty arrays)
- Maximum/minimum values
- Very long strings
- Special characters (Unicode, emojis, symbols)
- Zero quantities, prices

### 4. **Security Scenarios** (Protection Testing)

- Authentication failures (missing/invalid/expired tokens)
- Authorization violations (wrong role accessing protected resources)
- SQL injection attempts
- XSS payload attempts
- CSRF attacks
- Data access violations (user A accessing user B's data)

### 5. **Business Logic Scenarios** (Domain Rules)

- Business rule validations
- State transition rules
- Inventory constraints
- Pricing rules
- Order workflow constraints

### 6. **Other Scenarios** (Cross-Service)

- Database transaction handling
- External service failures
- Race conditions
- Concurrent access

## Universal Scenario Checklist

For **every** feature/endpoint being tested, ensure you cover these areas:

### Authentication & Authorization

#### Authentication & Authorization Tests

```
✓ Valid credentials should authenticate successfully
✓ Invalid credentials should return 401 Unauthorized
✓ Missing token should return 401 Unauthorized
✓ Expired token should return 401 Unauthorized
✓ Customer role accessing seller endpoint should return 403 Forbidden
✓ Seller should only access their own resources
✓ Admin should have full access
```

#### Input Validation Tests

```
✓ Valid input should succeed
✓ Missing required fields should return 400 Bad Request
✓ Invalid data types should return 400 Bad Request
✓ Values exceeding max length should return 400 Bad Request
✓ Negative numbers for positive-only fields should return 400 Bad Request
✓ SQL injection attempts should be sanitized
✓ XSS payloads should be escaped
```

#### Database & State Tests

```
✓ Concurrent updates should handle race conditions
✓ Transaction rollback on failure
✓ Soft deletes should preserve referential integrity
✓ Timestamps should be updated correctly
✓ Cascading deletes should work as expected
```

## API Endpoint Verification Checklist

For each endpoint, verify:

- [ ] **HTTP Methods**: Correct method (GET, POST, PUT, DELETE)
- [ ] **Status Codes**: Appropriate codes (200, 201, 400, 401, 403, 404, 500)
- [ ] **Request Validation**: All input validation rules
- [ ] **Response Format**: Correct JSON structure, required fields
- [ ] **Error Messages**: Clear, non-leaking error messages
- [ ] **Authentication**: Token required where necessary
- [ ] **Authorization**: Role permissions enforced
- [ ] **Pagination**: Correct page, limit, total count (if applicable)
- [ ] **Filtering**: All filter parameters work correctly (if applicable)
- [ ] **Sorting**: All sort options work correctly (if applicable)
- [ ] **Rate Limiting**: If applicable
- [ ] **Idempotency**: For create/update operations

## Example Scenario Document Format

When generating test scenarios, use this structure:

```markdown
# Test Scenarios: [Feature/Endpoint Name]

## Feature Information

- **Endpoint**: [API endpoint if applicable]
- **Method**: [HTTP method]
- **Authentication**: [Required/Optional/Public]
- **User Roles**: [Roles that can access]

---

## Test Scenarios

### [Happy Path] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Initial state and preconditions]
**When**: [Action being performed]
**Then**: [Expected outcome]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Validation point 1
- [ ] Validation point 2
- [ ] Validation point 3

---

### [Negative] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Initial state]
**When**: [Invalid action]
**Then**: [Expected error response]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Appropriate error message
- [ ] No sensitive data exposed
- [ ] Proper error code returned

---

### [Edge Case] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Edge case condition]
**When**: [Action]
**Then**: [Expected handling]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Edge case handled gracefully
- [ ] No system errors
- [ ] Response is valid

---

### [Security] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Security test setup]
**When**: [Security attack attempt]
**Then**: [Expected protection]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Attack is prevented
- [ ] No data leakage
- [ ] Security logged

---

### [Performance] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Load condition]
**When**: [Action under load]
**Then**: [Performance requirement met]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Response time within limits
- [ ] No resource exhaustion
- [ ] Scalable solution

---

### [Integration] - Scenario Name

**Scenario ID**: [ID]
**Given**: [Integration setup]
**When**: [Cross-service action]
**Then**: [Expected integration behavior]
**Expected Status Code**: [Code]

**Validation Points**:

- [ ] Services communicate correctly
- [ ] Data consistency maintained
- [ ] Error handling works

---

## Summary Checklist

Before promoting to upper environments, verify:

- [ ] All happy path scenarios pass
- [ ] All negative scenarios return appropriate errors
- [ ] Edge cases are handled gracefully
- [ ] Security vulnerabilities are addressed
- [ ] Performance meets requirements
- [ ] Integration with dependent services works
- [ ] API documentation is updated
- [ ] Monitoring/logging is in place
```

---

## Edge Cases Reference

Always include these edge cases in scenarios:

### Data Validation Edge Cases

- **Empty Values**: `""`, `null`, `undefined`, `[]`, `{}`
- **Whitespace**: `"   "`, `"\n\t"`, `" space before/after "`
- **Special Characters**: `<script>`, `'; DROP TABLE--`, `../../etc/passwd`
- **Unicode**: Emojis 😀, Arabic العربية, Chinese 中文, RTL text
- **Large Numbers**: `999999999999`, `-999999999`, `0.000000001`
- **Invalid Types**: String instead of number, array instead of object
- **Max Lengths**: Strings exceeding field limits (e.g., 256+ chars)

### Concurrency Edge Cases

- Two users updating same product simultaneously
- Stock level reaches zero during checkout
- Duplicate form submission
- Race condition in order creation

### State Transition Edge Cases

- Invalid state changes (e.g., "delivered" → "pending")
- State changes without required permissions
- Repeated state transitions

---

## Remember: A Good QA Engineer...

✓ Tests what users **will** do, not just what they **should** do
✓ Assumes nothing works until proven otherwise
✓ Documents assumptions and prerequisites clearly
✓ Thinks about security from day one
✓ Considers performance implications early
✓ Writes scenarios that developers and QA can execute
✓ Provides clear expected results and validation points
✓ Includes reproduction steps for failures
✓ Collaborates with developers to prevent bugs
✓ Continuously updates scenarios based on production issues
