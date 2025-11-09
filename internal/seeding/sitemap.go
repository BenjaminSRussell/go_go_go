package seeding

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/BenjaminSRussell/go_go_go/internal/parser"
)

// DiscoverFromSitemap discovers URLs from sitemap.xml
func DiscoverFromSitemap(startURL string, client *http.Client) ([]string, error) {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	sitemapURLs := []string{
		fmt.Sprintf("%s://%s/sitemap.xml", parsedURL.Scheme, parsedURL.Host),
		fmt.Sprintf("%s://%s/sitemap_index.xml", parsedURL.Scheme, parsedURL.Host),
		fmt.Sprintf("%s://%s/sitemap-index.xml", parsedURL.Scheme, parsedURL.Host),
	}

	allURLs := make([]string, 0)
	visited := make(map[string]bool)

	for _, sitemapURL := range sitemapURLs {
		urls, err := fetchSitemap(sitemapURL, client, visited)
		if err != nil {
			continue
		}
		allURLs = append(allURLs, urls...)
	}

	// Also check robots.txt for sitemap references
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
	resp, err := client.Get(robotsURL)
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		robotsContent := string(body)

		// Parse sitemap directives
		lines := strings.Split(robotsContent, "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
				sitemapURL := strings.TrimSpace(line[8:])
				urls, _ := fetchSitemap(sitemapURL, client, visited)
				allURLs = append(allURLs, urls...)
			}
		}
	}

	return allURLs, nil
}

func fetchSitemap(sitemapURL string, client *http.Client, visited map[string]bool) ([]string, error) {
	if visited[sitemapURL] {
		return nil, nil
	}
	visited[sitemapURL] = true

	resp, err := client.Get(sitemapURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	urls := parser.ExtractSitemapURLs(content)

	// Check if this is a sitemap index
	allURLs := make([]string, 0)
	for _, u := range urls {
		if strings.HasSuffix(u, ".xml") || strings.Contains(u, "sitemap") {
			// Recursively fetch nested sitemaps
			nestedURLs, _ := fetchSitemap(u, client, visited)
			allURLs = append(allURLs, nestedURLs...)
		} else {
			allURLs = append(allURLs, u)
		}
	}

	return allURLs, nil
}
