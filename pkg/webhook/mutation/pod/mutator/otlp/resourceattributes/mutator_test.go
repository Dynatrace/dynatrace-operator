package resourceattributes

import (
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	latestdynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/injection"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type mutatorTestCase struct {
	name           string
	objects        []runtime.Object
	namespace      corev1.Namespace
	pod            *corev1.Pod
	wantAttributes map[string][]string
}

func Test_Mutator_Mutate(t *testing.T) {
	const (
		testSecContextLabel          = "test-security-context-label"
		testCostCenterAnnotation     = "test-cost-center-annotation"
		testCustomMetadataLabel      = "test-custom-metadata-label"
		testCustomMetadataAnnotation = "test-custom-metadata-annotation"
	)
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)

	baseDK := latestdynakube.DynaKube{}
	baseDK.Status.KubeSystemUUID = "cluster-uid"
	baseDK.Status.KubernetesClusterName = "cluster-name"
	baseDK.Status.KubernetesClusterMEID = "cluster-meid"
	baseDK.Status.MetadataEnrichment.Rules = []metadataenrichment.Rule{
		{
			Type:   "LABEL",
			Source: testSecContextLabel,
			Target: "dt.security_context",
		},
		{
			Type:   "LABEL",
			Source: testCustomMetadataLabel,
		},
		{
			Type:   "ANNOTATION",
			Source: testCostCenterAnnotation,
			Target: "dt.cost.costcenter",
		},
		{
			Type:   "ANNOTATION",
			Source: testCustomMetadataAnnotation,
		},
	}

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ns"}}
	deploymentOwner := metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "web", Controller: new(true)}
	replicaSetOwned := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:            "web-1234567890",
		Namespace:       "ns",
		OwnerReferences: []metav1.OwnerReference{deploymentOwner},
	}}

	tests := []mutatorTestCase{
		{
			name: "adds Attributes with deployment workload via replicaset lookup",
			objects: []runtime.Object{
				replicaSetOwned,
				deployment,
			},
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
					Labels: map[string]string{
						testSecContextLabel:     "privileged",
						testCustomMetadataLabel: "custom-namespace-metadata",
					},
					Annotations: map[string]string{
						testCostCenterAnnotation:     "finance",
						testCustomMetadataAnnotation: "custom-namespace-annotation-metadata",
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "ns",
					Annotations: map[string]string{"metadata.dynatrace.com/foo": "bar"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       replicaSetOwned.Name,
							Controller: new(true),
						},
					},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
			},
			wantAttributes: map[string][]string{
				"c1": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c1",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
					"k8s.workload.kind=deployment",
					"dt.kubernetes.workload.kind=deployment",
					"k8s.workload.name=web",
					"dt.kubernetes.workload.name=web",
					"foo=bar",
					"dt.security_context=privileged",
					"dt.cost.costcenter=finance",
					"k8s.namespace.label." + testCustomMetadataLabel + "=custom-namespace-metadata",
					"k8s.namespace.annotation." + testCustomMetadataAnnotation + "=custom-namespace-annotation-metadata",
				},
			},
		},
		{
			name: "preserves existing Attributes and appends new ones (statefulset)",
			objects: []runtime.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "db", Namespace: "ns",
					},
				},
			},
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "ns",
					Annotations: map[string]string{"metadata.dynatrace.com": "xyz"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "StatefulSet",
							Name:       "db",
							Controller: new(true),
						},
					},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1", Env: []corev1.EnvVar{{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: "foo=bar,five=even,john=dow"}}}}},
			},
			wantAttributes: map[string][]string{
				"c1": {
					"foo=bar",   // pre-existing
					"five=even", // pre-existing
					"john=dow",  // pre-existing
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c1",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
					"k8s.workload.kind=statefulset",
					"dt.kubernetes.workload.kind=statefulset",
					"k8s.workload.name=db",
					"dt.kubernetes.workload.name=db",
				},
			},
		},
		{
			name:    "pod is it's own owner",
			objects: nil,

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
			},
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
			wantAttributes: map[string][]string{
				"c1": {
					"k8s.workload.name=pod1",
					"dt.kubernetes.workload.name=pod1",
					"k8s.workload.kind=pod",
					"dt.kubernetes.workload.kind=pod",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c1",
				},
			},
		},
		{
			name:      "multiple containers all mutated (job)",
			objects:   []runtime.Object{&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "jobx", Namespace: "ns"}}},
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "batch/v1",
							Kind:       "Job",
							Name:       "jobx",
							Controller: new(true),
						},
					},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}, {Name: "c2"}}},
			},
			wantAttributes: map[string][]string{
				"c1": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c1",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.workload.kind=job",
					"dt.kubernetes.workload.kind=job",
					"k8s.workload.name=jobx",
					"dt.kubernetes.workload.name=jobx",
					"k8s.node.name=$(K8S_NODE_NAME)",
				},
				"c2": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c2",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.workload.kind=job",
					"dt.kubernetes.workload.kind=job",
					"k8s.workload.name=jobx",
					"dt.kubernetes.workload.name=jobx",
					"k8s.node.name=$(K8S_NODE_NAME)",
				},
			},
		},
		{
			name:    "container excluded via annotation is skipped",
			objects: nil,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "ns",
					Annotations: map[string]string{"container.inject.dynatrace.com/c1": "false"},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
			},
			wantAttributes: map[string][]string{
				"c1": {}, // should be empty, nothing injected
			},
		},
	}

	t.Run("with deprecated annotations", func(t *testing.T) {
		dk := baseDK.DeepCopy()
		dk.Annotations = map[string]string{}
		runMutatorTests(t, *dk, tests, false)
	})

	t.Run("without deprecated annotations", func(t *testing.T) {
		dk := baseDK.DeepCopy()
		dk.Annotations = map[string]string{exp.EnrichmentEnableAttributesDTKubernetes: "false"}
		runMutatorTests(t, *dk, tests, true)
	})
}

