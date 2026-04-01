package wiki

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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

var titleRe = regexp.MustCompile(`(?m)^title:\s*"?([^"\n]+)"?`)
var dateRe = regexp.MustCompile(`(?m)^date:\s*(\S+)`)
var tagRe = regexp.MustCompile(`(?m)^  - (.+)`)
var typeRe = regexp.MustCompile(`(?m)^type:\s*bookmark`)

type articleInfo struct {
	path       string
	title      string
	date       string
	tags       []string
	isBookmark bool
}

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
		ai.date = normalizeDate(m[1])
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

func normalizeDate(raw string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return raw
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
