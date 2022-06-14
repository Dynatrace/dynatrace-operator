package utils

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

func isExecutable(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if !fileInfo.IsDir() && fileInfo.Mode().Perm()&0755 == 0755 {
			return true, nil
		}
		return false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func which(command string) (string, error) {
	fname, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}
	fname, err = filepath.Abs(fname)
	if err != nil {
		return "", err
	}
	return fname, nil
}

func GetCommand(commandName string) (string, error) {
	gobin, err := sh.Output("go", "env", "GOBIN")
	if err != nil {
		return "", err
	}

	if gobin != "" {
		commandPath := path.Join(gobin, "bin", commandName)
		isExec, err := isExecutable(commandPath)
		if err != nil {
			return "", err
		}
		if isExec {
			return commandPath, nil
		}
	}

	commandPath, err := which(commandName)
	if err != nil {
		return "", err
	}
	return commandPath, nil
}
