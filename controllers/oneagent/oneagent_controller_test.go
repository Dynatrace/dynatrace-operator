package oneagent

import (
	"context"
	"os"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testClusterID = "test-cluster-id"
)

var consoleLogger = zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

var sampleKubeSystemNS = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
		UID:  "01234-5678-9012-3456",
	},
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dkSpec := dynatracev1alpha1.DynaKubeSpec{
		APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
		Tokens: dkName,
		ClassicFullStack: dynatracev1alpha1.FullStackSpec{
			Enabled:   true,
			DNSPolicy: corev1.DNSClusterFirstWithHostNet,
			Labels: map[string]string{
				"label_key": "label_value",
			},
		},
	}

	dynakube := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec:       dkSpec,
	}
	fakeClient := fake.NewClient(
		dynakube,
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	dtClient := &dtclient.MockDynatraceClient{}

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance:  dynakube,
		feature:   daemonset.ClassicFeature,
		fullStack: &dynakube.Spec.ClassicFullStack,
	}

	dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dynakube}
	_, err := reconciler.Reconcile(context.TODO(), &dkState)
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: dkName + "-" + reconciler.feature, Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, dkName+"-"+reconciler.feature, dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcile_PhaseSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.APITokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1alpha1.PaaSTokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})

	t.Run("SetPhaseOnError called with different values, object and return value correctly modified", func(t *testing.T) {
		dk := base.DeepCopy()

		res := dk.Status.SetPhaseOnError(nil)
		assert.False(t, res)
		assert.Equal(t, dk.Status.Phase, dynatracev1alpha1.DynaKubePhaseType(""))

		res = dk.Status.SetPhaseOnError(errors.New("dummy error"))
		assert.True(t, res)

		if assert.NotNil(t, dk.Status.Phase) {
			assert.Equal(t, dynatracev1alpha1.Error, dk.Status.Phase)
		}
	})
}

func TestReconcile_TokensSetCorrectly(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"
	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
	c := fake.NewClient(
		NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		fullStack: &base.Spec.ClassicFullStack,
		feature:   daemonset.ClassicFeature,
		instance:  &base,
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = ""
		dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dk}

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
		dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dk}

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

		reconciler := &ReconcileOneAgent{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			logger:    consoleLogger,
			instance:  &base,
			feature:   daemonset.ClassicFeature,
			fullStack: &base.Spec.ClassicFullStack,
		}

		// arrange
		customTokenName := "custom-token-name"
		dk := base.DeepCopy()
		dk.Status.Tokens = dk.Tokens()
		dk.Spec.Tokens = customTokenName
		dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dk}

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
	base := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: name,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
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

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance:  &base,
		fullStack: &base.Spec.ClassicFullStack,
		feature:   daemonset.ClassicFeature,
	}

	t.Run(`reconileImp Instances set, if autoUpdate is true`, func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = oldVersion
		dsInfo := daemonset.NewClassicFullStack(dk, consoleLogger, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(name, reconciler.feature)
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()
		dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
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
		dk.Spec.OneAgent.AutoUpdate = &autoUpdate
		dk.Status.OneAgent.Version = oldVersion
		dsInfo := daemonset.NewClassicFullStack(dk, consoleLogger, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(name, reconciler.feature)
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = dk.Tokens()

		dkState := controllers.DynakubeState{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
		err = reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		reconciler.fullStack = &dk.Spec.ClassicFullStack
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

func TestOneAgent_Validate(t *testing.T) {
	dk := newDynaKube()
	assert.Error(t, validate(dk))
	dk.Spec.APIURL = "https://f.q.d.n/api"
	assert.NoError(t, validate(dk))
}

func TestMigrationForDaemonSetWithoutAnnotation(t *testing.T) {
	dkKey := metav1.ObjectMeta{Name: "my-dynakube", Namespace: "my-namespace"}

	ds1 := &appsv1.DaemonSet{ObjectMeta: dkKey}

	ds2, err := newDaemonSetForCR(consoleLogger, &dynatracev1alpha1.DynaKube{ObjectMeta: dkKey}, &dynatracev1alpha1.FullStackSpec{}, "classic", "cluster1")
	assert.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

	assert.True(t, kubeobjects.HasChanged(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	runTest := func(msg string, exp bool, mod func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube)) {
		t.Run(msg, func(t *testing.T) {
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			oldInstance := dynatracev1alpha1.DynaKube{ObjectMeta: key}
			newInstance := dynatracev1alpha1.DynaKube{ObjectMeta: key}

			mod(&oldInstance, &newInstance)

			ds1, err := newDaemonSetForCR(consoleLogger, &oldInstance, &oldInstance.Spec.ClassicFullStack, "cluster1", "classic")
			assert.NoError(t, err)

			ds2, err := newDaemonSetForCR(consoleLogger, &newInstance, &newInstance.Spec.ClassicFullStack, "cluster1", "classic")
			assert.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[kubeobjects.AnnotationHash])
			assert.NotEmpty(t, ds2.Annotations[kubeobjects.AnnotationHash])

			assert.Equal(t, exp, kubeobjects.HasChanged(ds1, ds2))
		})
	}

	runTest("no changes", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {})

	runTest("image added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image set but no change", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "registry.access.redhat.com/dynatrace/oneagent"
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("argument removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
		new.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("argument changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
		new.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=0"}
	})

	runTest("all arguments removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("resources added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("priority class added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class set but no change", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "class"
		new.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "some class"
		new.Spec.ClassicFullStack.PriorityClassName = "other class"
	})

	runTest("dns policy added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.DNSPolicy = corev1.DNSClusterFirst
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

func newDynaKube() *dynatracev1alpha1.DynaKube {
	return &dynatracev1alpha1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
	}
}

func TestInstanceStatus(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dynakube := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
				FullStackSpec: dynatracev1alpha1.FullStackSpec{Enabled: true},
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
				"operator.dynatrace.com/feature":  daemonset.InframonFeature,
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

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance:  dynakube,
		feature:   daemonset.InframonFeature,
		fullStack: &dynakube.Spec.InfraMonitoring.FullStackSpec,
	}

	upd, err := reconciler.reconcileInstanceStatuses(context.Background())
	assert.NoError(t, err)
	assert.True(t, upd)

	upd, err = reconciler.reconcileInstanceStatuses(context.Background())
	assert.NoError(t, err)
	assert.False(t, upd)
}
