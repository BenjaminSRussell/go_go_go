package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
}

// RetryHandler manages intelligent retries with exponential backoff
type RetryHandler struct {
	config RetryConfig

	// Per-host retry tracking
	hostRetries sync.Map // map[string]*hostRetryState
}

type hostRetryState struct {
	mu               sync.Mutex
	consecutiveFails int
	lastFailTime     time.Time
	backoffUntil     time.Time
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(config RetryConfig) *RetryHandler {
	return &RetryHandler{
		config: config,
	}
}

// ShouldRetry determines if a request should be retried
func (rh *RetryHandler) ShouldRetry(statusCode int, err error) bool {
	// Always retry on network errors
	if err != nil {
		return true
	}

	// Retry on specific status codes
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}

	return false
}

// GetBackoff calculates the backoff duration for a host
func (rh *RetryHandler) GetBackoff(host string, attempt int) time.Duration {
	state := rh.getOrCreateState(host)
	state.mu.Lock()
	defer state.mu.Unlock()

	// Check if we're in backoff period
	if time.Now().Before(state.backoffUntil) {
		return time.Until(state.backoffUntil)
	}

	// Calculate exponential backoff
	backoff := rh.config.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * rh.config.BackoffFactor)
		if backoff > rh.config.MaxBackoff {
			backoff = rh.config.MaxBackoff
			break
		}
	}

	// Add jitter (±20%)
	jitter := time.Duration(float64(backoff) * 0.2 * (2.0*float64(time.Now().UnixNano()%100)/100.0 - 1.0))
	backoff += jitter

	return backoff
}

// RecordFailure records a failed request for a host
func (rh *RetryHandler) RecordFailure(host string, statusCode int) {
	state := rh.getOrCreateState(host)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.consecutiveFails++
	state.lastFailTime = time.Now()

	// For rate limiting (429), apply aggressive backoff
	if statusCode == http.StatusTooManyRequests {
		backoff := rh.GetBackoff(host, state.consecutiveFails)
		state.backoffUntil = time.Now().Add(backoff * 2) // Double backoff for rate limits
	} else {
		backoff := rh.GetBackoff(host, state.consecutiveFails)
		state.backoffUntil = time.Now().Add(backoff)
	}
}

// RecordSuccess records a successful request for a host
func (rh *RetryHandler) RecordSuccess(host string) {
	state := rh.getOrCreateState(host)
	state.mu.Lock()
	defer state.mu.Unlock()

	// Reset failure counter on success
	state.consecutiveFails = 0
	state.backoffUntil = time.Time{}
}

// IsInBackoff checks if a host is currently in backoff
func (rh *RetryHandler) IsInBackoff(host string) (bool, time.Duration) {
	state := rh.getOrCreateState(host)
	state.mu.Lock()
	defer state.mu.Unlock()

	if time.Now().Before(state.backoffUntil) {
		return true, time.Until(state.backoffUntil)
	}

	return false, 0
}

func (rh *RetryHandler) getOrCreateState(host string) *hostRetryState {
	if val, ok := rh.hostRetries.Load(host); ok {
		return val.(*hostRetryState)
	}

	state := &hostRetryState{}
	actual, _ := rh.hostRetries.LoadOrStore(host, state)
	return actual.(*hostRetryState)
}

// GetStats returns retry statistics for a host
func (rh *RetryHandler) GetStats(host string) map[string]interface{} {
	state := rh.getOrCreateState(host)
	state.mu.Lock()
	defer state.mu.Unlock()

	return map[string]interface{}{
		"consecutive_fails": state.consecutiveFails,
		"last_fail_time":    state.lastFailTime,
		"backoff_until":     state.backoffUntil,
		"in_backoff":        time.Now().Before(state.backoffUntil),
	}
}

// RetryableError wraps an error with retry information
type RetryableError struct {
	Err        error
	StatusCode int
	Attempt    int
	MaxRetries int
}

func (e *RetryableError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("request failed (attempt %d/%d): %v", e.Attempt, e.MaxRetries, e.Err)
	}
	return fmt.Sprintf("request failed with status %d (attempt %d/%d)", e.StatusCode, e.Attempt, e.MaxRetries)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}
