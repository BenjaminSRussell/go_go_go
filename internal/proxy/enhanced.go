package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// EnhancedProxy extends Proxy with geolocation and session tracking
type EnhancedProxy struct {
	*Proxy

	// Geolocation
	Country     string
	City        string
	ISP         string

	// Capabilities
	SupportsHTTPS  bool
	SupportsH2     bool
	AnonymityLevel string // transparent, anonymous, elite

	// Session affinity
	LeasedTo      string    // Persona ID
	LeaseExpires  time.Time

	mu sync.Mutex
}

// EnhancedProxyManager extends ProxyManager with advanced features
type EnhancedProxyManager struct {
	*ProxyManager

	enhancedProxies map[string]*EnhancedProxy
	leases          map[string]string // persona ID -> proxy URL

	mu sync.RWMutex
}

// NewEnhancedProxyManager creates an enhanced proxy manager
func NewEnhancedProxyManager(sources []string) *EnhancedProxyManager {
	return &EnhancedProxyManager{
		ProxyManager:    NewProxyManager(sources),
		enhancedProxies: make(map[string]*EnhancedProxy),
		leases:          make(map[string]string),
	}
}

// LeaseProxy assigns a proxy to a persona for a session
func (epm *EnhancedProxyManager) LeaseProxy(personaID string, duration time.Duration, country string) (*EnhancedProxy, error) {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	// Check if persona already has a lease
	if proxyURL, exists := epm.leases[personaID]; exists {
		if proxy, ok := epm.enhancedProxies[proxyURL]; ok {
			proxy.mu.Lock()
			// Extend lease if still valid
			if time.Now().Before(proxy.LeaseExpires) {
				proxy.LeaseExpires = time.Now().Add(duration)
				proxy.mu.Unlock()
				return proxy, nil
			}
			proxy.mu.Unlock()
		}
	}

	// Find best available proxy
	var bestProxy *EnhancedProxy
	var bestScore float64

	for _, proxy := range epm.enhancedProxies {
		proxy.mu.Lock()

		// Skip if leased to someone else
		if proxy.LeasedTo != "" && time.Now().Before(proxy.LeaseExpires) {
			proxy.mu.Unlock()
			continue
		}

		// Skip if too many failures
		if proxy.FailCount >= epm.maxFailCount {
			proxy.mu.Unlock()
			continue
		}

		// Skip if country doesn't match (if specified)
		if country != "" && proxy.Country != country {
			proxy.mu.Unlock()
			continue
		}

		// Calculate score (prefer low fail count, high success rate, low latency)
		score := float64(proxy.SuccessCount) / float64(proxy.FailCount+1)
		if proxy.AvgLatency > 0 {
			score = score / float64(proxy.AvgLatency.Seconds())
		}

		// Prefer elite anonymity
		if proxy.AnonymityLevel == "elite" {
			score *= 1.5
		}

		proxy.mu.Unlock()

		if bestProxy == nil || score > bestScore {
			bestProxy = proxy
			bestScore = score
		}
	}

	if bestProxy == nil {
		return nil, fmt.Errorf("no available proxies matching criteria")
	}

	// Lease the proxy
	bestProxy.mu.Lock()
	bestProxy.LeasedTo = personaID
	bestProxy.LeaseExpires = time.Now().Add(duration)
	bestProxy.mu.Unlock()

	epm.leases[personaID] = bestProxy.URL

	return bestProxy, nil
}

// ReleaseProxy releases a persona's proxy lease
func (epm *EnhancedProxyManager) ReleaseProxy(personaID string) {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	if proxyURL, exists := epm.leases[personaID]; exists {
		if proxy, ok := epm.enhancedProxies[proxyURL]; ok {
			proxy.mu.Lock()
			proxy.LeasedTo = ""
			proxy.LeaseExpires = time.Time{}
			proxy.mu.Unlock()
		}
		delete(epm.leases, personaID)
	}
}

