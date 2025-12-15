package scheduler

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Job is the base job structure with command and payload.
type Job struct {
	// Unique identifier for this scheduled job (used for cancellation)
	JobID uuid.UUID `json:"jobId"`

	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

// NewJob creates a new Job with auto-generated UUID.
// Always use this constructor to ensure JobID is set.
//
// Example:
//
//	job := scheduler.NewJob("expire_reservation", json.RawMessage(`{"reservationId": 123}`))
func NewJob(command string, payload json.RawMessage) Job {
	return Job{
		JobID:   uuid.New(),
		Command: command,
		Payload: payload,
	}
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
