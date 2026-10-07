package seeding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Test hooks (override in httptest tests).
var (
	httpClient = http.DefaultClient
	crtShURL   = func(domain string) string {
		return fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", domain)
	}
)

// DiscoverFromCertificateTransparency discovers subdomains from CT logs
func DiscoverFromCertificateTransparency(startURL string) ([]string, error) {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	domain := parsedURL.Host
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	resp, err := httpClient.Get(crtShURL(domain))
	if err != nil {
		return nil, fmt.Errorf("failed to query CT logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CT query returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CT response: %w", err)
	}

	var certs []struct {
		NameValue string `json:"name_value"`
	}
	if err := json.Unmarshal(body, &certs); err != nil {
		return nil, fmt.Errorf("failed to parse CT response: %w", err)
	}

	subdomains := make(map[string]bool)
	for _, cert := range certs {
		for _, name := range strings.Split(cert.NameValue, "\n") {
			name = strings.TrimSpace(name)
			if strings.HasPrefix(name, "*.") {
				name = name[2:]
			}
			if strings.HasSuffix(name, domain) {
				subdomains[name] = true
			}
		}
	}

	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = "https"
	}
	urls := make([]string, 0, len(subdomains))
	for subdomain := range subdomains {
		urls = append(urls, fmt.Sprintf("%s://%s", scheme, subdomain))
	}
	return urls, nil
}
