package resilience

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestWithRetry_Success(t *testing.T) {
	attempts := 0
	config := RetryConfig{
		MaxAttempts:          3,
		InitialBackoff:       10 * time.Millisecond,
		MaxBackoff:           100 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{429, 503},
	}

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestWithRetry_AllFail(t *testing.T) {
	attempts := 0
	config := RetryConfig{
		MaxAttempts:          3,
		InitialBackoff:       10 * time.Millisecond,
		MaxBackoff:           100 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{429, 503},
	}

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return &HTTPError{StatusCode: 503, Message: "Service Unavailable"}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_NonRetryable(t *testing.T) {
	attempts := 0
	config := RetryConfig{
		MaxAttempts:          3,
		InitialBackoff:       10 * time.Millisecond,
		MaxBackoff:           100 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{429, 503},
	}

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return &HTTPError{StatusCode: 400, Message: "Bad Request"}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestWithRetry_SuccessAfterRetry(t *testing.T) {
	attempts := 0
	config := RetryConfig{
		MaxAttempts:          3,
		InitialBackoff:       10 * time.Millisecond,
		MaxBackoff:           100 * time.Millisecond,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{429, 503},
	}

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		if attempts < 3 {
			return &HTTPError{StatusCode: 503, Message: "Service Unavailable"}
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_CancelledContext(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:          5,
		InitialBackoff:       100 * time.Millisecond,
		MaxBackoff:           1 * time.Second,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{503},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WithRetry(ctx, config, func() error {
		return &HTTPError{StatusCode: 503, Message: "Service Unavailable"}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestIsRetryable_HTTPError(t *testing.T) {
	err := &HTTPError{StatusCode: 503, Message: "Service Unavailable"}
	if !isRetryable(err, []int{429, 503}) {
		t.Error("expected 503 to be retryable")
	}

	err2 := &HTTPError{StatusCode: 400, Message: "Bad Request"}
	if isRetryable(err2, []int{429, 503}) {
		t.Error("expected 400 to not be retryable")
	}
}

func TestIsRetryable_NetTimeout(t *testing.T) {
	err := &net.DNSError{Err: "timeout", Name: "example.com", IsTimeout: true}
	if !isRetryable(err, nil) {
		t.Error("expected DNS timeout to be retryable")
	}
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{StatusCode: 404, Message: "Not Found"}
	expected := "HTTP 404: Not Found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCircuitBreakerStates(t *testing.T) {
	if StateClosed.String() != "closed" {
		t.Errorf("expected closed, got %s", StateClosed.String())
	}
	if StateOpen.String() != "open" {
		t.Errorf("expected open, got %s", StateOpen.String())
	}
	if StateHalfOpen.String() != "half_open" {
		t.Errorf("expected half_open, got %s", StateHalfOpen.String())
	}
}

func TestBufferedRequest(t *testing.T) {
	body := "test body content"
	req, _ := fmt.Sscanf(body, "%s")
	_ = req

	buffered := BufferedRequest{
		Body: []byte(body),
	}

	if string(buffered.Body) != body {
		t.Errorf("expected %q, got %q", body, string(buffered.Body))
	}
}
