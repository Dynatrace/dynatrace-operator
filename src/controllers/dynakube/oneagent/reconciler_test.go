package oneagent

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testClusterID = "test-cluster-id"
)

type fakeVersionProvider struct {
	mock.Mock
}

func (f *fakeVersionProvider) Major() (string, error) {
	args := f.Called()
	return args.Get(0).(string), args.Error(1)
}

func (f *fakeVersionProvider) Minor() (string, error) {
	args := f.Called()
	return args.Get(0).(string), args.Error(1)
}

var sampleKubeSystemNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
		UID:  "01234-5678-9012-3456",
	},
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dkSpec := dynatracev1beta1.DynaKubeSpec{
		APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
		Tokens: dkName,
		OneAgent: dynatracev1beta1.OneAgentSpec{
			ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
				DNSPolicy: corev1.DNSClusterFirstWithHostNet,
				Labels: map[string]string{
					"label_key": "label_value",
				},
			},
		},
	}

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec:       dkSpec,
	}
	fakeClient := fake.NewClient(
		dynakube,
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	dtClient := &dtclient.MockDynatraceClient{}

	reconciler := &Reconciler{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
	}

	err := reconciler.Reconcile(context.TODO(), dynakube)
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: dynakube.OneAgentDaemonsetName(), Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, dynakube.OneAgentDaemonsetName(), dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcile_PhaseSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	base := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1beta1.APITokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "Ready",
	})
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1beta1.PaaSTokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "Ready",
	})
}

func TestReconcile_InstancesSet(t *testing.T) {
	const (
		namespace = "dynatrace"
		name      = "dynakube"
	)
	base := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: name,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
	c := fake.NewClient(
		NewSecret(name, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	componentVersion := "1.187"
	oldComponentVersion := "1.186"
	hostIP := "1.2.3.4"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(componentVersion, nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.DynatraceApiToken}, nil)

	reconciler := &Reconciler{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
	}

	expectedLabels := map[string]string{
		kubeobjects.AppNameLabel:      kubeobjects.OneAgentComponentLabel,
		kubeobjects.AppComponentLabel: "classicfullstack",
		kubeobjects.AppCreatedByLabel: name,
		kubeobjects.AppVersionLabel:   oldComponentVersion,
		kubeobjects.AppManagedByLabel: version.AppName,
	}

	t.Run("reconcileImp Instances set, if autoUpdate is true", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = oldComponentVersion
		dsInfo := daemonset.NewClassicFullStack(dk, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = expectedLabels
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		err = reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		err = reconciler.Reconcile(context.TODO(), dk)

		assert.NoError(t, err)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		dk := base.DeepCopy()
		autoUpdate := false
		dk.Spec.OneAgent.ClassicFullStack.AutoUpdate = &autoUpdate
		dk.Status.OneAgent.Version = oldComponentVersion
		dsInfo := daemonset.NewClassicFullStack(dk, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = expectedLabels
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP

		err = reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		err = reconciler.Reconcile(context.TODO(), dk)

		assert.NoError(t, err)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func TestMigrationForDaemonSetWithoutAnnotation(t *testing.T) {
	dkKey := metav1.ObjectMeta{Name: "my-dynakube", Namespace: "my-namespace"}
	ds1 := &appsv1.DaemonSet{ObjectMeta: dkKey}
	r := Reconciler{}

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: dkKey,
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}

	ds2, err := r.buildDesiredDaemonSet(dynakube)
	assert.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

	assert.True(t, kubeobjects.IsHashAnnotationDifferent(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	runTest := func(msg string, exp bool, mod func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube)) {
		t.Run(msg, func(t *testing.T) {
			r := Reconciler{}
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			oldInstance := dynatracev1beta1.DynaKube{
				ObjectMeta: key,
				Spec: dynatracev1beta1.DynaKubeSpec{
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
					},
				},
			}
			newInstance := dynatracev1beta1.DynaKube{
				ObjectMeta: key,
				Spec: dynatracev1beta1.DynaKubeSpec{
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
					},
				},
			}
			mod(&oldInstance, &newInstance)
			ds1, err := r.buildDesiredDaemonSet(&oldInstance)
			assert.NoError(t, err)

			ds2, err := r.buildDesiredDaemonSet(&newInstance)
			assert.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[kubeobjects.AnnotationHash])
			assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

			assert.Equal(t, exp, kubeobjects.IsHashAnnotationDifferent(ds1, ds2))
		})
	}

	runTest("no changes", false, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {})

	runTest("image present", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		new.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	})

	runTest("image set but no change", false, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		new.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	})

	runTest("image changed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Status.OneAgent.ImageID = "registry.access.redhat.com/dynatrace/oneagent:1.233.345@sha256:6ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		new.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	})

	runTest("argument removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
		new.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("argument changed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
		new.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=0"}
	})

	runTest("all arguments removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("resources added", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		new.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
	})

	runTest("priority class added", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		new.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
	})

	runTest("priority class removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
	})

	runTest("priority class set but no change", false, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
		new.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
	})

	runTest("priority class changed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.PriorityClassName = "some class"
		new.Spec.OneAgent.HostMonitoring.PriorityClassName = "other class"
	})

	runTest("dns policy added", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		new.Spec.OneAgent.HostMonitoring.DNSPolicy = corev1.DNSClusterFirst
	})
}

