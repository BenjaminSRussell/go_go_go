package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Proxy represents a proxy server
type Proxy struct {
	URL          string
	LastChecked  time.Time
	FailCount    int
	SuccessCount int
	AvgLatency   time.Duration
}

// ProxyManager manages a pool of proxies
type ProxyManager struct {
	mu sync.RWMutex

	proxies map[string]*Proxy
	sources []string

	// Configuration
	checkInterval     time.Duration
	maxFailCount      int
	validationURL     string
	validationTimeout time.Duration

	// Channels
	proxyQueue chan string
	stopChan   chan struct{}
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(sources []string) *ProxyManager {
	pm := &ProxyManager{
		proxies:           make(map[string]*Proxy),
		sources:           sources,
		checkInterval:     10 * time.Minute,
		maxFailCount:      3,
		validationURL:     "https://api.ipify.org",
		validationTimeout: 10 * time.Second,
		proxyQueue:        make(chan string, 1000),
		stopChan:          make(chan struct{}),
	}

	return pm
}

// Start starts the proxy manager
func (pm *ProxyManager) Start(ctx context.Context) error {
	// Start proxy scraper
	go pm.scraperLoop(ctx)

	// Start proxy validator
	for i := 0; i < 5; i++ {
		go pm.validatorLoop(ctx)
	}

	// Start periodic revalidation
	go pm.revalidationLoop(ctx)

	return nil
}

// Stop stops the proxy manager
func (pm *ProxyManager) Stop() {
	close(pm.stopChan)
}

// GetProxy returns a working proxy from the pool
func (pm *ProxyManager) GetProxy() (*Proxy, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Find best proxy (lowest fail count, highest success rate)
	var bestProxy *Proxy
	var bestScore float64

	for _, proxy := range pm.proxies {
		if proxy.FailCount >= pm.maxFailCount {
			continue
		}

		// Calculate score (higher is better)
		score := float64(proxy.SuccessCount) / float64(proxy.FailCount+1)
		if bestProxy == nil || score > bestScore {
			bestProxy = proxy
			bestScore = score
		}
	}

	if bestProxy == nil {
		return nil, fmt.Errorf("no working proxies available")
	}

	return bestProxy, nil
}

// RecordSuccess records a successful proxy usage
func (pm *ProxyManager) RecordSuccess(proxyURL string, latency time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proxy, exists := pm.proxies[proxyURL]; exists {
		proxy.SuccessCount++
		// Update average latency
		proxy.AvgLatency = (proxy.AvgLatency + latency) / 2
	}
}

// RecordFailure records a failed proxy usage
func (pm *ProxyManager) RecordFailure(proxyURL string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if proxy, exists := pm.proxies[proxyURL]; exists {
		proxy.FailCount++
		if proxy.FailCount >= pm.maxFailCount {
			delete(pm.proxies, proxyURL)
		}
	}
}

// scraperLoop periodically scrapes proxy sources
func (pm *ProxyManager) scraperLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.checkInterval)
	defer ticker.Stop()

	// Initial scrape
	pm.scrapeProxies(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.scrapeProxies(ctx)
		}
	}
}

// scrapeProxies scrapes all proxy sources
func (pm *ProxyManager) scrapeProxies(ctx context.Context) {
	for _, source := range pm.sources {
		go pm.scrapeSource(ctx, source)
	}
}

// scrapeSource scrapes a single proxy source
func (pm *ProxyManager) scrapeSource(ctx context.Context, source string) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse proxy URL
		proxyURL := pm.parseProxyLine(line)
		if proxyURL != "" {
			select {
			case pm.proxyQueue <- proxyURL:
			default:
				// Queue full, skip
			}
		}
	}
}

// parseProxyLine parses a proxy line into a URL
func (pm *ProxyManager) parseProxyLine(line string) string {
	// Support formats:
	// - ip:port
	// - http://ip:port
	// - socks5://ip:port

	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "socks5://") {
		return line
	}

	// Assume HTTP proxy if no scheme
	parts := strings.Split(line, ":")
	if len(parts) == 2 {
		return fmt.Sprintf("http://%s", line)
	}

	return ""
}

// validatorLoop validates proxies from the queue
func (pm *ProxyManager) validatorLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case proxyURL := <-pm.proxyQueue:
			if pm.validateProxy(ctx, proxyURL) {
				pm.addProxy(proxyURL)
			}
		}
	}
}

// validateProxy tests if a proxy is working
func (pm *ProxyManager) validateProxy(ctx context.Context, proxyURL string) bool {
	proxyParsed, err := url.Parse(proxyURL)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: pm.validationTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyParsed),
		},
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", pm.validationURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Check if response is valid
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return false
	}

	latency := time.Since(start)

	// Only accept proxies with reasonable latency
	if latency > 15*time.Second {
		return false
	}

	return true
}

// addProxy adds a validated proxy to the pool
func (pm *ProxyManager) addProxy(proxyURL string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.proxies[proxyURL]; !exists {
		pm.proxies[proxyURL] = &Proxy{
			URL:         proxyURL,
			LastChecked: time.Now(),
		}
	}
}

// revalidationLoop periodically revalidates proxies
func (pm *ProxyManager) revalidationLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.revalidateProxies(ctx)
		}
	}
}

// revalidateProxies revalidates all proxies in the pool
func (pm *ProxyManager) revalidateProxies(ctx context.Context) {
	pm.mu.RLock()
	proxies := make([]string, 0, len(pm.proxies))
	for url := range pm.proxies {
		proxies = append(proxies, url)
	}
	pm.mu.RUnlock()

	for _, proxyURL := range proxies {
		if !pm.validateProxy(ctx, proxyURL) {
			pm.mu.Lock()
			delete(pm.proxies, proxyURL)
			pm.mu.Unlock()
		}
	}
}

// GetStats returns proxy pool statistics
func (pm *ProxyManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	working := 0
	for _, proxy := range pm.proxies {
		if proxy.FailCount < pm.maxFailCount {
			working++
		}
	}

	return map[string]interface{}{
		"total_proxies":   len(pm.proxies),
		"working_proxies": working,
	}
}

// Default proxy sources (free proxy lists)
var DefaultProxySources = []string{
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
	"https://raw.githubusercontent.com/clarketm/proxy-list/master/proxy-list-raw.txt",
	"https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/http.txt",
	"https://raw.githubusercontent.com/sunny9577/proxy-scraper/master/proxies.txt",
}