func runMutatorTests(t *testing.T, dk latestdynakube.DynaKube, tests []mutatorTestCase, removeDeprecatedAttr bool) { //nolint:revive
	t.Helper()

	removeDTKubernetesAnnotations := func(attributes map[string][]string) map[string][]string {
		attributesCopy := maps.Clone(attributes)
		for containerName, attrs := range attributesCopy {
			filteredAttrs := slices.DeleteFunc(attrs, func(attr string) bool {
				return strings.HasPrefix(attr, "dt.kubernetes.")
			})
			attributesCopy[containerName] = filteredAttrs
		}

		return attributesCopy
	}

	for _, tt := range tests {
		wantAttributes := tt.wantAttributes
		if removeDeprecatedAttr {
			wantAttributes = removeDTKubernetesAnnotations(tt.wantAttributes)
		}

		t.Run(tt.name, func(t *testing.T) {
			pod := tt.pod.DeepCopy()

			builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
			if tt.objects != nil {
				builder = builder.WithRuntimeObjects(tt.objects...)
			}
			client := builder.Build()
			mut := New(client)

			req := dtwebhook.NewMutationRequest(
				t.Context(),
				tt.namespace,
				nil,
				pod,
				dk,
			)
			err := mut.Mutate(req)
			require.NoError(t, err)

			require.Len(t, pod.Spec.Containers, len(wantAttributes))

			for _, container := range pod.Spec.Containers {
				var resourceAttributes []string
				if env := k8senv.Find(container.Env, "OTEL_RESOURCE_ATTRIBUTES"); env != nil {
					resourceAttributes = slices.Sorted(strings.SplitSeq(env.Value, ","))
				}

				if len(wantAttributes[container.Name]) == 0 {
					assert.Empty(t, resourceAttributes, "container should be skipped, no Attributes injected")
					// also check pod/node env vars are not injected
					for _, envName := range []string{injection.K8sNodeNameEnv, injection.K8sPodNameEnv, injection.K8sPodUIDEnv} {
						assert.False(t, k8senv.Contains(container.Env, envName), "env var %s should not be injected", envName)
					}

					continue
				}

				require.NotEmpty(t, resourceAttributes)
				assert.Equal(t, resourceAttributes, slices.Compact(slices.Clone(resourceAttributes)), "contains duplicate elements")
				assert.Len(t, resourceAttributes, len(wantAttributes[container.Name]), "container should have right amount of attributes")

				for _, expected := range wantAttributes[container.Name] {
					assert.Contains(t, resourceAttributes, expected)
				}

				// verify env vars for pod/node references present with correct field paths
				podNameVar := k8senv.Find(container.Env, "K8S_PODNAME")
				podUIDVar := k8senv.Find(container.Env, "K8S_PODUID")
				nodeNameVar := k8senv.Find(container.Env, "K8S_NODE_NAME")

				require.NotNil(t, podNameVar, "missing K8S_PODNAME env var")
				require.NotNil(t, podUIDVar, "missing K8S_PODUID env var")
				require.NotNil(t, nodeNameVar, "missing K8S_NODE_NAME env var")

				assert.Equal(t, "metadata.name", podNameVar.ValueFrom.FieldRef.FieldPath)
				assert.Equal(t, "metadata.uid", podUIDVar.ValueFrom.FieldRef.FieldPath)
				assert.Equal(t, "spec.nodeName", nodeNameVar.ValueFrom.FieldRef.FieldPath)
			}
		})
	}
}

