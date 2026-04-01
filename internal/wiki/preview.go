package wiki

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview an article as HTML in browser",
	RunE:  runPreview,
}

var previewSlug string

func init() {
	previewCmd.Flags().StringVar(&previewSlug, "slug", "", "Article slug path to preview")
}

func runPreview(cmd *cobra.Command, args []string) error {
	root, err := rootDir()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "wiki-preview-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Run hugo to build the site into tmpDir
	hugoCmd := exec.Command("hugo", "-d", tmpDir)
	hugoCmd.Dir = root
	hugoCmd.Stderr = os.Stderr
	if err := hugoCmd.Run(); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	// Determine which HTML file to open
	var htmlPath string
	if previewSlug != "" {
		htmlPath = filepath.Join(tmpDir, "docs", previewSlug, "index.html")
	} else {
		htmlPath = filepath.Join(tmpDir, "index.html")
	}

	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		return fmt.Errorf("preview file not found: %s (check slug)", htmlPath)
	}

	// Open in browser
	openCmd := exec.Command("open", htmlPath)
	if err := openCmd.Run(); err != nil {
		fmt.Printf("Preview generated: %s\n", htmlPath)
		return nil
	}

	fmt.Printf("Preview opened in browser: %s\n", htmlPath)
	return nil
}
