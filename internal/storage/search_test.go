package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func seedSearch(t *testing.T) *SQLiteStorage {
	t.Helper()
	store, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "crawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	pages := []types.PageResult{
		{URL: "https://a.test/go", Title: "Golang crawler guide", MetaDescription: "Concurrency in Go"},
		{URL: "https://a.test/rust", Title: "Rust book", MetaDescription: "Mentions golang once"},
		{URL: "https://a.test/py", Title: "Python tips", MetaKeywords: "scripting"},
	}
	for _, p := range pages {
		p.StatusCode, p.CrawledAt = 200, time.Now()
		if err := store.SavePage(p); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func TestSearchRanksTitleMatchesFirst(t *testing.T) {
	store := seedSearch(t)
	defer store.Close()
	t.Logf("FTS5 enabled: %v", store.FTSEnabled())

	res, err := store.Search("golang", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("want 2 hits, got %d: %+v", len(res), res)
	}
	if res[0].URL != "https://a.test/go" {
		t.Fatalf("title match should rank first, got %+v", res)
	}
	if res[0].Score < res[1].Score {
		t.Fatalf("scores not descending: %+v", res)
	}
}

func TestSearchSeesIncrementalUpdates(t *testing.T) {
	store := seedSearch(t)
	defer store.Close()

	if err := store.SavePage(types.PageResult{URL: "https://a.test/py", Title: "Python and golang interop", StatusCode: 200, CrawledAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	res, err := store.Search("interop", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].URL != "https://a.test/py" {
		t.Fatalf("updated title not searchable: %+v", res)
	}
	if res, _ := store.Search("tips", 10); len(res) != 0 {
		t.Fatalf("stale title still indexed: %+v", res)
	}
}

func TestSearchEmptyAndSyntaxSafe(t *testing.T) {
	store := seedSearch(t)
	defer store.Close()
	if res, err := store.Search("   ", 10); err != nil || res != nil {
		t.Fatalf("blank query: %v %v", res, err)
	}
	if _, err := store.Search(`c++ "AND" title:`, 10); err != nil {
		t.Fatalf("special characters should not error: %v", err)
	}
}

func TestSearchBackfillsExistingDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crawl.db")
	store, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.SavePage(types.PageResult{URL: "https://b.test", Title: "Backfill me", StatusCode: 200, CrawledAt: time.Now()})
	store.Close()

	reopened, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if res, _ := reopened.Search("backfill", 5); len(res) != 1 {
		t.Fatalf("reopened DB search: %+v", res)
	}
}
