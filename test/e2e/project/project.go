//go:build e2e

package project

import (
	"os"
	"path/filepath"
)

var rootDir string

func RootDir() string {
	return rootDir
}

func TestDataDir() string {
	return filepath.Join(rootDir, "test", "e2e", "testdata")
}

func init() {
	rootDir = "."

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for dir != "" {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			rootDir = dir

			return
		}

		dir = filepath.Dir(dir)
	}
}
