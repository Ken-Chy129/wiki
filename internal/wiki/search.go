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
	Short: "Search articles and bookmarks",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

var (
	searchType  string
	searchField string
)

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "all", "Search scope: article, bookmark, all")
	searchCmd.Flags().StringVar(&searchField, "field", "all", "Search field: title, content, all")
}

func runSearch(cmd *cobra.Command, args []string) error {
	keyword := strings.ToLower(args[0])

	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	bmDir, _ := bookmarkDir()
	count := 0

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) == "_index.md" {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		isBookmark := bmDir != "" && strings.HasPrefix(path, bmDir)

		// Filter by type
		switch searchType {
		case "article":
			if isBookmark {
				return nil
			}
		case "bookmark":
			if !isBookmark {
				return nil
			}
		}

		ai := parseArticleInfo(path)

		// Match by field
		matched := false
		switch searchField {
		case "title":
			matched = strings.Contains(strings.ToLower(ai.title), keyword)
		case "content":
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			body := extractBody(string(data))
			matched = strings.Contains(strings.ToLower(body), keyword)
		default: // all
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			matched = strings.Contains(strings.ToLower(string(data)), keyword)
		}

		if matched {
			var slug string
			if isBookmark {
				slug = strings.TrimSuffix(filepath.Base(path), ".md")
			} else {
				relToContent, _ := filepath.Rel(docsDir, path)
				slug = strings.TrimSuffix(relToContent, ".md")
			}

			typeLabel := ""
			if isBookmark {
				typeLabel = " [bookmark]"
			}
			fmt.Printf("  %s  %s  slug:%s%s\n", ai.date, ai.title, slug, typeLabel)
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

// extractBody returns content after front matter.
func extractBody(content string) string {
	if loc := frontMatterRe.FindStringIndex(content); loc != nil {
		return content[loc[1]:]
	}
	return content
}
