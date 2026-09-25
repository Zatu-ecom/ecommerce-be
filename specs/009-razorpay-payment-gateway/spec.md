# Feature Specification: Razorpay Payment Gateway Integration

**Feature Branch**: `009-razorpay-payment-gateway`  
**Created**: 2026-08-15  
**Status**: Draft  
**Input**: User description: "Razorpay Payment Gateway Integration"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Customer pays for an order online (Priority: P1)

A customer has placed a pending order and needs to complete payment. The customer chooses to pay
online, is handed a payment session, completes the transaction with the payment provider, and the
order is automatically confirmed once the money is captured.

**Why this priority**: This is the core value — without it there is no online payment. Every other
capability (refunds, seller dashboards, additional providers) builds on this flow.

**Independent Test**: A customer can place a pending order, initiate payment, receive a payment
session, complete the provider's checkout, and observe the order move to confirmed without any
manual action.

**Acceptance Scenarios**:

1. **Given** a customer has a pending order, **When** they initiate payment for that order, **Then** the system creates a payment record in a pending state and returns the information the customer needs to complete the checkout.
2. **Given** a payment session has been initiated, **When** the payment provider confirms the money was captured, **Then** the payment is marked completed and the associated order is automatically confirmed.
3. **Given** a payment session has been initiated, **When** the payment provider reports the payment failed, **Then** the payment is marked failed and the associated order is marked failed.
4. **Given** an order is not owned by the customer or is not in a payable state, **When** they attempt to initiate payment, **Then** the attempt is rejected with a clear error.

---

### User Story 2 - Seller activates and manages a payment provider (Priority: P2)

A seller wants to accept online payments through their own provider account. The seller views the
available payment providers, sees which are already configured, supplies their own credentials for a
chosen provider, and can later deactivate it.

**Why this priority**: Sellers must be able to connect their own provider account before they can
receive money. This is required for the P1 flow to work for any real seller.

**Independent Test**: A seller can list all available providers, see the fields each provider
requires, save credentials for one provider, see it reflected as "configured", and deactivate it.

**Acceptance Scenarios**:

1. **Given** a seller has not configured any payment provider, **When** they view the list of available providers, **Then** they see each provider with its supported countries, currencies, payment methods, and a "not configured" state.
2. **Given** a seller selects an available provider, **When** they submit the required credentials, **Then** the credentials are stored securely and the provider shows as "configured" and available for use.
3. **Given** a seller has a configured provider, **When** they deactivate it, **Then** the provider no longer accepts new payments for that seller, without deleting the seller's saved credentials.
4. **Given** a seller submits incomplete or invalid credentials, **When** they attempt to save, **Then** the system rejects the submission and reports which fields are missing or invalid.

---

### User Story 3 - Seller refunds a customer (Priority: P3)

A seller needs to return money to a customer for a captured payment, either in full or partially.

**Why this priority**: Refunds are a necessary operational capability but are not required to take
the first payment. They can be delivered after the capture flow is live.

**Independent Test**: A seller can request a full or partial refund for a completed payment and
observe the refund progress to completion.

**Acceptance Scenarios**:

1. **Given** a payment is completed, **When** the seller requests a full or partial refund within the paid amount, **Then** a refund is initiated and tracked through to completion.
2. **Given** a payment is not completed, **When** a refund is requested, **Then** the request is rejected.
3. **Given** the provider reports the refund processed, **When** the system receives that notification, **Then** the refund is marked completed and the payment reflects the refunded amount.
4. **Given** the provider reports the refund failed, **When** the system receives that notification, **Then** the refund is marked failed and the payment remains completed.

---

### Edge Cases

