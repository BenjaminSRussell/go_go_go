package cli

import (
	"fmt"

	"github.com/BenjaminSRussell/go_go_go/internal/export"
	"github.com/spf13/cobra"
)

var (
	parquetDataDir   string
	parquetPagesFile string
	parquetLinksFile string
)

var exportParquetCmd = &cobra.Command{
	Use:   "export-parquet",
	Short: "Export crawl results to Parquet (DuckDB/pandas/Spark)",
	Long: `Export crawled pages (and the link graph, when the crawl used --enable-sqlite)
to Parquet files, e.g.:

  gogogoscraper export-parquet --data-dir ./data --output pages.parquet --links links.parquet
  duckdb -c "SELECT status_code, count(*) FROM 'pages.parquet' GROUP BY 1"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := export.ExportParquet(export.ParquetConfig{
			DataDir:   parquetDataDir,
			PagesFile: parquetPagesFile,
			LinksFile: parquetLinksFile,
		})
		if err != nil {
			return fmt.Errorf("parquet export failed: %w", err)
		}
		fmt.Printf("Exported %d pages to %s (from %s)\n", stats.Pages, parquetPagesFile, stats.Source)
		if parquetLinksFile != "" {
			fmt.Printf("Exported %d links to %s\n", stats.Links, parquetLinksFile)
		}
		return nil
	},
}

func init() {
	exportParquetCmd.Flags().StringVar(&parquetDataDir, "data-dir", "./data", "Crawl data directory")
	exportParquetCmd.Flags().StringVar(&parquetPagesFile, "output", "pages.parquet", "Pages Parquet output path")
	exportParquetCmd.Flags().StringVar(&parquetLinksFile, "links", "", "Optional links Parquet output path (requires crawl.db)")
}
