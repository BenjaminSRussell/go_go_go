package seeding

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCTSeederHTTPtest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name_value":"a.example.com\nb.example.com"},{"name_value":"*.cdn.example.com"}]`))
	}))
	defer srv.Close()

	oldClient, oldURL := httpClient, crtShURL
	httpClient = srv.Client()
	crtShURL = func(domain string) string { return srv.URL + "/?q=" + domain }
	t.Cleanup(func() { httpClient = oldClient; crtShURL = oldURL })

	urls, err := DiscoverFromCertificateTransparency("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(urls, ",")
	if !strings.Contains(joined, "a.example.com") || !strings.Contains(joined, "cdn.example.com") {
		t.Fatalf("unexpected urls: %v", urls)
	}
}

func TestCTSeederStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	oldClient, oldURL := httpClient, crtShURL
	httpClient = srv.Client()
	crtShURL = func(domain string) string { return srv.URL }
	t.Cleanup(func() { httpClient = oldClient; crtShURL = oldURL })

	_, err := DiscoverFromCertificateTransparency("https://example.com")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("want status 500 error, got %v", err)
	}
}

func TestCTSeederMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	oldClient, oldURL := httpClient, crtShURL
	httpClient = srv.Client()
	crtShURL = func(domain string) string { return srv.URL }
	t.Cleanup(func() { httpClient = oldClient; crtShURL = oldURL })

	_, err := DiscoverFromCertificateTransparency("https://example.com")
	if err == nil || !strings.Contains(err.Error(), "parse CT response") {
		t.Fatalf("want parse error, got %v", err)
	}
}

func TestCommonCrawlSeederHTTPtest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"url\":\"https://example.com/a\"}\n{\"url\":\"https://example.com/b\"}\n"))
	}))
	defer srv.Close()
	oldClient, oldURL := httpClient, commonCrawlURL
	httpClient = srv.Client()
	commonCrawlURL = func(domain string) string { return srv.URL }
	t.Cleanup(func() { httpClient = oldClient; commonCrawlURL = oldURL })

	urls, err := DiscoverFromCommonCrawl("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 {
		t.Fatalf("want 2 urls, got %v", urls)
	}
}

func TestCommonCrawlSeederStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	oldClient, oldURL := httpClient, commonCrawlURL
	httpClient = srv.Client()
	commonCrawlURL = func(domain string) string { return srv.URL }
	t.Cleanup(func() { httpClient = oldClient; commonCrawlURL = oldURL })

	_, err := DiscoverFromCommonCrawl("https://example.com")
	if err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("want status error, got %v", err)
	}
}
