package attributes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateEnvVarRef(t *testing.T) {
	t.Run("wraps env name in $() syntax", func(t *testing.T) {
		assert.Equal(t, "$(K8S_PODNAME)", createEnvVarRef("K8S_PODNAME"))
		assert.Equal(t, "$(MY_VAR)", createEnvVarRef("MY_VAR"))
	})
}

func TestGetPodAttributes(t *testing.T) {
	t.Run("appends three env vars with field-path references", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
			},
			DynaKube: dynakube.DynaKube{},
		}

		attrs.readPodAttributes(request)

		require.Len(t, attrs.podEnvVars, 3)

		podNameEnv := k8senv.Find(attrs.podEnvVars, K8sPodNameEnv)
		require.NotNil(t, podNameEnv)
		require.NotNil(t, podNameEnv.ValueFrom)
		assert.Equal(t, "metadata.name", podNameEnv.ValueFrom.FieldRef.FieldPath)

		podUIDEnv := k8senv.Find(attrs.podEnvVars, K8sPodUIDEnv)
		require.NotNil(t, podUIDEnv)
		require.NotNil(t, podUIDEnv.ValueFrom)
		assert.Equal(t, "metadata.uid", podUIDEnv.ValueFrom.FieldRef.FieldPath)

		nodeNameEnv := k8senv.Find(attrs.podEnvVars, K8sNodeNameEnv)
		require.NotNil(t, nodeNameEnv)
		require.NotNil(t, nodeNameEnv.ValueFrom)
		assert.Equal(t, "spec.nodeName", nodeNameEnv.ValueFrom.FieldRef.FieldPath)
	})

	t.Run("sets podInfo with env var references and namespace name", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
			},
			DynaKube: dynakube.DynaKube{},
		}

		attrs.readPodAttributes(request)

		assert.Equal(t, "$(K8S_PODNAME)", attrs.podInfo[K8sPodNameAttr])
		assert.Equal(t, "$(K8S_PODUID)", attrs.podInfo[K8sPodUIDAttr])
		assert.Equal(t, "$(K8S_NODE_NAME)", attrs.podInfo[K8sNodeNameAttr])
		assert.Equal(t, "my-ns", attrs.podInfo[K8sNamespaceNameAttr])
	})

	t.Run("sets clusterInfo from DynaKube status", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns"}},
			DynaKube: dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					KubeSystemUUID:        "uid-123",
					KubernetesClusterName: "my-cluster",
					KubernetesClusterMEID: "KUBERNETES_CLUSTER-ABC",
				},
			},
		}

		attrs.readPodAttributes(request)

		assert.Equal(t, "uid-123", attrs.clusterInfo[K8sClusterUIDAttr])
		assert.Equal(t, "my-cluster", attrs.clusterInfo[K8sClusterNameAttr])
		assert.Equal(t, "KUBERNETES_CLUSTER-ABC", attrs.clusterInfo[K8sDTClusterEntityAttr])
	})
}
