package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/stretchr/testify/assert"
)

const (
	expectedBaseInitArgsLen            = 12
	expectedBaseInitArgsLenWithoutMEID = 10
)

func assertResourceAttrArgsAreSorted(t *testing.T, args []string, attrs map[string]string, expectedAttrs []string) {
	t.Helper()

	attrArgs := args[len(args)-len(attrs):]
	assert.Equal(t, expectedAttrs, attrArgs)
}

func Test_getInitArgs(t *testing.T) {
	newDK := func() dynakube.DynaKube {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"

		return dk
	}

	t.Run("get base init args", func(t *testing.T) {
		dk := newDK()
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"

		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen)

		for _, arg := range args {
			assert.NotEmpty(t, arg)
		}
	})

	t.Run("add user defined args to existing init args", func(t *testing.T) {
		dk := newDK()
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Args: []string{
				"customArg1",
				"customArg2",
			},
		}
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen+len(dk.LogMonitoring().Args))

		for _, customArg := range dk.LogMonitoring().Args {
			assert.Contains(t, args, customArg)
		}
	})

	t.Run("get base init args when no MEID or not all necessary scopes are set", func(t *testing.T) {
		dk := newDK()

		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLenWithoutMEID)

		for _, arg := range args {
			assert.NotEmpty(t, arg)
		}
	})

	t.Run("propagate spec.resourceAttributes as -p args", func(t *testing.T) {
		dk := newDK()
		dk.Spec.ResourceAttributes = map[string]string{
			"team":    "platform",
			"env":     "staging",
			"service": "logmodule",
		}
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLenWithoutMEID+len(dk.Spec.ResourceAttributes))

		// Verify resourceAttributes are in sorted order
		assertResourceAttrArgsAreSorted(t, args, dk.Spec.ResourceAttributes, []string{
			"-p env=staging",
			"-p service=logmodule",
			"-p team=platform",
		})
	})

	t.Run("propagate spec.resourceAttributes as -p args together with user-defined args", func(t *testing.T) {
		dk := newDK()
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Args: []string{
				"customArg1",
				"customArg2",
			},
		}
		dk.Spec.ResourceAttributes = map[string]string{
			"team": "platform",
			"env":  "staging",
		}

		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLenWithoutMEID+len(dk.LogMonitoring().Args)+len(dk.Spec.ResourceAttributes))

		for _, customArg := range dk.LogMonitoring().Args {
			assert.Contains(t, args, customArg)
		}

		assertResourceAttrArgsAreSorted(t, args, dk.Spec.ResourceAttributes, []string{
			"-p env=staging",
			"-p team=platform",
		})
	})

	t.Run("propagate spec.resourceAttributes as -p args together with MEID args", func(t *testing.T) {
		dk := newDK()
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"
		dk.Spec.ResourceAttributes = map[string]string{
			"service": "logmodule",
			"team":    "platform",
			"env":     "staging",
		}

		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen+len(dk.Spec.ResourceAttributes))

		assert.Contains(t, args, "-p k8s.cluster.name=$(K8S_CLUSTER_NAME)")
		assert.Contains(t, args, "-p dt.entity.kubernetes_cluster=$(DT_ENTITY_KUBERNETES_CLUSTER)")

		assertResourceAttrArgsAreSorted(t, args, dk.Spec.ResourceAttributes, []string{
			"-p env=staging",
			"-p service=logmodule",
			"-p team=platform",
		})
	})
}
