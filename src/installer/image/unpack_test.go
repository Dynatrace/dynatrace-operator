package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPolicyContext(t *testing.T) {
	t.Run("not nil", func(t *testing.T) {
		policyContext, err := buildPolicyContext()
		require.NoError(t, err)
		require.NotNil(t, policyContext)
	})
}