func Test_Mutator_EncodesAttributeValues(t *testing.T) {
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name        string
		clusterName string
		annotations map[string]string
		expected    string
	}{
		{
			name:        "cluster name with special chars",
			clusterName: "bh-eks-test1 with space=equals,comma",
			expected:    "k8s.cluster.name=" + url.QueryEscape("bh-eks-test1 with space=equals,comma"),
		},
		{
			name:        "env ref in pod annotation value",
			clusterName: "cluster-name",
			annotations: map[string]string{"metadata.dynatrace.com/booom": "$(DT_API_TOKEN)"},
			expected:    "booom=" + url.QueryEscape("$(DT_API_TOKEN)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDK := latestdynakube.DynaKube{}
			baseDK.Status.KubeSystemUUID = "cluster-uid"
			baseDK.Status.KubernetesClusterName = tt.clusterName
			baseDK.Status.KubernetesClusterMEID = "cluster-meid"

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Annotations: tt.annotations},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
			}
			namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}}

			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			req := dtwebhook.NewMutationRequest(t.Context(), namespace, nil, pod, baseDK)
			require.NoError(t, New(client).Mutate(req))

			resourceAttributes := k8senv.Find(pod.Spec.Containers[0].Env, "OTEL_RESOURCE_ATTRIBUTES").Value
			require.NotEmpty(t, resourceAttributes, "OTEL_RESOURCE_ATTRIBUTES must be set")
			assert.Contains(t, resourceAttributes, tt.expected)
		})
	}
}

// Abort mutation if owner reference cannot be resolved, be consistent with metadata mutator
func Test_Mutator_MutateNoOwner(t *testing.T) {
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       "ghost-rs",
					Controller: new(true),
				},
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
	}
	baseDK := latestdynakube.DynaKube{}
	builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
	client := builder.Build()
	mut := New(client)
	req := dtwebhook.NewMutationRequest(
		t.Context(),
		namespace,
		nil,
		pod,
		baseDK,
	)
	err := mut.Mutate(req)
	require.Error(t, err)
}

