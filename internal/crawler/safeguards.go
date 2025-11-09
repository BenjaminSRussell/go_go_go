package crawler

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// SafeProcessor wraps URL processing with panic recovery
type SafeProcessor struct {
	ec         *EnhancedCrawler
	panicCount atomic.Int64
}

// NewSafeProcessor creates a safe processor wrapper
func NewSafeProcessor(ec *EnhancedCrawler) *SafeProcessor {
	return &SafeProcessor{
		ec: ec,
	}
}

// ProcessURLSafely wraps processURLEnhanced with panic recovery
func (sp *SafeProcessor) ProcessURLSafely(item types.URLItem) {
	defer func() {
		if r := recover(); r != nil {
			sp.panicCount.Add(1)

			// Log panic details
			fmt.Printf("\n[PANIC] URL: %s, Depth: %d\n", item.URL, item.Depth)
			fmt.Printf("[PANIC] Error: %v\n", r)
			fmt.Printf("[PANIC] Stack trace:\n%s\n", debug.Stack())

			// Create error result
			result := types.PageResult{
				URL:    item.URL,
				Depth:  item.Depth,
				Error:  fmt.Sprintf("panic during processing: %v", r),
			}

			// Try to save result (may also panic, but that's caught by outer handler)
			if sp.ec != nil && sp.ec.storage != nil {
				sp.ec.storage.SaveResult(result)
			}

			// Mark as error
			if sp.ec != nil {
				sp.ec.errors.Add(1)
				sp.ec.frontier.MarkProcessed()
			}
		}
	}()

	// Nil checks
	if sp.ec == nil {
		fmt.Printf("[ERROR] SafeProcessor has nil enhanced crawler\n")
		return
	}

	sp.ec.processURLEnhanced(item)
}

// GetPanicCount returns total number of panics recovered
func (sp *SafeProcessor) GetPanicCount() int64 {
	return sp.panicCount.Load()
}

// ValidateEnhancedCrawler performs comprehensive validation
func ValidateEnhancedCrawler(ec *EnhancedCrawler) error {
	if ec == nil {
		return fmt.Errorf("enhanced crawler is nil")
	}

	if ec.Crawler == nil {
		return fmt.Errorf("base crawler is nil")
	}

	if ec.frontier == nil {
		return fmt.Errorf("frontier is nil")
	}

	if ec.storage == nil {
		return fmt.Errorf("storage is nil")
	}

	if ec.client == nil {
		return fmt.Errorf("HTTP client is nil")
	}

	if ec.sem == nil {
		return fmt.Errorf("semaphore channel is nil")
	}

	if ec.ctx == nil {
		return fmt.Errorf("context is nil")
	}

	// Validate enabled features have their components
	if ec.enablePersonas && ec.personaPool == nil {
		return fmt.Errorf("personas enabled but personaPool is nil")
	}

	if ec.enableProxies && ec.proxyManager == nil {
		return fmt.Errorf("proxies enabled but proxyManager is nil")
	}

	if ec.enableWeightedNav && ec.navigator == nil {
		return fmt.Errorf("weighted nav enabled but navigator is nil")
	}

	if ec.enableJSRendering && ec.chromeRenderer == nil {
		return fmt.Errorf("JS rendering enabled but chromeRenderer is nil")
	}

	if ec.enableSQLite && ec.sqliteStorage == nil {
		return fmt.Errorf("SQLite enabled but sqliteStorage is nil")
	}

	return nil
}

// SafeClose safely closes resources with error handling
func SafeClose(ec *EnhancedCrawler) error {
	var lastErr error

	// Close Chrome renderer
	if ec.chromeRenderer != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("panic closing Chrome renderer: %v", r)
				}
			}()
			ec.chromeRenderer.Close()
		}()
	}

	// Close SQLite storage
	if ec.sqliteStorage != nil {
		if err := ec.sqliteStorage.Close(); err != nil {
			lastErr = fmt.Errorf("error closing SQLite: %w", err)
		}
	}

	// Stop proxy manager
	if ec.proxyManager != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("panic stopping proxy manager: %v", r)
				}
			}()
			ec.proxyManager.Stop()
		}()
	}

	// Close base storage
	if ec.storage != nil {
		if err := ec.storage.Close(); err != nil {
			lastErr = fmt.Errorf("error closing storage: %w", err)
		}
	}

	// Cancel context
	if ec.cancel != nil {
		ec.cancel()
	}

	return lastErr
}
