package navigation

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Link represents a weighted link in the document
type Link struct {
	URL          string
	AnchorText   string
	Weight       float64 // 0-1, higher = more likely to be clicked
	Position     Position
	InNavigation bool
	IsVisible    bool
}

// Position represents the DOM position of a link
type Position struct {
	IsAboveFold bool
	InNav       bool
	InFooter    bool
	InMain      bool
	Depth       int // DOM depth from body
}

// WeightedNavigator selects links based on human-like preferences
type WeightedNavigator struct {
	viewportHeight int
	viewportWidth  int
}

// NewWeightedNavigator creates a new weighted navigator
func NewWeightedNavigator() *WeightedNavigator {
	return &WeightedNavigator{
		viewportHeight: 1080,
		viewportWidth:  1920,
	}
}

// ExtractWeightedLinks extracts links with weights based on position and visibility
func (wn *WeightedNavigator) ExtractWeightedLinks(htmlContent, baseURL string) ([]Link, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	links := make([]Link, 0)

	// Calculate weights for each link
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		anchorText := strings.TrimSpace(s.Text())
		position := wn.analyzePosition(s)
		weight := wn.calculateWeight(s, position)

		link := Link{
			URL:          href,
			AnchorText:   anchorText,
			Weight:       weight,
			Position:     position,
			InNavigation: position.InNav,
			IsVisible:    wn.isVisible(s),
		}

		links = append(links, link)
	})

	return links, nil
}

// analyzePosition determines the structural position of a link
func (wn *WeightedNavigator) analyzePosition(s *goquery.Selection) Position {
	position := Position{}

	// Check parent elements
	parents := s.Parents()

	// Check if in navigation
	parents.Each(func(i int, parent *goquery.Selection) {
		tag := goquery.NodeName(parent)
		class, _ := parent.Attr("class")
		id, _ := parent.Attr("id")

		classLower := strings.ToLower(class)
		idLower := strings.ToLower(id)

		if tag == "nav" || strings.Contains(classLower, "nav") || strings.Contains(idLower, "nav") {
			position.InNav = true
		}

		if tag == "footer" || strings.Contains(classLower, "footer") || strings.Contains(idLower, "footer") {
			position.InFooter = true
		}

		if tag == "main" || tag == "article" || strings.Contains(classLower, "main") || strings.Contains(classLower, "content") {
			position.InMain = true
		}
	})

	// Calculate DOM depth
	position.Depth = parents.Length()

	// Check if above fold (simplified - in real browser we'd use coordinates)
	// We use a heuristic based on DOM position
	position.IsAboveFold = position.Depth < 10

	return position
}

// calculateWeight assigns a weight to a link based on various factors
func (wn *WeightedNavigator) calculateWeight(s *goquery.Selection, pos Position) float64 {
	weight := 0.5 // Base weight

	// Navigation links are highly weighted
	if pos.InNav {
		weight += 0.3
	}

	// Main content links are weighted
	if pos.InMain {
		weight += 0.2
	}

	// Footer links are less weighted
	if pos.InFooter {
		weight -= 0.3
	}

	// Above fold increases weight
	if pos.IsAboveFold {
		weight += 0.2
	}

	// Check visual prominence
	class, _ := s.Attr("class")
	classLower := strings.ToLower(class)

	// Buttons and CTAs are highly weighted
	if strings.Contains(classLower, "button") || strings.Contains(classLower, "btn") ||
		strings.Contains(classLower, "cta") {
		weight += 0.2
	}

	// Links with images are more visible
	if s.Find("img").Length() > 0 {
		weight += 0.1
	}

	// Check anchor text quality
	anchorText := strings.TrimSpace(s.Text())
	if len(anchorText) > 3 && len(anchorText) < 100 {
		weight += 0.1
	}

	// Check for common navigation patterns
	anchorLower := strings.ToLower(anchorText)
	if strings.Contains(anchorLower, "home") || strings.Contains(anchorLower, "about") ||
		strings.Contains(anchorLower, "contact") || strings.Contains(anchorLower, "products") {
		weight += 0.15
	}

	// Normalize weight to 0-1
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}

	return weight
}

