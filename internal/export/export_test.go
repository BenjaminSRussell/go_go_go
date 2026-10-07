package export

import (
	"encoding/csv"
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

func TestExporterExportCSVIncludesTitleAndError(t *testing.T) {
	tmpDir := t.TempDir()
	exporter, err := NewExporter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	results := []types.PageResult{
		{
			URL:           "https://example.com/ok",
			StatusCode:    200,
			ContentLength: 100,
			LinkCount:     2,
			CrawledAt:     time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
			Title:         "Hello, World",
			Error:         "",
		},
		{
			URL:           "https://example.com/fail",
			StatusCode:    0,
			ContentLength: 0,
			LinkCount:     0,
			CrawledAt:     time.Date(2026, 10, 7, 12, 1, 0, 0, time.UTC),
			Title:         "",
			Error:         "connection refused",
		},
	}

	outputFile := tmpDir + "/export.csv"
	if err := exporter.ExportCSV(results, outputFile); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}

	f, err := os.Open(outputFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header+2), got %d", len(rows))
	}
	wantHeader := []string{"URL", "StatusCode", "ContentLength", "LinkCount", "CrawledAt", "Title", "Error"}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Fatalf("header[%d]=%q want %q", i, rows[0][i], h)
		}
	}
	// Title with comma must round-trip via encoding/csv
	if rows[1][5] != "Hello, World" {
		t.Fatalf("title=%q", rows[1][5])
	}
	if rows[1][6] != "" {
		t.Fatalf("empty error field want \"\", got %q", rows[1][6])
	}
	if rows[2][6] != "connection refused" {
		t.Fatalf("error=%q", rows[2][6])
	}
}
