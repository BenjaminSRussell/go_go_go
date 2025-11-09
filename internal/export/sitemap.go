package export

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
)

// SitemapConfig holds export configuration
type SitemapConfig struct {
	DataDir           string
	OutputFile        string
	IncludeLastmod    bool
	IncludeChangefreq bool
	DefaultPriority   float64
}

// URLSet represents the XML sitemap structure
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a single URL in the sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	Lastmod    string  `xml:"lastmod,omitempty"`
	Changefreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// ExportSitemap exports crawl results to XML sitemap
func ExportSitemap(config SitemapConfig) (int, error) {
	store, err := storage.New(config.DataDir)
	if err != nil {
		return 0, fmt.Errorf("failed to open storage: %w", err)
	}
	defer store.Close()

	results, err := store.LoadResults()
	if err != nil {
		return 0, fmt.Errorf("failed to load results: %w", err)
	}

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]URL, 0),
	}

	for _, result := range results {
		// Only include successfully crawled pages
		if result.StatusCode != 200 {
			continue
		}

		u := URL{
			Loc:      result.URL,
			Priority: config.DefaultPriority,
		}

		if config.IncludeLastmod {
			u.Lastmod = result.CrawledAt.Format(time.RFC3339)
		}

		if config.IncludeChangefreq {
			u.Changefreq = "weekly"
		}

		urlSet.URLs = append(urlSet.URLs, u)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML header
	xmlContent := []byte(xml.Header + string(output))

	// Write to file
	if err := os.WriteFile(config.OutputFile, xmlContent, 0644); err != nil {
		return 0, fmt.Errorf("failed to write sitemap: %w", err)
	}

	return len(urlSet.URLs), nil
}
