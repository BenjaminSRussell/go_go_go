package types

import (
	"time"
)

// Config holds crawler configuration
type Config struct {
	StartURL        string
	Workers         int
	Timeout         time.Duration
	DataDir         string
	SeedingStrategy string
	IgnoreRobots    bool
	EnableRedis     bool
	RedisURL        string

	// Advanced features
	EnableProxies     bool
	EnableTLS         bool
	EnableJSRendering bool
	EnableSQLite      bool
	UseHeaderRotation bool
	MaxRetries        int
}

// Results contains crawl statistics
type Results struct {
	Discovered int
	Processed  int
	Errors     int
}

// URLItem represents a URL in the frontier
type URLItem struct {
	URL       string
	Depth     int
	ParentURL string
}

// PageResult contains information about a crawled page
type PageResult struct {
	URL           string    `json:"url"`
	Depth         int       `json:"depth"`
	StatusCode    int       `json:"status_code"`
	ContentLength int64     `json:"content_length"`
	Title         string    `json:"title"`
	LinkCount     int       `json:"link_count"`
	CrawledAt     time.Time `json:"crawled_at"`
	Error         string    `json:"error,omitempty"`
}
