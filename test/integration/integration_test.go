//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestBuildVariables(t *testing.T) {
	cmd := exec.Command("git", "branch", "--show-current")
	currentBranch, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'git branch': %v", err)
	}

	// get the current git commit
	currentGitCommit := exec.Command("git", "rev-parse", "HEAD")
	gitCommitOutput, err := currentGitCommit.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'git log': %v", err)
	}

	commitHash := strings.TrimSpace(string(gitCommitOutput))
	expectedGitCommitHash := version.Commit
	assert.Equal(t, expectedGitCommitHash, commitHash)

	expectedVersion := version.Version
	curBranch := strings.TrimSpace(string(currentBranch))
	assert.Equal(t, expectedVersion, curBranch)
}
