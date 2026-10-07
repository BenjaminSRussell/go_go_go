package renderer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// ChromeRenderer renders pages with headless Chrome. Concurrent tabs are
// capped by Config.MaxTabs and the browser is shut down after
// Config.IdleTimeout without renders, then relaunched on demand.
type ChromeRenderer struct {
	config Config
	pool   *tabPool
}

// NewChromeRenderer creates a renderer with DefaultConfig.
func NewChromeRenderer() (*ChromeRenderer, error) {
	return NewChromeRendererWithConfig(DefaultConfig())
}

// NewChromeRendererWithConfig creates a renderer with explicit pool settings.
func NewChromeRendererWithConfig(cfg Config) (*ChromeRenderer, error) {
	cfg = cfg.normalized()
	opts := allocatorOptions(cfg)
	start := func() (context.Context, context.CancelFunc) {
		return chromedp.NewExecAllocator(context.Background(), opts...)
	}
	return &ChromeRenderer{config: cfg, pool: newTabPool(cfg.MaxTabs, cfg.IdleTimeout, start)}, nil
}

func allocatorOptions(cfg Config) []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(defaultUserAgent),
	)
	if cfg.DisableGPU {
		opts = append(opts, chromedp.Flag("disable-gpu", true))
	}
	for _, raw := range cfg.ExtraFlags {
		name, value := ParseChromeFlag(raw)
		if name == "" {
			continue
		}
		opts = append(opts, chromedp.Flag(name, value))
	}
	return opts
}

// ParseChromeFlag turns "--name=value", "name=value" or "name" into a
// chromedp flag name and value (bool true when no value is given).
func ParseChromeFlag(raw string) (string, interface{}) {
	raw = strings.TrimLeft(strings.TrimSpace(raw), "-")
	if raw == "" {
		return "", nil
	}
	name, value, hasValue := strings.Cut(raw, "=")
	if !hasValue {
		return name, true
	}
	switch strings.ToLower(value) {
	case "true":
		return name, true
	case "false":
		return name, false
	}
	return name, value
}

// Config returns the effective configuration.
func (cr *ChromeRenderer) Config() Config { return cr.config }

// Stats returns render counters for metrics/logging.
func (cr *ChromeRenderer) Stats() Stats { return cr.pool.stats() }

// Render renders a URL and returns the final HTML
func (cr *ChromeRenderer) Render(url string, timeout time.Duration) (html string, err error) {
	waitCtx, waitCancel := context.WithTimeout(context.Background(), timeout)
	defer waitCancel()

	browser, release, err := cr.pool.acquire(waitCtx)
	if err != nil {
		return "", fmt.Errorf("render pool: %w", err)
	}
	defer func() { release(err) }()

	ctx, cancel := chromedp.NewContext(browser)
	defer cancel() // closes the tab

	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	var htmlContent string
	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			select {
			case <-time.After(2 * time.Second): // simple wait for JS execution
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
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

// Close shuts down Chrome and rejects further renders.
func (cr *ChromeRenderer) Close() {
	if cr.pool != nil {
		cr.pool.close()
	}
}
