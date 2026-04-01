package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search articles by keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	keyword := strings.ToLower(args[0])

	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	root, _ := rootDir()
	count := 0

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := strings.ToLower(string(data))
		if strings.Contains(content, keyword) {
			ai := parseArticleInfo(path)
			relPath, _ := filepath.Rel(root, path)
			fmt.Printf("  %s  %s\n", ai.title, relPath)
			count++
		}
		return nil
	})

	if count == 0 {
		fmt.Println("No results found.")
	} else {
		fmt.Printf("\nFound: %d\n", count)
	}
	return nil
}
