package scheduler

import "encoding/json"

// Job is the base job structure with command and payload.
type Job struct {
	// Unique identifier for this scheduled job (used for cancellation)
	JobID string `json:"jobId"`

	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

// ScheduledJob extends Job with metadata for tracing, context propagation, and cancellation.
// The JobID is used to cancel a scheduled job before it executes.
type ScheduledJob struct {
	Job

	// Propagated metadata for context reconstruction in workers
	UserID        string `json:"userId"`
	SellerID      string `json:"sellerId"`
	CorrelationId string `json:"correlationId"`
}
