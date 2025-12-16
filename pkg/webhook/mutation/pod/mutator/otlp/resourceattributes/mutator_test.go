package resourceattributes

import (
	"strings"
	"testing"

	latestdynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_Mutator_Mutate(t *testing.T) { //nolint:gocognit,revive
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
	deploymentOwner := metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "web", Controller: ptr.To(true)}
	replicaSetOwned := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:            "web-1234567890",
		Namespace:       "ns",
		OwnerReferences: []metav1.OwnerReference{deploymentOwner},
	}}

	tests := []struct {
		name           string
		objects        []runtime.Object
		namespace      corev1.Namespace
		pod            *corev1.Pod
		wantAttributes map[string][]string
	}{
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
							Controller: ptr.To(true),
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
							Controller: ptr.To(true),
						},
					},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c1", Env: []corev1.EnvVar{{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: "foo=bar"}}}}},
			},
			wantAttributes: map[string][]string{
				"c1": {
					"foo=bar", // pre-existing
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
			name:    "no workload info when no owners",
			objects: nil,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns"},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
			},
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
			wantAttributes: map[string][]string{
				"c1": {
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
							Controller: ptr.To(true),
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
					"k8s.workload.kind=job",
					"dt.kubernetes.workload.kind=job",
					"k8s.workload.name=jobx",
					"dt.kubernetes.workload.name=jobx",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
				},
				"c2": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-name",
					"dt.entity.kubernetes_cluster=cluster-meid",
					"k8s.container.name=c2",
					"k8s.workload.kind=job",
					"dt.kubernetes.workload.kind=job",
					"k8s.workload.name=jobx",
					"dt.kubernetes.workload.name=jobx",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
				},
			},
		},
		{
			name:      "container excluded via annotation is skipped",
			objects:   nil,
			namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				tt.pod,
				baseDK,
			)
			err := mut.Mutate(req)
			require.NoError(t, err)

			for i := range tt.pod.Spec.Containers {
				container := &tt.pod.Spec.Containers[i]
				var rawResourceAttributes string
				for _, e := range container.Env {
					if e.Name == "OTEL_RESOURCE_ATTRIBUTES" {
						rawResourceAttributes = e.Value

						break
					}
				}

				if len(tt.wantAttributes[container.Name]) == 0 {
					assert.Empty(t, rawResourceAttributes, "container should be skipped, no Attributes injected")
					// also check pod/node env vars are not injected
					for _, envName := range []string{injection.K8sNodeNameEnv, injection.K8sPodNameEnv, injection.K8sPodUIDEnv} {
						assert.False(t, k8senv.Contains(container.Env, envName), "env var %s should not be injected", envName)
					}

					continue
				}

				require.NotEmpty(t, rawResourceAttributes)

				resourceAttributes := strings.Split(rawResourceAttributes, ",")

				assert.Len(t, resourceAttributes, len(tt.wantAttributes[container.Name]), "container should have right amount of attributes")
				for _, expected := range tt.wantAttributes[container.Name] {
					count := 0
					for _, attr := range resourceAttributes {
						if attr == expected {
							count++
						}
					}
					// ensure that each expected attribute appears exactly once
					assert.Equal(t, 1, count, "expected attr %s to appear exactly once; got %v", expected, resourceAttributes)
				}
				// verify env vars for pod/node references present with correct field paths
				var podNameVar, podUIDVar, nodeNameVar *corev1.EnvVar

				for i := range container.Env {
					if container.Env[i].Name == "K8S_PODNAME" {
						podNameVar = &container.Env[i]
					}
					if container.Env[i].Name == "K8S_PODUID" {
						podUIDVar = &container.Env[i]
					}
					if container.Env[i].Name == "K8S_NODE_NAME" {
						nodeNameVar = &container.Env[i]
					}
				}

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
					Controller: ptr.To(true),
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
					Controller: ptr.To(true),
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

	result := mut.Reinvoke(req)
	assert.False(t, result, "Reinvoke should return false when mutation occurs")
}
