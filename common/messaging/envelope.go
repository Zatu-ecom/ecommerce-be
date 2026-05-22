package messaging

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope is a shared wire contract for asynchronous messages.
type Envelope struct {
	MessageID     string          `json:"messageId"`
	CorrelationID string          `json:"correlationId,omitempty"`
	TraceID       string          `json:"traceId,omitempty"`
	TenantID      string          `json:"tenantId,omitempty"`
	ActorID       string          `json:"actorId,omitempty"`
	EventType     string          `json:"eventType"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	RetryCount    int             `json:"retryCount"`
	Payload       json.RawMessage `json:"payload"`
}

// NewEnvelope creates a standard envelope around a domain payload.
func NewEnvelope(eventType string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		MessageID:  uuid.NewString(),
		EventType:  eventType,
		Version:    1,
		OccurredAt: time.Now().UTC(),
		Payload:    raw,
	}, nil
}

// DecodePayload unmarshals payload into destination type.
func (e Envelope) DecodePayload(dst any) error {
	return json.Unmarshal(e.Payload, dst)
}
