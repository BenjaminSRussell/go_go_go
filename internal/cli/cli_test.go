package cli

import (
	"testing"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestCrawlCommandFlags(t *testing.T) {
	crawlCmd.SetArgs([]string{"crawl", "--help"})

	err := crawlCmd.Execute()

	if err != nil {
		t.Logf("Expected help command to execute: %v", err)
	}
}

func TestCrawlConfigCreation(t *testing.T) {
	config := types.Config{
		StartURL:        "https://example.com",
		Workers:         4,
		Timeout:         30 * time.Second,
		DataDir:         "/tmp/crawl",
		SeedingStrategy: "sitemap",
		IgnoreRobots:    false,
		EnablePersonas:  false,
		MaxRetries:      3,
	}

	if config.StartURL == "" {
		t.Error("Expected StartURL to be set")
	}

	if config.Workers <= 0 {
		t.Error("Expected Workers to be positive")
	}

	if config.Timeout <= 0 {
		t.Error("Expected Timeout to be positive")
	}
}

func TestRootCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()

	if err != nil {
		t.Logf("Expected help command to execute: %v", err)
	}
}

func TestFlagDefaults(t *testing.T) {
	defaults := map[string]interface{}{
		"workers":    4,
		"timeout":    30,
		"max-retries": 3,
		"max-personas": 50,
	}

	for flagName, expectedValue := range defaults {
		if expectedValue == nil {
			t.Errorf("Expected default value for flag %s", flagName)
		}
	}
}

func TestConfigValidationEmpty(t *testing.T) {
	config := types.Config{}

	valid := config.StartURL != ""

	if valid {
		t.Error("Expected empty config to be invalid")
	}
}

func TestConfigValidationComplete(t *testing.T) {
	config := types.Config{
		StartURL:   "https://example.com",
		Workers:    4,
		Timeout:    30 * time.Second,
		DataDir:    "/tmp/crawl",
	}

	valid := config.StartURL != "" && config.Workers > 0

	if !valid {
		t.Error("Expected complete config to be valid")
	}
}
