package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestRobotsAgentToken(t *testing.T) {
	cases := map[string]string{
		"":                             "GoGoGoBot",
		"GoGoGoBot/1.0 (+https://x.y)": "GoGoGoBot",
		"MyBot":                        "MyBot",
		"Acme Crawler":                 "Acme",
	}
	for in, want := range cases {
		if got := robotsAgentToken(in); got != want {
			t.Errorf("robotsAgentToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHostGovernorCapsConcurrencyPerHost(t *testing.T) {
	g := newHostGovernor(2)
	var cur, peak atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := g.acquire(context.Background(), "a.example", 0)
			if err != nil {
				t.Error(err)
				return
			}
			n := cur.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			cur.Add(-1)
			release()
		}()
	}
	wg.Wait()
	if p := peak.Load(); p != 2 {
		t.Fatalf("peak concurrency = %d, want 2", p)
	}
}

func TestHostGovernorSpacesStartsByDelay(t *testing.T) {
	g := newHostGovernor(4)
	const delay = 50 * time.Millisecond
	var mu sync.Mutex
	var starts []time.Time
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := g.acquire(context.Background(), "b.example", delay)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			starts = append(starts, time.Now())
			mu.Unlock()
			release()
		}()
	}
	wg.Wait()
	assertSpacing(t, starts, delay)
}

func TestHostGovernorAcquireHonorsCancel(t *testing.T) {
	g := newHostGovernor(1)
	release, _ := g.acquire(context.Background(), "c.example", 0)
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := g.acquire(ctx, "c.example", 0); err == nil {
		t.Fatal("expected context error while host slot is held")
	}
}

// Integration: Disallow is skipped and Crawl-Delay spaces page fetches.
func TestCrawlHonorsRobotsDisallowAndCrawlDelay(t *testing.T) {
	const delay = 200 * time.Millisecond
	var mu sync.Mutex
	var pageHits []time.Time
	var privateHits atomic.Int32
	var gotUA atomic.Value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			fmt.Fprint(w, "User-agent: PoliteTestBot\nDisallow: /private\nCrawl-delay: 0.2\n\nUser-agent: *\nDisallow: /\n")
			return
		case "/private":
			privateHits.Add(1)
		}
		gotUA.Store(r.Header.Get("User-Agent"))
		mu.Lock()
		pageHits = append(pageHits, time.Now())
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			fmt.Fprint(w, `<html><body><a href="/a">a</a><a href="/b">b</a><a href="/c">c</a><a href="/private">p</a></body></html>`)
			return
		}
		fmt.Fprint(w, "<html><body>leaf</body></html>")
	}))
	defer srv.Close()

	c, err := New(types.Config{
		StartURL:        srv.URL + "/",
		Workers:         8,
		Timeout:         5 * time.Second,
		DataDir:         t.TempDir(),
		SeedingStrategy: "none",
		MaxDepth:        1,
		UserAgent:       "PoliteTestBot/0.1",
		// Even with a generous per-host cap, Crawl-Delay must space starts.
		PerHostConcurrency: 8,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Crawl(); err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	if n := privateHits.Load(); n != 0 {
		t.Fatalf("/private fetched %d times despite Disallow", n)
	}
	if ua, _ := gotUA.Load().(string); ua != "PoliteTestBot/0.1" {
		t.Fatalf("User-Agent = %q, want configured UA", ua)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(pageHits) != 4 { // "/", /a, /b, /c
		t.Fatalf("page hits = %d, want 4", len(pageHits))
	}
	assertSpacing(t, pageHits, delay)
}

func assertSpacing(t *testing.T, starts []time.Time, delay time.Duration) {
	t.Helper()
	sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
	// Small tolerance for timer granularity.
	min := delay - 10*time.Millisecond
	for i := 1; i < len(starts); i++ {
		if gap := starts[i].Sub(starts[i-1]); gap < min {
			t.Fatalf("request %d started %v after previous, want >= %v", i, gap, delay)
		}
	}
}
