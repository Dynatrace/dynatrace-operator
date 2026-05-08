package attributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyDeprecatedAttributes(t *testing.T) {
	t.Run("copies workload kind, workload name, and cluster UID to deprecated keys", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.workloadInfo[K8sWorkloadKindAttr] = "deployment"
		attrs.workloadInfo[K8sWorkloadNameAttr] = "my-deployment"
		attrs.clusterInfo[K8sClusterUIDAttr] = "cluster-uid-123"

		attrs.applyDeprecatedAttributes()

		assert.Equal(t, "deployment", attrs.deprecated[DeprecatedWorkloadKindKey])
		assert.Equal(t, "my-deployment", attrs.deprecated[DeprecatedWorkloadNameKey])
		assert.Equal(t, "cluster-uid-123", attrs.deprecated[DeprecatedClusterIDKey])
	})

	t.Run("uses empty string when workload info is not set", func(t *testing.T) {
		attrs := newTestPodAttributes()

		attrs.applyDeprecatedAttributes()

		assert.Equal(t, "", attrs.deprecated[DeprecatedWorkloadKindKey])
		assert.Equal(t, "", attrs.deprecated[DeprecatedWorkloadNameKey])
		assert.Equal(t, "", attrs.deprecated[DeprecatedClusterIDKey])
	})
}
