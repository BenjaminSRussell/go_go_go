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

	customhttp "github.com/BenjaminSRussell/go_go_go/internal/http"
	"github.com/BenjaminSRussell/go_go_go/internal/navigation"
	"github.com/BenjaminSRussell/go_go_go/internal/parser"
	"github.com/BenjaminSRussell/go_go_go/internal/persona"
	"github.com/BenjaminSRussell/go_go_go/internal/renderer"
	"github.com/BenjaminSRussell/go_go_go/internal/seeding"
	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/temoto/robotstxt"
)

// Crawler handles web crawling with optional advanced features
type Crawler struct {
	config   types.Config
	frontier *Frontier
	storage  *storage.Storage
	client   *http.Client

	robotsCache sync.Map

	discovered atomic.Int64
	processed  atomic.Int64
	errors     atomic.Int64

	sem      chan struct{}
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown atomic.Bool

	// Advanced components (optional)
	personaPool    *persona.PersonaPool
	navigator      *navigation.WeightedNavigator
	retryHandler   *customhttp.RetryHandler
	headerRotator  *customhttp.HeaderRotator
	chromeRenderer *renderer.ChromeRenderer
	sqliteStorage  *storage.SQLiteStorage

	// Feature flags
	enablePersonas    bool
	enableWeightedNav bool
	enableJSRendering bool
	enableSQLite      bool
}

