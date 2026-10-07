package export

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/parquet-go/parquet-go"
)

// PageRow is the Parquet schema for crawled pages (readable by DuckDB,
// pandas/pyarrow, Spark).
type PageRow struct {
	URL             string    `parquet:"url"`
	Depth           int32     `parquet:"depth"`
	StatusCode      int32     `parquet:"status_code"`
	ContentLength   int64     `parquet:"content_length"`
	ContentHash     string    `parquet:"content_hash,optional"`
	Title           string    `parquet:"title,optional"`
	LinkCount       int32     `parquet:"link_count"`
	CrawledAt       time.Time `parquet:"crawled_at,timestamp(millisecond)"`
	Error           string    `parquet:"error,optional"`
	MetaDescription string    `parquet:"meta_description,optional"`
}

// LinkRow is the Parquet schema for the link graph.
type LinkRow struct {
	FromURL    string `parquet:"from_url"`
	ToURL      string `parquet:"to_url"`
	AnchorText string `parquet:"anchor_text,optional"`
}

// ParquetConfig controls ExportParquet.
type ParquetConfig struct {
	DataDir   string // crawl data dir (crawl.db preferred, sitemap.jsonl fallback)
	PagesFile string // output path for pages (required)
	LinksFile string // output path for links; empty skips (needs crawl.db)
}

// ParquetStats reports what ExportParquet wrote.
type ParquetStats struct {
	Pages  int
	Links  int
	Source string
}

// ExportParquet writes crawl results to Parquet. It reads crawl.db when the
// crawl ran with --enable-sqlite and falls back to sitemap.jsonl otherwise.
func ExportParquet(cfg ParquetConfig) (ParquetStats, error) {
	var stats ParquetStats
	if cfg.PagesFile == "" {
		return stats, fmt.Errorf("pages output file is required")
	}

	var (
		results []types.PageResult
		links   []storage.LinkRow
	)
	dbPath := filepath.Join(cfg.DataDir, "crawl.db")
	if _, err := os.Stat(dbPath); err == nil {
		db, err := storage.NewSQLiteStorage(dbPath)
		if err != nil {
			return stats, fmt.Errorf("open %s: %w", dbPath, err)
		}
		defer db.Close()
		if results, err = db.QueryPages(map[string]interface{}{}); err != nil {
			return stats, fmt.Errorf("read pages: %w", err)
		}
		if cfg.LinksFile != "" {
			if links, err = db.AllLinks(); err != nil {
				return stats, fmt.Errorf("read links: %w", err)
			}
		}
		stats.Source = dbPath
	} else {
		store, err := storage.New(cfg.DataDir)
		if err != nil {
			return stats, fmt.Errorf("open data dir: %w", err)
		}
		defer store.Close()
		if results, err = store.LoadResults(); err != nil {
			return stats, fmt.Errorf("read sitemap.jsonl: %w", err)
		}
		if cfg.LinksFile != "" {
			return stats, fmt.Errorf("links export needs %s (crawl with --enable-sqlite)", dbPath)
		}
		stats.Source = filepath.Join(cfg.DataDir, "sitemap.jsonl")
	}

	pageRows := make([]PageRow, 0, len(results))
	for _, r := range results {
		pageRows = append(pageRows, PageRow{
			URL:             r.URL,
			Depth:           int32(r.Depth),
			StatusCode:      int32(r.StatusCode),
			ContentLength:   r.ContentLength,
			ContentHash:     r.ContentHash,
			Title:           r.Title,
			LinkCount:       int32(r.LinkCount),
			CrawledAt:       r.CrawledAt.UTC(),
			Error:           r.Error,
			MetaDescription: r.MetaDescription,
		})
	}
	if err := writeParquetFile(cfg.PagesFile, pageRows); err != nil {
		return stats, err
	}
	stats.Pages = len(pageRows)

	if cfg.LinksFile != "" {
		linkRows := make([]LinkRow, 0, len(links))
		for _, l := range links {
			linkRows = append(linkRows, LinkRow{FromURL: l.SourceURL, ToURL: l.TargetURL, AnchorText: l.AnchorText})
		}
		if err := writeParquetFile(cfg.LinksFile, linkRows); err != nil {
			return stats, err
		}
		stats.Links = len(linkRows)
	}
	return stats, nil
}

func writeParquetFile[T any](path string, rows []T) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := parquet.WriteFile(path, rows, parquet.Compression(&parquet.Zstd)); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
