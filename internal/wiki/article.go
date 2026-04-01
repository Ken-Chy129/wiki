package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var articleCmd = &cobra.Command{
	Use:   "article",
	Short: "Manage articles",
}

// --- create ---

var articleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new article draft",
	RunE:  runArticleCreate,
}

var (
	acTitle    string
	acCategory string
	acTags     string
	acSummary  string
	acFile     string
	acContent  string
)

// --- edit ---

var articleEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing article",
	RunE:  runArticleEdit,
}

var (
	aeSlug    string
	aeFile    string
	aeContent string
)

// --- show ---

var articleShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print article content by slug",
	RunE:  runArticleShow,
}

var asSlug string

// --- list ---

var articleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List articles",
	RunE:  runArticleList,
}

var (
	alCategory string
	alTags     string
)

// --- delete ---

var articleDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an article by slug",
	RunE:  runArticleDelete,
}

var adSlug string

// --- preview ---

var articlePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview an article as HTML in browser",
	RunE:  runArticlePreview,
}

var apSlug string

// --- categories ---

var articleCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List all existing categories",
	RunE:  runArticleCategories,
}

func init() {
	articleCmd.AddCommand(articleCreateCmd)
	articleCmd.AddCommand(articleEditCmd)
	articleCmd.AddCommand(articleShowCmd)
	articleCmd.AddCommand(articleListCmd)
	articleCmd.AddCommand(articleDeleteCmd)
	articleCmd.AddCommand(articlePreviewCmd)
	articleCmd.AddCommand(articleCategoriesCmd)

	// create flags
	articleCreateCmd.Flags().StringVar(&acTitle, "title", "", "Article title (required)")
	articleCreateCmd.Flags().StringVar(&acCategory, "category", "", "Category path, e.g. AI/LLM (required)")
	articleCreateCmd.Flags().StringVar(&acTags, "tags", "", "Comma-separated tags")
	articleCreateCmd.Flags().StringVar(&acSummary, "summary", "", "Article summary")
	articleCreateCmd.Flags().StringVar(&acFile, "file", "", "Read content from file")
	articleCreateCmd.Flags().StringVar(&acContent, "content", "", "Article content as string")
	articleCreateCmd.MarkFlagRequired("title")
	articleCreateCmd.MarkFlagRequired("category")

	// edit flags
	articleEditCmd.Flags().StringVar(&aeSlug, "slug", "", "Article slug path (required)")
	articleEditCmd.Flags().StringVar(&aeFile, "file", "", "Read new content from file")
	articleEditCmd.Flags().StringVar(&aeContent, "content", "", "New content as string")
	articleEditCmd.MarkFlagRequired("slug")

	// show flags
	articleShowCmd.Flags().StringVar(&asSlug, "slug", "", "Article slug path (required)")
	articleShowCmd.MarkFlagRequired("slug")

	// list flags
	articleListCmd.Flags().StringVar(&alCategory, "category", "", "Filter by category")
	articleListCmd.Flags().StringVar(&alTags, "tags", "", "Filter by tag")

	// delete flags
	articleDeleteCmd.Flags().StringVar(&adSlug, "slug", "", "Article slug path (required)")
	articleDeleteCmd.MarkFlagRequired("slug")

	// preview flags
	articlePreviewCmd.Flags().StringVar(&apSlug, "slug", "", "Article slug path to preview")
}

// --- implementations ---

func runArticleCreate(cmd *cobra.Command, args []string) error {
	content, err := resolveContent(acFile, acContent)
	if err != nil {
		return err
	}

	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	categoryPath := filepath.Join(docsDir, filepath.FromSlash(acCategory))
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	if err := ensureCategoryIndex(categoryPath, acCategory); err != nil {
		return err
	}

	slug := slugify(acTitle)
	filePath := filepath.Join(categoryPath, slug+".md")

	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file already exists: %s", filePath)
	}

	frontMatter := buildFrontMatter(acTitle, acTags, acSummary)
	fullContent := frontMatter + content

	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	relSlug := filepath.ToSlash(filepath.Join(acCategory, slug))
	fmt.Printf("Draft created: %s\n", relSlug)
	return nil
}

func runArticleEdit(cmd *cobra.Command, args []string) error {
	newBody, err := resolveContent(aeFile, aeContent)
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

	filePath := filepath.Join(docsDir, filepath.FromSlash(aeSlug)+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("article not found: %s", aeSlug)
	}

	existing, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	frontMatter := extractFrontMatter(string(existing))
	fullContent := frontMatter + newBody

	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Updated: %s\n", aeSlug)
	return nil
}

func runArticleShow(cmd *cobra.Command, args []string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(docsDir, filepath.FromSlash(asSlug)+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("article not found: %s", asSlug)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read article: %w", err)
	}

	fmt.Print(string(data))
	return nil
}

func runArticleList(cmd *cobra.Command, args []string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	bmDir, _ := bookmarkDir()
	var articles []articleInfo

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) == "_index.md" {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip bookmarks
		if bmDir != "" && strings.HasPrefix(path, bmDir) {
			return nil
		}

		ai := parseArticleInfo(path)
		relToContent, _ := filepath.Rel(docsDir, path)
		ai.path = strings.TrimSuffix(relToContent, ".md")

		if alCategory != "" && !strings.HasPrefix(ai.path, alCategory) {
			return nil
		}
		if alTags != "" && !hasTag(ai.tags, alTags) {
			return nil
		}

		articles = append(articles, ai)
		return nil
	})

	if len(articles) == 0 {
		fmt.Println("No articles found.")
		return nil
	}

	for _, a := range articles {
		fmt.Printf("  %s  %s  slug:%s\n", a.date, a.title, a.path)
	}
	fmt.Printf("\nTotal: %d\n", len(articles))
	return nil
}

func runArticleDelete(cmd *cobra.Command, args []string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(docsDir, filepath.FromSlash(adSlug)+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("article not found: %s", adSlug)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	fmt.Printf("Deleted: %s\n", adSlug)
	return nil
}

func runArticlePreview(cmd *cobra.Command, args []string) error {
	root, err := rootDir()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "wiki-preview-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	hugoCmd := execCommand("hugo", "-d", tmpDir, "--buildFuture")
	hugoCmd.Dir = root
	hugoCmd.Stderr = os.Stderr
	if err := hugoCmd.Run(); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	var htmlPath string
	if apSlug != "" {
		htmlPath = filepath.Join(tmpDir, "docs", strings.ToLower(apSlug), "index.html")
	} else {
		htmlPath = filepath.Join(tmpDir, "index.html")
	}

	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		return fmt.Errorf("preview file not found: %s (check slug)", htmlPath)
	}

	openCmd := execCommand("open", htmlPath)
	if err := openCmd.Run(); err != nil {
		fmt.Printf("Preview generated: %s\n", htmlPath)
		return nil
	}

	fmt.Printf("Preview opened in browser: %s\n", htmlPath)
	return nil
}

func runArticleCategories(cmd *cobra.Command, args []string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}
	bmDir, _ := bookmarkDir()

	seen := make(map[string]bool)

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		// Skip bookmark directory
		if bmDir != "" && strings.HasPrefix(path, bmDir) {
			return nil
		}
		if path == docsDir {
			return nil
		}

		rel, _ := filepath.Rel(docsDir, path)
		seen[rel] = true
		return nil
	})

	if len(seen) == 0 {
		fmt.Println("No categories found.")
		return nil
	}

	for cat := range seen {
		fmt.Println(cat)
	}
	return nil
}
