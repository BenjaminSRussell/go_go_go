package http

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.InitialBackoff != 1*time.Second {
		t.Errorf("Expected InitialBackoff=1s, got %v", config.InitialBackoff)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor=2.0, got %v", config.BackoffFactor)
	}
}

func TestRetryHandlerShouldRetry(t *testing.T) {
	rh := NewRetryHandler(DefaultRetryConfig())

	tests := []struct {
		statusCode int
		err        error
		shouldRetry bool
	}{
		{http.StatusOK, nil, false},
		{http.StatusNotFound, nil, false},
		{http.StatusTooManyRequests, nil, true},
		{http.StatusInternalServerError, nil, true},
		{http.StatusBadGateway, nil, true},
		{http.StatusServiceUnavailable, nil, true},
		{http.StatusGatewayTimeout, nil, true},
		{0, errors.New("network error"), true},
	}

	for _, tt := range tests {
		result := rh.ShouldRetry(tt.statusCode, tt.err)
		if result != tt.shouldRetry {
			t.Errorf("ShouldRetry(%d, %v): expected %v, got %v", tt.statusCode, tt.err, tt.shouldRetry, result)
		}
	}
}

func TestRetryHandlerGetBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
	rh := NewRetryHandler(config)

	backoff := rh.GetBackoff("example.com", 0)

	if backoff <= 0 {
		t.Errorf("Expected positive backoff, got %v", backoff)
	}
}

func TestRetryHandlerRecordSuccess(t *testing.T) {
	rh := NewRetryHandler(DefaultRetryConfig())

	if rh == nil {
		t.Error("Expected RetryHandler to be created")
	}
}

func TestRetryHandlerRecordFailure(t *testing.T) {
	rh := NewRetryHandler(DefaultRetryConfig())

	if rh == nil {
		t.Error("Expected RetryHandler to be created")
	}
}

func TestNewHeaderRotator(t *testing.T) {
	hr := NewHeaderRotator()

	if hr == nil {
		t.Error("Expected HeaderRotator to be created")
	}
}

func TestTLSFingerprinter(t *testing.T) {
	tf := NewTLSFingerprinter()

	if tf == nil {
		t.Error("Expected TLSFingerprinter to be created")
	}

	profile := tf.GetRandomProfile()

	if profile.Name == "" {
		t.Error("Expected profile name to be set")
	}
}
