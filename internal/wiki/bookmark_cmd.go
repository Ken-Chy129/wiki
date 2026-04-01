package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var bookmarkGroupCmd = &cobra.Command{
	Use:   "bookmark",
	Short: "Manage bookmarks",
}

// --- add ---

var bookmarkAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Save a URL for later reading",
	RunE:  runBookmarkAdd,
}

var (
	baURL   string
	baTitle string
	baTags  string
	baDesc  string
)

// --- list ---

var bookmarkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bookmarks",
	RunE:  runBookmarkList,
}

var blTags string

// --- delete ---

var bookmarkDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a bookmark by slug",
	RunE:  runBookmarkDelete,
}

var bdSlug string

func init() {
	bookmarkGroupCmd.AddCommand(bookmarkAddCmd)
	bookmarkGroupCmd.AddCommand(bookmarkListCmd)
	bookmarkGroupCmd.AddCommand(bookmarkDeleteCmd)

	// add flags
	bookmarkAddCmd.Flags().StringVar(&baURL, "url", "", "URL to bookmark (required)")
	bookmarkAddCmd.Flags().StringVar(&baTitle, "title", "", "Bookmark title")
	bookmarkAddCmd.Flags().StringVar(&baTags, "tags", "", "Comma-separated tags")
	bookmarkAddCmd.Flags().StringVar(&baDesc, "desc", "", "Brief description of the content")
	bookmarkAddCmd.MarkFlagRequired("url")

	// list flags
	bookmarkListCmd.Flags().StringVar(&blTags, "tags", "", "Filter by tag")

	// delete flags
	bookmarkDeleteCmd.Flags().StringVar(&bdSlug, "slug", "", "Bookmark slug (required)")
	bookmarkDeleteCmd.MarkFlagRequired("slug")
}

func runBookmarkAdd(cmd *cobra.Command, args []string) error {
	bmDir, err := bookmarkDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bmDir, 0755); err != nil {
		return fmt.Errorf("failed to create bookmark directory: %w", err)
	}

	indexPath := filepath.Join(bmDir, "_index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		idx := "---\ntitle: \"收藏\"\nbookCollapseSection: true\n---\n"
		os.WriteFile(indexPath, []byte(idx), 0644)
	}

	title := baTitle
	if title == "" {
		title = baURL
	}

	slug := slugify(title)
	if len(slug) > 60 {
		slug = slug[:60]
	}
	filePath := filepath.Join(bmDir, slug+".md")

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	sb.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("url: %q\n", baURL))
	if baTags != "" {
		tagList := strings.Split(baTags, ",")
		trimmed := make([]string, len(tagList))
		for i, t := range tagList {
			trimmed[i] = fmt.Sprintf("%q", strings.TrimSpace(t))
		}
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(trimmed, ", ")))
	}
	sb.WriteString("type: bookmark\n")
	sb.WriteString("---\n\n")

	if baDesc != "" {
		sb.WriteString(baDesc + "\n\n")
	}
	sb.WriteString(fmt.Sprintf("[Read original](%s)\n", baURL))

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write bookmark: %w", err)
	}

	fmt.Printf("Bookmark saved: %s\n", slug)
	return nil
}

func runBookmarkList(cmd *cobra.Command, args []string) error {
	bmDir, err := bookmarkDir()
	if err != nil {
		return err
	}

	var bookmarks []articleInfo

	filepath.Walk(bmDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) == "_index.md" {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		ai := parseArticleInfo(path)
		ai.path = strings.TrimSuffix(filepath.Base(path), ".md")

		if blTags != "" && !hasTag(ai.tags, blTags) {
			return nil
		}

		bookmarks = append(bookmarks, ai)
		return nil
	})

	if len(bookmarks) == 0 {
		fmt.Println("No bookmarks found.")
		return nil
	}

	for _, b := range bookmarks {
		fmt.Printf("  %s  %s  slug:%s\n", b.date, b.title, b.path)
	}
	fmt.Printf("\nTotal: %d\n", len(bookmarks))
	return nil
}

func runBookmarkDelete(cmd *cobra.Command, args []string) error {
	bmDir, err := bookmarkDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(bmDir, bdSlug+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("bookmark not found: %s", bdSlug)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	fmt.Printf("Deleted: %s\n", bdSlug)
	return nil
}
