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
		URL:             "https://example.com",
		StatusCode:      200,
		ContentLength:   1024,
		LinkCount:       5,
		CrawledAt:       time.Now(),
		MetaDescription: "A sample page",
		MetaKeywords:    "a,b,c",
		ImageCount:      3,
		ScriptCount:     2,
	}

	err = store.SavePage(result)
	if err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}

	pages, err := store.QueryPages(map[string]interface{}{})
	if err != nil {
		t.Fatalf("QueryPages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	got := pages[0]
	if got.MetaDescription != "A sample page" {
		t.Fatalf("meta_description=%q", got.MetaDescription)
	}
	if got.MetaKeywords != "a,b,c" {
		t.Fatalf("meta_keywords=%q", got.MetaKeywords)
	}
	if got.ImageCount != 3 || got.ScriptCount != 2 {
		t.Fatalf("counts image=%d script=%d", got.ImageCount, got.ScriptCount)
	}

	// Recrawl updates rather than wiping
	result.MetaDescription = "Updated"
	result.ImageCount = 9
	if err := store.SavePage(result); err != nil {
		t.Fatalf("resave: %v", err)
	}
	pages, err = store.QueryPages(map[string]interface{}{})
	if err != nil {
		t.Fatalf("QueryPages2: %v", err)
	}
	if pages[0].MetaDescription != "Updated" || pages[0].ImageCount != 9 {
		t.Fatalf("upsert failed: %+v", pages[0])
	}
}
