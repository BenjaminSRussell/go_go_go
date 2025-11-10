package types

import (
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid config",
			config: Config{
				StartURL: "https://example.com",
				Workers:  4,
				Timeout:  30 * time.Second,
				DataDir:  "/tmp/crawl",
			},
			valid: true,
		},
		{
			name: "empty start url",
			config: Config{
				Workers: 4,
				Timeout: 30 * time.Second,
				DataDir: "/tmp/crawl",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.config.StartURL != ""
			if valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, valid)
			}
		})
	}
}

func TestResults(t *testing.T) {
	r := Results{
		Discovered: 100,
		Processed:  50,
		Errors:     5,
	}

	if r.Discovered != 100 {
		t.Errorf("Expected Discovered=100, got %d", r.Discovered)
	}
	if r.Processed != 50 {
		t.Errorf("Expected Processed=50, got %d", r.Processed)
	}
	if r.Errors != 5 {
		t.Errorf("Expected Errors=5, got %d", r.Errors)
	}
}

func TestURLItem(t *testing.T) {
	item := URLItem{
		URL:       "https://example.com/page",
		Depth:     2,
		ParentURL: "https://example.com",
	}

	if item.URL != "https://example.com/page" {
		t.Errorf("Expected URL=https://example.com/page, got %s", item.URL)
	}
	if item.Depth != 2 {
		t.Errorf("Expected Depth=2, got %d", item.Depth)
	}
}

func TestPageResult(t *testing.T) {
	pr := PageResult{
		URL:           "https://example.com",
		StatusCode:    200,
		ContentLength: 1024,
		LinkCount:     5,
		Title:         "Example",
	}

	if pr.StatusCode != 200 {
		t.Errorf("Expected StatusCode=200, got %d", pr.StatusCode)
	}
	if pr.LinkCount != 5 {
		t.Errorf("Expected LinkCount=5, got %d", pr.LinkCount)
	}
}
