package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	customhttp "github.com/BenjaminSRussell/go_go_go/internal/http"
	"github.com/BenjaminSRussell/go_go_go/internal/navigation"
	"github.com/BenjaminSRussell/go_go_go/internal/parser"
	"github.com/BenjaminSRussell/go_go_go/internal/persona"
	"github.com/BenjaminSRussell/go_go_go/internal/proxy"
	"github.com/BenjaminSRussell/go_go_go/internal/renderer"
	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// EnhancedCrawler extends Crawler with advanced features
type EnhancedCrawler struct {
	*Crawler

	// Advanced components
	personaPool    *persona.PersonaPool
	proxyManager   *proxy.EnhancedProxyManager
	navigator      *navigation.WeightedNavigator
	retryHandler   *customhttp.RetryHandler
	headerRotator  *customhttp.HeaderRotator
	chromeRenderer *renderer.ChromeRenderer
	sqliteStorage  *storage.SQLiteStorage

	// Feature flags from config
	enablePersonas    bool
	enableProxies     bool
	enableWeightedNav bool
	enableJSRendering bool
	enableSQLite      bool
}

// NewEnhanced creates an enhanced crawler with advanced features
func NewEnhanced(config types.Config) (*EnhancedCrawler, error) {
	// Create base crawler
	baseCrawler, err := New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base crawler: %w", err)
	}

	enhanced := &EnhancedCrawler{
		Crawler:           baseCrawler,
		enablePersonas:    config.EnablePersonas,
		enableProxies:     config.EnableProxies,
		enableWeightedNav: config.EnableWeightedNav,
		enableJSRendering: config.EnableJSRendering,
		enableSQLite:      config.EnableSQLite,
	}

	// Initialize persona pool if enabled
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

		enhanced.personaPool = persona.NewPersonaPool(maxPersonas, lifetime, reuseLimit)
		fmt.Printf("Persona system enabled: max=%d, lifetime=%v, reuse=%d\n",
			maxPersonas, lifetime, reuseLimit)
	}

	// Initialize enhanced proxy manager if enabled
	if config.EnableProxies {
		enhanced.proxyManager = proxy.NewEnhancedProxyManager(proxy.DefaultProxySources)

		// Start proxy manager with graceful degradation
		ctx := context.Background()
		if err := enhanced.proxyManager.Start(ctx); err != nil {
			fmt.Printf("Warning: failed to start proxy manager: %v\n", err)
			fmt.Println("Continuing without proxy rotation...")
			enhanced.enableProxies = false
			enhanced.proxyManager = nil
		} else {
			fmt.Println("Enhanced proxy manager started")
		}
	}

	// Initialize weighted navigator if enabled
	if config.EnableWeightedNav {
		enhanced.navigator = navigation.NewWeightedNavigator()
		fmt.Println("Weighted navigation enabled")
	}

	// Initialize retry handler (always enabled)
	retryConfig := customhttp.DefaultRetryConfig()
	if config.MaxRetries > 0 {
		retryConfig.MaxRetries = config.MaxRetries
	}
	enhanced.retryHandler = customhttp.NewRetryHandler(retryConfig)

	// Initialize header rotator (enabled by default)
	if config.UseHeaderRotation {
		enhanced.headerRotator = customhttp.NewHeaderRotator()
		fmt.Println("Header rotation enabled")
	}

	// Initialize Chrome renderer if enabled
	if config.EnableJSRendering {
		chromeRenderer, err := renderer.NewChromeRenderer()
		if err != nil {
			fmt.Printf("Warning: failed to initialize Chrome renderer: %v\n", err)
			fmt.Println("Continuing without JavaScript rendering...")
			enhanced.enableJSRendering = false
		} else {
			enhanced.chromeRenderer = chromeRenderer
			fmt.Println("JavaScript rendering enabled")
		}
	}

	// Initialize SQLite storage if enabled
	if config.EnableSQLite {
		dbPath := fmt.Sprintf("%s/crawl.db", config.DataDir)
		sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
		if err != nil {
			fmt.Printf("Warning: failed to initialize SQLite storage: %v\n", err)
			fmt.Println("Continuing with JSONL storage only...")
			enhanced.enableSQLite = false
		} else {
			enhanced.sqliteStorage = sqliteStorage
			fmt.Println("SQLite storage enabled")
		}
	}

	// Final validation before returning
	if err := ValidateEnhancedCrawler(enhanced); err != nil {
		SafeClose(enhanced)
		return nil, fmt.Errorf("crawler validation failed: %w", err)
	}

	return enhanced, nil
}

