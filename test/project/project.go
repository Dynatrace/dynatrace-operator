//go:build e2e

package project

import (
	"os"
	"path"
)

var rootDir string

func RootDir() string {
	return rootDir
}

func TestDataDir() string {
	return path.Join(rootDir, "test", "testdata")
}

func init() {
	rootDir = "."

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for dir != "" {
		goModPath := path.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			rootDir = dir

			return
		}

		dir = path.Dir(dir)
	}
}
