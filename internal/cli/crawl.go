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
		}

		c, err := crawler.New(config)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}

		results, err := c.Crawl()
		if err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}

		fmt.Printf("Crawl completed!\n")
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

	crawlCmd.MarkFlagRequired("start-url")
}
