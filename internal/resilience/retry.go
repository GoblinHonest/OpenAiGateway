package resilience

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"time"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

type RetryConfig struct {
	MaxAttempts          int
	InitialBackoff       time.Duration
	MaxBackoff           time.Duration
	BackoffMultiplier    float64
	RetryableStatusCodes []int
}

func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryable(err, config.RetryableStatusCodes) {
			return err
		}

		if attempt == config.MaxAttempts-1 {
			break
		}

		backoff := config.InitialBackoff * time.Duration(math.Pow(config.BackoffMultiplier, float64(attempt)))
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error, retryableCodes []int) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		for _, code := range retryableCodes {
			if httpErr.StatusCode == code {
				return true
			}
		}
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	return false
}
