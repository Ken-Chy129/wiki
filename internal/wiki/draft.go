package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var draftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Create a new article draft",
	RunE:  runDraft,
}

var (
	draftTitle    string
	draftCategory string
	draftTags     string
	draftSummary  string
	draftFile     string
	draftContent  string
)

func init() {
	draftCmd.Flags().StringVar(&draftTitle, "title", "", "Article title (required)")
	draftCmd.Flags().StringVar(&draftCategory, "category", "", "Category path, e.g. AI/LLM (required)")
	draftCmd.Flags().StringVar(&draftTags, "tags", "", "Comma-separated tags")
	draftCmd.Flags().StringVar(&draftSummary, "summary", "", "Article summary")
	draftCmd.Flags().StringVar(&draftFile, "file", "", "Read content from file")
	draftCmd.Flags().StringVar(&draftContent, "content", "", "Article content as string")
	draftCmd.MarkFlagRequired("title")
	draftCmd.MarkFlagRequired("category")
}

func runDraft(cmd *cobra.Command, args []string) error {
	content, err := resolveContent(draftFile, draftContent)
	if err != nil {
		return err
	}

	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	// Build target directory
	categoryPath := filepath.Join(docsDir, filepath.FromSlash(draftCategory))
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	// Ensure category _index.md exists
	if err := ensureCategoryIndex(categoryPath, draftCategory); err != nil {
		return err
	}

	// Generate filename from title
	slug := slugify(draftTitle)
	filePath := filepath.Join(categoryPath, slug+".md")

	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file already exists: %s", filePath)
	}

	// Build front matter
	frontMatter := buildFrontMatter(draftTitle, draftTags, draftSummary)
	fullContent := frontMatter + content

	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	root, _ := rootDir()
	relPath, _ := filepath.Rel(root, filePath)
	fmt.Printf("Draft created: %s\n", relPath)
	return nil
}

func resolveContent(file, content string) (string, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), nil
	}
	if content != "" {
		return content, nil
	}
	return "", nil
}

func buildFrontMatter(title, tags, summary string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	sb.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString("draft: true\n")
	if summary != "" {
		sb.WriteString(fmt.Sprintf("summary: %q\n", summary))
	}
	if tags != "" {
		tagList := strings.Split(tags, ",")
		trimmed := make([]string, len(tagList))
		for i, t := range tagList {
			trimmed[i] = fmt.Sprintf("%q", strings.TrimSpace(t))
		}
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(trimmed, ", ")))
	}
	sb.WriteString("---\n\n")
	return sb.String()
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	// Keep alphanumeric, hyphens, and CJK characters
	var result []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r > 127 {
			result = append(result, r)
		}
	}
	return strings.Trim(string(result), "-")
}

func ensureCategoryIndex(categoryPath, category string) error {
	indexPath := filepath.Join(categoryPath, "_index.md")
	if _, err := os.Stat(indexPath); err == nil {
		return nil
	}
	parts := strings.Split(category, "/")
	title := parts[len(parts)-1]
	content := fmt.Sprintf("---\ntitle: %q\nbookCollapseSection: true\n---\n", title)
	return os.WriteFile(indexPath, []byte(content), 0644)
}