func Test_Mutator_Reinvoke(t *testing.T) {
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)

	baseDK := latestdynakube.DynaKube{}
	baseDK.Status.KubeSystemUUID = "cluster-uid"
	baseDK.Status.KubernetesClusterName = "cluster-name"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Name:       "db",
					Controller: new(true),
				},
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
	}

	builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
	client := builder.Build()
	mut := New(client)

	req := &dtwebhook.ReinvocationRequest{
		BaseRequest: &dtwebhook.BaseRequest{
			Pod:       pod,
			DynaKube:  baseDK,
			Namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: pod.Namespace}},
		},
	}

	result := mut.Reinvoke(t.Context(), req)
	assert.False(t, result, "Reinvoke should return false when mutation occurs")
}

func TestMutate_OTLPResourceAttributes(t *testing.T) {
	const (
		testNamespace = "ns"
		containerName = "c1"
		globalKey     = "global-key"
		globalValue   = "global-value"
		otlpKey       = "otlp-key"
		otlpValue     = "otlp-value"
		collisionKey  = "collision-key"
		globalCollVal = "global-collision-value"
		otlpCollVal   = "otlp-collision-value"
		containerVal  = "container-value"
	)

	baseNamespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}

	newPod := func(containerEnv ...corev1.EnvVar) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: testNamespace},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: containerName, Env: containerEnv}},
			},
		}
	}

	type testCase struct {
		name        string
		dk          latestdynakube.DynaKube
		pod         *corev1.Pod
		wantAttrs   []string
		notWantKeys []string
	}

	cases := []testCase{
		{
			name: "no Dynakube resource attributes, only operator semantic attrs present",
			dk:   latestdynakube.DynaKube{},
			pod:  newPod(),
			wantAttrs: []string{
				"k8s.namespace.name=ns",
				"k8s.pod.name=$(K8S_PODNAME)",
			},
		},
		{
			name: "global resource attributes applied",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					ResourceAttributes: map[string]string{globalKey: globalValue},
				},
			},
			pod:       newPod(),
			wantAttrs: []string{globalKey + "=" + globalValue},
		},
		{
			name: "OTLP additionalResourceAttributes applied",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
						AdditionalResourceAttributes: map[string]string{otlpKey: otlpValue},
					},
				},
			},
			pod:       newPod(),
			wantAttrs: []string{otlpKey + "=" + otlpValue},
		},
		{
			name: "both Dynakube fields set, key collision - OTLP-additional wins",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					ResourceAttributes: map[string]string{collisionKey: globalCollVal},
					OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
						AdditionalResourceAttributes: map[string]string{collisionKey: otlpCollVal},
					},
				},
			},
			pod:         newPod(),
			wantAttrs:   []string{collisionKey + "=" + otlpCollVal},
			notWantKeys: []string{collisionKey + "=" + globalCollVal},
		},
		{
			name: "Dynakube field collides with operator semantic key - user wins",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					ResourceAttributes: map[string]string{"k8s.namespace.name": "user-override"},
				},
			},
			pod:         newPod(),
			wantAttrs:   []string{"k8s.namespace.name=user-override"},
			notWantKeys: []string{"k8s.namespace.name=" + testNamespace},
		},
		{
			name: "container pre-existing OTEL_RESOURCE_ATTRIBUTES wins on collision with Dynakube",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					ResourceAttributes: map[string]string{collisionKey: globalCollVal},
				},
			},
			pod: newPod(corev1.EnvVar{
				Name:  OTELResourceAttributesEnv,
				Value: collisionKey + "=" + containerVal,
			}),
			wantAttrs:   []string{collisionKey + "=" + containerVal},
			notWantKeys: []string{collisionKey + "=" + globalCollVal},
		},
		{
			name: "empty key and empty value in Dynakube field are filtered out",
			dk: latestdynakube.DynaKube{
				Spec: latestdynakube.DynaKubeSpec{
					ResourceAttributes: map[string]string{
						"":          "empty-key-value",
						"empty-val": "",
						globalKey:   globalValue,
					},
				},
			},
			pod:       newPod(),
			wantAttrs: []string{globalKey + "=" + globalValue},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pod := tc.pod.DeepCopy()

			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			mut := New(client)

			req := dtwebhook.NewMutationRequest(t.Context(), baseNamespace, nil, pod, tc.dk)
			err := mut.Mutate(req)
			require.NoError(t, err)

			require.NotNil(t, req.AnnotationWriter, "AnnotationWriter must be set for deferred annotation writing")

			container := pod.Spec.Containers[0]
			env := k8senv.Find(container.Env, OTELResourceAttributesEnv)
			require.NotNil(t, env, "OTEL_RESOURCE_ATTRIBUTES must be set")

			attrs := slices.Sorted(strings.SplitSeq(env.Value, ","))

			for _, want := range tc.wantAttrs {
				assert.Contains(t, attrs, want, "expected attribute to be present")
			}
			for _, notWant := range tc.notWantKeys {
				assert.NotContains(t, attrs, notWant, "expected attribute to NOT be present")
			}
		})
	}
}

