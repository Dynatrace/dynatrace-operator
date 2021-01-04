package oneagent

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
		OneAgent: dynatracev1alpha1.OneAgentSpec{
			Enabled: true,
			DNSPolicy: corev1.DNSClusterFirstWithHostNet,
			Labels: map[string]string{
				"label_key": "label_value",
			},
		},
	}

	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec:       dkSpec,
		},
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetLatestAgentVersion", "unix", "default").Return("42", nil)
	dtClient.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	dtClient.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
	dtClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)

	reconciler := &ReconcileOneAgent{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: dkName, Namespace: namespace}})
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
			OneAgent: dynatracev1alpha1.OneAgentSpec{
				Enabled: true,
			},
		},
	}
	base.Status.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
		Type:    dynatracev1alpha1.APITokenConditionType,
		Status:  corev1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})
	base.Status.Conditions.SetCondition(dynatracev1alpha1.Condition{
		Type:    dynatracev1alpha1.PaaSTokenConditionType,
		Status:  corev1.ConditionTrue,
		Reason:  dynatracev1alpha1.ReasonTokenReady,
		Message: "Ready",
	})

	t.Run("SetPhaseOnError called with different values, object and return value correctly modified", func(t *testing.T) {
		dk := base.DeepCopy()

		res := dk.Status.OneAgentStatus.SetPhaseOnError(nil)
		assert.False(t, res)
		assert.Equal(t, dk.Status.OneAgentStatus.Phase, dynatracev1alpha1.OneAgentPhaseType(""))

		res = dk.Status.OneAgentStatus.SetPhaseOnError(errors.New("dummy error"))
		assert.True(t, res)

		if assert.NotNil(t, dk.Status.OneAgentStatus.Phase) {
			assert.Equal(t, dynatracev1alpha1.Error, dk.Status.OneAgentStatus.Phase)
		}
	})

	// arrange
	c := fake.NewFakeClientWithScheme(scheme.Scheme,
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileRollout Phase is set to deploying, if agent version is not set on OneAgent object", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.OneAgentStatus.Version = ""

		// act
		updateCR, err := reconciler.reconcileRollout(consoleLogger, oa, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, err, nil)
		assert.Equal(t, dynatracev1alpha1.Deploying, oa.Status.OneAgentStatus.Phase)
		assert.Equal(t, version, oa.Status.OneAgentStatus.Version)
	})

	t.Run("reconcileRollout Phase not changing, if agent version is already set on OneAgent object", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Status.OneAgentStatus.Version = version
		dk.Status.Tokens = utils.GetTokensName(*dk)

		// act
		updateCR, err := reconciler.reconcileRollout(consoleLogger, dk, dtcMock)

		// assert
		assert.False(t, updateCR)
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.OneAgentPhaseType(""), dk.Status.OneAgentStatus.Phase)
	})

	t.Run("reconcileVersion Phase not changing", func(t *testing.T) {
		// arrange
		oa := base.DeepCopy()
		oa.Status.OneAgentStatus.Version = version

		// act
		_, err := reconciler.reconcileVersion(consoleLogger, oa, dtcMock)

		// assert
		assert.Equal(t, nil, err)
		assert.Equal(t, dynatracev1alpha1.OneAgentPhaseType(""), oa.Status.OneAgentStatus.Phase)
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
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Enabled: true,
				},
			},
	}
	c := fake.NewFakeClientWithScheme(scheme.Scheme,
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
	dtcMock := &dtclient.MockDynatraceClient{}
	version := "1.187"
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(version, nil)

	reconciler := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileRollout Tokens status set, if empty", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = ""

		// act
		updateCR, err := reconciler.reconcileRollout(consoleLogger, dk, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(*dk), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})
	t.Run("reconcileRollout Tokens status set, if status has wrong name", func(t *testing.T) {
		// arrange
		dk := base.DeepCopy()
		dk.Spec.Tokens = ""
		dk.Status.Tokens = "not the actual name"

		// act
		updateCR, err := reconciler.reconcileRollout(consoleLogger, dk, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(*dk), dk.Status.Tokens)
		assert.Equal(t, nil, err)
	})

	t.Run("reconcileRollout Tokens status set, not equal to defined name", func(t *testing.T) {
		c = fake.NewFakeClientWithScheme(scheme.Scheme,
			NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
			sampleKubeSystemNS)

		reconciler := &ReconcileOneAgent{
			client:    c,
			apiReader: c,
			scheme:    scheme.Scheme,
			logger:    consoleLogger,
			dtcReconciler: &utils.DynatraceClientReconciler{
				Client:              c,
				DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
				UpdatePaaSToken:     true,
				UpdateAPIToken:      true,
			},
		}

		// arrange
		customTokenName := "custom-token-name"
		dk := base.DeepCopy()
		dk.Status.Tokens = utils.GetTokensName(*dk)
		dk.Spec.Tokens = customTokenName

		// act
		updateCR, err := reconciler.reconcileRollout(consoleLogger, dk, dtcMock)

		// assert
		assert.True(t, updateCR)
		assert.Equal(t, utils.GetTokensName(*dk), dk.Status.Tokens)
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
			OneAgent: dynatracev1alpha1.OneAgentSpec{
				Enabled: true,
			},
		},
	}

	// arrange
	c := fake.NewFakeClientWithScheme(scheme.Scheme,
		NewSecret(dkName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		sampleKubeSystemNS)
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
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is false", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Spec.OneAgent.DisableAgentUpdate = false
		dk.Status.OneAgentStatus.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName)
		pod.Spec = newPodSpecForCR(dk, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = utils.GetTokensName(*dk)

		rec := reconciliation{log: consoleLogger, instance: dk, requeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.reconcileImpl(&rec)

		assert.NotNil(t, dk.Status.OneAgentStatus.Instances)
		assert.NotEmpty(t, dk.Status.OneAgentStatus.Instances)
	})

	t.Run("reconcileImpl Instances set, if agentUpdateDisabled is true", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Spec.OneAgent.DisableAgentUpdate = true
		dk.Status.OneAgentStatus.Version = oldVersion
		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = buildLabels(dkName)
		pod.Spec = newPodSpecForCR(dk, false, consoleLogger, "cluster1")
		pod.Status.HostIP = hostIP
		dk.Status.Tokens = utils.GetTokensName(*dk)

		rec := reconciliation{log: consoleLogger, instance: dk, requeueAfter: 30 * time.Minute}
		err := reconciler.client.Create(context.TODO(), pod)

		assert.NoError(t, err)

		reconciler.reconcileImpl(&rec)

		assert.NotNil(t, dk.Status.OneAgentStatus.Instances)
		assert.NotEmpty(t, dk.Status.OneAgentStatus.Instances)
	})
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
