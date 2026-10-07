package cli

import (
	"fmt"
	"path/filepath"

	"github.com/BenjaminSRussell/go_go_go/internal/crawler"
	"github.com/spf13/cobra"
)

var resumeDataDir string

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a previous crawl",
	Long:  `Resume crawling from saved state`,
	RunE: func(cmd *cobra.Command, args []string) error {
		absDir, err := filepath.Abs(resumeDataDir)
		if err != nil {
			absDir = resumeDataDir
		}
		fmt.Printf("Resuming crawl from data-dir: %s\n", absDir)

		c, err := crawler.Resume(resumeDataDir)
		if err != nil {
			return fmt.Errorf("failed to resume crawler from %s: %w", absDir, err)
		}

		frontier := 0
		if c != nil {
			frontier = c.FrontierSize()
		}
		fmt.Printf("Restored frontier size: %d\n", frontier)
		if frontier == 0 {
			return fmt.Errorf("resume data-dir %s has an empty frontier — nothing to crawl", absDir)
		}

		results, err := c.Crawl()
		if err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}

		fmt.Printf("Crawl resumed and completed!\n")
		fmt.Printf("Discovered: %d, Processed: %d, Errors: %d\n",
			results.Discovered, results.Processed, results.Errors)

		return nil
	},
}

func init() {
	resumeCmd.Flags().StringVar(&resumeDataDir, "data-dir", "./data", "Data storage directory")
}
