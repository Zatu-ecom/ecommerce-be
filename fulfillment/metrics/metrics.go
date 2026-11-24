package metrics

import (
	"context"
	"time"

	"ecommerce-be/common/log"
)

// Webhook outcomes — closed set so dashboards stay stable.
const (
	WebhookApplied     = "applied"
	WebhookDuplicate   = "duplicate"
	WebhookIgnored     = "ignored"
	WebhookUnverified  = "unverified"
	WebhookUnknownAWB  = "unknown_awb"
	WebhookApplyFailed = "apply_failed"
	WebhookError       = "error"
)

// Sweep results — closed set.
const (
	SweepOK    = "ok"
	SweepError = "error"
)

// Recorder receives one event per webhook push and one summary per sweep.
// Implementations must be cheap and non-blocking; the log sink below is
// the default and the Prometheus-labeled sink later keeps the same call
// shape (mirrors cachekit.Recorder posture).
type Recorder interface {
	// RecordWebhook logs one courier push outcome. action is the normalized
	// action ("" when the push never normalized: unknown AWB/unverified).
	RecordWebhook(ctx context.Context, providerCode, outcome, action string, latency time.Duration)
	// RecordSweep logs one cron sweep summary (reconcile or NDR).
	RecordSweep(ctx context.Context, sweep, providerCode, result string, count int, latency time.Duration)
}

// LogRecorder implements Recorder over structured logs: outcomes at info
// (webhook volume is courier-driven and low), errors always. Fields carry
// provider/outcome/action labels so log queries substitute for metrics
// until the Prometheus sink lands.
func LogRecorder() Recorder { return logRecorder{} }

type logRecorder struct{}

// RecordWebhook implements Recorder.
func (logRecorder) RecordWebhook(
	ctx context.Context,
	providerCode, outcome, action string,
	latency time.Duration,
) {
	entry := log.WithContext(ctx).
		WithField("module", "fulfillment").
		WithField("op", "webhook").
		WithField("provider", providerCode).
		WithField("outcome", outcome).
		WithField("latency_ms", latency.Milliseconds())
	if action != "" {
		entry = entry.WithField("action", action)
	}
	if outcome == WebhookError || outcome == WebhookUnverified || outcome == WebhookApplyFailed {
		entry.Warn("fulfillment webhook outcome")
		return
	}
	entry.Info("fulfillment webhook outcome")
}

// RecordSweep implements Recorder.
func (logRecorder) RecordSweep(
	ctx context.Context,
	sweep, providerCode, result string,
	count int,
	latency time.Duration,
) {
	entry := log.WithContext(ctx).
		WithField("module", "fulfillment").
		WithField("op", sweep).
		WithField("provider", providerCode).
		WithField("result", result).
		WithField("count", count).
		WithField("latency_ms", latency.Milliseconds())
	if result == SweepError {
		entry.Warn("fulfillment sweep summary")
		return
	}
	entry.Info("fulfillment sweep summary")
}
