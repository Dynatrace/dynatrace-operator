package nodemetadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseAttributes(t *testing.T) {
	t.Run("parses valid key=value pairs", func(t *testing.T) {
		content, err := parseAttributes("k8s.cluster.name=test-cluster,k8s.node.name=test-node")

		require.NoError(t, err)
		assert.Equal(t, "k8s.cluster.name=test-cluster\nk8s.node.name=test-node\n", content)
	})

	t.Run("trims whitespace from pairs", func(t *testing.T) {
		content, err := parseAttributes(" k8s.cluster.name=test , k8s.node.name=node ")

		require.NoError(t, err)
		assert.Equal(t, "k8s.cluster.name=test\nk8s.node.name=node\n", content)
	})

	t.Run("returns error for empty attributes flag", func(t *testing.T) {
		content, err := parseAttributes("")

		require.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("returns error for whitespace-only attributes", func(t *testing.T) {
		content, err := parseAttributes("  ,  ,  ")

		require.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "no valid attributes")
	})

	t.Run("returns error for invalid format without equals", func(t *testing.T) {
		content, err := parseAttributes("k8s.cluster.name")

		require.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "invalid attribute format")
		assert.Contains(t, err.Error(), "k8s.cluster.name")
	})

	t.Run("returns error for mixed valid and invalid pairs", func(t *testing.T) {
		content, err := parseAttributes("k8s.cluster.name=test,invalid-key")

		require.Error(t, err)
		assert.Empty(t, content)
		assert.Contains(t, err.Error(), "invalid attribute format")
	})

	t.Run("handles values with equals signs", func(t *testing.T) {
		content, err := parseAttributes("key=value=with=equals")

		require.NoError(t, err)
		assert.Equal(t, "key=value=with=equals\n", content)
	})

	t.Run("handles single pair", func(t *testing.T) {
		content, err := parseAttributes("k8s.cluster.name=test-cluster")

		require.NoError(t, err)
		assert.Equal(t, "k8s.cluster.name=test-cluster\n", content)
	})
}

func Test_writeMetadataFile(t *testing.T) {
	t.Run("creates directory and writes file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "dt_node_metadata.properties")
		content := "k8s.cluster.name=test-cluster\nk8s.node.name=test-node\n"

		err = writeMetadataFile(filePath, content)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "metadata.properties")

		err = os.MkdirAll(filepath.Dir(filePath), 0755)
		require.NoError(t, err)
		// #nosec G306 -- node metadata file is not sensitive, 0644 is intentional
		err = os.WriteFile(filePath, []byte("old=content\n"), 0644)
		require.NoError(t, err)

		newContent := "new=content\n"

		err = writeMetadataFile(filePath, newContent)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, newContent, string(data))
		assert.NotContains(t, string(data), "old=content")
	})

	t.Run("writes empty file for empty content", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "empty.properties")
		content := ""

		err = writeMetadataFile(filePath, content)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Empty(t, string(data))
	})
}

func Test_run(t *testing.T) {
	t.Run("successfully generates metadata file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "metadata.properties")

		nodeMetadataFileFlagValue = filePath
		nodeAttributesFlagValue = "k8s.node.name=test-node,k8s.cluster.uid=test-uid"

		runFunc := run()
		err = runFunc(nil, nil)

		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "k8s.node.name=test-node\nk8s.cluster.uid=test-uid\n", string(data))
	})

	t.Run("returns error for invalid attributes", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		nodeMetadataFileFlagValue = filepath.Join(tmpDir, "metadata.properties")
		nodeAttributesFlagValue = ""

		runFunc := run()
		err = runFunc(nil, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "nodemetadata-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		nodeMetadataFileFlagValue = filepath.Join(tmpDir, "metadata.properties")
		nodeAttributesFlagValue = "invalid-without-equals"

		runFunc := run()
		err = runFunc(nil, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute format")
	})
}
