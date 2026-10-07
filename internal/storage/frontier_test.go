package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestFrontierPendingSkipsCompleted(t *testing.T) {
	s, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "crawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for i, u := range []string{"https://x.test/", "https://x.test/a", "https://x.test/b"} {
		if err := s.EnqueuePending(types.URLItem{URL: u, Depth: i}); err != nil {
			t.Fatal(err)
		}
	}
	// Duplicate enqueue is ignored.
	if err := s.EnqueuePending(types.URLItem{URL: "https://x.test/a", Depth: 9}); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePage(types.PageResult{URL: "https://x.test/", StatusCode: 200, CrawledAt: time.Now(), ContentHash: "abc"}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkDone("https://x.test/"); err != nil {
		t.Fatal(err)
	}
	// A page saved without MarkDone (crash between the two) is still skipped.
	if err := s.SavePage(types.PageResult{URL: "https://x.test/b", StatusCode: 200, CrawledAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	pending, err := s.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].URL != "https://x.test/a" || pending[0].Depth != 1 {
		t.Fatalf("pending = %+v, want only /a at depth 1", pending)
	}

	pages, err := s.QueryPages(map[string]interface{}{"status_code": 200})
	if err != nil {
		t.Fatal(err)
	}
	var gotHash string
	for _, p := range pages {
		if p.URL == "https://x.test/" {
			gotHash = p.ContentHash
		}
	}
	if gotHash != "abc" {
		t.Fatalf("content_hash = %q, want abc", gotHash)
	}
}

func TestContentHashMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	// Pre-#3 pages schema without content_hash.
	if _, err := db.Exec(`CREATE TABLE pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, url TEXT UNIQUE NOT NULL, depth INTEGER NOT NULL,
		status_code INTEGER, content_length INTEGER, title TEXT, link_count INTEGER,
		crawled_at TIMESTAMP, error TEXT, meta_description TEXT, meta_keywords TEXT,
		image_count INTEGER, script_count INTEGER)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatalf("open old db: %v", err)
	}
	defer s.Close()
	if err := s.SavePage(types.PageResult{URL: "https://x.test/", CrawledAt: time.Now(), ContentHash: "h"}); err != nil {
		t.Fatalf("SavePage after migration: %v", err)
	}
}
