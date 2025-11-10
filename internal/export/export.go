package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

type Exporter struct {
	outputDir string
}

func NewExporter(outputDir string) (*Exporter, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Exporter{
		outputDir: outputDir,
	}, nil
}

func (e *Exporter) ExportJSON(results []types.PageResult, outputFile string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

func (e *Exporter) ExportCSV(results []types.PageResult, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"URL", "StatusCode", "ContentLength", "LinkCount", "CrawledAt"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, result := range results {
		record := []string{
			result.URL,
			fmt.Sprintf("%d", result.StatusCode),
			fmt.Sprintf("%d", result.ContentLength),
			fmt.Sprintf("%d", result.LinkCount),
			result.CrawledAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

func (e *Exporter) ExportSitemap(results []types.PageResult, outputFile string) error {
	config := SitemapConfig{
		OutputFile:        outputFile,
		DefaultPriority:   0.8,
		IncludeLastmod:    true,
		IncludeChangefreq: true,
	}

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]URL, 0),
	}

	for _, result := range results {
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

	output, err := marshalXML(urlSet)
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	xmlContent := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + output)

	if err := os.WriteFile(outputFile, xmlContent, 0644); err != nil {
		return fmt.Errorf("failed to write sitemap: %w", err)
	}

	return nil
}

func marshalXML(urlSet URLSet) (string, error) {
	var result string
	result += "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n"
	for _, url := range urlSet.URLs {
		result += "  <url>\n"
		result += fmt.Sprintf("    <loc>%s</loc>\n", url.Loc)
		if url.Lastmod != "" {
			result += fmt.Sprintf("    <lastmod>%s</lastmod>\n", url.Lastmod)
		}
		if url.Changefreq != "" {
			result += fmt.Sprintf("    <changefreq>%s</changefreq>\n", url.Changefreq)
		}
		if url.Priority > 0 {
			result += fmt.Sprintf("    <priority>%.1f</priority>\n", url.Priority)
		}
		result += "  </url>\n"
	}
	result += "</urlset>"
	return result, nil
}
