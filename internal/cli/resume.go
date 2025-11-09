package cli

import (
	"fmt"

	"github.com/BenjaminSRussell/go_go_go/internal/crawler"
	"github.com/spf13/cobra"
)

var resumeDataDir string

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a previous crawl",
	Long:  `Resume crawling from saved state`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := crawler.Resume(resumeDataDir)
		if err != nil {
			return fmt.Errorf("failed to resume crawler: %w", err)
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
