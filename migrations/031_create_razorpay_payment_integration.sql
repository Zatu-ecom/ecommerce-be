-- ============================================================================
-- Migration: 031_create_razorpay_payment_integration.sql
-- Status: FOLDED into 009 (010-payment-gateway-platform)
--
-- The final payment schema now lives entirely in
-- migrations/009_create_payment_tables.sql, which creates payment_transaction
-- with reference_*, gateway_session_id, gateway_payment_id,
-- payment_method_details and environment, the payment_transaction_event
-- ledger, logo_file_id on payment_gateway, and the unconditional unique
-- (gateway_id, event_id) webhook index.
--
-- This file is intentionally a no-op: it is kept so migration numbering stays
-- stable for tooling that discovers migrations/*.sql in lexical order.
-- Payment is not in production — recreate dev/test databases after the 009
-- rewrite instead of applying incremental ALTERs here.
-- Created: 2026-08-15 / Slimmed: 2026-09-06
-- ============================================================================

SELECT 1;