// processURLEnhanced processes a URL with advanced features
func (ec *EnhancedCrawler) processURLEnhanced(item types.URLItem) {
	defer ec.wg.Done()
	defer func() { <-ec.sem }()

	result := types.PageResult{
		URL:       item.URL,
		Depth:     item.Depth,
		CrawledAt: time.Now(),
	}

	// Get or create persona for this crawl
	var currentPersona *persona.Persona
	if ec.enablePersonas {
		parsedURL, err := url.Parse(item.URL)
		if err != nil {
			result.Error = fmt.Sprintf("invalid URL: %v", err)
			ec.saveResult(result)
			ec.errors.Add(1)
			return
		}

		currentPersona, err = ec.personaPool.GetOrCreatePersona(parsedURL.Host)
		if err != nil {
			result.Error = fmt.Sprintf("persona creation failed: %v", err)
			ec.saveResult(result)
			ec.errors.Add(1)
			return
		}

		// Get think time and wait (behavioral delay)
		thinkTime := currentPersona.GetThinkTime()
		time.Sleep(thinkTime)
	}

	// Lease proxy if persona + proxy enabled
	if ec.enablePersonas && ec.enableProxies && currentPersona != nil {
		leaseDuration := ec.config.ProxyLeaseDuration
		if leaseDuration <= 0 {
			leaseDuration = 15 * time.Minute
		}

		proxyLease, err := ec.proxyManager.LeaseProxy(currentPersona.ID, leaseDuration, "")
		if err == nil && proxyLease != nil {
			currentPersona.AssignProxy(proxyLease.URL)
		}
	}

	// Check robots.txt
	if !ec.config.IgnoreRobots && !ec.isAllowedByRobots(item.URL) {
		result.Error = "blocked by robots.txt"
		ec.saveResult(result)
		ec.errors.Add(1)
		return
	}

	// Retry logic with exponential backoff
	var resp *http.Response
	var err error
	maxRetries := ec.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	parsedURL, _ := url.Parse(item.URL)
	host := parsedURL.Host

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if host is in backoff
		if inBackoff, duration := ec.retryHandler.IsInBackoff(host); inBackoff {
			if attempt > 0 {
				time.Sleep(duration)
			}
		}

		// Make request
		resp, err = ec.makeRequest(item.URL, currentPersona)

		if err == nil && resp != nil {
			if resp.StatusCode == http.StatusOK {
				ec.retryHandler.RecordSuccess(host)
				break
			}

			// Check if should retry
			if ec.retryHandler.ShouldRetry(resp.StatusCode, nil) {
				ec.retryHandler.RecordFailure(host, resp.StatusCode)
				resp.Body.Close()

				if attempt < maxRetries {
					backoff := ec.retryHandler.GetBackoff(host, attempt)
					time.Sleep(backoff)
					continue
				}
			}

			// Non-retryable error
			break
		}

		// Network error
		if err != nil && ec.retryHandler.ShouldRetry(0, err) {
			ec.retryHandler.RecordFailure(host, 0)
			if attempt < maxRetries {
				backoff := ec.retryHandler.GetBackoff(host, attempt)
				time.Sleep(backoff)
				continue
			}
		}

		break
	}

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		ec.saveResult(result)
		ec.errors.Add(1)
		return
	}

	if resp == nil {
		result.Error = "nil response"
		ec.saveResult(result)
		ec.errors.Add(1)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("non-200 status: %d", resp.StatusCode)
		ec.saveResult(result)
		ec.errors.Add(1)
		return
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("body read failed: %v", err)
		ec.saveResult(result)
		ec.errors.Add(1)
		return
	}

	result.ContentLength = int64(len(body))
	htmlContent := string(body)

	// Check if JS rendering needed
	if ec.enableJSRendering && renderer.ShouldRender(htmlContent) {
		timeout := 30 * time.Second
		if currentPersona != nil {
			timeout = currentPersona.GetPageLoadTimeout()
		}

		renderedHTML, err := ec.chromeRenderer.Render(item.URL, timeout)
		if err == nil {
			htmlContent = renderedHTML
			result.ContentLength = int64(len(renderedHTML))
		}
	}

	// Extract links with weighted navigation if enabled
	var links []string
	if ec.enableWeightedNav {
		weightedLinks, err := ec.navigator.ExtractWeightedLinks(htmlContent, item.URL)
		if err == nil {
			// Filter to visible links
			visibleLinks := ec.navigator.FilterVisibleLinks(weightedLinks)

			// Select links based on persona's click bias
			for _, wLink := range visibleLinks {
				shouldFollow := true
				if currentPersona != nil {
					shouldFollow = currentPersona.ShouldFollowLink(wLink.Weight)
				}

				if shouldFollow && ec.shouldCrawl(wLink.URL, item.URL) {
					links = append(links, wLink.URL)
				}
			}
		}
	} else {
		// Standard link extraction
		links, result.Title = parser.ExtractLinks(htmlContent, item.URL)
	}

	result.LinkCount = len(links)

	// Add links to frontier
	for _, link := range links {
		if ec.shouldCrawl(link, item.URL) {
			ec.frontier.Add(types.URLItem{
				URL:       link,
				Depth:     item.Depth + 1,
				ParentURL: item.URL,
			})
			ec.discovered.Add(1)
		}
	}

	// Save result
	ec.saveResult(result)
	ec.frontier.MarkProcessed()
	ec.processed.Add(1)
}

// makeRequest creates and executes an HTTP request with persona/proxy support
func (ec *EnhancedCrawler) makeRequest(urlStr string, p *persona.Persona) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ec.ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Apply persona fingerprint if available
	if p != nil {
		p.ApplyToRequest(req)
	} else if ec.headerRotator != nil {
		// Fallback to header rotation
		ec.headerRotator.ApplyHeaders(req)
	}

	// Use persona's proxy if available
	client := ec.client
	if p != nil && p.ProxyURL != "" {
		proxyURL, err := url.Parse(p.ProxyURL)
		if err == nil {
			transport := &http.Transport{
				Proxy:               http.ProxyURL(proxyURL),
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			}
			client = &http.Client{
				Timeout:   ec.config.Timeout,
				Transport: transport,
			}
		}
	}

	return client.Do(req)
}

// saveResult saves a result to appropriate storage
func (ec *EnhancedCrawler) saveResult(result types.PageResult) {
	// Save to JSONL (always)
	ec.storage.SaveResult(result)

	// Save to SQLite if enabled
	if ec.enableSQLite && ec.sqliteStorage != nil {
		ec.sqliteStorage.SavePage(result)
	}
}

// Close cleans up resources
func (ec *EnhancedCrawler) Close() error {
	if ec.chromeRenderer != nil {
		ec.chromeRenderer.Close()
	}

	if ec.sqliteStorage != nil {
		ec.sqliteStorage.Close()
	}

	if ec.proxyManager != nil {
		ec.proxyManager.Stop()
	}

	return ec.storage.Close()
}
