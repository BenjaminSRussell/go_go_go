package crawler

import (
	"fmt"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

// CrawlerInterface defines the common crawler interface
type CrawlerInterface interface {
	Crawl() (*types.Results, error)
	Close() error
}

// NewFromConfig creates appropriate crawler based on configuration
func NewFromConfig(config types.Config) (CrawlerInterface, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if advanced features are enabled
	hasAdvancedFeatures := config.EnablePersonas ||
		config.EnableProxies ||
		config.EnableWeightedNav ||
		config.EnableJSRendering ||
		config.EnableSQLite

	if hasAdvancedFeatures {
		fmt.Println("Creating enhanced crawler with advanced features...")
		return NewEnhanced(config)
	}

	fmt.Println("Creating standard crawler...")
	return New(config)
}

// validateConfig validates crawler configuration
func validateConfig(config types.Config) error {
	if config.StartURL == "" {
		return fmt.Errorf("start URL is required")
	}

	if config.Workers <= 0 {
		return fmt.Errorf("workers must be positive, got %d", config.Workers)
	}

	if config.Workers > 1000 {
		return fmt.Errorf("workers too high (max 1000), got %d", config.Workers)
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", config.Timeout)
	}

	if config.DataDir == "" {
		return fmt.Errorf("data directory is required")
	}

	// Validate persona settings if enabled
	if config.EnablePersonas {
		if config.MaxPersonas <= 0 {
			config.MaxPersonas = 50
		}
		if config.MaxPersonas > 1000 {
			return fmt.Errorf("max personas too high (max 1000), got %d", config.MaxPersonas)
		}
	}

	// Validate retry settings
	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative, got %d", config.MaxRetries)
	}

	if config.MaxRetries > 10 {
		return fmt.Errorf("max retries too high (max 10), got %d", config.MaxRetries)
	}

	return nil
}
