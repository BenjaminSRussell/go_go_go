package export

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/parquet-go/parquet-go"
)

func TestExportParquetFromSQLite(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.NewSQLiteStorage(filepath.Join(dir, "crawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, p := range []types.PageResult{
		{URL: "https://x.test/", StatusCode: 200, ContentLength: 10, ContentHash: "h1", Title: "Home", LinkCount: 2, CrawledAt: now},
		{URL: "https://x.test/a", Depth: 1, StatusCode: 404, CrawledAt: now, Error: "non-200 status: 404"},
	} {
		if err := db.SavePage(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveLinks("https://x.test/", []types.Link{{TargetURL: "https://x.test/a"}, {TargetURL: "https://x.test/b"}}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	pagesOut := filepath.Join(dir, "out", "pages.parquet")
	linksOut := filepath.Join(dir, "out", "links.parquet")
	stats, err := ExportParquet(ParquetConfig{DataDir: dir, PagesFile: pagesOut, LinksFile: linksOut})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Pages != 2 || stats.Links != 2 {
		t.Fatalf("stats = %+v", stats)
	}

	pages, err := parquet.ReadFile[PageRow](pagesOut)
	if err != nil {
		t.Fatal(err)
	}
	byURL := map[string]PageRow{}
	for _, p := range pages {
		byURL[p.URL] = p
	}
	home := byURL["https://x.test/"]
	if home.StatusCode != 200 || home.ContentHash != "h1" || home.Title != "Home" || !home.CrawledAt.Equal(now) {
		t.Fatalf("home row = %+v", home)
	}
	if byURL["https://x.test/a"].StatusCode != 404 {
		t.Fatalf("404 row = %+v", byURL["https://x.test/a"])
	}

	links, err := parquet.ReadFile[LinkRow](linksOut)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || links[0].FromURL != "https://x.test/" {
		t.Fatalf("links = %+v", links)
	}
}

func TestExportParquetLinksRequireSQLite(t *testing.T) {
	dir := t.TempDir()
	_, err := ExportParquet(ParquetConfig{DataDir: dir, PagesFile: filepath.Join(dir, "p.parquet"), LinksFile: filepath.Join(dir, "l.parquet")})
	if err == nil {
		t.Fatal("expected error exporting links without crawl.db")
	}
}
