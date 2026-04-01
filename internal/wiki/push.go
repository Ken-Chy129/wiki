package wiki

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Commit and push changes to deploy",
	RunE:  runPush,
}

var (
	pushMessage string
	pushDiff    bool
)

func init() {
	pushCmd.Flags().StringVar(&pushMessage, "message", "", "Commit message (default: auto-generated)")
	pushCmd.Flags().BoolVar(&pushDiff, "diff", false, "Show pending changes without deploying")
}

func runPush(cmd *cobra.Command, args []string) error {
	root, err := rootDir()
	if err != nil {
		return err
	}

	if pushDiff {
		return showPendingChanges(root)
	}

	// Stage all content changes
	gitAdd := exec.Command("git", "add", "content/")
	gitAdd.Dir = root
	gitAdd.Stdout = os.Stdout
	gitAdd.Stderr = os.Stderr
	if err := gitAdd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are staged changes
	gitDiffCheck := exec.Command("git", "diff", "--cached", "--quiet")
	gitDiffCheck.Dir = root
	if err := gitDiffCheck.Run(); err == nil {
		fmt.Println("No changes to push.")
		return nil
	}

	msg := pushMessage
	if msg == "" {
		msg = "docs: update knowledge base"
	}

	gitCommit := exec.Command("git", "commit", "-m", msg)
	gitCommit.Dir = root
	gitCommit.Stdout = os.Stdout
	gitCommit.Stderr = os.Stderr
	if err := gitCommit.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	gitPush := exec.Command("git", "push")
	gitPush.Dir = root
	gitPush.Stdout = os.Stdout
	gitPush.Stderr = os.Stderr
	if err := gitPush.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Println("Published successfully!")
	return nil
}

func showPendingChanges(root string) error {
	docsDir, err := contentDir()
	if err != nil {
		return err
	}
	bmDir, _ := bookmarkDir()

	// Get changed files from git status
	gitStatus := exec.Command("git", "-c", "core.quotePath=false", "status", "--short", "-u", "content/")
	gitStatus.Dir = root
	out, err := gitStatus.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	if len(out) == 0 {
		fmt.Println("No pending changes.")
		return nil
	}

	fmt.Println("Pending changes:")
	fmt.Println()

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse status and path, e.g. "?? content/docs/收藏/xxx.md" or "M  content/docs/AI/foo.md"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		filePath := strings.TrimSpace(parts[1])

		// Skip non-.md and _index.md
		if !strings.HasSuffix(filePath, ".md") || strings.HasSuffix(filePath, "_index.md") {
			continue
		}

		// If it's a directory entry (ends with /), skip
		if strings.HasSuffix(filePath, "/") {
			continue
		}

		absPath := filepath.Join(root, filePath)
		ai := parseArticleInfo(absPath)

		isBookmark := bmDir != "" && strings.HasPrefix(absPath, bmDir)

		var slug string
		if isBookmark {
			slug = strings.TrimSuffix(filepath.Base(absPath), ".md")
		} else {
			relToContent, _ := filepath.Rel(docsDir, absPath)
			slug = strings.TrimSuffix(relToContent, ".md")
		}

		statusLabel := ""
		switch {
		case strings.Contains(status, "?"):
			statusLabel = "new"
		case strings.Contains(status, "M"):
			statusLabel = "modified"
		case strings.Contains(status, "D"):
			statusLabel = "deleted"
		case strings.Contains(status, "A"):
			statusLabel = "new"
		default:
			statusLabel = status
		}

		typeLabel := "article"
		if isBookmark {
			typeLabel = "bookmark"
		}

		title := ai.title
		if title == "" {
			title = slug
		}

		fmt.Printf("  [%s] [%s] %s (%s)\n", statusLabel, typeLabel, title, slug)
	}

	return nil
}
