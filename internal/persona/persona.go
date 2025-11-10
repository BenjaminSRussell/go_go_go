package persona

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	customhttp "github.com/BenjaminSRussell/go_go_go/internal/http"
	"golang.org/x/net/publicsuffix"
)

// Persona represents a consistent browsing identity
type Persona struct {
	ID           string
	Created      time.Time
	LastUsed     time.Time
	RequestCount int

	// Identity components
	TLSProfile    customhttp.TLSProfile
	HeaderProfile customhttp.BrowserProfile
	CookieJar     http.CookieJar
	ProxyURL      string

	// Behavioral traits
	AvgThinkTime time.Duration // Average time between actions
	Patience     time.Duration // How long to wait for pages
	ClickBias    float64       // Preference for visible links (0-1)

	mu sync.Mutex
}

// PersonaPool manages a pool of personas
type PersonaPool struct {
	personas map[string]*Persona
	mu       sync.RWMutex

	tlsFingerprinter *customhttp.TLSFingerprinter
	headerRotator    *customhttp.HeaderRotator

	// Configuration
	maxPersonas     int
	personaLifetime time.Duration
	reuseThreshold  int // Max requests per persona
}

// NewPersonaPool creates a new persona pool
func NewPersonaPool(maxPersonas int, lifetime time.Duration, reuseThreshold int) *PersonaPool {
	return &PersonaPool{
		personas:         make(map[string]*Persona),
		tlsFingerprinter: customhttp.NewTLSFingerprinter(),
		headerRotator:    customhttp.NewHeaderRotator(),
		maxPersonas:      maxPersonas,
		personaLifetime:  lifetime,
		reuseThreshold:   reuseThreshold,
	}
}

// GetOrCreatePersona retrieves an existing persona or creates a new one
func (pp *PersonaPool) GetOrCreatePersona(host string) (*Persona, error) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	// Try to find a suitable existing persona for this host
	for _, p := range pp.personas {
		p.mu.Lock()
		isExpired := time.Since(p.Created) > pp.personaLifetime
		isOverused := p.RequestCount >= pp.reuseThreshold
		p.mu.Unlock()

		if !isExpired && !isOverused {
			p.mu.Lock()
			p.LastUsed = time.Now()
			p.RequestCount++
			p.mu.Unlock()
			return p, nil
		}
	}

	// Clean up expired personas
	if len(pp.personas) >= pp.maxPersonas {
		pp.cleanupExpired()
	}

	// Create new persona
	persona, err := pp.createPersona()
	if err != nil {
		return nil, err
	}

	pp.personas[persona.ID] = persona
	return persona, nil
}

// createPersona creates a new persona with matching TLS/header profiles
func (pp *PersonaPool) createPersona() (*Persona, error) {
	// Generate unique ID
	id := generateID()

	// Get matching TLS and header profiles
	tlsProfile := pp.tlsFingerprinter.GetRandomProfile()
	headerProfile := pp.tlsFingerprinter.GetMatchingHeaderProfile(tlsProfile)

	// Create cookie jar
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Generate behavioral traits
	avgThinkTime := randomLogNormal(3.0, 0.8) // Mean 3s, sigma 0.8
	patience := randomLogNormal(15.0, 0.5)    // Mean 15s timeout
	clickBias := 0.7 + randomFloat()*0.3      // 0.7-1.0 (prefer visible links)

	persona := &Persona{
		ID:            id,
		Created:       time.Now(),
		LastUsed:      time.Now(),
		RequestCount:  1,
		TLSProfile:    tlsProfile,
		HeaderProfile: headerProfile,
		CookieJar:     jar,
		AvgThinkTime:  avgThinkTime,
		Patience:      patience,
		ClickBias:     clickBias,
	}

	return persona, nil
}

// AssignProxy assigns a proxy to this persona
func (p *Persona) AssignProxy(proxyURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ProxyURL = proxyURL
}

// GetThinkTime returns a realistic delay based on log-normal distribution
func (p *Persona) GetThinkTime() time.Duration {
	p.mu.Lock()
	avg := p.AvgThinkTime
	p.mu.Unlock()

	// Generate log-normal distributed delay around average
	// This models real human behavior better than uniform random
	seconds := float64(avg.Seconds())
	return randomLogNormal(seconds, 0.5)
}

// GetPageLoadTimeout returns how long this persona will wait for a page
func (p *Persona) GetPageLoadTimeout() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Patience
}

// ShouldFollowLink determines if this persona would follow a given link
// based on its position and visibility in the DOM
func (p *Persona) ShouldFollowLink(weight float64) bool {
	p.mu.Lock()
	bias := p.ClickBias
	p.mu.Unlock()

	// Weight is 0-1 based on link position (higher = more visible)
	// Bias is 0-1 (higher = prefer visible links)
	// Combine them: high weight + high bias = high probability

	threshold := bias * weight
	return randomFloat() < threshold
}

