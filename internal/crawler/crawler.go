package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/parser"
	"github.com/BenjaminSRussell/go_go_go/internal/seeding"
	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/temoto/robotstxt"
)

// Crawler is the main crawler engine
type Crawler struct {
	config   types.Config
	frontier *Frontier
	storage  *storage.Storage
	client   *http.Client

	// Robot exclusion
	robotsCache sync.Map // map[string]*robotstxt.RobotsData

	// Stats
	discovered atomic.Int64
	processed  atomic.Int64
	errors     atomic.Int64

	// Concurrency control
	sem      chan struct{}
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown atomic.Bool
}

// New creates a new crawler instance
func New(config types.Config) (*Crawler, error) {
	// Validate config
	if config.StartURL == "" {
		return nil, fmt.Errorf("start URL is required")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize storage
	store, err := storage.New(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Crawler{
		config:   config,
		frontier: NewFrontier(),
		storage:  store,
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        config.Workers * 2,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		sem:    make(chan struct{}, config.Workers),
		ctx:    ctx,
		cancel: cancel,
	}

	// Add start URL
	c.frontier.Add(types.URLItem{URL: config.StartURL, Depth: 0})

	// Run seeding strategies
	if err := c.runSeeding(); err != nil {
		return nil, fmt.Errorf("seeding failed: %w", err)
	}

	return c, nil
}

// Resume restores a crawler from saved state
func Resume(dataDir string) (*Crawler, error) {
	// Load config from storage
	store, err := storage.New(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load storage: %w", err)
	}

	config, err := store.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	c, err := New(config)
	if err != nil {
		return nil, err
	}

	// Restore frontier from storage
	urls, err := store.LoadPendingURLs()
	if err != nil {
		return nil, fmt.Errorf("failed to load pending URLs: %w", err)
	}

	for _, item := range urls {
		c.frontier.Add(item)
	}

	return c, nil
}

// Crawl starts the crawling process
func (c *Crawler) Crawl() (*types.Results, error) {
	defer c.storage.Close()

	// Save initial config
	if err := c.storage.SaveConfig(c.config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Starting crawl with %d workers\n", c.config.Workers)
	fmt.Printf("Initial frontier size: %d URLs\n", c.frontier.Size())

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Start progress reporter
	go c.reportProgress(ticker)

	// Main crawl loop
	for !c.shutdown.Load() {
		// Check if frontier is empty
		if c.frontier.IsEmpty() {
			// Wait a bit for workers to finish
			time.Sleep(1 * time.Second)
			if c.frontier.IsEmpty() {
				fmt.Println("\nFrontier exhausted, finishing...")
				break
			}
		}

		// Get next URL
		item, ok := c.frontier.Next()
		if !ok {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Acquire semaphore
		c.sem <- struct{}{}
		c.wg.Add(1)

		// Process URL in goroutine
		go c.processURL(item)
	}

	// Wait for all workers to finish
	c.wg.Wait()

	results := &types.Results{
		Discovered: int(c.discovered.Load()),
		Processed:  int(c.processed.Load()),
		Errors:     int(c.errors.Load()),
	}

	return results, nil
}

// processURL crawls a single URL
func (c *Crawler) processURL(item types.URLItem) {
	defer c.wg.Done()
	defer func() { <-c.sem }()

	result := types.PageResult{
		URL:       item.URL,
		Depth:     item.Depth,
		CrawledAt: time.Now(),
	}

	// Check robots.txt
	if !c.config.IgnoreRobots && !c.isAllowedByRobots(item.URL) {
		result.Error = "blocked by robots.txt"
		c.storage.SaveResult(result)
		c.errors.Add(1)
		return
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(c.ctx, "GET", item.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("request creation failed: %v", err)
		c.storage.SaveResult(result)
		c.errors.Add(1)
		return
	}

	req.Header.Set("User-Agent", "GoGoGoBot/1.0 (+https://github.com/BenjaminSRussell/go_go_go)")

	resp, err := c.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		c.storage.SaveResult(result)
		c.errors.Add(1)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Only process successful responses
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("non-200 status: %d", resp.StatusCode)
		c.storage.SaveResult(result)
		c.errors.Add(1)
		return
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("body read failed: %v", err)
		c.storage.SaveResult(result)
		c.errors.Add(1)
		return
	}

	result.ContentLength = int64(len(body))

	// Parse HTML and extract links
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		links, title := parser.ExtractLinks(string(body), item.URL)
		result.Title = title
		result.LinkCount = len(links)

		// Add links to frontier
		for _, link := range links {
			if c.shouldCrawl(link, item.URL) {
				c.frontier.Add(types.URLItem{
					URL:       link,
					Depth:     item.Depth + 1,
					ParentURL: item.URL,
				})
				c.discovered.Add(1)
			}
		}
	}

	// Save result
	c.storage.SaveResult(result)
	c.frontier.MarkProcessed()
	c.processed.Add(1)
}

// shouldCrawl determines if a URL should be crawled
func (c *Crawler) shouldCrawl(link, baseURL string) bool {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return false
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	// Only crawl HTTP(S)
	if parsedLink.Scheme != "http" && parsedLink.Scheme != "https" {
		return false
	}

	// Stay on same domain (or subdomain)
	if !strings.HasSuffix(parsedLink.Host, parsedBase.Host) &&
		parsedLink.Host != parsedBase.Host {
		return false
	}

	return true
}

// isAllowedByRobots checks robots.txt
func (c *Crawler) isAllowedByRobots(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

	// Check cache
	if data, ok := c.robotsCache.Load(robotsURL); ok {
		if robots, ok := data.(*robotstxt.RobotsData); ok {
			return robots.TestAgent(parsedURL.Path, "GoGoGoBot")
		}
	}

	// Fetch robots.txt
	resp, err := c.client.Get(robotsURL)
	if err != nil {
		// If robots.txt doesn't exist, allow crawling
		return true
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return true
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return true
	}

	robots, err := robotstxt.FromBytes(body)
	if err != nil {
		return true
	}

	c.robotsCache.Store(robotsURL, robots)

	return robots.TestAgent(parsedURL.Path, "GoGoGoBot")
}

// runSeeding executes seeding strategies
func (c *Crawler) runSeeding() error {
	if c.config.SeedingStrategy == "none" {
		return nil
	}

	strategies := []string{}
	if c.config.SeedingStrategy == "all" {
		strategies = []string{"sitemap", "ct", "commoncrawl"}
	} else {
		strategies = strings.Split(c.config.SeedingStrategy, ",")
	}

	for _, strategy := range strategies {
		fmt.Printf("Running seeding strategy: %s\n", strategy)

		var urls []string
		var err error

		switch strategy {
		case "sitemap":
			urls, err = seeding.DiscoverFromSitemap(c.config.StartURL, c.client)
		case "ct":
			urls, err = seeding.DiscoverFromCertificateTransparency(c.config.StartURL)
		case "commoncrawl":
			urls, err = seeding.DiscoverFromCommonCrawl(c.config.StartURL)
		default:
			fmt.Printf("Unknown seeding strategy: %s\n", strategy)
			continue
		}

		if err != nil {
			fmt.Printf("Seeding strategy %s failed: %v\n", strategy, err)
			continue
		}

		added := 0
		for _, url := range urls {
			if c.frontier.Add(types.URLItem{URL: url, Depth: 0}) {
				added++
			}
		}
		fmt.Printf("Added %d URLs from %s\n", added, strategy)
	}

	return nil
}

// reportProgress prints crawl progress
func (c *Crawler) reportProgress(ticker *time.Ticker) {
	for range ticker.C {
		discovered := c.discovered.Load()
		processed := c.processed.Load()
		errors := c.errors.Load()
		pending := c.frontier.Size()

		fmt.Printf("\rDiscovered: %d | Processed: %d | Errors: %d | Pending: %d",
			discovered, processed, errors, pending)
	}
}

// Close closes the crawler and releases resources
func (c *Crawler) Close() error {
	c.shutdown.Store(true)
	if c.cancel != nil {
		c.cancel()
	}
	if c.storage != nil {
		return c.storage.Close()
	}
	return nil
}
