package oneagent

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
}

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
			Enabled: true,
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
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		dynakube,
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS).Build()

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetLatestAgentVersion", "unix", "default").Return("42", nil)

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance: dynakube,
		webhookInjection: false,
		dtc: dtClient,
		fullStack: &dynakube.Spec.ClassicFullStack,
	}

	rec := utils.Reconciliation{}
	_, err := reconciler.Reconcile(context.TODO(), &rec)
	assert.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: dkName, Namespace: namespace}, dsActual)
	assert.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, dkName, dsActual.GetObjectMeta().GetName(), "wrong name")
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

	// arrange
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS).Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		instance: &base,
		webhookInjection: false,
		dtc: dtcMock,
		fullStack: &base.Spec.ClassicFullStack,
	}

	t.Run("reconcileRollout Phase is set to deploying, if agent version is not set on OneAgent object", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = ""

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, dk, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, err, nil)
		assert.Equal(t, dynatracev1alpha1.Deploying, dk.Status.Phase)
		assert.Equal(t, version, dk.Status.OneAgent.Version)
	})

	t.Run("reconcileRollout Phase not changing, if agent version is already set on OneAgent object", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = version
		dk.Status.Tokens = utils.GetTokensName(dk)

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, dk, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.False(t, updateCR)
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.DynaKubePhaseType(""), dk.Status.Phase)
	})

	t.Run("reconcileVersion Phase not changing", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.OneAgent.Version = version

		// act
		_, err := reconciler.reconcileVersion(context.TODO(), consoleLogger, oa, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.DynaKubePhaseType(""), oa.Status.Phase)
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
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS).Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtc: dtcMock,
		fullStack: &base.Spec.ClassicFullStack,
		webhookInjection: false,
		instance: &base,
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = ""

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, dk, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(dk), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
	t.Run("reconcileRollout Tokens status set, if status has wrong name", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = "not the actual name"

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, dk, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(dk), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})

	t.Run("reconcileRollout Tokens status set, not equal to defined name", func(t *testing.T) {
		c = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS).Build()

		reconciler := &ReconcileOneAgent{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			logger:    consoleLogger,
			instance: &base,
			webhookInjection: false,
			fullStack: &base.Spec.ClassicFullStack,
			dtc: dtcMock,
		}

		// arrange
		customTokenName := "custom-token-name"
		dk := base.DeepCopy()
		dk.Status.Tokens = utils.GetTokensName(dk)
		dk.Spec.Tokens = customTokenName

		// act
		updateCR, err := reconciler.reconcileRollout(context.TODO(), consoleLogger, dk, &dynatracev1alpha1.FullStackSpec{}, false, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(dk), dk.Status.Tokens)
		assert.Equal(t, customTokenName, dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
}

func TestReconcile_InstancesSet(t *testing.T) {
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

	// arrange
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS).Build()
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	oldVersion := "1.186"
	hostIP := "1.2.3.4"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)
	dtcMock.On("GetAgentVersionForIP", hostIP).Return(version, nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{utils.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{utils.DynatraceApiToken}, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtc: dtcMock,
		instance: &base,
		fullStack: &base.Spec.ClassicFullStack,
		webhookInjection: false,
	}

	t.Run("reconcileImpl Instances set, if autoUpdate is true", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Status.OneAgent.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName)
		pod.Spec = newPodSpecForCR(dk, &dynatracev1alpha1.FullStackSpec{}, false, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = utils.GetTokensName(dk)

		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		reconciler.Reconcile(context.TODO(), &rec)

		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		dk := base.DeepCopy()
		autoUpdate := false
		dk.Spec.OneAgent.AutoUpdate = &autoUpdate
		dk.Status.OneAgent.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName)
		pod.Spec = newPodSpecForCR(dk, &dynatracev1alpha1.FullStackSpec{}, false, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = utils.GetTokensName(dk)

		rec := utils.Reconciliation{Log: consoleLogger, Instance: dk, RequeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.instance = dk
		reconciler.fullStack = &dk.Spec.ClassicFullStack
		reconciler.Reconcile(context.TODO(), &rec)

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