// ApplyToRequest applies this persona's fingerprint to an HTTP request
func (p *Persona) ApplyToRequest(req *http.Request) {
	p.mu.Lock()
	profile := p.HeaderProfile
	p.mu.Unlock()

	// Apply headers
	req.Header.Set("User-Agent", profile.UserAgent)
	req.Header.Set("Accept", profile.Accept)
	req.Header.Set("Accept-Language", profile.AcceptLanguage)
	req.Header.Set("Accept-Encoding", profile.AcceptEncoding)

	if profile.SecChUA != "" {
		req.Header.Set("Sec-Ch-Ua", profile.SecChUA)
	}
	if profile.SecChUAPlatform != "" {
		req.Header.Set("Sec-Ch-Ua-Platform", profile.SecChUAPlatform)
	}
	if profile.SecChUAMobile != "" {
		req.Header.Set("Sec-Ch-Ua-Mobile", profile.SecChUAMobile)
	}
	if profile.SecFetchSite != "" {
		req.Header.Set("Sec-Fetch-Site", profile.SecFetchSite)
	}
	if profile.SecFetchMode != "" {
		req.Header.Set("Sec-Fetch-Mode", profile.SecFetchMode)
	}
	if profile.SecFetchDest != "" {
		req.Header.Set("Sec-Fetch-Dest", profile.SecFetchDest)
	}
	if profile.UpgradeInsecure != "" {
		req.Header.Set("Upgrade-Insecure-Requests", profile.UpgradeInsecure)
	}
}

// GetStats returns statistics about this persona
func (p *Persona) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"id":            p.ID,
		"age":           time.Since(p.Created).String(),
		"request_count": p.RequestCount,
		"last_used":     time.Since(p.LastUsed).String(),
		"tls_profile":   p.TLSProfile.Name,
		"proxy":         p.ProxyURL != "",
	}
}

// cleanupExpired removes expired personas
func (pp *PersonaPool) cleanupExpired() {
	now := time.Now()
	for id, p := range pp.personas {
		p.mu.Lock()
		isExpired := now.Sub(p.Created) > pp.personaLifetime
		isOverused := p.RequestCount >= pp.reuseThreshold
		p.mu.Unlock()

		if isExpired || isOverused {
			delete(pp.personas, id)
		}
	}
}

// GetStats returns pool statistics
func (pp *PersonaPool) GetStats() map[string]interface{} {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	active := 0
	for _, p := range pp.personas {
		p.mu.Lock()
		if time.Since(p.LastUsed) < 5*time.Minute {
			active++
		}
		p.mu.Unlock()
	}

	return map[string]interface{}{
		"total_personas":  len(pp.personas),
		"active_personas": active,
		"max_personas":    pp.maxPersonas,
	}
}

// NewPersona creates a new persona with default values
func NewPersona() *Persona {
	id := generateID()

	avgThinkTime := randomLogNormal(3.0, 0.8)
	patience := randomLogNormal(15.0, 0.5)
	clickBias := 0.7 + randomFloat()*0.3

	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	fingerprinter := customhttp.NewTLSFingerprinter()
	tlsProfile := fingerprinter.GetRandomProfile()
	headerProfile := fingerprinter.GetMatchingHeaderProfile(tlsProfile)

	return &Persona{
		ID:            id,
		Created:       time.Now(),
		LastUsed:      time.Now(),
		RequestCount:  0,
		TLSProfile:    tlsProfile,
		HeaderProfile: headerProfile,
		CookieJar:     jar,
		AvgThinkTime:  avgThinkTime,
		Patience:      patience,
		ClickBias:     clickBias,
	}
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// randomLogNormal generates a log-normally distributed duration
// This models real human timing better than uniform distribution
func randomLogNormal(mean float64, sigma float64) time.Duration {
	// Using Box-Muller transform to generate normal distribution
	u1 := randomFloat()
	u2 := randomFloat()

	// Standard normal
	z := (-2.0 * logFloat(u1))
	z = sqrtFloat(z) * cosFloat(2.0*3.14159265359*u2)

	// Convert to log-normal
	logValue := logFloat(mean) + sigma*z
	value := expFloat(logValue)

	// Clamp to reasonable bounds (0.5s to 30s)
	if value < 0.5 {
		value = 0.5
	}
	if value > 30.0 {
		value = 30.0
	}

	return time.Duration(value * float64(time.Second))
}

func randomFloat() float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1<<53))
	return float64(n.Int64()) / float64(1<<53)
}

func logFloat(x float64) float64 {
	// Simple approximation of natural log
	if x <= 0 {
		return 0
	}
	return 0.693147 * (x - 1) // Rough approximation
}

func sqrtFloat(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

func cosFloat(x float64) float64 {
	// Taylor series approximation
	x = x - 6.283185307179586*float64(int(x/6.283185307179586))
	result := 1.0
	term := 1.0
	for i := 1; i <= 10; i++ {
		term *= -x * x / float64((2*i-1)*(2*i))
		result += term
	}
	return result
}

func expFloat(x float64) float64 {
	// Taylor series approximation
	result := 1.0
	term := 1.0
	for i := 1; i <= 20; i++ {
		term *= x / float64(i)
		result += term
	}
	return result
}
