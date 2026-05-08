package attributes

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newTestPodAttributes creates a PodAttributes with all maps initialized so tests can set
// individual fields without triggering nil-map panics.
func newTestPodAttributes() *PodAttributes {
	return &PodAttributes{
		rules:                make(map[string]string),
		rulesPropagate:       make(map[string]string),
		namespaceAnnotations: make(map[string]string),
		podAnnotations:       make(map[string]string),
		custom:               make(map[string]string),
		workloadInfo:         make(map[string]string),
		clusterInfo:          make(map[string]string),
		podInfo:              make(map[string]string),
		deprecated:           make(map[string]string),
		podEnvVars:           []corev1.EnvVar{},
	}
}

// toResultMap converts the slice of "key=value" strings produced by Convert into a map.
func toResultMap(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, _ := strings.Cut(p, "=")
		m[k] = v
	}

	return m
}

// simpleConvertFunc formats each attribute as "key=value".
func simpleConvertFunc(k, v string) string { return k + "=" + v }

// ---- AddCustomAttribute / AddCustomAttributes / GetPodEnvVars / Convert ----

func TestAddCustomAttribute(t *testing.T) {
	t.Run("adds a single key-value to custom", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.AddCustomAttribute("my.key", "my-value")
		assert.Equal(t, "my-value", attrs.custom["my.key"])
	})

	t.Run("overwrites existing custom key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.AddCustomAttribute("my.key", "first")
		attrs.AddCustomAttribute("my.key", "second")
		assert.Equal(t, "second", attrs.custom["my.key"])
	})
}

func TestAddCustomAttributes(t *testing.T) {
	t.Run("bulk copies all entries into custom", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.AddCustomAttributes(map[string]string{"a": "1", "b": "2"})
		assert.Equal(t, "1", attrs.custom["a"])
		assert.Equal(t, "2", attrs.custom["b"])
	})

	t.Run("does not touch unrelated existing custom keys", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.custom["existing"] = "kept"
		attrs.AddCustomAttributes(map[string]string{"new": "val"})
		assert.Equal(t, "kept", attrs.custom["existing"])
		assert.Equal(t, "val", attrs.custom["new"])
	})
}

func TestGetPodEnvVars(t *testing.T) {
	t.Run("returns the internal podEnvVars slice", func(t *testing.T) {
		attrs := newTestPodAttributes()
		env := corev1.EnvVar{Name: "FOO", Value: "bar"}
		attrs.podEnvVars = append(attrs.podEnvVars, env)
		result := attrs.GetPodEnvVars()
		require.Len(t, result, 1)
		assert.Equal(t, env, result[0])
	})

	t.Run("returns empty slice when no env vars set", func(t *testing.T) {
		attrs := newTestPodAttributes()
		assert.Empty(t, attrs.GetPodEnvVars())
	})
}

func TestConvert_Method(t *testing.T) {
	t.Run("combines and converts attributes to key=value strings", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.workloadInfo["k8s.workload.kind"] = "deployment"
		attrs.custom["my.attr"] = "custom-val"

		result := attrs.Convert(simpleConvertFunc)

		m := toResultMap(result)
		assert.Equal(t, "deployment", m["k8s.workload.kind"])
		assert.Equal(t, "custom-val", m["my.attr"])
	})

	t.Run("passes ContainerAttributes into the result", func(t *testing.T) {
		attrs := newTestPodAttributes()
		container := ContainerAttributes{ContainerName: "my-container"}

		result := attrs.Convert(simpleConvertFunc, container)

		m := toResultMap(result)
		assert.Equal(t, "my-container", m[K8sContainerNameAttr])
	})
}

// ---- combineAll() precedence tests ----
//
// Each sub-test sets the SAME key in two adjacent precedence levels.
// The higher-priority source must win.
//
// Order (lowest → highest):
//   deprecated → workloadInfo → podInfo → clusterInfo
//   → container → rules → rulesPropagate
//   → namespaceAnnotations → podAnnotations → custom

