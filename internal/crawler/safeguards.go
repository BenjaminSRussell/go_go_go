package crawler

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// SafeProcessor wraps URL processing with panic recovery
type SafeProcessor struct {
	c          *Crawler
	panicCount atomic.Int64
}

// NewSafeProcessor creates a safe processor wrapper
func NewSafeProcessor(c *Crawler) *SafeProcessor {
	return &SafeProcessor{
		c: c,
	}
}

// ProcessURLSafely wraps processURL with panic recovery
func (sp *SafeProcessor) ProcessURLSafely(item types.URLItem) {
	defer func() {
		if r := recover(); r != nil {
			sp.panicCount.Add(1)

			fmt.Printf("\n[PANIC] URL: %s, Depth: %d\n", item.URL, item.Depth)
			fmt.Printf("[PANIC] Error: %v\n", r)
			fmt.Printf("[PANIC] Stack trace:\n%s\n", debug.Stack())

			result := types.PageResult{
				URL:   item.URL,
				Depth: item.Depth,
				Error: fmt.Sprintf("panic during processing: %v", r),
			}

			if sp.c != nil && sp.c.storage != nil {
				sp.c.storage.SaveResult(result)
			}

			if sp.c != nil {
				sp.c.errors.Add(1)
				sp.c.frontier.MarkProcessed()
			}
		}
	}()

	if sp.c == nil {
		fmt.Printf("[ERROR] SafeProcessor has nil crawler\n")
		return
	}

	sp.c.processURL(item)
}

// GetPanicCount returns total number of panics recovered
func (sp *SafeProcessor) GetPanicCount() int64 {
	return sp.panicCount.Load()
}
