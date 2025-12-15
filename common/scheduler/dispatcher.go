package scheduler

import (
	"context"
	"fmt"

	"ecommerce-be/common/constants"
	"ecommerce-be/common/log"
)

// Context key type to avoid collisions
type contextKey string

const (
	UserIDKey        contextKey = contextKey(constants.USER_ID_KEY)
	SellerIDKey      contextKey = contextKey(constants.SELLER_ID_KEY)
	CorrelationIDKey contextKey = contextKey(constants.CORRELATION_ID_KEY)
)

func Dispatch(job ScheduledJob) error {
	handler, ok := Get(job.Command)
	if !ok {
		return fmt.Errorf("unknown command: %s", job.Command)
	}

	ctx := GetContextWithKeys(job)

	err := handler(ctx, job.Payload)
	if err != nil {
		log.ErrorWithContext(
			ctx,
			fmt.Sprintf("error executing command %s: %s", job.Command, err.Error()),
			err,
		)
	}

	return err
}

func GetContextWithKeys(job ScheduledJob) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, UserIDKey, job.UserID)
	ctx = context.WithValue(ctx, SellerIDKey, job.SellerID)
	ctx = context.WithValue(ctx, CorrelationIDKey, job.CorrelationId)
	return ctx
}
