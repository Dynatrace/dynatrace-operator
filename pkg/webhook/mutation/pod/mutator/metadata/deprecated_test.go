package metadata

import (
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_setDeprecatedAttributes(t *testing.T) {
	t.Run("set deprecated attribute according to current attribute", func(t *testing.T) {
		attrs := podattr.Attributes{
			WorkloadInfo: podattr.WorkloadInfo{
				WorkloadKind: "kind",
				WorkloadName: "name",
			},
		}

		SetDeprecatedAttributes(&attrs)

		assertDeprecatedAttributes(t, attrs)
	})
}

func assertDeprecatedAttributes(t *testing.T, attrs podattr.Attributes) {
	t.Helper()

	depWorkloadKind, ok := attrs.UserDefined[DeprecatedWorkloadKindKey]
	require.True(t, ok)
	assert.Equal(t, attrs.WorkloadKind, depWorkloadKind)

	depWorkloadName, ok := attrs.UserDefined[DeprecatedWorkloadNameKey]
	require.True(t, ok)
	assert.Equal(t, attrs.WorkloadName, depWorkloadName)
}
