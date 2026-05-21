package download

type RetryPolicy struct {
	MaxAttempts int
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 3}
}

func (p RetryPolicy) CanRetry(attempt int, retryable bool) bool {
	if p.MaxAttempts <= 0 {
		return false
	}
	return retryable && attempt < p.MaxAttempts
}