// isVisible checks if a link is likely visible (simplified heuristic)
func (wn *WeightedNavigator) isVisible(s *goquery.Selection) bool {
	// Check for display: none
	style, exists := s.Attr("style")
	if exists && strings.Contains(strings.ToLower(style), "display:none") {
		return false
	}

	// Check for hidden class
	class, exists := s.Attr("class")
	if exists && strings.Contains(strings.ToLower(class), "hidden") {
		return false
	}

	// Check if has text or image
	text := strings.TrimSpace(s.Text())
	hasImage := s.Find("img").Length() > 0

	return len(text) > 0 || hasImage
}

// SelectWeightedLink selects a link based on weighted probability
func (wn *WeightedNavigator) SelectWeightedLink(links []Link, clickBias float64) *Link {
	if len(links) == 0 {
		return nil
	}

	// Apply click bias to weights
	adjustedWeights := make([]float64, len(links))
	totalWeight := 0.0

	for i, link := range links {
		// Bias increases the weight of high-weight links
		adjustedWeight := math.Pow(link.Weight, 1.0/clickBias)
		adjustedWeights[i] = adjustedWeight
		totalWeight += adjustedWeight
	}

	// Normalize weights
	if totalWeight == 0 {
		return &links[0]
	}

	for i := range adjustedWeights {
		adjustedWeights[i] /= totalWeight
	}

	// Select link using weighted random selection
	r := randomFloat()
	cumulative := 0.0

	for i, weight := range adjustedWeights {
		cumulative += weight
		if r <= cumulative {
			return &links[i]
		}
	}

	// Fallback
	return &links[len(links)-1]
}

// FilterVisibleLinks filters links to only visible ones
func (wn *WeightedNavigator) FilterVisibleLinks(links []Link) []Link {
	visible := make([]Link, 0)
	for _, link := range links {
		if link.IsVisible && link.Weight > 0.1 {
			visible = append(visible, link)
		}
	}
	return visible
}

// GetTopLinks returns the N highest weighted links
func (wn *WeightedNavigator) GetTopLinks(links []Link, n int) []Link {
	if len(links) <= n {
		return links
	}

	// Sort by weight (simple bubble sort for small N)
	sorted := make([]Link, len(links))
	copy(sorted, links)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Weight < sorted[j+1].Weight {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted[:n]
}

// GetLinkStats returns statistics about link distribution
func (wn *WeightedNavigator) GetLinkStats(links []Link) map[string]interface{} {
	totalLinks := len(links)
	visibleLinks := 0
	navLinks := 0
	aboveFoldLinks := 0

	totalWeight := 0.0
	maxWeight := 0.0
	minWeight := 1.0

	for _, link := range links {
		if link.IsVisible {
			visibleLinks++
		}
		if link.InNavigation {
			navLinks++
		}
		if link.Position.IsAboveFold {
			aboveFoldLinks++
		}

		totalWeight += link.Weight
		if link.Weight > maxWeight {
			maxWeight = link.Weight
		}
		if link.Weight < minWeight {
			minWeight = link.Weight
		}
	}

	avgWeight := 0.0
	if totalLinks > 0 {
		avgWeight = totalWeight / float64(totalLinks)
	}

	return map[string]interface{}{
		"total_links":      totalLinks,
		"visible_links":    visibleLinks,
		"navigation_links": navLinks,
		"above_fold_links": aboveFoldLinks,
		"avg_weight":       avgWeight,
		"max_weight":       maxWeight,
		"min_weight":       minWeight,
	}
}

func randomFloat() float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1<<53))
	return float64(n.Int64()) / float64(1<<53)
}
