package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage provides SQLite-based storage for queryable data
type SQLiteStorage struct {
	db *sql.DB
	// fts is true when the FTS5 index is available. mattn/go-sqlite3 only
	// compiles FTS5 in with `-tags sqlite_fts5`; without it Search falls back
	// to a weighted LIKE scan (#8).
	fts bool
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// WAL + busy timeout: crawler workers write concurrently (pages, links,
	// frontier); without these go-sqlite3 returns "database is locked".
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
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
		script_count INTEGER,
		content_hash TEXT
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
		UNIQUE(source_url, target_url),
		FOREIGN KEY (source_url) REFERENCES pages(url)
	);

	CREATE INDEX IF NOT EXISTS idx_source_url ON links(source_url);
	CREATE INDEX IF NOT EXISTS idx_target_url ON links(target_url);

	-- Pending crawl work for resume (#3): a row is inserted when a URL is
	-- enqueued and deleted once its result is saved to pages.
	CREATE TABLE IF NOT EXISTS frontier (
		url TEXT PRIMARY KEY,
		depth INTEGER NOT NULL,
		parent_url TEXT
	);

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

	// Databases created before content_hash existed (#3).
	if err := addColumnIfMissing(db, "pages", "content_hash", "TEXT"); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_content_hash ON pages(content_hash)"); err != nil {
		return nil, fmt.Errorf("failed to create content_hash index: %w", err)
	}

	store := &SQLiteStorage{db: db}
	store.fts = store.ensureFTS() == nil
	return store, nil
}

const ftsSchema = `
	CREATE VIRTUAL TABLE pages_fts USING fts5(
		url UNINDEXED, title, meta_description, meta_keywords,
		content='pages', content_rowid='id'
	);
	CREATE TRIGGER IF NOT EXISTS pages_fts_ai AFTER INSERT ON pages BEGIN
		INSERT INTO pages_fts(rowid, url, title, meta_description, meta_keywords)
		VALUES (new.id, new.url, new.title, new.meta_description, new.meta_keywords);
	END;
	CREATE TRIGGER IF NOT EXISTS pages_fts_ad AFTER DELETE ON pages BEGIN
		INSERT INTO pages_fts(pages_fts, rowid, url, title, meta_description, meta_keywords)
		VALUES ('delete', old.id, old.url, old.title, old.meta_description, old.meta_keywords);
	END;
	CREATE TRIGGER IF NOT EXISTS pages_fts_au AFTER UPDATE ON pages BEGIN
		INSERT INTO pages_fts(pages_fts, rowid, url, title, meta_description, meta_keywords)
		VALUES ('delete', old.id, old.url, old.title, old.meta_description, old.meta_keywords);
		INSERT INTO pages_fts(rowid, url, title, meta_description, meta_keywords)
		VALUES (new.id, new.url, new.title, new.meta_description, new.meta_keywords);
	END;
	INSERT INTO pages_fts(pages_fts) VALUES ('rebuild');
`

// ensureFTS creates the FTS5 index and its sync triggers, backfilling rows
// already in pages. Returns an error when FTS5 is not compiled in.
func (s *SQLiteStorage) ensureFTS() error {
	var name string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='pages_fts'`).Scan(&name)
	if err == nil {
		return nil // already created (and kept in sync by triggers)
	}
	if err != sql.ErrNoRows {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ftsSchema); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// FTSEnabled reports whether ranked FTS5 search is available.
func (s *SQLiteStorage) FTSEnabled() bool { return s.fts }

// SearchResult is one ranked hit from Search.
type SearchResult struct {
	URL             string
	Title           string
	MetaDescription string
	Score           float64 // higher is better
}

// Search finds pages whose title / meta description / keywords match query,
// best matches first. Uses FTS5 (bm25) when available, else a weighted LIKE.
func (s *SQLiteStorage) Search(query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if s.fts {
		return s.searchFTS(query, limit)
	}
	return s.searchLike(query, limit)
}

func (s *SQLiteStorage) searchFTS(query string, limit int) ([]SearchResult, error) {
	// Quote each term so user input like "c++" or "a:b" is not parsed as
	// FTS5 query syntax; terms are implicitly ANDed.
	terms := strings.Fields(query)
	for i, t := range terms {
		terms[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	rows, err := s.db.Query(`
		SELECT p.url, COALESCE(p.title, ''), COALESCE(p.meta_description, ''),
		       -bm25(pages_fts, 0.0, 10.0, 3.0, 1.0) AS score
		FROM pages_fts JOIN pages p ON p.id = pages_fts.rowid
		WHERE pages_fts MATCH ?
		ORDER BY score DESC
		LIMIT ?`, strings.Join(terms, " "), limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	return scanSearch(rows)
}

func (s *SQLiteStorage) searchLike(query string, limit int) ([]SearchResult, error) {
	like := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.Query(`
		SELECT url, COALESCE(title, ''), COALESCE(meta_description, ''),
		       (CASE WHEN LOWER(COALESCE(title, '')) LIKE ?1 THEN 10 ELSE 0 END) +
		       (CASE WHEN LOWER(COALESCE(meta_description, '')) LIKE ?1 THEN 3 ELSE 0 END) +
		       (CASE WHEN LOWER(COALESCE(meta_keywords, '')) LIKE ?1 THEN 1 ELSE 0 END) AS score
		FROM pages
		WHERE score > 0
		ORDER BY score DESC, url
		LIMIT ?2`, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return scanSearch(rows)
}

func scanSearch(rows *sql.Rows) ([]SearchResult, error) {
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.URL, &r.Title, &r.MetaDescription, &r.Score); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SavePage saves a page result to SQLite
func (s *SQLiteStorage) SavePage(result types.PageResult) error {
	// UPSERT so recrawls refresh meta columns without wiping unspecified fields
	// the way INSERT OR REPLACE can when the column list is incomplete.
	query := `
		INSERT INTO pages
		(url, depth, status_code, content_length, title, link_count, crawled_at, error,
		 meta_description, meta_keywords, image_count, script_count, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			depth = excluded.depth,
			status_code = excluded.status_code,
			content_length = excluded.content_length,
			title = excluded.title,
			link_count = excluded.link_count,
			crawled_at = excluded.crawled_at,
			error = excluded.error,
			meta_description = excluded.meta_description,
			meta_keywords = excluded.meta_keywords,
			image_count = excluded.image_count,
			script_count = excluded.script_count,
			content_hash = excluded.content_hash
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
		result.MetaDescription,
		result.MetaKeywords,
		result.ImageCount,
		result.ScriptCount,
		nullIfEmpty(result.ContentHash),
	)

	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// addColumnIfMissing runs ALTER TABLE ADD COLUMN when table lacks column.
func addColumnIfMissing(db *sql.DB, table, column, decl string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name, typ string
			notNull   int
			dflt      sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl))
	return err
}

