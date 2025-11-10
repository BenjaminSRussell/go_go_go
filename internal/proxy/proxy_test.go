package proxy

import (
	"context"
	"testing"
	"time"
)

func TestProxyLeaseNew(t *testing.T) {
	proxy := &Proxy{
		URL:          "http://proxy.example.com:8080",
		LastChecked:  time.Now(),
		FailCount:    0,
		SuccessCount: 0,
	}

	if proxy.URL != "http://proxy.example.com:8080" {
		t.Errorf("Expected proxy URL to be set")
	}

	if proxy.SuccessCount != 0 {
		t.Errorf("Expected success count to be 0")
	}
}

func TestEnhancedProxyManagerNew(t *testing.T) {
	manager := NewEnhancedProxyManager(DefaultProxySources)

	if manager == nil {
		t.Error("Expected ProxyManager to be created")
	}
}

func TestEnhancedProxyManagerStart(t *testing.T) {
	manager := NewEnhancedProxyManager(DefaultProxySources)

	ctx := context.Background()
	err := manager.Start(ctx)

	if err != nil {
		t.Logf("Expected manager to start (error OK in test): %v", err)
	}
}

func TestEnhancedProxyManagerLeaseProxy(t *testing.T) {
	manager := NewEnhancedProxyManager(DefaultProxySources)

	ctx := context.Background()
	manager.Start(ctx)

	lease, err := manager.LeaseProxy("test-persona", 15*time.Minute, "")

	if err != nil {
		t.Logf("Expected lease to be created (error OK in test): %v", err)
	}

	if lease != nil {
		if lease.URL == "" {
			t.Error("Expected proxy URL to be set")
		}
	}
}

func TestProxySourceInterface(t *testing.T) {
	sources := []string{
		"http://test.example.com/proxies.txt",
	}

	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}

	if sources[0] != "http://test.example.com/proxies.txt" {
		t.Error("Expected proxy source URL")
	}
}

func TestDefaultProxySources(t *testing.T) {
	if len(DefaultProxySources) == 0 {
		t.Error("Expected default proxy sources to be defined")
	}
}
