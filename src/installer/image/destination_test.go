package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDestinationInfo(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		testDir := "dir"
		context, reference, err := getDestinationInfo(testDir)
		require.NoError(t, err)
		require.NotNil(t, context)
		require.NotNil(t, reference)
	})
}

func TestBuildDestinationContext(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		testDir := "dir"
		destinationContext := buildDestinationContext(testDir)
		require.NotNil(t, destinationContext)
		assert.Equal(t, testDir, destinationContext.BlobInfoCacheDir)
		assert.True(t, destinationContext.DirForceDecompress)
	})
}

func TestGetDestinationReference(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		testDir := "dir"
		destinationReference, err := getDestinationReference(testDir)
		require.NoError(t, err)
		require.NotNil(t, destinationReference)
	})
}
