package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// Fresh crawl populates crawl.db; an interrupted crawl resumes from the
// persisted frontier and never refetches completed URLs (#3).
func TestSQLiteCrawlResumeSkipsCompleted(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, `<html><head><title>home</title></head><body><a href="/a">a</a><a href="/b">b</a><a href="/c">c</a></body></html>`)
		default:
			fmt.Fprintf(w, "<html><body>leaf %s</body></html>", r.URL.Path)
		}
	}))
	defer srv.Close()

	dataDir := t.TempDir()
	c, err := New(types.Config{
		StartURL:        srv.URL + "/",
		Workers:         1,
		Timeout:         5 * time.Second,
		DataDir:         dataDir,
		SeedingStrategy: "none",
		IgnoreRobots:    true,
		MaxDepth:        2,
		MaxPages:        2, // stop early to simulate an interrupted crawl
		EnableSQLite:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Crawl(); err != nil {
		t.Fatal(err)
	}

	db, err := storage.NewSQLiteStorage(filepath.Join(dataDir, "crawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	firstDone, _ := db.CompletedURLs()
	pending, _ := db.LoadPending()
	links, _ := db.AllLinks()
	db.Close()
	if len(firstDone) != 2 || len(pending) != 2 {
		t.Fatalf("after first run: completed=%v pending=%v", firstDone, pending)
	}
	if len(links) != 3 {
		t.Fatalf("links from / = %d, want 3", len(links))
	}

	r, err := Resume(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if r.FrontierSize() != 2 {
		t.Fatalf("resumed frontier = %d, want 2", r.FrontierSize())
	}
	if _, err := r.Crawl(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, p := range []string{"/", "/a", "/b", "/c"} {
		if hits[p] != 1 {
			t.Fatalf("hits = %v: %s fetched %d times, want exactly 1", hits, p, hits[p])
		}
	}

	db, err = storage.NewSQLiteStorage(filepath.Join(dataDir, "crawl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pages, err := db.QueryPages(map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 4 {
		t.Fatalf("pages = %d, want 4", len(pages))
	}
	for _, p := range pages {
		if len(p.ContentHash) != 64 {
			t.Fatalf("page %s content_hash = %q, want sha256 hex", p.URL, p.ContentHash)
		}
	}
}
