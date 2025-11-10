package cli

import (
	"fmt"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/crawler"
	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/spf13/cobra"
)

var (
	startURL        string
	workers         int
	timeout         int
	dataDir         string
	seedingStrategy string
	ignoreRobots    bool
	enableRedis     bool
	redisURL        string

	// Advanced features
	enableProxies     bool
	enableTLS         bool
	enableJSRendering bool
	enableSQLite      bool
	useHeaderRotation bool
	maxRetries        int

	// Persona & behavioral features
	enablePersonas     bool
	maxPersonas        int
	personaLifetime    int
	personaReuseLimit  int
	enableWeightedNav  bool
	proxyLeaseDuration int
)

var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "Start a new crawl",
	Long:  `Start crawling from a given URL with specified options`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := types.Config{
			StartURL:        startURL,
			Workers:         workers,
			Timeout:         time.Duration(timeout) * time.Second,
			DataDir:         dataDir,
			SeedingStrategy: seedingStrategy,
			IgnoreRobots:    ignoreRobots,
			EnableRedis:     enableRedis,
			RedisURL:        redisURL,

			// Advanced features
			EnableProxies:     enableProxies,
			EnableTLS:         enableTLS,
			EnableJSRendering: enableJSRendering,
			EnableSQLite:      enableSQLite,
			UseHeaderRotation: useHeaderRotation,
			MaxRetries:        maxRetries,

			// Persona & behavioral features
			EnablePersonas:     enablePersonas,
			MaxPersonas:        maxPersonas,
			PersonaLifetime:    time.Duration(personaLifetime) * time.Minute,
			PersonaReuseLimit:  personaReuseLimit,
			EnableWeightedNav:  enableWeightedNav,
			ProxyLeaseDuration: time.Duration(proxyLeaseDuration) * time.Minute,
		}

		c, err := crawler.NewFromConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}
		defer c.Close()

		results, err := c.Crawl()
		if err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}

		fmt.Printf("\nCrawl completed!\n")
		fmt.Printf("Discovered: %d, Processed: %d, Errors: %d\n",
			results.Discovered, results.Processed, results.Errors)

		return nil
	},
}

func init() {
	crawlCmd.Flags().StringVar(&startURL, "start-url", "", "Starting URL (required)")
	crawlCmd.Flags().IntVar(&workers, "workers", 256, "Number of concurrent workers")
	crawlCmd.Flags().IntVar(&timeout, "timeout", 20, "Request timeout in seconds")
	crawlCmd.Flags().StringVar(&dataDir, "data-dir", "./data", "Data storage directory")
	crawlCmd.Flags().StringVar(&seedingStrategy, "seeding-strategy", "all", "Seeding strategy: none/sitemap/ct/commoncrawl/all")
	crawlCmd.Flags().BoolVar(&ignoreRobots, "ignore-robots", false, "Ignore robots.txt")
	crawlCmd.Flags().BoolVar(&enableRedis, "enable-redis", false, "Enable distributed crawling with Redis")
	crawlCmd.Flags().StringVar(&redisURL, "redis-url", "", "Redis connection URL")

	// Advanced features
	crawlCmd.Flags().BoolVar(&enableProxies, "enable-proxies", false, "Enable proxy rotation (uses free proxy lists)")
	crawlCmd.Flags().BoolVar(&enableTLS, "enable-tls-fingerprint", false, "Enable TLS fingerprinting to mimic real browsers")
	crawlCmd.Flags().BoolVar(&enableJSRendering, "enable-js-rendering", false, "Enable JavaScript rendering with headless Chrome")
	crawlCmd.Flags().BoolVar(&enableSQLite, "enable-sqlite", false, "Use SQLite for queryable storage instead of JSONL")
	crawlCmd.Flags().BoolVar(&useHeaderRotation, "use-header-rotation", true, "Rotate browser headers")
	crawlCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Maximum retry attempts per URL")

	// Persona & behavioral features
	crawlCmd.Flags().BoolVar(&enablePersonas, "enable-personas", false, "Enable persona-based crawling with session persistence")
	crawlCmd.Flags().IntVar(&maxPersonas, "max-personas", 50, "Maximum number of concurrent personas")
	crawlCmd.Flags().IntVar(&personaLifetime, "persona-lifetime", 30, "Persona lifetime in minutes")
	crawlCmd.Flags().IntVar(&personaReuseLimit, "persona-reuse-limit", 100, "Maximum requests per persona")
	crawlCmd.Flags().BoolVar(&enableWeightedNav, "enable-weighted-nav", false, "Use weighted navigation (prefer visible/important links)")
	crawlCmd.Flags().IntVar(&proxyLeaseDuration, "proxy-lease-duration", 15, "Proxy lease duration in minutes (session affinity)")

	crawlCmd.MarkFlagRequired("start-url")
}
