package wiki

import (
	"fmt"
	"os"
	"path/filepath"
)

// rootDir returns the project root directory (where hugo.toml lives).
func rootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "hugo.toml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no hugo.toml found)")
		}
		dir = parent
	}
}

func contentDir() (string, error) {
	root, err := rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "content", "docs"), nil
}

func bookmarkDir() (string, error) {
	root, err := rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "content", "docs", "收藏"), nil
}
