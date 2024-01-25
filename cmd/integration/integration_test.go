package integration

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"os/exec"
	"strings"
	"testing"
)

func TestBuildVariables(t *testing.T) {
	cmd := exec.Command("git", "branch")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'git branch': %v", err)
	}

	currentBranch, err := getCurrentBranch(string(output))
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// get the current git commit
	currentGitCommit := exec.Command("git", "log", "-n", "1", currentBranch)
	gitCommitOutput, err := currentGitCommit.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'git log': %v", err)
	}

	commitHash, err := extractCommitHash(string(gitCommitOutput))
	if err != nil {
		t.Fatalf("Failed to run 'git branch': %v", err)
	}

	expectedGitCommitHash := version.Commit
	if commitHash != expectedGitCommitHash {
		t.Errorf("Git commits not equal. Expected: %s, Actual: %s", expectedGitCommitHash, commitHash)
	}

	expectedVersion := version.Version
	if !strings.Contains(expectedVersion, currentBranch) {
		t.Errorf("Versions not equal. Expected: %s, Actual: %s", expectedVersion, currentBranch)
	}
}

func getCurrentBranch(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "*") {
			return strings.TrimSpace(strings.TrimPrefix(line, "*")), nil
		}
	}
	return "", fmt.Errorf("could not find current branch")
}

func extractCommitHash(input string) (string, error) {
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		if strings.Contains(line, "commit") {
			commitHash := strings.Split(line, " ")
			return strings.TrimSpace(commitHash[1]), nil
		}
	}

	return "", fmt.Errorf("commit hash not found in input")
}
