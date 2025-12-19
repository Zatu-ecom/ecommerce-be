package scheduler

import (
	"context"
	"fmt"

	"ecommerce-be/common/log"
)

func Dispatch(job ScheduledJob, ctx context.Context) error {
	handler, ok := Get(job.Command)
	if !ok {
		return fmt.Errorf("unknown command: %s", job.Command)
	}

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