func TestCombine_WorkloadInfoWinsOverDeprecated(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.useDeprecated = true
	attrs.deprecated["shared.key"] = "from-deprecated"
	attrs.workloadInfo["shared.key"] = "from-workload"

	result := attrs.combineAll()

	assert.Equal(t, "from-workload", result["shared.key"])
}

func TestCombine_PodInfoWinsOverWorkloadInfo(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.workloadInfo["shared.key"] = "from-workload"
	attrs.podInfo["shared.key"] = "from-pod-info"

	result := attrs.combineAll()

	assert.Equal(t, "from-pod-info", result["shared.key"])
}

func TestCombine_ClusterInfoWinsOverPodInfo(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.podInfo["shared.key"] = "from-pod-info"
	attrs.clusterInfo["shared.key"] = "from-cluster"

	result := attrs.combineAll()

	assert.Equal(t, "from-cluster", result["shared.key"])
}

func TestCombine_ContainerWinsOverClusterInfo(t *testing.T) {
	attrs := newTestPodAttributes()
	// use the actual container name key for an apples-to-apples comparison
	attrs.clusterInfo[K8sContainerNameAttr] = "from-cluster"
	container := ContainerAttributes{ContainerName: "from-container"}

	result := attrs.combineAll(container)

	assert.Equal(t, "from-container", result[K8sContainerNameAttr])
}

func TestCombine_RulesWinsOverContainer(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.rules[K8sContainerNameAttr] = "from-rules"
	container := ContainerAttributes{ContainerName: "from-container"}

	result := attrs.combineAll(container)

	assert.Equal(t, "from-rules", result[K8sContainerNameAttr])
}

func TestCombine_RulesPropagateWinsOverRules(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.rules["shared.key"] = "from-rules"
	attrs.rulesPropagate["shared.key"] = "from-rules-propagate"

	result := attrs.combineAll()

	assert.Equal(t, "from-rules-propagate", result["shared.key"])
}

func TestCombine_NamespaceAnnotationWinsOverRulesPropagate(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.rulesPropagate["shared.key"] = "from-rules-propagate"
	attrs.namespaceAnnotations["shared.key"] = "from-namespace"

	result := attrs.combineAll()

	assert.Equal(t, "from-namespace", result["shared.key"])
}

func TestCombine_PodAnnotationWinsOverNamespaceAnnotation(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.namespaceAnnotations["shared.key"] = "from-namespace"
	attrs.podAnnotations["shared.key"] = "from-pod"

	result := attrs.combineAll()

	assert.Equal(t, "from-pod", result["shared.key"])
}

func TestCombine_CustomWinsOverPodAnnotation(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.podAnnotations["shared.key"] = "from-pod"
	attrs.custom["shared.key"] = "from-custom"

	result := attrs.combineAll()

	assert.Equal(t, "from-custom", result["shared.key"])
}

func TestCombine_DeprecatedExcludedWhenDisabled(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.useDeprecated = false
	attrs.deprecated[DeprecatedWorkloadKindKey] = "some-kind"

	result := attrs.combineAll()

	assert.NotContains(t, result, DeprecatedWorkloadKindKey)
}

func TestCombine_DeprecatedIncludedAndOverriddenWhenEnabled(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.useDeprecated = true
	attrs.deprecated[DeprecatedWorkloadKindKey] = "from-deprecated"
	attrs.workloadInfo[DeprecatedWorkloadKindKey] = "from-workload"

	result := attrs.combineAll()

	// deprecated is present but the same key in workloadInfo wins
	assert.Equal(t, "from-workload", result[DeprecatedWorkloadKindKey])
}