// New creates a new crawler instance
func New(config types.Config) (*Crawler, error) {
	if config.StartURL == "" {
		return nil, fmt.Errorf("start URL is required")
	}

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	store, err := storage.New(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Crawler{
		config:            config,
		frontier:          NewFrontier(),
		storage:           store,
		client:            &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        config.Workers * 2,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		sem:               make(chan struct{}, config.Workers),
		ctx:               ctx,
		cancel:            cancel,
		enablePersonas:    config.EnablePersonas,
		enableWeightedNav: config.EnableWeightedNav,
		enableJSRendering: config.EnableJSRendering,
		enableSQLite:      config.EnableSQLite,
	}

	// Persona management
	if config.EnablePersonas {
		maxPersonas := config.MaxPersonas
		if maxPersonas <= 0 {
			maxPersonas = 50
		}
		lifetime := config.PersonaLifetime
		if lifetime <= 0 {
			lifetime = 30 * time.Minute
		}
		reuseLimit := config.PersonaReuseLimit
		if reuseLimit <= 0 {
			reuseLimit = 100
		}
		c.personaPool = persona.NewPersonaPool(maxPersonas, lifetime, reuseLimit)
		fmt.Printf("Persona system enabled: max=%d, lifetime=%v, reuse=%d\n",
			maxPersonas, lifetime, reuseLimit)
	}

	// Weighted navigation
	if config.EnableWeightedNav {
		c.navigator = navigation.NewWeightedNavigator()
		fmt.Println("Weighted navigation enabled")
	}

	// Retry handler
	retryConfig := customhttp.DefaultRetryConfig()
	if config.MaxRetries > 0 {
		retryConfig.MaxRetries = config.MaxRetries
	}
	c.retryHandler = customhttp.NewRetryHandler(retryConfig)

	// Header rotation
	if config.UseHeaderRotation {
		c.headerRotator = customhttp.NewHeaderRotator()
		fmt.Println("Header rotation enabled")
	}

	// JavaScript rendering
	if config.EnableJSRendering {
		chromeRenderer, err := renderer.NewChromeRenderer()
		if err != nil {
			fmt.Printf("Warning: failed to initialize Chrome renderer: %v\n", err)
			fmt.Println("Continuing without JavaScript rendering...")
			c.enableJSRendering = false
		} else {
			c.chromeRenderer = chromeRenderer
			fmt.Println("JavaScript rendering enabled")
		}
	}

	// SQLite storage
	if config.EnableSQLite {
		dbPath := fmt.Sprintf("%s/crawl.db", config.DataDir)
		sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
		if err != nil {
			fmt.Printf("Warning: failed to initialize SQLite storage: %v\n", err)
			fmt.Println("Continuing with JSONL storage only...")
			c.enableSQLite = false
		} else {
			c.sqliteStorage = sqliteStorage
			fmt.Println("SQLite storage enabled")
		}
	}

	c.frontier.Add(types.URLItem{URL: config.StartURL, Depth: 0})

	if err := c.runSeeding(); err != nil {
		return nil, fmt.Errorf("seeding failed: %w", err)
	}

	return c, nil
}

// Resume restores crawler from saved state
func Resume(dataDir string) (*Crawler, error) {
	// Load config
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

	// Restore frontier
	urls, err := store.LoadPendingURLs()
	if err != nil {
		return nil, fmt.Errorf("failed to load pending URLs: %w", err)
	}

	for _, item := range urls {
		c.frontier.Add(item)
	}

	return c, nil
}

// Crawl starts crawling
func (c *Crawler) Crawl() (*types.Results, error) {
	defer c.storage.Close()
	defer c.Close()

	if err := c.storage.SaveConfig(c.config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Starting crawl with %d workers\n", c.config.Workers)
	fmt.Printf("Initial frontier size: %d URLs\n", c.frontier.Size())
	c.printFeatureStatus()

	safeProcessor := NewSafeProcessor(c)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go c.reportProgress(ticker)

	for !c.shutdown.Load() {
		if c.frontier.IsEmpty() {
			time.Sleep(1 * time.Second)
			if c.frontier.IsEmpty() {
				fmt.Println("\nFrontier exhausted, finishing...")
				break
			}
		}

		item, ok := c.frontier.Next()
		if !ok {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		c.sem <- struct{}{}
		c.wg.Add(1)

		go safeProcessor.ProcessURLSafely(item)
	}

	c.wg.Wait()

	results := &types.Results{
		Discovered: int(c.discovered.Load()),
		Processed:  int(c.processed.Load()),
		Errors:     int(c.errors.Load()),
	}

	panicCount := safeProcessor.GetPanicCount()
	if panicCount > 0 {
		fmt.Printf("\n[WARNING] Total panics recovered: %d\n", panicCount)
	}

	c.printFinalStats(results)

	return results, nil
}

// processURL processes a single URL
func (c *Crawler) processURL(item types.URLItem) {
	defer c.wg.Done()
	defer func() { <-c.sem }()

	result := types.PageResult{
		URL:       item.URL,
		Depth:     item.Depth,
		CrawledAt: time.Now(),
	}

	// Get persona if enabled
	var currentPersona *persona.Persona
	if c.enablePersonas {
		parsedURL, err := url.Parse(item.URL)
		if err != nil {
			result.Error = fmt.Sprintf("invalid URL: %v", err)
			c.saveResult(result)
			c.errors.Add(1)
			return
		}

		currentPersona, err = c.personaPool.GetOrCreatePersona(parsedURL.Host)
		if err != nil {
			result.Error = fmt.Sprintf("persona creation failed: %v", err)
			c.saveResult(result)
			c.errors.Add(1)
			return
		}

		// Behavioral delay
		time.Sleep(currentPersona.GetThinkTime())
	}

	if !c.config.IgnoreRobots && !c.isAllowedByRobots(item.URL) {
		result.Error = "blocked by robots.txt"
		c.saveResult(result)
		c.errors.Add(1)
		return
	}

	// Retry with exponential backoff
	var resp *http.Response
	var err error
	maxRetries := c.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	parsedURL, _ := url.Parse(item.URL)
	host := parsedURL.Host

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if inBackoff, duration := c.retryHandler.IsInBackoff(host); inBackoff && attempt > 0 {
			time.Sleep(duration)
		}

		resp, err = c.makeRequest(item.URL, currentPersona)

		if err == nil && resp != nil {
			if resp.StatusCode == http.StatusOK {
				c.retryHandler.RecordSuccess(host)
				break
			}

			if c.retryHandler.ShouldRetry(resp.StatusCode, nil) {
				c.retryHandler.RecordFailure(host, resp.StatusCode)
				resp.Body.Close()

				if attempt < maxRetries {
					time.Sleep(c.retryHandler.GetBackoff(host, attempt))
					continue
				}
			}
			break
		}

		if err != nil && c.retryHandler.ShouldRetry(0, err) {
			c.retryHandler.RecordFailure(host, 0)
			if attempt < maxRetries {
				time.Sleep(c.retryHandler.GetBackoff(host, attempt))
				continue
			}
		}
		break
	}

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		c.saveResult(result)
		c.errors.Add(1)
		return
	}

	if resp == nil {
		result.Error = "nil response"
		c.saveResult(result)
		c.errors.Add(1)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("non-200 status: %d", resp.StatusCode)
		c.saveResult(result)
		c.errors.Add(1)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("body read failed: %v", err)
		c.saveResult(result)
		c.errors.Add(1)
		return
	}

	result.ContentLength = int64(len(body))
	htmlContent := string(body)

	// JavaScript rendering if needed
	if c.enableJSRendering && renderer.ShouldRender(htmlContent) {
		timeout := 30 * time.Second
		if currentPersona != nil {
			timeout = currentPersona.GetPageLoadTimeout()
		}

		renderedHTML, err := c.chromeRenderer.Render(item.URL, timeout)
		if err == nil {
			htmlContent = renderedHTML
			result.ContentLength = int64(len(renderedHTML))
		}
	}

	// Extract links
	var links []string
	if c.enableWeightedNav {
		weightedLinks, err := c.navigator.ExtractWeightedLinks(htmlContent, item.URL)
		if err == nil {
			visibleLinks := c.navigator.FilterVisibleLinks(weightedLinks)

			for _, wLink := range visibleLinks {
				shouldFollow := true
				if currentPersona != nil {
					shouldFollow = currentPersona.ShouldFollowLink(wLink.Weight)
				}

				if shouldFollow && c.shouldCrawl(wLink.URL, item.URL) {
					links = append(links, wLink.URL)
				}
			}
		}
	} else {
		links, result.Title = parser.ExtractLinks(htmlContent, item.URL)
	}

	result.LinkCount = len(links)

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

	c.saveResult(result)
	c.frontier.MarkProcessed()
	c.processed.Add(1)
}

// makeRequest creates and executes HTTP request
func (c *Crawler) makeRequest(urlStr string, p *persona.Persona) (*http.Response, error) {
	req, err := http.NewRequestWithContext(c.ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Apply headers
	if p != nil {
		p.ApplyToRequest(req)
	} else if c.headerRotator != nil {
		c.headerRotator.ApplyHeaders(req)
	} else {
		req.Header.Set("User-Agent", "GoGoGoBot/1.0 (+https://github.com/BenjaminSRussell/go_go_go)")
	}

	return c.client.Do(req)
}

// saveResult saves result to storage
func (c *Crawler) saveResult(result types.PageResult) {
	c.storage.SaveResult(result)

	if c.enableSQLite && c.sqliteStorage != nil {
		c.sqliteStorage.SavePage(result)
	}
}

// shouldCrawl checks if URL should be crawled
func (c *Crawler) shouldCrawl(link, baseURL string) bool {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return false
	}

	if parsedLink.Scheme != "http" && parsedLink.Scheme != "https" {
		return false
	}

	if c.config.CrawlExternalLinks {
		return true
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

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

	// Check robots.txt cache
	if data, ok := c.robotsCache.Load(robotsURL); ok {
		if robots, ok := data.(*robotstxt.RobotsData); ok {
			return robots.TestAgent(parsedURL.Path, "GoGoGoBot")
		}
	}

	// Fetch and parse robots.txt
	resp, err := c.client.Get(robotsURL)
	if err != nil {
		// Allow if robots.txt doesn't exist
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

// runSeeding runs seeding strategies
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

// printFeatureStatus prints enabled features
func (c *Crawler) printFeatureStatus() {
	fmt.Println("\n=== Enabled Features ===")
	if c.enablePersonas {
		fmt.Println("[+] Persona management")
		if c.personaPool != nil {
			stats := c.personaPool.GetStats()
			fmt.Printf("  Max personas: %v\n", stats["max_personas"])
		}
	}
	if c.enableWeightedNav {
		fmt.Println("[+] Weighted navigation")
	}
	if c.enableJSRendering {
		fmt.Println("[+] JavaScript rendering")
	}
	if c.enableSQLite {
		fmt.Println("[+] SQLite storage")
	}
	if c.headerRotator != nil {
		fmt.Println("[+] Header rotation")
	}
	if c.retryHandler != nil {
		fmt.Println("[+] Intelligent retry")
	}
	fmt.Println("========================")
}

// reportProgress prints progress
func (c *Crawler) reportProgress(ticker *time.Ticker) {
	for range ticker.C {
		discovered := c.discovered.Load()
		processed := c.processed.Load()
		errors := c.errors.Load()
		pending := c.frontier.Size()

		fmt.Printf("\r[Progress] Discovered: %d | Processed: %d | Errors: %d | Pending: %d",
			discovered, processed, errors, pending)

		if c.enablePersonas && c.personaPool != nil {
			stats := c.personaPool.GetStats()
			fmt.Printf(" | Personas: %v/%v", stats["active_personas"], stats["total_personas"])
		}
	}
}

// printFinalStats prints final statistics
func (c *Crawler) printFinalStats(results *types.Results) {
	fmt.Println("\n\n=== Final Statistics ===")
	fmt.Printf("Total discovered: %d\n", results.Discovered)
	fmt.Printf("Total processed:  %d\n", results.Processed)
	fmt.Printf("Total errors:     %d\n", results.Errors)

	if results.Processed > 0 {
		successRate := float64(results.Processed-results.Errors) / float64(results.Processed) * 100
		fmt.Printf("Success rate:     %.1f%%\n", successRate)
	}

	if c.enablePersonas && c.personaPool != nil {
		fmt.Println("\n--- Persona Statistics ---")
		stats := c.personaPool.GetStats()
		fmt.Printf("Total personas created: %v\n", stats["total_personas"])
		fmt.Printf("Active personas:        %v\n", stats["active_personas"])
	}

	if c.enableSQLite && c.sqliteStorage != nil {
		fmt.Println("\n--- Database Statistics ---")
		stats, err := c.sqliteStorage.GetStats()
		if err == nil {
			fmt.Printf("Total pages in DB:       %v\n", stats["total_pages"])
			fmt.Printf("Successful pages in DB:  %v\n", stats["successful_pages"])
			fmt.Printf("Failed pages in DB:      %v\n", stats["failed_pages"])
		}
	}

	fmt.Println("========================")
}

// Close releases resources
func (c *Crawler) Close() error {
	c.shutdown.Store(true)
	if c.cancel != nil {
		c.cancel()
	}
	if c.chromeRenderer != nil {
		c.chromeRenderer.Close()
	}
	if c.sqliteStorage != nil {
		c.sqliteStorage.Close()
	}
	if c.storage != nil {
		return c.storage.Close()
	}
	return nil
}
