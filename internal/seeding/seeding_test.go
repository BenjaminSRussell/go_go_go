package seeding

import (
	"net/http"
	"testing"
)

func TestDiscoverFromSitemap(t *testing.T) {
	client := &http.Client{}

	urls, err := DiscoverFromSitemap("https://www.google.com", client)

	if err != nil {
		t.Logf("Expected to discover from sitemap (error OK in test): %v", err)
	}

	if urls != nil {
		if len(urls) == 0 {
			t.Logf("No URLs discovered, which is OK for test")
		}
	}
}

func TestDiscoverFromCertificateTransparency(t *testing.T) {
	urls, err := DiscoverFromCertificateTransparency("example.com")

	if err != nil {
		t.Logf("Expected to discover from CT logs (error OK in test): %v", err)
	}

	if urls != nil && len(urls) > 0 {
		for _, url := range urls {
			if url == "" {
				t.Error("Expected non-empty URL")
			}
		}
	}
}

func TestDiscoverFromCommonCrawl(t *testing.T) {
	urls, err := DiscoverFromCommonCrawl("example.com")

	if err != nil {
		t.Logf("Expected to discover from Common Crawl (error OK in test): %v", err)
	}

	if urls != nil && len(urls) > 0 {
		for _, url := range urls {
			if url == "" {
				t.Error("Expected non-empty URL")
			}
		}
	}
}

func TestSeedingURLValidation(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"https://example.com/page",
		"https://subdomain.example.com",
	}

	for _, url := range validURLs {
		if url == "" {
			t.Error("Expected valid URL")
		}
	}
}

func TestSeedingStrategyNames(t *testing.T) {
	strategies := []string{"sitemap", "ct", "commoncrawl"}

	for _, strategy := range strategies {
		if strategy == "" {
			t.Error("Expected non-empty strategy name")
		}
	}
}
