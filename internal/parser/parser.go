package parser

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractLinks extracts all links and the title from an HTML document
func ExtractLinks(htmlContent, baseURL string) ([]string, string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, ""
	}

	links := make([]string, 0)
	title := ""
	visited := make(map[string]bool)

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				// Extract href from anchor tags
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						if link := normalizeURL(attr.Val, baseURL); link != "" {
							if !visited[link] {
								links = append(links, link)
								visited[link] = true
							}
						}
					}
				}
			case "link":
				// Extract href from link tags (for alternate pages, etc.)
				var href, rel string
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						href = attr.Val
					}
					if attr.Key == "rel" {
						rel = attr.Val
					}
				}
				if rel == "alternate" || rel == "canonical" {
					if link := normalizeURL(href, baseURL); link != "" {
						if !visited[link] {
							links = append(links, link)
							visited[link] = true
						}
					}
				}
			case "title":
				// Extract title
				if n.FirstChild != nil {
					title = n.FirstChild.Data
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return links, title
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
	doc, err := html.Parse(strings.NewReader(xmlContent))
	if err != nil {
		return nil
	}

	urls := make([]string, 0)

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "loc" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				urls = append(urls, n.FirstChild.Data)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return urls
}
