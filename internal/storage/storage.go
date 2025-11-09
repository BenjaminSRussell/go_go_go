package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// Storage manages persistent storage of crawl data
type Storage struct {
	dataDir string
	mu      sync.Mutex
	jsonl   *os.File
}

// New creates a new storage instance
func New(dataDir string) (*Storage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	jsonlPath := filepath.Join(dataDir, "sitemap.jsonl")
	file, err := os.OpenFile(jsonlPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %w", err)
	}

	return &Storage{
		dataDir: dataDir,
		jsonl:   file,
	}, nil
}

// SaveResult saves a page result to storage
func (s *Storage) SaveResult(result types.PageResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if _, err := s.jsonl.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// SaveConfig saves crawler configuration
func (s *Storage) SaveConfig(config types.Config) error {
	configPath := filepath.Join(s.dataDir, "config.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadConfig loads crawler configuration
func (s *Storage) LoadConfig() (types.Config, error) {
	configPath := filepath.Join(s.dataDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return types.Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return types.Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// LoadPendingURLs loads pending URLs (for resume functionality)
func (s *Storage) LoadPendingURLs() ([]types.URLItem, error) {
	// For simplicity, we'll just return empty slice
	// In a production system, you'd want to persist the frontier state
	return []types.URLItem{}, nil
}

// LoadResults loads all crawl results
func (s *Storage) LoadResults() ([]types.PageResult, error) {
	jsonlPath := filepath.Join(s.dataDir, "sitemap.jsonl")

	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.PageResult{}, nil
		}
		return nil, fmt.Errorf("failed to read JSONL file: %w", err)
	}

	lines := make([]byte, 0)
	results := make([]types.PageResult, 0)

	for i, b := range data {
		if b == '\n' {
			if len(lines) > 0 {
				var result types.PageResult
				if err := json.Unmarshal(lines, &result); err == nil {
					results = append(results, result)
				}
				lines = lines[:0]
			}
		} else {
			lines = append(lines, data[i])
		}
	}

	// Handle last line
	if len(lines) > 0 {
		var result types.PageResult
		if err := json.Unmarshal(lines, &result); err == nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jsonl != nil {
		return s.jsonl.Close()
	}

	return nil
}
