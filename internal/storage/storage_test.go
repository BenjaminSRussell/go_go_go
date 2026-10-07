package storage

import (
	"os"
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestStorageNew(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if store == nil {
		t.Error("Expected storage to be created")
	}

	store.Close()
}

func TestStorageSaveResult(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	result := types.PageResult{
		URL:           "https://example.com",
		StatusCode:    200,
		ContentLength: 1024,
		LinkCount:     5,
		CrawledAt:     time.Now(),
	}

	err = store.SaveResult(result)
	if err != nil {
		t.Errorf("Failed to save result: %v", err)
	}
}

func TestStorageSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	config := types.Config{
		StartURL: "https://example.com",
		Workers:  4,
		Timeout:  30 * time.Second,
		DataDir:  tmpDir,
	}

	err = store.SaveConfig(config)
	if err != nil {
		t.Errorf("Failed to save config: %v", err)
	}
}

func TestStorageLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	config := types.Config{
		StartURL: "https://example.com",
		Workers:  4,
		Timeout:  30 * time.Second,
		DataDir:  tmpDir,
	}

	err = store.SaveConfig(config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := store.LoadConfig()
	if err != nil {
		t.Errorf("Failed to load config: %v", err)
	}

	if loaded.StartURL != config.StartURL {
		t.Errorf("Expected StartURL %s, got %s", config.StartURL, loaded.StartURL)
	}
}

func TestStorageSQLiteNew(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	store, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}

	if store == nil {
		t.Error("Expected SQLite storage to be created")
	}

	store.Close()
	os.Remove(tmpFile)
}

func TestStorageSQLiteSavePage(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	defer os.Remove(tmpFile)

	store, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer store.Close()

	result := types.PageResult{
		URL:           "https://example.com",
		StatusCode:    200,
		ContentLength: 1024,
		LinkCount:     5,
		CrawledAt:     time.Now(),
	}

	err = store.SavePage(result)
	if err != nil {
		t.Errorf("Failed to save page: %v", err)
	}
}

func TestSaveMetaTagsReplacesOnRecrawl(t *testing.T) {
	tmpFile := t.TempDir() + "/meta.db"
	store, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_ = store.SavePage(types.PageResult{URL: "https://ex.com", CrawledAt: time.Now()})
	if err := store.SaveMetaTags("https://ex.com", map[string]string{"a": "1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMetaTags("https://ex.com", map[string]string{"a": "2", "b": "3"}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM meta_tags WHERE url = ?", "https://ex.com").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 meta rows after replace, got %d", n)
	}
}

func TestSaveLinksReplacesBySource(t *testing.T) {
	tmpFile := t.TempDir() + "/links.db"
	store, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_ = store.SavePage(types.PageResult{URL: "https://ex.com", CrawledAt: time.Now()})
	if err := store.SaveLinks("https://ex.com", []types.Link{{TargetURL: "https://a", AnchorText: "A"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLinks("https://ex.com", []types.Link{{TargetURL: "https://b", AnchorText: "B"}, {TargetURL: "https://c"}}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM links WHERE source_url = ?", "https://ex.com").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 links, got %d", n)
	}
}
