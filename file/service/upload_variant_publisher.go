package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/constants"
	"ecommerce-be/common/log"
	"ecommerce-be/common/messaging"
	fileError "ecommerce-be/file/error"
	fileMessaging "ecommerce-be/file/messaging"
)

// VariantPublisher publishes a file.image.process.requested command to RabbitMQ.
// It wraps the common/messaging.Publisher interface and handles envelope construction
// and correlation ID propagation (CA2).
type VariantPublisher interface {
	// Publish sends an ImageProcessRequested command to the ecom.commands exchange.
	// CA2: correlationID is embedded in the envelope's CorrelationID field so the
	// variant worker can propagate it, satisfying Constitution §VI.
	Publish(
		ctx context.Context,
		msg fileMessaging.ImageProcessRequested,
		correlationID string,
	) error
}

type variantPublisher struct {
	publisher messaging.Publisher
}

// NewVariantPublisher creates a VariantPublisher backed by the given messaging.Publisher.
// The caller is responsible for injecting the correct RabbitMQ publisher instance.
func NewVariantPublisher(p messaging.Publisher) VariantPublisher {
	return &variantPublisher{publisher: p}
}

// Publish constructs a common/messaging.Envelope wrapping msg, sets the CorrelationID
// (CA2), and publishes to exchange "ecom.commands" with routing key
// "file.image.process.requested".
//
// On publish failure:
//   - The error is wrapped with a loggable message (no provider credentials included).
//   - The caller (complete-upload service) logs it and inserts file_job{FAILED_TO_PUBLISH}
//     but still returns HTTP 200 to the client (FR-019).
func (v *variantPublisher) Publish(
	ctx context.Context,
	msg fileMessaging.ImageProcessRequested,
	correlationID string,
) error {
	env, err := messaging.NewEnvelope(constants.ROUTING_KEY_FILE_IMAGE_PROCESS_REQUESTED, msg)
	if err != nil {
		return fileError.ErrFileUploadInternal.WithMessagef(
			"variant publisher: marshal payload: %v", err,
		)
	}

	// CA2: propagate the HTTP request's correlation ID into the message envelope
	// (Constitution §VI — correlation IDs MUST be propagated to message queues).
	env.CorrelationID = correlationID

	if err := v.publisher.Publish(
		ctx,
		constants.DEFAULT_COMMANDS_EXCHANGE,
		constants.ROUTING_KEY_FILE_IMAGE_PROCESS_REQUESTED,
		env,
	); err != nil {
		// Do NOT include raw provider error details in the wrapped message (SC-006).
		log.ErrorWithContext(ctx, "variant publisher: publish failed", err)
		return fmt.Errorf(
			"variant publisher: publish to %s exchange failed; check logs for details",
			constants.DEFAULT_COMMANDS_EXCHANGE,
		)
	}

	return nil
}
