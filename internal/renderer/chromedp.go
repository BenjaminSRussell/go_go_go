package renderer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ChromeRenderer renders pages with headless Chrome
type ChromeRenderer struct {
	allocCtx  context.Context
	allocCancel context.CancelFunc
}

// NewChromeRenderer creates a new Chrome renderer
func NewChromeRenderer() (*ChromeRenderer, error) {
	// Create allocator context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	return &ChromeRenderer{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
	}, nil
}

// Render renders a URL and returns the final HTML
func (cr *ChromeRenderer) Render(url string, timeout time.Duration) (string, error) {
	ctx, cancel := chromedp.NewContext(cr.allocCtx)
	defer cancel()

	// Set timeout
	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	var htmlContent string

	// Navigate and wait for network idle
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		// Wait for network to be mostly idle (2 connections or less for 500ms)
		chromedp.ActionFunc(func(ctx context.Context) error {
			time.Sleep(2 * time.Second) // Simple wait for JS execution
			return nil
		}),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return "", fmt.Errorf("failed to render page: %w", err)
	}

	return htmlContent, nil
}

// ShouldRender determines if a page needs JS rendering
func ShouldRender(htmlContent string) bool {
	// Check if page is mostly empty or has JS framework indicators
	if len(htmlContent) < 500 {
		return true
	}

	// Check for common JS framework indicators
	jsIndicators := []string{
		"<div id=\"root\"></div>",
		"<div id=\"app\"></div>",
		"<noscript>You need to enable JavaScript",
		"JavaScript is required",
		"Please enable JavaScript",
		"__NEXT_DATA__",
		"ng-app",
		"v-app",
		"data-reactroot",
	}

	lowerContent := strings.ToLower(htmlContent)
	for _, indicator := range jsIndicators {
		if strings.Contains(lowerContent, strings.ToLower(indicator)) {
			return true
		}
	}

	return false
}

// Close closes the renderer
func (cr *ChromeRenderer) Close() {
	if cr.allocCancel != nil {
		cr.allocCancel()
	}
}