func TestCombine_UniqueKeysFromAllSourcesMerged(t *testing.T) {
	attrs := newTestPodAttributes()
	attrs.useDeprecated = true
	attrs.deprecated["dep.key"] = "dep-val"
	attrs.workloadInfo["workload.key"] = "workload-val"
	attrs.podInfo["pod.key"] = "pod-val"
	attrs.clusterInfo["cluster.key"] = "cluster-val"
	attrs.rules["rules.key"] = "rules-val"
	attrs.rulesPropagate["rulespropagate.key"] = "rulespropagate-val"
	attrs.namespaceAnnotations["ns.key"] = "ns-val"
	attrs.podAnnotations["pod-anno.key"] = "pod-anno-val"
	attrs.custom["custom.key"] = "custom-val"
	container := ContainerAttributes{ContainerName: "my-container"}

	result := attrs.combineAll(container)

	assert.Equal(t, "dep-val", result["dep.key"])
	assert.Equal(t, "workload-val", result["workload.key"])
	assert.Equal(t, "pod-val", result["pod.key"])
	assert.Equal(t, "cluster-val", result["cluster.key"])
	assert.Equal(t, "my-container", result[K8sContainerNameAttr])
	assert.Equal(t, "rules-val", result["rules.key"])
	assert.Equal(t, "rulespropagate-val", result["rulespropagate.key"])
	assert.Equal(t, "ns-val", result["ns.key"])
	assert.Equal(t, "pod-anno-val", result["pod-anno.key"])
	assert.Equal(t, "custom-val", result["custom.key"])
}

func TestCombine_MultipleContainerAttrs_LaterOneWins(t *testing.T) {
	attrs := newTestPodAttributes()
	first := ContainerAttributes{ContainerName: "first-container"}
	second := ContainerAttributes{ContainerName: "second-container"}

	result := attrs.combineAll(first, second)

	assert.Equal(t, "second-container", result[K8sContainerNameAttr])
}

func TestCombine_EmptyAttributes(t *testing.T) {
	attrs := newTestPodAttributes()

	result := attrs.combineAll()

	assert.Empty(t, result)
}

// ---- Constructor-based end-to-end precedence test ----
//
// This test uses NewPodAttributes (the real constructor) to verify that attribute
// collection and precedence work correctly end-to-end. In particular it confirms
// that namespace and pod "metadata.dynatrace.com/" annotations override attributes
// that are otherwise automatically collected (k8s.pod.name, k8s.cluster.uid, etc.).

