# 📋 User Service - Product Requirements Document (PRD)

> **Version**: 2.0  
> **Last Updated**: January 3, 2026  
> **Status**: Draft  
> **Author**: Development Team

---

## 📑 Table of Contents

1. [Overview](#overview)
2. [Current State Analysis](#current-state-analysis)
3. [Feature Requirements](#feature-requirements)
   - [Auth APIs](#1-auth-apis)
   - [Seller Registration & Profile](#2-seller-registration--profile)
   - [Admin APIs](#3-admin-apis)
4. [Data Models](#data-models)
5. [API Specifications](#api-specifications)
6. [Business Rules](#business-rules)
7. [Implementation Priority](#implementation-priority)
8. [Database Migration](#database-migration)

---

## 🎯 Overview

### Purpose

This PRD defines the requirements for completing the User Service, including authentication flows, seller onboarding, and admin capabilities.

> **Note**: Plan and Subscription management has been moved to the **Subscription Service** ([SUBSCRIPTION_SERVICE_PRD.md](../subscription/SUBSCRIPTION_SERVICE_PRD.md)).

### Scope

| In Scope                                                        | Out of Scope                                          |
| --------------------------------------------------------------- | ----------------------------------------------------- |
| Authentication APIs (forgot/reset password, email verification) | Payment processing (handled by Payment Service)       |
| Seller registration with profile                                | Plan & Subscription management (Subscription Service) |
| Seller settings management                                      | Usage tracking & limits (Subscription Service)        |
| Admin user & seller management                                  | SSO/OAuth integration                                 |
| Email verification flow                                         | Two-factor authentication (Phase 2)                   |

### Goals

1. Complete authentication flow for production readiness
2. Enable seller self-registration with business profile
3. Provide admin tools for user/seller management
4. Implement secure password reset and email verification

### Service Dependencies

```
┌──────────────────┐     ┌──────────────────────────────┐
│   USER SERVICE   │     │    SUBSCRIPTION SERVICE      │
│                  │     │    (Separate Module)         │
│  • Auth          │     │                              │
│  • Profile       │────▶│  • Plans                     │
│  • Address       │     │  • Subscriptions             │
│  • Seller Setup  │     │  • Usage Tracking            │
│  • Admin Users   │     │  • Limit Enforcement         │
└──────────────────┘     └──────────────────────────────┘
```

---

## 📊 Current State Analysis

### ✅ Existing Features

| Feature                   | Status         | Notes                                             |
| ------------------------- | -------------- | ------------------------------------------------- |
| Customer Registration     | ✅ Done        | `POST /api/user/auth/register`                    |
| Login (all roles)         | ✅ Done        | `POST /api/user/auth/login`                       |
| Logout                    | ✅ Done        | `POST /api/user/auth/logout`                      |
| Token Refresh             | ✅ Done        | `POST /api/user/auth/refresh`                     |
| User Profile (Get/Update) | ✅ Done        | `GET/PUT /api/user/profile`                       |
| Change Password           | ✅ Done        | `PATCH /api/user/password`                        |
| Address CRUD              | ✅ Done        | `/api/user/addresses/*`                           |
| Country/Currency APIs     | ✅ Done        | `/api/user/countries/*`, `/api/user/currencies/*` |
| Seller Settings Routes    | 📝 Routes only | Handlers TODO                                     |

### ❌ Missing Features

| Feature                   | Priority  | Complexity | Service
| ------------------------- | --------- | ---------- | ---------------------- |
| Forgot/Reset Password     | 🔴 High   | Medium     | User Service           |
| Email Verification        | 🟡 Medium | Medium     | User Service           |
| Seller Registration       | 🔴 High   | High       | User Service           |
| Seller Profile Management | 🔴 High   | Medium     | User Service           |
| Plan APIs                 | 🔴 High   | Low        | → Subscription Service |
| Subscription APIs         | 🔴 High   | High       | → Subscription Service |
| Admin User Management     | 🟡 Medium | Medium     | User Service           |

### Existing Entities

```
✅ User (with RoleID, SellerID, CountryID, CurrencyID, Locale)
✅ Role (ADMIN, SELLER, CUSTOMER)
✅ SellerProfile (BusinessName, BusinessLogo, TaxID, IsVerified)
✅ SellerSettings (BusinessCountryID, BaseCurrencyID, etc.)
✅ Address
✅ Country, Currency, CountryCurrency
```

> **Note**: Plan and Subscription entities are managed by the Subscription Service.

---

## 📦 Feature Requirements

### 1. Auth APIs

#### 1.1 Forgot Password

**User Story**: As a user, I want to request a password reset link when I forget my password.

**Flow**:

```
1. User submits email
2. System validates email exists
3. System generates reset token (expires in 1 hour)
4. System sends email with reset link
5. Return success (don't reveal if email exists for security)
```

**Requirements**:

- [ ] Generate secure random token (32 bytes, base64 encoded)
- [ ] Store token hash in database with expiry
- [ ] Rate limit: Max 3 requests per email per hour
- [ ] Email template with reset link
- [ ] Token valid for 1 hour

#### 1.2 Reset Password

**User Story**: As a user, I want to set a new password using the reset link.

**Flow**:

```
1. User clicks reset link with token
2. System validates token (not expired, not used)
3. User submits new password
4. System updates password, invalidates token
5. Optionally: Invalidate all existing sessions
```

**Requirements**:

- [ ] Validate token exists and not expired
- [ ] Password strength validation (min 6 chars, as per existing)
- [ ] Mark token as used after successful reset
- [ ] Log password reset event for security audit

#### 1.3 Email Verification

**User Story**: As a new user, I want to verify my email address.

**Flow**:

```
1. After registration, system sends verification email
2. User clicks verification link
3. System marks email as verified
4. User can now access full features
```

**Requirements**:

- [ ] Add `IsEmailVerified` field to User entity
- [ ] Generate verification token on registration
- [ ] Token valid for 24 hours
- [ ] Resend verification endpoint (rate limited)
- [ ] Optional: Restrict certain actions until verified

#### 1.4 Resend Verification Email

**User Story**: As a user who hasn't received verification email, I want to request a new one.

**Requirements**:

- [ ] Rate limit: Max 3 requests per hour
- [ ] Invalidate previous tokens
- [ ] Only for unverified users

---

### 2. Seller Registration & Profile

#### 2.1 Seller Registration

**User Story**: As a business owner, I want to register as a seller with my business details.

**Flow**:

```
1. User submits registration with business details
2. System creates User with SELLER role
3. System creates SellerProfile
4. System creates default SellerSettings
5. Send verification email
6. Return auth token
```

> **Note**: Subscription/trial creation is handled by **Subscription Service** after seller registration.

**Request Model**:

```json
{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john@business.com",
  "password": "securepass123",
  "confirmPassword": "securepass123",
  "phone": "+1234567890",
  "businessName": "John's Store",
  "businessLogo": "https://...",
  "taxId": "TAX123456",
  "businessCountryId": 1,
  "baseCurrencyId": 1
}
```

**Requirements**:

- [ ] Validate all required fields
- [ ] Check email uniqueness
- [ ] Check taxId uniqueness (if provided)
- [ ] Create User, SellerProfile, SellerSettings in transaction
- [ ] Assign SELLER role (RoleID = 2)
- [ ] User.SellerID = User.ID (seller is their own seller)
- [ ] Send verification email

#### 2.2 Get Seller Profile

**User Story**: As a seller, I want to view my business profile.

**Response Model**:

```json
{
  "user": {
    "id": 1,
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@business.com",
    "phone": "+1234567890",
    "isActive": true,
    "isEmailVerified": true
  },
  "profile": {
    "businessName": "John's Store",
    "businessLogo": "https://...",
    "taxId": "TAX123456",
    "isVerified": false
  },
  "settings": {
    "businessCountry": { "id": 1, "name": "United States", "code": "US" },
    "baseCurrency": { "id": 1, "code": "USD", "symbol": "$" },
    "settlementCurrency": { "id": 1, "code": "USD", "symbol": "$" },
    "displayPricesInBuyerCurrency": false
  }
}
```

#### 2.3 Update Seller Profile

**User Story**: As a seller, I want to update my business details.

**Endpoint**: `PUT /api/user/seller/profile`

**Request Model**:

```json
{
  "businessName": "John's Updated Store",
  "businessLogo": "https://new-logo.png",
  "taxId": "NEWTAX123"
}
```

**Requirements**:

- [ ] Only update provided fields (use pointers)
- [ ] Validate taxId uniqueness if changed
- [ ] Cannot update if seller is verified (some fields locked)

#### 2.4 Seller Settings

**Endpoints**:

- `GET /api/user/seller/settings` - Get current settings
- `PUT /api/user/seller/settings` - Update settings

**Request Model**:

```json
{
  "businessCountryId": 1,
  "baseCurrencyId": 1,
  "settlementCurrencyId": 1,
  "displayPricesInBuyerCurrency": true
}
```

**Requirements**:

- [ ] Validate country and currency IDs exist
- [ ] Currency must be valid for selected country
- [ ] Log settings changes for audit

---

### 3. Admin APIs

#### 3.1 List Users (Enhanced)

**Endpoint**: `GET /api/user/admin/users`

**Query Parameters**:

```

?page=1
&limit=20
&role=customer|seller|admin
&status=active|inactive
&search=email or name
&sortBy=createdAt
&sortOrder=desc

```

**Requirements**:

- [ ] Admin-only access
- [ ] Filter by role, status
- [ ] Search by email, name
- [ ] Pagination with total count

#### 3.2 Get User Details

**Endpoint**: `GET /api/user/admin/users/:id`

**Response**:

```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "firstName": "John",
    "lastName": "Doe",
    "role": "customer",
    "isActive": true,
    "isEmailVerified": true,
    "createdAt": "2026-01-01T00:00:00Z",
    "lastLoginAt": "2026-01-02T10:00:00Z"
  },
  "addresses": [...],
  "activityLog": [...]
}
```

#### 3.3 Update User Status

**Endpoint**: `PUT /api/user/admin/users/:id/status`

**Request**:

```json
{
  "isActive": false,
  "reason": "Violated terms of service"
}
```

**Requirements**:

- [ ] Log status change with reason
- [ ] If deactivating: Invalidate all user sessions
- [ ] Send notification email to user
- [ ] Cannot deactivate own account

#### 3.4 List Sellers

**Endpoint**: `GET /api/user/admin/sellers`

**Query Parameters**:

```
?page=1
&limit=20
&verified=true|false
&search=business name or email
```

**Response includes**:

- User info
- Seller profile
- Verification status

> **Note**: Subscription status filtering is available via Subscription Service API.

#### 3.5 Get Seller Details

**Endpoint**: `GET /api/user/admin/sellers/:id`

**Response**:

```json
{
  "user": {
    "id": 2,
    "email": "seller@example.com",
    "firstName": "Jane",
    "lastName": "Merchant"
  },
  "profile": {
    "businessName": "Jane's Shop",
    "businessLogo": "https://...",
    "taxId": "TAX123",
    "isVerified": true,
    "verifiedAt": "2026-01-01T00:00:00Z",
    "verifiedBy": 1
  },
  "settings": {
    "businessCountry": { "id": 1, "name": "United States" },
    "baseCurrency": { "id": 1, "code": "USD" }
  }
}
```

#### 3.6 Verify Seller

**Endpoint**: `PUT /api/user/admin/sellers/:id/verify`

**Request**:

```json
{
  "isVerified": true,
  "notes": "Documents verified on 2026-01-01"
}
```

**Requirements**:

- [ ] Update SellerProfile.IsVerified
- [ ] Log verification with admin ID and notes
- [ ] Send notification to seller

---

## 📐 Data Models

### Summary of Entities

| Entity                   | Status     | Purpose                     |
| ------------------------ | ---------- | --------------------------- |
| `User`                   | 🔄 Updated | Added `IsEmailVerified`     |
| `SellerProfile`          | ✅ Exists  | Seller business information |
| `SellerSettings`         | ✅ Exists  | Seller preferences          |
| `PasswordResetToken`     | 🆕 New     | Password reset flow         |
| `EmailVerificationToken` | 🆕 New     | Email verification flow     |

> **Note**: Plan, Subscription, and Usage entities are managed by the **Subscription Service**.

### Entity Definitions

#### PasswordResetToken

```go
type PasswordResetToken struct {
    db.BaseEntity
    UserID    uint      `json:"userId"    gorm:"index;not null"`
    TokenHash string    `json:"-"         gorm:"size:64;not null;uniqueIndex"`
    ExpiresAt time.Time `json:"expiresAt" gorm:"not null"`
    UsedAt    *time.Time `json:"usedAt"`
}
```

#### EmailVerificationToken

```go
type EmailVerificationToken struct {
    db.BaseEntity
    UserID    uint       `json:"userId"    gorm:"index;not null"`
    TokenHash string     `json:"-"         gorm:"size:64;not null;uniqueIndex"`
    ExpiresAt time.Time  `json:"expiresAt" gorm:"not null"`
    UsedAt    *time.Time `json:"usedAt"`
}
```

#### User (Updated)

```go
type User struct {
    // ... existing fields ...
    IsEmailVerified bool `json:"isEmailVerified" gorm:"default:false"`
}
```

---

## 🛣️ API Specifications

### Route Summary

```
/api/user/
│
├── auth/
│   ├── POST   /register              ✅ Exists (Customer)
│   ├── POST   /login                 ✅ Exists
│   ├── POST   /logout                ✅ Exists
│   ├── POST   /refresh               ✅ Exists
│   ├── POST   /forgot-password       🆕 New
│   ├── POST   /reset-password        🆕 New
│   ├── POST   /verify-email          🆕 New
│   └── POST   /resend-verification   🆕 New
│
├── profile                           ✅ Exists
├── addresses/*                       ✅ Exists
├── countries/*                       ✅ Exists
├── currencies/*                      ✅ Exists
│
├── seller/
│   ├── POST   /register              🆕 New (creates user + profile + settings)
│   ├── GET    /profile               🆕 New (full seller profile)
│   ├── PUT    /profile               🆕 New
│   ├── GET    /settings              📝 Handler TODO
│   └── PUT    /settings              📝 Handler TODO
│
└── admin/
    ├── GET    /users                 ✅ Partial → Enhance
    ├── GET    /users/:id             🆕 New
    ├── PUT    /users/:id/status      🆕 New
    ├── GET    /sellers               🆕 New
    ├── GET    /sellers/:id           🆕 New
    └── PUT    /sellers/:id/verify    🆕 New
```

> **Note**: Plan and Subscription admin APIs are in the **Subscription Service**.

---

## 📏 Business Rules

### Authentication

| Rule     | Description                                      |
| -------- | ------------------------------------------------ |
| AUTH-001 | Password reset tokens expire in 1 hour           |
| AUTH-002 | Email verification tokens expire in 24 hours     |
| AUTH-003 | Max 3 password reset requests per email per hour |
| AUTH-004 | Max 3 verification email resends per hour        |
| AUTH-005 | Password reset invalidates all existing sessions |

### Seller Registration

| Rule       | Description                                           |
| ---------- | ----------------------------------------------------- |
| SELLER-001 | Email must be unique across all users                 |
| SELLER-002 | TaxID must be unique if provided                      |
| SELLER-003 | New seller gets SellerID = UserID                     |
| SELLER-004 | Seller must complete settings before listing products |

### Admin

| Rule      | Description                                  |
| --------- | -------------------------------------------- |
| ADMIN-001 | Cannot deactivate own account                |
| ADMIN-002 | All admin actions must be logged with reason |
| ADMIN-003 | Seller verification requires notes           |

---

## 🎯 Implementation Priority

### Phase 1 - Core Auth & Seller (Week 1-2)

| Priority | Feature                     | Effort |
| -------- | --------------------------- | ------ |
| 🔴 P0    | Seller Registration         | 3 days |
| 🔴 P0    | Seller Profile (Get/Update) | 2 days |
| 🔴 P0    | Seller Settings Handlers    | 2 days |
| 🔴 P0    | Forgot/Reset Password       | 2 days |

### Phase 2 - Admin & Verification (Week 3-4)

| Priority | Feature                 | Effort |
| -------- | ----------------------- | ------ |
| 🟡 P1    | Email Verification      | 2 days |
| 🟡 P1    | Admin User Management   | 2 days |
| 🟡 P1    | Admin Seller Management | 2 days |
| 🟡 P1    | Admin Seller Verify     | 1 day  |

### Phase 3 - Enhancements (Week 5+)

| Priority | Feature                          | Effort |
| -------- | -------------------------------- | ------ |
| 🟢 P2    | Notification Service Integration | 2 days |
| 🟢 P2    | Activity Logging                 | 2 days |
| ⚪ P3    | Two-Factor Authentication        | 3 days |

---

## 📊 Database Migration

### Migration 009: User Service Enhancements

```sql
-- migrations/009_user_service_enhancements.sql
-- Description: User service auth and seller enhancements
-- Author: Development Team
-- Date: 2026-01-03

-- =============================================
-- 1. USER ENHANCEMENTS
-- =============================================

-- Add email verification to user
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN DEFAULT FALSE;

-- =============================================
-- 2. PASSWORD RESET TOKENS
-- =============================================

CREATE TABLE IF NOT EXISTS password_reset_token (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_password_reset_token UNIQUE (token_hash)
);
CREATE INDEX IF NOT EXISTS idx_password_reset_token_user ON password_reset_token(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_token_expires ON password_reset_token(expires_at);

-- =============================================
-- 3. EMAIL VERIFICATION TOKENS
-- =============================================

CREATE TABLE IF NOT EXISTS email_verification_token (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_email_verification_token UNIQUE (token_hash)
);
CREATE INDEX IF NOT EXISTS idx_email_verification_token_user ON email_verification_token(user_id);
```

---

## ✅ Success Metrics

| Metric                              | Target      |
| ----------------------------------- | ----------- |
| Seller registration completion rate | > 80%       |
| Password reset success rate         | > 95%       |
| Email verification rate             | > 70%       |
| Average time to seller onboarding   | < 5 minutes |

---

## 📚 References

- [Architecture Documentation](../ARCHITECTURE.md)
- [Coding Standards](../CODING_STANDARDS.md)
- [Subscription Service PRD](../subscription/SUBSCRIPTION_SERVICE_PRD.md)

---

**Document Status**: Ready for Review  
**Version**: 2.0 (Refactored - Plan/Subscription moved to Subscription Service)  
**Next Steps**:

1. Review with team
2. Create implementation tasks
3. Begin development
