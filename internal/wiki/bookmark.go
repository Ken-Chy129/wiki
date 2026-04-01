package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var bookmarkCmd = &cobra.Command{
	Use:   "bookmark",
	Short: "Save a URL for later reading",
	RunE:  runBookmark,
}

var (
	bmURL   string
	bmTitle string
	bmTags  string
	bmDesc  string
)

func init() {
	bookmarkCmd.Flags().StringVar(&bmURL, "url", "", "URL to bookmark (required)")
	bookmarkCmd.Flags().StringVar(&bmTitle, "title", "", "Bookmark title")
	bookmarkCmd.Flags().StringVar(&bmTags, "tags", "", "Comma-separated tags")
	bookmarkCmd.Flags().StringVar(&bmDesc, "desc", "", "Brief description of the content")
	bookmarkCmd.MarkFlagRequired("url")
}

func runBookmark(cmd *cobra.Command, args []string) error {
	bmDir, err := bookmarkDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bmDir, 0755); err != nil {
		return fmt.Errorf("failed to create bookmark directory: %w", err)
	}

	// Ensure _index.md
	indexPath := filepath.Join(bmDir, "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		idx := "---\ntitle: \"收藏\"\nbookCollapseSection: true\n---\n"
		os.WriteFile(indexPath, []byte(idx), 0644)
	}

	title := bmTitle
	if title == "" {
		title = bmURL
	}

	slug := slugify(title)
	if len(slug) > 60 {
		slug = slug[:60]
	}
	filePath := filepath.Join(bmDir, slug+".md")

	// Build content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	sb.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("url: %q\n", bmURL))
	if bmTags != "" {
		tagList := strings.Split(bmTags, ",")
		trimmed := make([]string, len(tagList))
		for i, t := range tagList {
			trimmed[i] = fmt.Sprintf("%q", strings.TrimSpace(t))
		}
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(trimmed, ", ")))
	}
	sb.WriteString("type: bookmark\n")
	sb.WriteString("---\n\n")

	if bmDesc != "" {
		sb.WriteString(bmDesc + "\n\n")
	}
	sb.WriteString(fmt.Sprintf("[Read original](%s)\n", bmURL))

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write bookmark: %w", err)
	}

	root, _ := rootDir()
	relPath, _ := filepath.Rel(root, filePath)
	fmt.Printf("Bookmark saved: %s\n", relPath)
	return nil
}
