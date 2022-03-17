package oneagent

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
			ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
				HostInjectSpec: dynatracev1beta1.HostInjectSpec{
					DNSPolicy: corev1.DNSClusterFirstWithHostNet,
					Labels: map[string]string{
						"label_key": "label_value",
					},
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

	reconciler := &OneAgentReconciler{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		instance:  dynakube,
		feature:   daemonset.ClassicFeature,
	}

	dkState := status.DynakubeState{Instance: dynakube}
	_, err := reconciler.Reconcile(context.TODO(), &dkState)
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
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
				},
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

	t.Run("SetPhaseOnError called with different values, object and return value correctly modified", func(t *testing.T) {
		dk := base.DeepCopy()

		res := dk.Status.SetPhaseOnError(nil)
		assert.False(t, res)
		assert.Equal(t, dk.Status.Phase, dynatracev1beta1.DynaKubePhaseType(""))

		res = dk.Status.SetPhaseOnError(errors.New("dummy error"))
		assert.True(t, res)

		if assert.NotNil(t, dk.Status.Phase) {
			assert.Equal(t, dynatracev1beta1.Error, dk.Status.Phase)
		}
	})
}

func TestReconcile_TokensSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"
	base := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
				},
			},
		},
	}
	c := fake.NewClient(
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &OneAgentReconciler{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		feature:   daemonset.ClassicFeature,
		instance:  &base,
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = ""
		dkState := status.DynakubeState{Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&dkState)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
	t.Run("reconcileRollout Tokens status set, if status has wrong name", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = "not the actual name"
		dkState := status.DynakubeState{Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&dkState)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})

	t.Run("reconcileRollout Tokens status set, not equal to defined name", func(t *testing.T) {
		c = fake.NewClient(
			NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
			sampleKubeSystemNS)

		reconciler := &OneAgentReconciler{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			instance:  &base,
			feature:   daemonset.ClassicFeature,
		}

		// arrange
		customTokenName := "custom-token-name"
		dk := base.DeepCopy()
		dk.Status.Tokens = dk.Tokens()
		dk.Spec.Tokens = customTokenName
		dkState := status.DynakubeState{Instance: dk}

		// act
		updateCR, err := reconciler.reconcileRollout(&dkState)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, dk.Tokens(), dk.Status.Tokens)
		assert.Equal(t, customTokenName, dk.Status.Tokens)
		assert.Equal(t, nil, err)
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
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
				},
			},
		},
	}
	c := fake.NewClient(
		NewSecret(name, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	oldVersion := "1.186"
	hostIP := "1.2.3.4"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.DynatraceApiToken}, nil)

	reconciler := &OneAgentReconciler{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		instance:  &base,
		feature:   daemonset.ClassicFeature,
	}

	t.Run(`reconileImp Instances set, if autoUpdate is true`, func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = oldVersion
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
		pod.Labels = daemonset.BuildLabels(name, reconciler.feature)
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()
		dkState := status.DynakubeState{Instance: dk, RequeueAfter: 30 * time.Minute}
		err = reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		upd, err := reconciler.Reconcile(context.TODO(), &dkState)

		assert.NoError(t, err)
		assert.True(t, upd)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		dk := base.DeepCopy()
		autoUpdate := false
		dk.Spec.OneAgent.ClassicFullStack.AutoUpdate = &autoUpdate
		dk.Status.OneAgent.Version = oldVersion
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
		pod.Labels = daemonset.BuildLabels(name, reconciler.feature)
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()

		dkState := status.DynakubeState{Instance: dk, RequeueAfter: 30 * time.Minute}
		err = reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		upd, err := reconciler.Reconcile(context.TODO(), &dkState)

		assert.NoError(t, err)
		assert.True(t, upd)
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
	r := OneAgentReconciler{
		feature: daemonset.HostMonitoringFeature,
	}
	dkState := &status.DynakubeState{
		Instance: &dynatracev1beta1.DynaKube{
			ObjectMeta: dkKey,
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		},
	}

	ds2, err := r.newDaemonSetForCR(dkState, "cluster1")
	assert.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

	assert.True(t, kubeobjects.HasChanged(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	runTest := func(msg string, exp bool, mod func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube)) {
		t.Run(msg, func(t *testing.T) {
			r := OneAgentReconciler{
				feature: daemonset.HostMonitoringFeature,
			}
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			oldInstance := dynatracev1beta1.DynaKube{
				ObjectMeta: key,
				Spec: dynatracev1beta1.DynaKubeSpec{
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
					},
				},
			}
			newInstance := dynatracev1beta1.DynaKube{
				ObjectMeta: key,
				Spec: dynatracev1beta1.DynaKubeSpec{
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
					},
				},
			}
			mod(&oldInstance, &newInstance)

			dkState := &status.DynakubeState{
				Instance: &oldInstance,
			}
			ds1, err := r.newDaemonSetForCR(dkState, "cluster1")
			assert.NoError(t, err)

			dkState.Instance = &newInstance
			ds2, err := r.newDaemonSetForCR(dkState, "cluster1")
			assert.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[kubeobjects.AnnotationHash])
			assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

			assert.Equal(t, exp, kubeobjects.HasChanged(ds1, ds2))
		})
	}

	runTest("no changes", false, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {})

	runTest("image added", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		new.Spec.OneAgent.HostMonitoring.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image set but no change", false, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Image = "docker.io/dynatrace/oneagent"
		new.Spec.OneAgent.HostMonitoring.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image removed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image changed", true, func(old *dynatracev1beta1.DynaKube, new *dynatracev1beta1.DynaKube) {
		old.Spec.OneAgent.HostMonitoring.Image = "registry.access.redhat.com/dynatrace/oneagent"
		new.Spec.OneAgent.HostMonitoring.Image = "docker.io/dynatrace/oneagent"
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
		r := OneAgentReconciler{
			feature: daemonset.HostMonitoringFeature,
		}
		dkState := &status.DynakubeState{
			Instance: newDynaKube(),
		}
		versionProvider.On("Major").Return("1", nil)
		versionProvider.On("Minor").Return("20+", nil)
		ds, err := r.newDaemonSetForCR(dkState, "cluster1")

		assert.NoError(t, err)
		assert.NotNil(t, ds)

		affinity := ds.Spec.Template.Spec.Affinity

		assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "beta.kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64", "arm64"},
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
					Values:   []string{"amd64", "arm64"},
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
				HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
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
				HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: namespace,
			Labels: map[string]string{
				"dynatrace.com/component":         "operator",
				"operator.dynatrace.com/instance": dkName,
				"operator.dynatrace.com/feature":  daemonset.HostMonitoringFeature,
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

	reconciler := &OneAgentReconciler{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		instance:  dynakube,
		feature:   daemonset.HostMonitoringFeature,
	}

	upd, err := reconciler.reconcileInstanceStatuses(context.Background(), reconciler.instance)
	assert.NoError(t, err)
	assert.True(t, upd)

	upd, err = reconciler.reconcileInstanceStatuses(context.Background(), reconciler.instance)
	assert.NoError(t, err)
	assert.False(t, upd)
}
