package attributes

import (
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_setDeprecatedClusterAttributes(t *testing.T) {
	t.Run("set deprecated attribute according to current attribute", func(t *testing.T) {
		attrs := podattr.Attributes{
			ClusterInfo: podattr.ClusterInfo{
				ClusterUID: "testID",
			},
		}

		attrs = setDeprecatedClusterAttributes(attrs)

		assertDeprecatedClusterAttributes(t, attrs)
	})
}

func Test_setDeprecatedAttributes(t *testing.T) {
	t.Run("set deprecated attribute according to current attribute", func(t *testing.T) {
		attrs := podattr.Attributes{
			WorkloadInfo: podattr.WorkloadInfo{
				WorkloadKind: "kind",
				WorkloadName: "name",
			},
		}

		attrs = setDeprecatedWorkloadAttributes(attrs)

		assertDeprecatedWorkloadAttributes(t, attrs)
	})
}

func assertDeprecatedWorkloadAttributes(t *testing.T, attrs podattr.Attributes) {
	depWorkloadKind, ok := attrs.UserDefined[DeprecatedWorkloadKindKey]
	require.True(t, ok)
	assert.Equal(t, attrs.WorkloadKind, depWorkloadKind)

	depWorkloadName, ok := attrs.UserDefined[DeprecatedWorkloadNameKey]
	require.True(t, ok)
	assert.Equal(t, attrs.WorkloadName, depWorkloadName)
}

func assertDeprecatedClusterAttributes(t *testing.T, attrs podattr.Attributes) {
	depValue, ok := attrs.UserDefined[DeprecatedClusterIDKey]
	require.True(t, ok)
	assert.Equal(t, attrs.ClusterUID, depValue)
}
