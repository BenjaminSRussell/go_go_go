package seeding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DiscoverFromCertificateTransparency discovers subdomains from CT logs
func DiscoverFromCertificateTransparency(startURL string) ([]string, error) {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	domain := parsedURL.Host
	// Remove port if present
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Query crt.sh for certificates
	ctURL := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", domain)

	resp, err := http.Get(ctURL)
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

	// Extract unique subdomains
	subdomains := make(map[string]bool)
	for _, cert := range certs {
		names := strings.Split(cert.NameValue, "\n")
		for _, name := range names {
			name = strings.TrimSpace(name)
			// Skip wildcards
			if strings.HasPrefix(name, "*.") {
				name = name[2:]
			}
			// Must end with our domain
			if strings.HasSuffix(name, domain) {
				subdomains[name] = true
			}
		}
	}

	// Convert to URLs
	urls := make([]string, 0, len(subdomains))
	for subdomain := range subdomains {
		urls = append(urls, fmt.Sprintf("%s://%s", parsedURL.Scheme, subdomain))
	}

	return urls, nil
}
