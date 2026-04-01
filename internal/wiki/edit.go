package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing article",
	RunE:  runEdit,
}

var (
	editSlug    string
	editFile    string
	editContent string
)

func init() {
	editCmd.Flags().StringVar(&editSlug, "slug", "", "Article slug path, e.g. ai/llm/transformer (required)")
	editCmd.Flags().StringVar(&editFile, "file", "", "Read new content from file")
	editCmd.Flags().StringVar(&editContent, "content", "", "New content as string")
	editCmd.MarkFlagRequired("slug")
}

func runEdit(cmd *cobra.Command, args []string) error {
	newBody, err := resolveContent(editFile, editContent)
	if err != nil {
		return err
	}
	if newBody == "" {
		return fmt.Errorf("must provide --file or --content")
	}

	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(docsDir, filepath.FromSlash(editSlug)+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("article not found: %s", filePath)
	}

	existing, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Preserve front matter, replace body
	frontMatter := extractFrontMatter(string(existing))
	fullContent := frontMatter + newBody

	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	root, _ := rootDir()
	relPath, _ := filepath.Rel(root, filePath)
	fmt.Printf("Updated: %s\n", relPath)
	return nil
}

var frontMatterRe = regexp.MustCompile(`(?s)^---\n.*?\n---\n+`)

func extractFrontMatter(content string) string {
	if loc := frontMatterRe.FindStringIndex(content); loc != nil {
		fm := content[loc[0]:loc[1]]
		if !strings.HasSuffix(fm, "\n\n") {
			fm += "\n"
		}
		return fm
	}
	return ""
}
