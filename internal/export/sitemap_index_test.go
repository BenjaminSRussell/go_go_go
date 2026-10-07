package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeURLs(n int) []URL {
	urls := make([]URL, n)
	for i := range urls {
		urls[i] = URL{Loc: fmt.Sprintf("https://example.com/p/%d", i)}
	}
	return urls
}

func TestWriteSitemapsSingleFileUnderLimit(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sitemap.xml")
	if err := WriteSitemaps(makeURLs(3), SitemapConfig{OutputFile: out, MaxURLsPerFile: 5}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if !strings.Contains(string(b), "<urlset") {
		t.Fatalf("expected urlset, got %s", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "sitemap-1.xml")); err == nil {
		t.Fatal("did not expect child sitemap")
	}
}

func TestWriteSitemapsEmitsIndexOverLimit(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sitemap.xml")
	cfg := SitemapConfig{OutputFile: out, MaxURLsPerFile: 2, BaseURL: "https://example.com/"}
	if err := WriteSitemaps(makeURLs(5), cfg); err != nil {
		t.Fatal(err)
	}
	idx, _ := os.ReadFile(out)
	s := string(idx)
	if !strings.Contains(s, "<sitemapindex") {
		t.Fatalf("expected sitemapindex, got %s", s)
	}
	for i := 1; i <= 3; i++ {
		want := fmt.Sprintf("https://example.com/sitemap-%d.xml", i)
		if !strings.Contains(s, want) {
			t.Errorf("index missing %s", want)
		}
		child, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("sitemap-%d.xml", i)))
		if err != nil {
			t.Fatalf("child %d: %v", i, err)
		}
		if !strings.Contains(string(child), "<urlset") {
			t.Errorf("child %d not a urlset", i)
		}
	}
	last, _ := os.ReadFile(filepath.Join(dir, "sitemap-3.xml"))
	if strings.Count(string(last), "<url>") != 1 {
		t.Errorf("last child should hold 1 url")
	}
}