- What happens when the same provider notification is delivered more than once? The system must process it only once and ignore the duplicate.
- What happens when a notification arrives after the payment has already reached its final state? The system must not regress a completed or failed payment.
- What happens when a notification cannot be matched to a known payment? The system must record the notification for later investigation without crashing.
- What happens when a provider notification signature is missing or invalid? The system must reject it as untrusted and must not change any records.
- What happens when the customer's order currency or the seller's business country is not supported by the configured provider? Payment initiation must fail with a clear message.
- What happens when two customers initiate payment for the same order at the same time? Only one payment should be accepted; the other must be rejected.
- What happens when a seller has no active provider configured? Payment initiation must fail with a clear message rather than silently creating an orphaned record.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow a customer to initiate payment for one of their own pending orders, creating a payment record whose amount and currency are derived from the order and the seller's configured currency — never from client-supplied values.
- **FR-002**: System MUST support multiple payment providers through a single, provider-independent payment flow, so that a new provider can be added without changing the payment or notification processing logic.
- **FR-003**: System MUST store, for every payment, its current state, the amount and currency, the seller and customer, and a provider-neutral reference to the checkout session and the captured payment.
- **FR-004**: System MUST maintain an append-only history of every state change a payment goes through, including who or what triggered it and the corresponding provider request and response, for reconciliation and dispute handling.
- **FR-005**: System MUST securely store seller provider credentials so that secret values are not readable by anyone who can view the database.
- **FR-006**: System MUST let a seller view all available payment providers, including each provider's supported countries, currencies, payment methods, required configuration fields, and whether the seller has already configured it.
- **FR-007**: System MUST let a seller save, update, and deactivate credentials for a payment provider, validating required fields before saving.
- **FR-008**: System MUST verify the authenticity of every incoming payment notification before acting on it, and reject untrusted notifications.
- **FR-009**: System MUST process each unique provider notification exactly once, even when the provider retries delivery.
- **FR-010**: System MUST automatically mark an order confirmed when its payment is captured, and automatically mark an order failed when its payment fails, using the existing order rules for those transitions. A payment that is only authorized (not yet captured) MUST NOT confirm the order.
- **FR-011**: System MUST let a seller request a full or partial refund for a completed payment and track the refund through pending, processing, completed, or failed states.
- **FR-012**: System MUST update refund and payment state from provider notifications when a refund is created, processed, or fails.
- **FR-013**: System MUST ensure a seller can only see and manage their own payments, refunds, and provider configurations, with no cross-seller data exposure.
- **FR-014**: System MUST let a seller list their own payments and retrieve a single payment's current state, scoped to that seller.

### Key Entities *(include if feature involves data)*

- **Payment Provider**: A supported provider (e.g., Razorpay). Captures its name, the countries and currencies it supports, the payment methods it supports, a logo reference, and the configuration fields a seller must supply.
- **Provider Configuration**: A seller's credentials and settings for a specific provider, including whether it is active. Secret values are stored encrypted.
- **Payment**: A single payment attempt, tracking current state, amount, currency, customer, seller, and provider references to the checkout session and captured payment.
- **Payment Event**: An append-only record of each state change on a payment, with the trigger source and provider request/response.
- **Refund**: A full or partial return of money against a completed payment, tracking amount, reason, requester, and current state.
- **Notification Log**: A record of every incoming provider notification, used to guarantee exactly-once processing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A customer can complete an online payment and see their order confirmed automatically, without contacting support or waiting for manual intervention.
- **SC-002**: A seller can connect their own provider account and begin accepting payments without engineering assistance.
- **SC-003**: 100% of untrusted or invalid payment notifications are rejected without modifying any payment or order state.
- **SC-004**: A duplicate payment notification changes the resulting payment and order state no more than a single notification does.
- **SC-005**: A seller can request a refund for a completed payment and see the refund reach a final state without manual database changes.
- **SC-006**: A seller can view only their own payments, refunds, and provider configurations; no cross-seller data is ever returned.

## Assumptions

- The first supported payment provider is Razorpay, which supports India (INR). Additional providers are expected but out of scope for the initial delivery.
- The payment module is the single owner of payment records; the order module continues to own order state and exposes the operations needed to confirm or fail an order on payment outcome.
- Seller credentials are supplied by the seller from their own provider account (not a platform-managed account).
- Saved payment methods (cards, UPI, wallets) are out of scope for this feature and will be handled separately.
- Cash on delivery and recurring/subscription billing are out of scope for this feature, though the payment record model is designed to accommodate them later.
- Payment amounts are always expressed in the currency's minor unit (for INR, paise).
