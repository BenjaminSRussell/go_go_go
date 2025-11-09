package http

import (
	"math/rand"
	"net/http"
	"time"
)

// BrowserProfile represents a complete browser fingerprint
type BrowserProfile struct {
	UserAgent           string
	AcceptLanguage      string
	AcceptEncoding      string
	Accept              string
	SecChUA             string
	SecChUAPlatform     string
	SecChUAMobile       string
	SecFetchSite        string
	SecFetchMode        string
	SecFetchDest        string
	UpgradeInsecure     string
	CacheControl        string
	Pragma              string
}

var browserProfiles = []BrowserProfile{
	// Chrome on Windows
	{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		SecChUA:             `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		SecChUAPlatform:     `"Windows"`,
		SecChUAMobile:       "?0",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
		CacheControl:        "max-age=0",
	},
	// Chrome on macOS
	{
		UserAgent:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		SecChUA:             `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		SecChUAPlatform:     `"macOS"`,
		SecChUAMobile:       "?0",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
	},
	// Firefox on Windows
	{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:134.0) Gecko/20100101 Firefox/134.0",
		AcceptLanguage:      "en-US,en;q=0.5",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
		CacheControl:        "max-age=0",
	},
	// Firefox on macOS
	{
		UserAgent:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:134.0) Gecko/20100101 Firefox/134.0",
		AcceptLanguage:      "en-US,en;q=0.5",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
	},
	// Safari on macOS
	{
		UserAgent:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
	},
	// Edge on Windows
	{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
		SecChUA:             `"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		SecChUAPlatform:     `"Windows"`,
		SecChUAMobile:       "?0",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
	},
	// Chrome on Linux
	{
		UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		SecChUA:             `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		SecChUAPlatform:     `"Linux"`,
		SecChUAMobile:       "?0",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
	},
	// Chrome Mobile on Android
	{
		UserAgent:           "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		SecChUA:             `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
		SecChUAPlatform:     `"Android"`,
		SecChUAMobile:       "?1",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
		UpgradeInsecure:     "1",
	},
	// Safari on iOS
	{
		UserAgent:           "Mozilla/5.0 (iPhone; CPU iPhone OS 18_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Mobile/15E148 Safari/604.1",
		AcceptLanguage:      "en-US,en;q=0.9",
		AcceptEncoding:      "gzip, deflate, br",
		Accept:              "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		SecFetchSite:        "none",
		SecFetchMode:        "navigate",
		SecFetchDest:        "document",
	},
}

// HeaderRotator manages browser header rotation
type HeaderRotator struct {
	profiles []BrowserProfile
	rnd      *rand.Rand
}

// NewHeaderRotator creates a new header rotator
func NewHeaderRotator() *HeaderRotator {
	return &HeaderRotator{
		profiles: browserProfiles,
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetRandomProfile returns a random browser profile
func (hr *HeaderRotator) GetRandomProfile() BrowserProfile {
	return hr.profiles[hr.rnd.Intn(len(hr.profiles))]
}

// ApplyHeaders applies browser headers to an HTTP request
func (hr *HeaderRotator) ApplyHeaders(req *http.Request) {
	profile := hr.GetRandomProfile()

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
	if profile.CacheControl != "" {
		req.Header.Set("Cache-Control", profile.CacheControl)
	}
	if profile.Pragma != "" {
		req.Header.Set("Pragma", profile.Pragma)
	}

	// Add DNT header randomly
	if hr.rnd.Float32() < 0.3 {
		req.Header.Set("DNT", "1")
	}
}
