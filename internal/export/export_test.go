package export

import (
	"os"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestExporterNew(t *testing.T) {
	tmpDir := t.TempDir()

	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	if exporter == nil {
		t.Error("Expected exporter to be created")
	}
}

func TestExporterExportJSON(t *testing.T) {
	tmpDir := t.TempDir()

	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	results := []types.PageResult{
		{
			URL:           "https://example.com/page1",
			StatusCode:    200,
			ContentLength: 1024,
			LinkCount:     5,
			CrawledAt:     time.Now(),
		},
		{
			URL:           "https://example.com/page2",
			StatusCode:    200,
			ContentLength: 2048,
			LinkCount:     3,
			CrawledAt:     time.Now(),
		},
	}

	outputFile := tmpDir + "/export.json"
	err = exporter.ExportJSON(results, outputFile)

	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected export file to be created")
	}
}

func TestExporterExportCSV(t *testing.T) {
	tmpDir := t.TempDir()

	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	results := []types.PageResult{
		{
			URL:           "https://example.com/page1",
			StatusCode:    200,
			ContentLength: 1024,
			LinkCount:     5,
			CrawledAt:     time.Now(),
		},
	}

	outputFile := tmpDir + "/export.csv"
	err = exporter.ExportCSV(results, outputFile)

	if err != nil {
		t.Errorf("Failed to export CSV: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected export file to be created")
	}
}

func TestExporterExportSitemap(t *testing.T) {
	tmpDir := t.TempDir()

	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	results := []types.PageResult{
		{
			URL:       "https://example.com/page1",
			CrawledAt: time.Now(),
		},
		{
			URL:       "https://example.com/page2",
			CrawledAt: time.Now(),
		},
	}

	outputFile := tmpDir + "/sitemap.xml"
	err = exporter.ExportSitemap(results, outputFile)

	if err != nil {
		t.Errorf("Failed to export sitemap: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected sitemap file to be created")
	}
}

func TestExporterExportEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	results := []types.PageResult{}

	outputFile := tmpDir + "/export.json"
	err = exporter.ExportJSON(results, outputFile)

	if err != nil {
		t.Logf("Expected to handle empty results: %v", err)
	}
}
