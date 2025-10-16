package resourceattributes

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	"strings"
	"testing"

	latestdynakube "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_getWorkloadInfo(t *testing.T) {
	// Ensure schemes are registered
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)

	deploymentOwner := metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "my-deploy", Controller: ptr.To(true)}
	replicaSetWithDeployment := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deploy-1234567890",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{
				deploymentOwner,
			},
		},
	}

	tests := []struct {
		name        string
		objects     []runtime.Object
		pod         *corev1.Pod
		wantKind    string
		wantName    string
		expectError bool
	}{
		{
			name:     "nil pod",
			objects:  nil,
			pod:      nil,
			wantKind: "",
			wantName: "",
		},
		{
			name:     "replicaset owned by deployment (api lookup)",
			objects:  []runtime.Object{replicaSetWithDeployment},
			pod:      &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: replicaSetWithDeployment.Name, Controller: ptr.To(true)}}}},
			wantKind: "Deployment",
			wantName: "my-deploy",
		},
		{
			name:        "replicaset fallback when not found",
			objects:     nil, // no RS object -> fallback
			pod:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "missing-rs", Controller: ptr.To(true)}}}},
			wantKind:    "",
			wantName:    "",
			expectError: true,
		},
		{
			name:     "statefulset direct",
			objects:  nil,
			pod:      &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "StatefulSet", Name: "db", Controller: ptr.To(true)}}}},
			wantKind: "StatefulSet",
			wantName: "db",
		},
		{
			name:     "non-controller owner ignored",
			objects:  nil,
			pod:      &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", OwnerReferences: []metav1.OwnerReference{{APIVersion: "batch/v1", Kind: "Job", Name: "job1", Controller: nil}}}},
			wantKind: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
			if tt.objects != nil {
				builder = builder.WithRuntimeObjects(tt.objects...)
			}
			client := builder.Build()
			kind, name, err := getWorkloadInfo(context.Background(), client, tt.pod)

			if tt.expectError {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantKind, kind)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestMutatorMutate(t *testing.T) { //nolint:gocognit
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)

	baseDK := latestdynakube.DynaKube{}
	baseDK.Status.KubeSystemUUID = "cluster-uid"
	baseDK.Status.KubernetesClusterName = "cluster-name"

	deploymentOwner := metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "web", Controller: ptr.To(true)}
	replicaSetOwned := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "web-1234567890", Namespace: "ns", OwnerReferences: []metav1.OwnerReference{deploymentOwner}}}

	tests := []struct {
		name           string
		objects        []runtime.Object
		pod            *corev1.Pod
		wantAttributes map[string][]string
	}{
		{
			name:    "adds attributes with deployment workload via replicaset lookup",
			objects: []runtime.Object{replicaSetOwned},
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
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c1",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
					"k8s.workload.kind=Deployment",
					"k8s.workload.name=web",
					"foo=bar",
				},
			},
		},
		{
			name:    "preserves existing attributes and appends new ones (statefulset)",
			objects: nil,
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
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c1",
					"k8s.pod.name=$(K8S_PODNAME)",
					"k8s.pod.uid=$(K8S_PODUID)",
					"k8s.node.name=$(K8S_NODE_NAME)",
					"k8s.workload.kind=StatefulSet",
					"k8s.workload.name=db",
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
			wantAttributes: map[string][]string{
				"c1": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c1",
				},
			},
		},
		{
			name:    "multiple containers all mutated (job)",
			objects: nil,
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
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c1",
					"k8s.workload.kind=Job",
					"k8s.workload.name=jobx",
				},
				"c2": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c2",
					"k8s.workload.kind=Job",
					"k8s.workload.name=jobx",
				},
			},
		},
		{
			name:    "RS owner missing (no deployment) - add other attributes",
			objects: nil,
			pod: &corev1.Pod{
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
			},
			wantAttributes: map[string][]string{
				"c1": {
					"k8s.namespace.name=ns",
					"k8s.cluster.uid=cluster-uid",
					"dt.kubernetes.cluster.id=cluster-uid",
					"k8s.cluster.name=cluster-uid",
					"dt.kubernetes.cluster.name=cluster-name",
					"k8s.container.name=c1",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
			if tt.objects != nil {
				builder = builder.WithRuntimeObjects(tt.objects...)
			}
			client := builder.Build()
			mut := New(client)

			req := dtwebhook.NewMutationRequest(
				context.Background(),
				corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tt.pod.Namespace}},
				nil,
				tt.pod,
				baseDK,
			)
			err := mut.Mutate(req)
			require.NoError(t, err)

			for i := range tt.pod.Spec.Containers {
				container := &tt.pod.Spec.Containers[i]
				var val string
				for _, e := range container.Env {
					if e.Name == "OTEL_RESOURCE_ATTRIBUTES" {
						val = e.Value
						break
					}
				}

				if len(tt.wantAttributes[container.Name]) == 0 {
					assert.Empty(t, val, "container should be skipped, no attributes injected")
					// also check pod/node env vars are not injected
					for _, envName := range []string{"K8S_PODNAME", "K8S_PODUID", "K8S_NODE_NAME"} {
						assert.False(t, env.IsIn(container.Env, envName), "env var %s should not be injected", envName)
					}
					continue
				}

				require.NotEmpty(t, val)

				attrs := strings.Split(val, ",")

				for _, expected := range tt.wantAttributes[container.Name] {
					assert.Contains(t, attrs, expected, "expected attr %s missing; got %v", expected, attrs)
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

func TestMutatorReinvoke(t *testing.T) {
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
	assert.True(t, result, "Reinvoke should return true when mutation occurs")
}

func Test_appendAttribute(t *testing.T) {

	b := &strings.Builder{}

	b.WriteString("key=value")

	type args struct {
		b        *strings.Builder
		existing *corev1.EnvVar
		key      string
		value    string
	}
	tests := []struct {
		name            string
		args            args
		want            bool
		wantEnvVarValue string
	}{
		{
			name: "empty value",
			args: args{
				b:        &strings.Builder{},
				existing: nil,
				key:      "key1",
				value:    "",
			},
			want:            false,
			wantEnvVarValue: "",
		},
		{
			name: "appends to empty builder",
			args: args{
				b:        &strings.Builder{},
				existing: nil,
				key:      "key1",
				value:    "value1",
			},
			want:            true,
			wantEnvVarValue: "key1=value1",
		},
		{
			name: "appends to builder",
			args: args{
				b:        b,
				existing: nil,
				key:      "key1",
				value:    "value1",
			},
			want:            true,
			wantEnvVarValue: "key=value,key1=value1",
		},
		{
			name: "do not overwrite existing key",
			args: args{
				b:        &strings.Builder{},
				existing: &corev1.EnvVar{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: "key1=oldvalue"},
				key:      "key1",
				value:    "newvalue",
			},
			want:            false,
			wantEnvVarValue: "key1=oldvalue",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, appendAttribute(tt.args.b, tt.args.existing, tt.args.key, tt.args.value), "appendAttribute(%v, %v, %v, %v)", tt.args.b, tt.args.existing, tt.args.key, tt.args.value)
		})
	}
}
