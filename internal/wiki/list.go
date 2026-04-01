package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List articles or bookmarks",
	RunE:  runList,
}

var (
	listCategory string
	listSaved    bool
	listTags     string
)

func init() {
	listCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
	listCmd.Flags().BoolVar(&listSaved, "saved", false, "List bookmarks only")
	listCmd.Flags().StringVar(&listTags, "tags", "", "Filter by tag")
}

type articleInfo struct {
	path     string
	title    string
	date     string
	tags     []string
	isBookmark bool
}

func runList(cmd *cobra.Command, args []string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}

	root, _ := rootDir()
	var articles []articleInfo

	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) == "_index.md" {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		ai := parseArticleInfo(path)
		relPath, _ := filepath.Rel(root, path)
		ai.path = relPath

		// Apply filters
		if listSaved && !ai.isBookmark {
			return nil
		}
		if listCategory != "" {
			relToContent, _ := filepath.Rel(docsDir, path)
			if !strings.HasPrefix(relToContent, listCategory) {
				return nil
			}
		}
		if listTags != "" && !hasTag(ai.tags, listTags) {
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
		typeTag := ""
		if a.isBookmark {
			typeTag = " [bookmark]"
		}
		fmt.Printf("  %s  %-40s %s%s\n", a.date, a.title, a.path, typeTag)
	}
	fmt.Printf("\nTotal: %d\n", len(articles))
	return nil
}

var titleRe = regexp.MustCompile(`(?m)^title:\s*"?([^"\n]+)"?`)
var dateRe = regexp.MustCompile(`(?m)^date:\s*(\S+)`)
var tagRe = regexp.MustCompile(`(?m)^  - (.+)`)
var typeRe = regexp.MustCompile(`(?m)^type:\s*bookmark`)

func parseArticleInfo(path string) articleInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return articleInfo{}
	}
	content := string(data)

	ai := articleInfo{}
	if m := titleRe.FindStringSubmatch(content); len(m) > 1 {
		ai.title = m[1]
	}
	if m := dateRe.FindStringSubmatch(content); len(m) > 1 {
		ai.date = m[1]
	}
	if matches := tagRe.FindAllStringSubmatch(content, -1); matches != nil {
		for _, m := range matches {
			ai.tags = append(ai.tags, strings.TrimSpace(m[1]))
		}
	}
	ai.isBookmark = typeRe.MatchString(content)
	return ai
}

func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}