// ValidateProxyEnhanced performs multi-tier validation
func (epm *EnhancedProxyManager) ValidateProxyEnhanced(ctx context.Context, proxyURL string) (*EnhancedProxy, error) {
	enhanced := &EnhancedProxy{
		Proxy: &Proxy{
			URL:         proxyURL,
			LastChecked: time.Now(),
		},
	}

	proxyParsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: epm.validationTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyParsed),
		},
	}

	// Tier 1: Connectivity test
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/ip", nil)
	if err != nil {
		return nil, fmt.Errorf("connectivity test failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connectivity test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("connectivity test returned status %d", resp.StatusCode)
	}

	enhanced.AvgLatency = time.Since(start)

	// Tier 2: Anonymity check
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ipResponse struct {
		Origin string `json:"origin"`
	}
	if err := json.Unmarshal(body, &ipResponse); err == nil {
		// Check if proxy IP is exposed (for anonymity level)
		// This is a simplified check - real implementation would be more sophisticated
		enhanced.AnonymityLevel = "anonymous"
	}

	// Tier 3: HTTPS capability test
	httpsReq, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/ip", nil)
	if err == nil {
		httpsResp, err := client.Do(httpsReq)
		if err == nil && httpsResp.StatusCode == http.StatusOK {
			enhanced.SupportsHTTPS = true
			httpsResp.Body.Close()
		}
	}

	// Tier 4: Geolocation lookup
	geoInfo, err := epm.lookupGeolocation(ipResponse.Origin)
	if err == nil {
		enhanced.Country = geoInfo.Country
		enhanced.City = geoInfo.City
		enhanced.ISP = geoInfo.ISP
	}

	// Only accept proxies with reasonable latency
	if enhanced.AvgLatency > 15*time.Second {
		return nil, fmt.Errorf("latency too high: %v", enhanced.AvgLatency)
	}

	return enhanced, nil
}

// GeoInfo holds geolocation information
type GeoInfo struct {
	Country string
	City    string
	ISP     string
}

// lookupGeolocation performs IP geolocation lookup
func (epm *EnhancedProxyManager) lookupGeolocation(ip string) (*GeoInfo, error) {
	// Use a free geolocation API (ip-api.com allows 45 req/min for free)
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geolocation lookup failed: %d", resp.StatusCode)
	}

	var result struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		City    string `json:"city"`
		ISP     string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("geolocation lookup unsuccessful")
	}

	return &GeoInfo{
		Country: result.Country,
		City:    result.City,
		ISP:     result.ISP,
	}, nil
}

// AddEnhancedProxy adds a validated proxy to the enhanced pool
func (epm *EnhancedProxyManager) AddEnhancedProxy(proxy *EnhancedProxy) {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	epm.enhancedProxies[proxy.URL] = proxy
	epm.proxies[proxy.URL] = proxy.Proxy
}

// GetProxyByCountry returns a proxy from a specific country
func (epm *EnhancedProxyManager) GetProxyByCountry(country string) (*EnhancedProxy, error) {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	for _, proxy := range epm.enhancedProxies {
		proxy.mu.Lock()

		// Check if available and matches country
		isAvailable := proxy.LeasedTo == "" || time.Now().After(proxy.LeaseExpires)
		matchesCountry := proxy.Country == country
		isHealthy := proxy.FailCount < epm.maxFailCount

		proxy.mu.Unlock()

		if isAvailable && matchesCountry && isHealthy {
			return proxy, nil
		}
	}

	return nil, fmt.Errorf("no available proxies from %s", country)
}

// GetEnhancedStats returns detailed statistics
func (epm *EnhancedProxyManager) GetEnhancedStats() map[string]interface{} {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	total := len(epm.enhancedProxies)
	available := 0
	leased := 0
	byCountry := make(map[string]int)
	byAnonymity := make(map[string]int)

	for _, proxy := range epm.enhancedProxies {
		proxy.mu.Lock()

		if proxy.LeasedTo == "" || time.Now().After(proxy.LeaseExpires) {
			available++
		} else {
			leased++
		}

		if proxy.Country != "" {
			byCountry[proxy.Country]++
		}

		if proxy.AnonymityLevel != "" {
			byAnonymity[proxy.AnonymityLevel]++
		}

		proxy.mu.Unlock()
	}

	return map[string]interface{}{
		"total_proxies":     total,
		"available_proxies": available,
		"leased_proxies":    leased,
		"active_leases":     len(epm.leases),
		"by_country":        byCountry,
		"by_anonymity":      byAnonymity,
	}
}

// CleanupExpiredLeases releases expired proxy leases
func (epm *EnhancedProxyManager) CleanupExpiredLeases() {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	now := time.Now()

	for personaID, proxyURL := range epm.leases {
		if proxy, ok := epm.enhancedProxies[proxyURL]; ok {
			proxy.mu.Lock()
			if now.After(proxy.LeaseExpires) {
				proxy.LeasedTo = ""
				proxy.LeaseExpires = time.Time{}
				delete(epm.leases, personaID)
			}
			proxy.mu.Unlock()
		}
	}
}
