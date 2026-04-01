package wiki

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Commit and push changes to deploy",
	RunE:  runPush,
}

var pushMessage string

func init() {
	pushCmd.Flags().StringVar(&pushMessage, "message", "", "Commit message (default: auto-generated)")
}

func runPush(cmd *cobra.Command, args []string) error {
	root, err := rootDir()
	if err != nil {
		return err
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
	gitDiff := exec.Command("git", "diff", "--cached", "--quiet")
	gitDiff.Dir = root
	if err := gitDiff.Run(); err == nil {
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
