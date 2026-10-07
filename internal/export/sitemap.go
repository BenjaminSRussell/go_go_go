package export

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	// MaxURLsPerFile caps each sitemap file (protocol limit 50,000). When the
	// export exceeds it, OutputFile becomes a sitemap index pointing at
	// numbered child sitemaps (#11). Zero means MaxSitemapURLs.
	MaxURLsPerFile int
	// BaseURL, when set, prefixes child sitemap locations in the index
	// (e.g. "https://example.com/"). Otherwise child file names are used.
	BaseURL string
}

// MaxSitemapURLs is the sitemaps.org protocol limit per sitemap file.
const MaxSitemapURLs = 50000

// SitemapIndex is the <sitemapindex> document.
type SitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	XMLNS    string         `xml:"xmlns,attr"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

// SitemapEntry is one child sitemap reference in an index.
type SitemapEntry struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod,omitempty"`
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

	if err := WriteSitemaps(urlSet.URLs, config); err != nil {
		return 0, err
	}
	return len(urlSet.URLs), nil
}

// WriteSitemaps writes urls to config.OutputFile, splitting into numbered
// child sitemaps plus a sitemap index when len(urls) exceeds the per-file cap.
func WriteSitemaps(urls []URL, config SitemapConfig) error {
	limit := config.MaxURLsPerFile
	if limit <= 0 || limit > MaxSitemapURLs {
		limit = MaxSitemapURLs
	}
	if len(urls) <= limit {
		return writeURLSet(config.OutputFile, urls)
	}

	ext := filepath.Ext(config.OutputFile)
	stem := strings.TrimSuffix(config.OutputFile, ext)
	if ext == "" {
		ext = ".xml"
	}
	index := SitemapIndex{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, part := 0, 1; i < len(urls); i, part = i+limit, part+1 {
		end := i + limit
		if end > len(urls) {
			end = len(urls)
		}
		childPath := fmt.Sprintf("%s-%d%s", stem, part, ext)
		if err := writeURLSet(childPath, urls[i:end]); err != nil {
			return err
		}
		loc := filepath.Base(childPath)
		if config.BaseURL != "" {
			loc = strings.TrimSuffix(config.BaseURL, "/") + "/" + loc
		}
		index.Sitemaps = append(index.Sitemaps, SitemapEntry{Loc: loc, Lastmod: now})
	}
	return writeXML(config.OutputFile, index)
}

func writeURLSet(path string, urls []URL) error {
	return writeXML(path, URLSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
}

func writeXML(path string, v interface{}) error {
	output, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}
	if err := os.WriteFile(path, []byte(xml.Header+string(output)), 0644); err != nil {
		return fmt.Errorf("failed to write sitemap: %w", err)
	}
	return nil
}
