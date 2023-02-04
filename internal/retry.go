package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v4"
)

var (
	StandardRetryOnCodes = []int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusUnprocessableEntity}
)

func TryHTTPCall(ctx context.Context, numberOfTries uint64, operation func() (*http.Response, error), retryOnCodes ...int) error {
	if len(retryOnCodes) == 0 {
		retryOnCodes = StandardRetryOnCodes
	}
	count := 0
	doOp := func() error {
		resp, err := operation()
		if err == nil {
			return nil
		}
		if resp == nil {
			return backoff.Permanent(fmt.Errorf("response was nil: %w", err))
		}
		select {
		case <-ctx.Done():
			return backoff.Permanent(fmt.Errorf("context was cancelled: %w", err))
		default:
		}
		shouldRetry := false
		for _, c := range retryOnCodes {
			if c == resp.StatusCode {
				shouldRetry = true
				break
			}
		}
		if shouldRetry {
			count = count + 1
			return fmt.Errorf("retry %d due to HTTP %d: %w", count, resp.StatusCode, err)
		}
		return backoff.Permanent(fmt.Errorf("retry %d permanent: %w", count, err))
	}
	return backoff.Retry(doOp, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), numberOfTries))
}