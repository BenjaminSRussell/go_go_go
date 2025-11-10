package crawler

import (
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestNewCrawler(t *testing.T) {
	config := types.Config{
		StartURL: "http://example.com",
		Workers:  1,
		Timeout:  10 * time.Second,
		DataDir:  t.TempDir(),
	}
	crawler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if crawler == nil {
		t.Fatal("New() returned nil")
	}
}
