package crawler

import (
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
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


func TestShouldCrawlHostScope(t *testing.T) {
	c := &Crawler{config: types.Config{CrawlExternalLinks: false}}

	cases := []struct {
		link string
		base string
		want bool
	}{
		{"https://example.com/a", "https://example.com/", true},
		{"https://www.example.com/a", "https://example.com/", true},
		{"https://notexample.com/a", "https://example.com/", false},
		{"https://evil-example.com/a", "https://example.com/", false},
		{"https://example.com.evil/a", "https://example.com/", false},
		{"https://other.com/a", "https://example.com/", false},
	}
	for _, tc := range cases {
		got := c.shouldCrawl(tc.link, tc.base)
		if got != tc.want {
			t.Errorf("shouldCrawl(%q, %q)=%v want %v", tc.link, tc.base, got, tc.want)
		}
	}
}

func TestFrontierExhaustedRequiresZeroInFlight(t *testing.T) {
	// Documents the Crawl() exit condition: empty frontier alone is not enough
	// while workers may still discover links.
	c := &Crawler{}
	c.frontier = NewFrontier()
	if !(c.frontier.IsEmpty() && c.inFlight.Load() == 0) {
		t.Fatal("expected exhausted with zero in-flight")
	}
	c.inFlight.Store(1)
	if c.frontier.IsEmpty() && c.inFlight.Load() == 0 {
		t.Fatal("must not treat as exhausted while work is in flight")
	}
	c.inFlight.Store(0)
	if !(c.frontier.IsEmpty() && c.inFlight.Load() == 0) {
		t.Fatal("expected exhausted after in-flight drained")
	}
}

func TestResumeEmptyFrontierReportsZeroSize(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.Config{StartURL: "https://example.com", Workers: 1, Timeout: time.Second, DataDir: dir}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	store.Close()

	c, err := Resume(dir)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if sz := c.FrontierSize(); sz != 0 {
		t.Fatalf("expected empty frontier, got %d", sz)
	}
}