func TestNewDaemonset_Affinity(t *testing.T) {
	t.Run(`adds correct affinities`, func(t *testing.T) {
		versionProvider := &fakeVersionProvider{}
		r := Reconciler{}
		dynakube := newDynaKube()
		versionProvider.On("Major").Return("1", nil)
		versionProvider.On("Minor").Return("20+", nil)
		ds, err := r.buildDesiredDaemonSet(dynakube)

		assert.NoError(t, err)
		assert.NotNil(t, ds)

		affinity := ds.Spec.Template.Spec.Affinity

		assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "beta.kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64", "arm64", "ppc64le"},
				},
				{
					Key:      "beta.kubernetes.io/os",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"linux"},
				},
			},
		})
		assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64", "arm64", "ppc64le"},
				},
				{
					Key:      "kubernetes.io/os",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"linux"},
				},
			},
		})
	})
}

func newResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    parseQuantity("10m"),
			"memory": parseQuantity("100Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    parseQuantity("20m"),
			"memory": parseQuantity("200Mi"),
		},
	}
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}

func newDynaKube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
}

func TestInstanceStatus(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":          "dynatrace-operator",
				"app.kubernetes.io/component":     "oneagent",
				"app.kubernetes.io/created-by":    dkName,
				"app.kubernetes.io/version":       "snapshot",
				"component.dynatrace.com/feature": deploymentmetadata.HostMonitoringDeploymentType,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
		Status: corev1.PodStatus{
			HostIP: "123.123.123.123",
		},
	}

	fakeClient := fake.NewClient(
		dynakube,
		pod,
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	reconciler := &Reconciler{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
	}

	err := reconciler.reconcileInstanceStatuses(context.Background(), dynakube)
	assert.NoError(t, err)
	assert.NotEmpty(t, t, dynakube.Status.OneAgent.Instances)
	instances := dynakube.Status.OneAgent.Instances

	err = reconciler.reconcileInstanceStatuses(context.Background(), dynakube)
	assert.NoError(t, err)
	assert.Equal(t, instances, dynakube.Status.OneAgent.Instances)
}

func TestEmptyInstancesWithWrongLabels(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: namespace,
			Labels: map[string]string{
				"wrongLabel": "dynatrace-operator",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
		Status: corev1.PodStatus{
			HostIP: "123.123.123.123",
		},
	}

	fakeClient := fake.NewClient(
		dynakube,
		pod,
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	reconciler := &Reconciler{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
	}

	err := reconciler.reconcileInstanceStatuses(context.Background(), dynakube)
	assert.NoError(t, err)
	assert.Empty(t, dynakube.Status.OneAgent.Instances)
}

func TestReconcile_ActivegateConfigMap(t *testing.T) {
	const (
		testNamespace       = "test-namespace"
		testTenantToken     = "test-token"
		testTenantUUID      = "test-uuid"
		testTenantEndpoints = "test-endpoints"
	)

	dynakube := newDynaKube()
	dynakube.Status = dynatracev1beta1.DynaKubeStatus{
		OneAgent: dynatracev1beta1.OneAgentStatus{
			ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
				ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID:  testTenantUUID,
					Endpoints:   testTenantEndpoints,
					LastRequest: metav1.Time{},
				},
			},
		},
	}

	fakeClient := fake.NewClient(
		dynakube,
		NewSecret(dynakube.Name, dynakube.Namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	t.Run(`create OneAgent connection info ConfigMap`, func(t *testing.T) {
		reconciler := NewOneAgentReconciler(fakeClient, fakeClient, scheme.Scheme, "")

		err := reconciler.Reconcile(context.TODO(), dynakube)
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneAgentConnectionInfoConfigMapName(), Namespace: dynakube.Namespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDName])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsName])
	})
}
