package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BenjaminSRussell/go_go_go/internal/storage"
	"github.com/spf13/cobra"
)

var (
	searchDataDir string
	searchLimit   int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search crawled page titles and meta (requires --enable-sqlite crawl)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := filepath.Join(searchDataDir, "crawl.db")
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("no SQLite database at %s (crawl with --enable-sqlite first): %w", dbPath, err)
		}
		store, err := storage.NewSQLiteStorage(dbPath)
		if err != nil {
			return err
		}
		defer store.Close()

		results, err := store.Search(strings.Join(args, " "), searchLimit)
		if err != nil {
			return err
		}
		mode := "LIKE fallback (build with -tags sqlite_fts5 for ranked FTS5)"
		if store.FTSEnabled() {
			mode = "FTS5"
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "%d result(s) [%s]\n", len(results), mode)
		for i, r := range results {
			fmt.Fprintf(out, "%2d. %s\n    %s\n", i+1, r.Title, r.URL)
			if r.MetaDescription != "" {
				fmt.Fprintf(out, "    %s\n", r.MetaDescription)
			}
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchDataDir, "data-dir", "./data", "Data storage directory containing crawl.db")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum results")
}
