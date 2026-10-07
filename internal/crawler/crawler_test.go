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
