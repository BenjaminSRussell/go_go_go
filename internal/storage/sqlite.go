package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage provides SQLite-based storage for queryable data
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		depth INTEGER NOT NULL,
		status_code INTEGER,
		content_length INTEGER,
		title TEXT,
		link_count INTEGER,
		crawled_at TIMESTAMP,
		error TEXT,
		meta_description TEXT,
		meta_keywords TEXT,
		image_count INTEGER,
		script_count INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_url ON pages(url);
	CREATE INDEX IF NOT EXISTS idx_status_code ON pages(status_code);
	CREATE INDEX IF NOT EXISTS idx_crawled_at ON pages(crawled_at);
	CREATE INDEX IF NOT EXISTS idx_depth ON pages(depth);

	CREATE TABLE IF NOT EXISTS links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_url TEXT NOT NULL,
		target_url TEXT NOT NULL,
		anchor_text TEXT,
		FOREIGN KEY (source_url) REFERENCES pages(url)
	);

	CREATE INDEX IF NOT EXISTS idx_source_url ON links(source_url);
	CREATE INDEX IF NOT EXISTS idx_target_url ON links(target_url);

	CREATE TABLE IF NOT EXISTS meta_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		name TEXT NOT NULL,
		content TEXT,
		FOREIGN KEY (url) REFERENCES pages(url)
	);

	CREATE INDEX IF NOT EXISTS idx_meta_url ON meta_tags(url);

	CREATE TABLE IF NOT EXISTS structured_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		json_data TEXT NOT NULL,
		FOREIGN KEY (url) REFERENCES pages(url)
	);

	CREATE INDEX IF NOT EXISTS idx_structured_url ON structured_data(url);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// SavePage saves a page result to SQLite
func (s *SQLiteStorage) SavePage(result types.PageResult) error {
	query := `
		INSERT OR REPLACE INTO pages
		(url, depth, status_code, content_length, title, link_count, crawled_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		result.URL,
		result.Depth,
		result.StatusCode,
		result.ContentLength,
		result.Title,
		result.LinkCount,
		result.CrawledAt,
		result.Error,
	)

	return err
}

// SaveMetaTags saves meta tags for a URL
func (s *SQLiteStorage) SaveMetaTags(url string, metaTags map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO meta_tags (url, name, content) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for name, content := range metaTags {
		if _, err := stmt.Exec(url, name, content); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveStructuredData saves JSON-LD structured data
func (s *SQLiteStorage) SaveStructuredData(url, jsonData string) error {
	query := "INSERT INTO structured_data (url, json_data) VALUES (?, ?)"
	_, err := s.db.Exec(query, url, jsonData)
	return err
}

// QueryPages queries pages with filters
func (s *SQLiteStorage) QueryPages(filters map[string]interface{}) ([]types.PageResult, error) {
	query := "SELECT url, depth, status_code, content_length, title, link_count, crawled_at, error FROM pages WHERE 1=1"
	args := make([]interface{}, 0)

	if statusCode, ok := filters["status_code"]; ok {
		query += " AND status_code = ?"
		args = append(args, statusCode)
	}

	if depth, ok := filters["depth"]; ok {
		query += " AND depth = ?"
		args = append(args, depth)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]types.PageResult, 0)
	for rows.Next() {
		var result types.PageResult
		var crawledAt string
		err := rows.Scan(
			&result.URL,
			&result.Depth,
			&result.StatusCode,
			&result.ContentLength,
			&result.Title,
			&result.LinkCount,
			&crawledAt,
			&result.Error,
		)
		if err != nil {
			continue
		}
		result.CrawledAt, _ = time.Parse(time.RFC3339, crawledAt)
		results = append(results, result)
	}

	return results, nil
}

// GetStats returns crawl statistics
func (s *SQLiteStorage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total pages
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_pages"] = total

	// Successful pages
	var successful int
	err = s.db.QueryRow("SELECT COUNT(*) FROM pages WHERE status_code = 200").Scan(&successful)
	if err != nil {
		return nil, err
	}
	stats["successful_pages"] = successful

	// Failed pages
	var failed int
	err = s.db.QueryRow("SELECT COUNT(*) FROM pages WHERE status_code != 200 OR error IS NOT NULL").Scan(&failed)
	if err != nil {
		return nil, err
	}
	stats["failed_pages"] = failed

	return stats, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
