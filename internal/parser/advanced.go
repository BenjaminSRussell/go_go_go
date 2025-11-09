package parser

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractData extracts structured data from HTML using CSS selectors
type ExtractedData struct {
	Links       []string
	Title       string
	MetaTags    map[string]string
	Images      []string
	Scripts     []string
	Stylesheets []string
	JSONData    []string // JSON-LD structured data
}

// ExtractLinks extracts all links and the title from an HTML document
func ExtractLinks(htmlContent, baseURL string) ([]string, string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, ""
	}

	links := make([]string, 0)
	visited := make(map[string]bool)
	title := doc.Find("title").First().Text()

	// Extract links from <a> tags
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if link := normalizeURL(href, baseURL); link != "" {
				if !visited[link] {
					links = append(links, link)
					visited[link] = true
				}
			}
		}
	})

	// Extract links from <link> tags (alternate, canonical)
	doc.Find("link[rel='alternate'], link[rel='canonical']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if link := normalizeURL(href, baseURL); link != "" {
				if !visited[link] {
					links = append(links, link)
					visited[link] = true
				}
			}
		}
	})

	return links, title
}

// ExtractStructuredData extracts comprehensive data from HTML
func ExtractStructuredData(htmlContent, baseURL string) (*ExtractedData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	data := &ExtractedData{
		Links:       make([]string, 0),
		MetaTags:    make(map[string]string),
		Images:      make([]string, 0),
		Scripts:     make([]string, 0),
		Stylesheets: make([]string, 0),
		JSONData:    make([]string, 0),
	}

	visited := make(map[string]bool)

	// Extract title
	data.Title = doc.Find("title").First().Text()

	// Extract meta tags
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, exists := s.Attr("name"); exists {
			if content, exists := s.Attr("content"); exists {
				data.MetaTags[name] = content
			}
		}
		if property, exists := s.Attr("property"); exists {
			if content, exists := s.Attr("content"); exists {
				data.MetaTags[property] = content
			}
		}
	})

	// Extract links
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if link := normalizeURL(href, baseURL); link != "" {
				if !visited[link] {
					data.Links = append(data.Links, link)
					visited[link] = true
				}
			}
		}
	})

	// Extract images
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if img := normalizeURL(src, baseURL); img != "" {
				data.Images = append(data.Images, img)
			}
		}
	})

	// Extract scripts
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if script := normalizeURL(src, baseURL); script != "" {
				data.Scripts = append(data.Scripts, script)
			}
		}
	})

	// Extract stylesheets
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if stylesheet := normalizeURL(href, baseURL); stylesheet != "" {
				data.Stylesheets = append(data.Stylesheets, stylesheet)
			}
		}
	})

	// Extract JSON-LD structured data
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		data.JSONData = append(data.JSONData, s.Text())
	})

	return data, nil
}

// ExtractBySelector extracts content using custom CSS selectors
func ExtractBySelector(htmlContent, selector string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	results := make([]string, 0)
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		results = append(results, strings.TrimSpace(s.Text()))
	})

	return results, nil
}

// normalizeURL converts relative URLs to absolute and cleans them
func normalizeURL(href, baseURL string) string {
	// Skip empty, javascript, mailto, tel, etc.
	if href == "" || strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") {
		return ""
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Parse href
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Resolve relative URL
	resolved := base.ResolveReference(u)

	// Remove fragment
	resolved.Fragment = ""

	// Normalize path
	resolvedStr := resolved.String()

	// Remove common tracking parameters
	resolvedStr = removeTrackingParams(resolvedStr)

	return resolvedStr
}

// removeTrackingParams removes common tracking parameters
func removeTrackingParams(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	trackingParams := map[string]bool{
		"utm_source":   true,
		"utm_medium":   true,
		"utm_campaign": true,
		"utm_term":     true,
		"utm_content":  true,
		"fbclid":       true,
		"gclid":        true,
		"msclkid":      true,
		"mc_cid":       true,
		"mc_eid":       true,
	}

	q := u.Query()
	for param := range trackingParams {
		q.Del(param)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// ExtractSitemapURLs extracts URLs from a sitemap XML
func ExtractSitemapURLs(xmlContent string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(xmlContent))
	if err != nil {
		return nil
	}

	urls := make([]string, 0)
	doc.Find("loc").Each(func(i int, s *goquery.Selection) {
		urls = append(urls, strings.TrimSpace(s.Text()))
	})

	return urls
}
