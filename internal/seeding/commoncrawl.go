package seeding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DiscoverFromCommonCrawl queries Common Crawl index for known URLs
func DiscoverFromCommonCrawl(startURL string) ([]string, error) {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	domain := parsedURL.Host
	// Remove port if present
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Query Common Crawl Index API
	ccURL := fmt.Sprintf("https://index.commoncrawl.org/CC-MAIN-2024-10-index?url=%s&output=json&limit=1000", domain)

	resp, err := http.Get(ccURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query Common Crawl: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Common Crawl query returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Common Crawl response: %w", err)
	}

	// Parse JSONL (one JSON object per line)
	lines := strings.Split(string(body), "\n")
	urls := make([]string, 0)
	seen := make(map[string]bool)

	for _, line := range lines {
		if line == "" {
			continue
		}

		var result struct {
			URL string `json:"url"`
		}

		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		if result.URL != "" && !seen[result.URL] {
			urls = append(urls, result.URL)
			seen[result.URL] = true
		}
	}

	return urls, nil
}