// TestMutate_AnnotationWriter verifies that annotation writing is deferred via AnnotationWriter
// and that OTLP additionalResourceAttributes win in the JSON annotation even when the pod already
// carries metadata.dynatrace.com/ annotations (as the metadata mutator would have written them
// before this mutator runs in the old, unfixed flow).
func TestMutate_AnnotationWriter(t *testing.T) {
	const (
		testNamespace = "ns"
		collisionKey  = "collision-key"
		globalCollVal = "global-collision-value"
		otlpCollVal   = "otlp-collision-value"
	)

	_ = appsv1.AddToScheme(scheme.Scheme)

	dk := latestdynakube.DynaKube{
		Spec: latestdynakube.DynaKubeSpec{
			ResourceAttributes: map[string]string{collisionKey: globalCollVal},
			OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
				AdditionalResourceAttributes: map[string]string{collisionKey: otlpCollVal},
			},
		},
	}
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}

	t.Run("AnnotationWriter is set after Mutate", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: testNamespace},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
		}

		client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		req := dtwebhook.NewMutationRequest(t.Context(), namespace, nil, pod, dk)

		require.NoError(t, New(client).Mutate(req))
		require.NotNil(t, req.AnnotationWriter)
	})

	t.Run("OTLP additionalResourceAttributes win in JSON annotation (clean pod)", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: testNamespace},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
		}

		client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		req := dtwebhook.NewMutationRequest(t.Context(), namespace, nil, pod, dk)

		require.NoError(t, New(client).Mutate(req))
		require.NotNil(t, req.AnnotationWriter)
		require.NoError(t, req.AnnotationWriter.ApplyAnnotationsToPod(pod))

		jsonAnnotation := pod.Annotations[metadataenrichment.Annotation]
		require.NotEmpty(t, jsonAnnotation)

		var annotationAttrs map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonAnnotation), &annotationAttrs))

		assert.Equal(t, otlpCollVal, annotationAttrs[collisionKey], "OTLP additionalResourceAttributes must win in JSON annotation")
		assert.NotEqual(t, globalCollVal, annotationAttrs[collisionKey])
	})

	t.Run("user-set metadata.dynatrace.com/ annotation still wins over dynakube attrs", func(t *testing.T) {
		const userValue = "user-set-value"

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: testNamespace,
				Annotations: map[string]string{
					metadataenrichment.Prefix + collisionKey: userValue,
				},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
		}

		client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		req := dtwebhook.NewMutationRequest(t.Context(), namespace, nil, pod, dk)

		require.NoError(t, New(client).Mutate(req))

		env := k8senv.Find(pod.Spec.Containers[0].Env, OTELResourceAttributesEnv)
		require.NotNil(t, env)
		assert.Contains(t, env.Value, collisionKey+"="+userValue, "user pod annotation must win over dynakube attrs")
	})
}
