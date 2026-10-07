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

	// Crawl bounds (MaxDepth default 5; MaxPages 0 = unlimited)
	MaxDepth int
	MaxPages int64

	// Advanced features
	EnableJSRendering bool
	JSMaxTabs         int           // concurrent Chrome tabs (0 = default 4)
	JSIdleTimeout     time.Duration // shut Chrome down after idle (0 = default 60s)
	ChromeFlags       []string      // extra Chrome switches, "name" or "name=value"
	EnableSQLite      bool
	UseHeaderRotation bool
	MaxRetries        int

	// Persona & behavioral features
	EnablePersonas    bool
	MaxPersonas       int
	PersonaLifetime   time.Duration
	PersonaReuseLimit int
	EnableWeightedNav bool
	CrawlExternalLinks bool

	// MetricsAddr if non-empty (e.g. ":9090") serves Prometheus text at /metrics
	MetricsAddr string
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
	URL              string    `json:"url"`
	Depth            int       `json:"depth"`
	StatusCode       int       `json:"status_code"`
	ContentLength    int64     `json:"content_length"`
	Title            string    `json:"title"`
	LinkCount        int       `json:"link_count"`
	CrawledAt        time.Time `json:"crawled_at"`
	Error            string    `json:"error,omitempty"`
	MetaDescription  string    `json:"meta_description,omitempty"`
	MetaKeywords     string    `json:"meta_keywords,omitempty"`
	ImageCount       int       `json:"image_count,omitempty"`
	ScriptCount      int       `json:"script_count,omitempty"`
}


// Link is an outbound hyperlink from a crawled page.
type Link struct {
	TargetURL  string `json:"target_url"`
	AnchorText string `json:"anchor_text,omitempty"`
}
