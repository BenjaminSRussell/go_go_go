package crawler

import (
	"fmt"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// Crawl starts the crawling process for enhanced crawler
func (ec *EnhancedCrawler) Crawl() (*types.Results, error) {
	defer ec.storage.Close()
	defer ec.Close()

	// Save initial config
	if err := ec.storage.SaveConfig(ec.config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Starting enhanced crawl with %d workers\n", ec.config.Workers)
	fmt.Printf("Initial frontier size: %d URLs\n", ec.frontier.Size())

	// Print feature status
	ec.printFeatureStatus()

	// Create safe processor for panic recovery
	safeProcessor := NewSafeProcessor(ec)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Start progress reporter
	go ec.reportProgressEnhanced(ticker)

	// Start cleanup goroutine if proxy manager enabled
	if ec.enableProxies && ec.proxyManager != nil {
		go ec.periodicCleanup()
	}

	// Main crawl loop
	for !ec.shutdown.Load() {
		// Check if frontier is empty
		if ec.frontier.IsEmpty() {
			// Wait a bit for workers to finish
			time.Sleep(1 * time.Second)
			if ec.frontier.IsEmpty() {
				fmt.Println("\nFrontier exhausted, finishing...")
				break
			}
		}

		// Get next URL
		item, ok := ec.frontier.Next()
		if !ok {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Acquire semaphore
		ec.sem <- struct{}{}
		ec.wg.Add(1)

		// Process URL with panic recovery
		go safeProcessor.ProcessURLSafely(item)
	}

	// Wait for all workers to finish
	ec.wg.Wait()

	results := &types.Results{
		Discovered: int(ec.discovered.Load()),
		Processed:  int(ec.processed.Load()),
		Errors:     int(ec.errors.Load()),
	}

	// Log panic statistics if any occurred
	panicCount := safeProcessor.GetPanicCount()
	if panicCount > 0 {
		fmt.Printf("\n[WARNING] Total panics recovered: %d\n", panicCount)
	}

	// Print final stats
	ec.printFinalStats(results)

	return results, nil
}

// printFeatureStatus prints enabled features
func (ec *EnhancedCrawler) printFeatureStatus() {
	fmt.Println("\n=== Enabled Features ===")
	if ec.enablePersonas {
		fmt.Println("✓ Persona management")
		if ec.personaPool != nil {
			stats := ec.personaPool.GetStats()
			fmt.Printf("  Max personas: %v\n", stats["max_personas"])
		}
	}
	if ec.enableProxies {
		fmt.Println("✓ Proxy rotation")
	}
	if ec.enableWeightedNav {
		fmt.Println("✓ Weighted navigation")
	}
	if ec.enableJSRendering {
		fmt.Println("✓ JavaScript rendering")
	}
	if ec.enableSQLite {
		fmt.Println("✓ SQLite storage")
	}
	if ec.headerRotator != nil {
		fmt.Println("✓ Header rotation")
	}
	if ec.retryHandler != nil {
		fmt.Println("✓ Intelligent retry")
	}
	fmt.Println("========================")
}

// reportProgressEnhanced prints detailed progress with enhanced stats
func (ec *EnhancedCrawler) reportProgressEnhanced(ticker *time.Ticker) {
	for range ticker.C {
		discovered := ec.discovered.Load()
		processed := ec.processed.Load()
		errors := ec.errors.Load()
		pending := ec.frontier.Size()

		fmt.Printf("\r[Progress] Discovered: %d | Processed: %d | Errors: %d | Pending: %d",
			discovered, processed, errors, pending)

		// Print additional stats if features enabled
		if ec.enablePersonas && ec.personaPool != nil {
			stats := ec.personaPool.GetStats()
			fmt.Printf(" | Personas: %v/%v", stats["active_personas"], stats["total_personas"])
		}

		if ec.enableProxies && ec.proxyManager != nil {
			stats := ec.proxyManager.GetEnhancedStats()
			fmt.Printf(" | Proxies: %v avail", stats["available_proxies"])
		}
	}
}

// printFinalStats prints final crawl statistics
func (ec *EnhancedCrawler) printFinalStats(results *types.Results) {
	fmt.Println("\n\n=== Final Statistics ===")
	fmt.Printf("Total discovered: %d\n", results.Discovered)
	fmt.Printf("Total processed:  %d\n", results.Processed)
	fmt.Printf("Total errors:     %d\n", results.Errors)

	if results.Processed > 0 {
		successRate := float64(results.Processed-results.Errors) / float64(results.Processed) * 100
		fmt.Printf("Success rate:     %.1f%%\n", successRate)
	}

	// Persona stats
	if ec.enablePersonas && ec.personaPool != nil {
		fmt.Println("\n--- Persona Statistics ---")
		stats := ec.personaPool.GetStats()
		fmt.Printf("Total personas created: %v\n", stats["total_personas"])
		fmt.Printf("Active personas:        %v\n", stats["active_personas"])
	}

	// Proxy stats
	if ec.enableProxies && ec.proxyManager != nil {
		fmt.Println("\n--- Proxy Statistics ---")
		stats := ec.proxyManager.GetEnhancedStats()
		fmt.Printf("Total proxies:     %v\n", stats["total_proxies"])
		fmt.Printf("Available proxies: %v\n", stats["available_proxies"])
		fmt.Printf("Leased proxies:    %v\n", stats["leased_proxies"])
		if byCountry, ok := stats["by_country"].(map[string]int); ok && len(byCountry) > 0 {
			fmt.Println("By country:")
			for country, count := range byCountry {
				fmt.Printf("  %s: %d\n", country, count)
			}
		}
	}

	// SQLite stats
	if ec.enableSQLite && ec.sqliteStorage != nil {
		fmt.Println("\n--- Database Statistics ---")
		stats, err := ec.sqliteStorage.GetStats()
		if err == nil {
			fmt.Printf("Total pages in DB:       %v\n", stats["total_pages"])
			fmt.Printf("Successful pages in DB:  %v\n", stats["successful_pages"])
			fmt.Printf("Failed pages in DB:      %v\n", stats["failed_pages"])
		}
	}

	fmt.Println("========================")
}

// periodicCleanup runs periodic maintenance tasks
func (ec *EnhancedCrawler) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if ec.shutdown.Load() {
			return
		}

		// Cleanup expired proxy leases
		if ec.proxyManager != nil {
			ec.proxyManager.CleanupExpiredLeases()
		}
	}
}