func TestCombine_ViaConstructors_AnnotationsOverrideAutoCollected(t *testing.T) {
	const (
		podName       = "my-pod"
		namespaceName = "my-ns"
		clusterUID    = "real-cluster-uid"
		clusterName   = "real-cluster-name"
		clusterMEID   = "KUBERNETES_CLUSTER-REAL"
	)

	makePod := func(podAnnotations map[string]string) *corev1.Pod {
		return &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        podName,
				Namespace:   namespaceName,
				Annotations: podAnnotations,
			},
		}
	}

	makeDynaKube := func() dynakube.DynaKube {
		return dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID:        clusterUID,
				KubernetesClusterName: clusterName,
				KubernetesClusterMEID: clusterMEID,
			},
		}
	}

	t.Run("namespace annotation overrides auto-collected cluster attributes", func(t *testing.T) {
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
				Annotations: map[string]string{
					metadataenrichment.Prefix + K8sClusterUIDAttr:  "uid-from-ns",
					metadataenrichment.Prefix + K8sClusterNameAttr: "name-from-ns",
				},
			},
		}
		request := dtwebhook.BaseRequest{
			Pod:       makePod(nil),
			Namespace: ns,
			DynaKube:  makeDynaKube(),
		}

		attrs, err := NewPodAttributes(t.Context(), request, fake.NewClient())
		require.NoError(t, err)

		container := *NewContainerAttributes(corev1.Container{Name: "my-container"})
		result := toResultMap(attrs.Convert(simpleConvertFunc, container))

		// namespace annotation (precedence 8) beats clusterInfo (precedence 4)
		assert.Equal(t, "uid-from-ns", result[K8sClusterUIDAttr])
		assert.Equal(t, "name-from-ns", result[K8sClusterNameAttr])
		// unoverridden auto-collected cluster attribute survives
		assert.Equal(t, clusterMEID, result[K8sDTClusterEntityAttr])
	})

	t.Run("pod annotation overrides auto-collected pod attributes", func(t *testing.T) {
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
		request := dtwebhook.BaseRequest{
			Pod: makePod(map[string]string{
				metadataenrichment.Prefix + K8sPodNameAttr: "overridden-pod-name",
			}),
			Namespace: ns,
			DynaKube:  makeDynaKube(),
		}

		attrs, err := NewPodAttributes(t.Context(), request, fake.NewClient())
		require.NoError(t, err)

		container := *NewContainerAttributes(corev1.Container{Name: "my-container"})
		result := toResultMap(attrs.Convert(simpleConvertFunc, container))

		// pod annotation (precedence 9) beats podInfo (precedence 3)
		assert.Equal(t, "overridden-pod-name", result[K8sPodNameAttr])
	})

	t.Run("pod annotation beats namespace annotation for the same key", func(t *testing.T) {
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
				Annotations: map[string]string{
					metadataenrichment.Prefix + K8sClusterUIDAttr: "uid-from-ns",
				},
			},
		}
		request := dtwebhook.BaseRequest{
			Pod: makePod(map[string]string{
				metadataenrichment.Prefix + K8sClusterUIDAttr: "uid-from-pod",
			}),
			Namespace: ns,
			DynaKube:  makeDynaKube(),
		}

		attrs, err := NewPodAttributes(t.Context(), request, fake.NewClient())
		require.NoError(t, err)

		result := toResultMap(attrs.Convert(simpleConvertFunc))

		// pod annotation (9) beats namespace annotation (8) beats clusterInfo (4)
		assert.Equal(t, "uid-from-pod", result[K8sClusterUIDAttr])
	})

	t.Run("custom attribute beats pod annotation for the same key", func(t *testing.T) {
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
		request := dtwebhook.BaseRequest{
			Pod: makePod(map[string]string{
				metadataenrichment.Prefix + K8sClusterUIDAttr: "uid-from-pod",
			}),
			Namespace: ns,
			DynaKube:  makeDynaKube(),
		}

		attrs, err := NewPodAttributes(t.Context(), request, fake.NewClient())
		require.NoError(t, err)
		attrs.AddCustomAttribute(K8sClusterUIDAttr, "uid-from-custom")

		result := toResultMap(attrs.Convert(simpleConvertFunc))

		// custom (10) beats pod annotation (9)
		assert.Equal(t, "uid-from-custom", result[K8sClusterUIDAttr])
	})

	t.Run("auto-collected attributes not touched by annotations survive unchanged", func(t *testing.T) {
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
		request := dtwebhook.BaseRequest{
			Pod:       makePod(nil),
			Namespace: ns,
			DynaKube:  makeDynaKube(),
		}

		attrs, err := NewPodAttributes(t.Context(), request, fake.NewClient())
		require.NoError(t, err)

		container := *NewContainerAttributes(corev1.Container{Name: "my-container"})
		result := toResultMap(attrs.Convert(simpleConvertFunc, container))

		assert.Equal(t, namespaceName, result[K8sNamespaceNameAttr])
		assert.Equal(t, clusterUID, result[K8sClusterUIDAttr])
		assert.Equal(t, clusterName, result[K8sClusterNameAttr])
		assert.Equal(t, clusterMEID, result[K8sDTClusterEntityAttr])
		assert.Equal(t, "my-container", result[K8sContainerNameAttr])
		// workload kind is "pod" because the pod has no owner references
		assert.Equal(t, "pod", result[K8sWorkloadKindAttr])
		assert.Equal(t, podName, result[K8sWorkloadNameAttr])
	})
}
