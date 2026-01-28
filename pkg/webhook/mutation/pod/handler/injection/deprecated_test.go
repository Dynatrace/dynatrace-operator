package injection

import (
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_setDeprecatedAttributes(t *testing.T) {
	t.Run("set deprecated attribute according to current attribute", func(t *testing.T) {
		attrs := podattr.Attributes{
			ClusterInfo: podattr.ClusterInfo{
				ClusterUID: "testID",
			},
		}

		setDeprecatedAttributes(&attrs)

		assertDeprecatedAttributes(t, attrs)
	})
}

func assertDeprecatedAttributes(t *testing.T, attrs podattr.Attributes) {
	t.Helper()

	depValue, ok := attrs.UserDefined[deprecatedClusterIDKey]
	require.True(t, ok)
	assert.Equal(t, attrs.ClusterUID, depValue)
}
