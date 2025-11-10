package navigation

import (
	"testing"
)

func TestWeightedNavigatorNew(t *testing.T) {
	wn := NewWeightedNavigator()

	if wn == nil {
		t.Error("Expected WeightedNavigator to be created")
	}
}

func TestExtractWeightedLinks(t *testing.T) {
	wn := NewWeightedNavigator()

	html := `
	<html>
		<body>
			<a href="/page1">Important Link</a>
			<a href="/page2">Normal Link</a>
			<a href="/page3" style="display:none">Hidden Link</a>
		</body>
	</html>
	`

	links, err := wn.ExtractWeightedLinks(html, "https://example.com")

	if err != nil {
		t.Errorf("Failed to extract links: %v", err)
	}

	if len(links) == 0 {
		t.Error("Expected to extract weighted links")
	}
}

func TestFilterVisibleLinks(t *testing.T) {
	wn := NewWeightedNavigator()

	links := []Link{
		{URL: "https://example.com/page1", Weight: 0.9, IsVisible: true},
		{URL: "https://example.com/page2", Weight: 0.5, IsVisible: false},
		{URL: "https://example.com/page3", Weight: 0.8, IsVisible: true},
	}

	visible := wn.FilterVisibleLinks(links)

	if len(visible) != 2 {
		t.Errorf("Expected 2 visible links, got %d", len(visible))
	}

	for _, link := range visible {
		if !link.IsVisible {
			t.Error("Expected only visible links")
		}
	}
}

func TestExtractWeightedLinksInvalidHTML(t *testing.T) {
	wn := NewWeightedNavigator()

	links, err := wn.ExtractWeightedLinks("", "https://example.com")

	if err != nil {
		t.Errorf("Expected no error for empty HTML, got %v", err)
	}

	if len(links) != 0 {
		t.Errorf("Expected 0 links for empty HTML, got %d", len(links))
	}
}

func TestWeightedLinkStructure(t *testing.T) {
	link := Link{
		URL:        "https://example.com/page",
		Weight:     0.75,
		IsVisible:  true,
		AnchorText: "Click here",
	}

	if link.Weight < 0 || link.Weight > 1 {
		t.Error("Weight should be between 0 and 1")
	}

	if link.URL == "" {
		t.Error("URL should not be empty")
	}
}