// EnqueuePending records a URL as pending crawl work (for resume).
func (s *SQLiteStorage) EnqueuePending(item types.URLItem) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO frontier (url, depth, parent_url) VALUES (?, ?, ?)",
		item.URL, item.Depth, nullIfEmpty(item.ParentURL),
	)
	return err
}

// MarkDone removes a URL from the pending frontier once its result is saved.
func (s *SQLiteStorage) MarkDone(url string) error {
	_, err := s.db.Exec("DELETE FROM frontier WHERE url = ?", url)
	return err
}

// LoadPending returns frontier URLs that have no saved page yet, i.e. work a
// resumed crawl still has to do. Completed URLs are skipped.
func (s *SQLiteStorage) LoadPending() ([]types.URLItem, error) {
	rows, err := s.db.Query(`
		SELECT f.url, f.depth, COALESCE(f.parent_url, '')
		FROM frontier f
		WHERE NOT EXISTS (SELECT 1 FROM pages p WHERE p.url = f.url)
		ORDER BY f.depth, f.rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.URLItem
	for rows.Next() {
		var it types.URLItem
		if err := rows.Scan(&it.URL, &it.Depth, &it.ParentURL); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// CompletedURLs returns every URL that already has a saved page.
func (s *SQLiteStorage) CompletedURLs() ([]string, error) {
	rows, err := s.db.Query("SELECT url FROM pages")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

// LinkRow is one edge of the crawl link graph.
type LinkRow struct {
	SourceURL  string
	TargetURL  string
	AnchorText string
}

// AllLinks returns the full link graph.
func (s *SQLiteStorage) AllLinks() ([]LinkRow, error) {
	rows, err := s.db.Query("SELECT source_url, target_url, COALESCE(anchor_text, '') FROM links ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LinkRow
	for rows.Next() {
		var l LinkRow
		if err := rows.Scan(&l.SourceURL, &l.TargetURL, &l.AnchorText); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SaveMetaTags saves meta tags for a URL
func (s *SQLiteStorage) SaveMetaTags(url string, metaTags map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM meta_tags WHERE url = ?", url); err != nil {
		return err
	}

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

// SaveStructuredData saves JSON-LD structured data (replace-by-URL).
func (s *SQLiteStorage) SaveStructuredData(url, jsonData string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM structured_data WHERE url = ?", url); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO structured_data (url, json_data) VALUES (?, ?)", url, jsonData); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveLinks replaces outbound links for a source URL.
func (s *SQLiteStorage) SaveLinks(source string, links []types.Link) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM links WHERE source_url = ?", source); err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		"INSERT OR IGNORE INTO links (source_url, target_url, anchor_text) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, link := range links {
		if link.TargetURL == "" {
			continue
		}
		if _, err := stmt.Exec(source, link.TargetURL, link.AnchorText); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// QueryPages queries pages with filters
func (s *SQLiteStorage) QueryPages(filters map[string]interface{}) ([]types.PageResult, error) {
	query := "SELECT url, depth, status_code, content_length, title, link_count, crawled_at, error, meta_description, meta_keywords, image_count, script_count, content_hash FROM pages WHERE 1=1"
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
		var metaDesc, metaKw, contentHash sql.NullString
		var imgCount, scriptCount sql.NullInt64
		err := rows.Scan(
			&result.URL,
			&result.Depth,
			&result.StatusCode,
			&result.ContentLength,
			&result.Title,
			&result.LinkCount,
			&crawledAt,
			&result.Error,
			&metaDesc,
			&metaKw,
			&imgCount,
			&scriptCount,
			&contentHash,
		)
		result.ContentHash = contentHash.String
		if metaDesc.Valid {
			result.MetaDescription = metaDesc.String
		}
		if metaKw.Valid {
			result.MetaKeywords = metaKw.String
		}
		if imgCount.Valid {
			result.ImageCount = int(imgCount.Int64)
		}
		if scriptCount.Valid {
			result.ScriptCount = int(scriptCount.Int64)
		}
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
