package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gogogoscraper",
	Short: "A high-performance web scraper written in Go",
	Long:  `Go Go Go Scraper - A concurrent web crawler optimized for URL discovery and sitemap generation`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(crawlCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(exportCmd)
}
