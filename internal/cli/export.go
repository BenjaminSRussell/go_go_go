package cli

import (
	"fmt"

	"github.com/BenjaminSRussell/go_go_go/internal/export"
	"github.com/spf13/cobra"
)

var (
	exportDataDir     string
	outputFile        string
	includeLastmod    bool
	includeChangefreq bool
	defaultPriority   float64
)

var exportCmd = &cobra.Command{
	Use:   "export-sitemap",
	Short: "Export crawl results to sitemap",
	Long:  `Export crawled URLs to XML sitemap format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := export.SitemapConfig{
			DataDir:           exportDataDir,
			OutputFile:        outputFile,
			IncludeLastmod:    includeLastmod,
			IncludeChangefreq: includeChangefreq,
			DefaultPriority:   defaultPriority,
		}

		count, err := export.ExportSitemap(config)
		if err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

		fmt.Printf("Successfully exported %d URLs to %s\n", count, outputFile)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportDataDir, "data-dir", "./data", "Data storage directory")
	exportCmd.Flags().StringVar(&outputFile, "output", "sitemap.xml", "Output file path")
	exportCmd.Flags().BoolVar(&includeLastmod, "include-lastmod", true, "Include lastmod in sitemap")
	exportCmd.Flags().BoolVar(&includeChangefreq, "include-changefreq", true, "Include changefreq in sitemap")
	exportCmd.Flags().Float64Var(&defaultPriority, "default-priority", 0.5, "Default priority value")
}
