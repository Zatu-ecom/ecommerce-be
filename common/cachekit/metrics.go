package cachekit

import (
	"context"
	"time"

	"ecommerce-be/common/log"
)

// Result labels for Record. Keep the set closed so dashboards stay stable.
const (
	ResultHit     = "hit"
	ResultMiss    = "miss"
	ResultError   = "error"
	ResultDropped = "dropped"
)

// Recorder receives one event per cache operation. Implementations must be
// cheap and non-blocking; the log sink below is the Phase 1 default and the
// Prometheus-labeled sink later keeps the same call shape.
type Recorder interface {
	Record(ctx context.Context, module, op, store, result string, latency time.Duration)
}

// LogRecorder implements Recorder over structured logs: errors always,
// drops as warnings, hits/misses at debug (hit volume would spam info).
// Fields carry module/op/store labels so log queries substitute for metrics
// until the Prometheus sink lands.
func LogRecorder() Recorder { return logRecorder{} }

type logRecorder struct{}

// Record implements Recorder.
func (logRecorder) Record(ctx context.Context, module, op, store, result string, latency time.Duration) {
	entry := log.WithContext(ctx).WithField("module", module).
		WithField("op", op).
		WithField("store", store).
		WithField("latency_ms", latency.Milliseconds())
	switch result {
	case ResultError:
		entry.Error("cache op error")
	case ResultDropped:
		entry.Warn("cache write dropped")
	default:
		entry.Debug("cache op " + result)
	}
}

// NopRecorder discards events. Used in unit tests.
func NopRecorder() Recorder { return nopRecorder{} }

type nopRecorder struct{}

// Record implements Recorder by doing nothing.
func (nopRecorder) Record(context.Context, string, string, string, string, time.Duration) {
}
