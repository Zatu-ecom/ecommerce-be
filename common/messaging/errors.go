package messaging

// RetryableError marks processing failures that should be retried.
type RetryableError struct {
	Err error
}

const retryableErrorMsg = "retryable error"

func (e RetryableError) Error() string {
	if e.Err == nil {
		return retryableErrorMsg
	}
	return e.Err.Error()
}

// IsRetryable returns true if error should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(RetryableError)
	return ok
}
