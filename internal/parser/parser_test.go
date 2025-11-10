package parser

import (
	"testing"
)

func TestExtractLinks(t *testing.T) {
	html := `
	<html>
		<head><title>Test Page</title></head>
		<body>
			<a href="https://example.com/page1">Link 1</a>
			<a href="/page2">Link 2</a>
			<a href="page3">Link 3</a>
		</body>
	</html>
	`

	links, title := ExtractLinks(html, "https://example.com")

	if title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got %s", title)
	}

	if len(links) == 0 {
		t.Error("Expected to extract links, got 0")
	}
}

func TestExtractLinksEmptyHTML(t *testing.T) {
	links, title := ExtractLinks("", "https://example.com")

	if len(links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(links))
	}

	if title != "" {
		t.Errorf("Expected empty title, got %s", title)
	}
}

func TestExtractLinksWithTitle(t *testing.T) {
	html := `<html><head><title>My Title</title></head></html>`

	_, title := ExtractLinks(html, "https://example.com")

	if title != "My Title" {
		t.Errorf("Expected title 'My Title', got %s", title)
	}
}

func TestExtractLinksNoDuplicates(t *testing.T) {
	html := `
	<html>
		<body>
			<a href="https://example.com/page">Link 1</a>
			<a href="https://example.com/page">Link 2</a>
		</body>
	</html>
	`

	links, _ := ExtractLinks(html, "https://example.com")

	if len(links) != 1 {
		t.Errorf("Expected 1 unique link, got %d", len(links))
	}
}

func TestExtractStructuredData(t *testing.T) {
	html := `
	<html>
		<head><title>Test</title></head>
		<body>
			<a href="https://example.com/page">Link</a>
			<img src="/image.png" />
			<script src="/script.js"></script>
			<link rel="stylesheet" href="/style.css" />
		</body>
	</html>
	`

	data, err := ExtractStructuredData(html, "https://example.com")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if data == nil {
		t.Error("Expected data to not be nil")
	}

	if data.Title != "Test" {
		t.Errorf("Expected title 'Test', got %s", data.Title)
	}
}

func TestExtractStructuredDataInvalidHTML(t *testing.T) {
	_, err := ExtractStructuredData("", "https://example.com")

	if err != nil {
		t.Errorf("Expected no error for empty HTML, got %v", err)
	}
}
