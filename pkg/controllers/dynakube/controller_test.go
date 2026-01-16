package dynakube

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	ag "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring"
	oneagentcontroller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	dynakubemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID       = "test-uid"
	testAPIToken  = "test-api-token"
	testUUID      = "test-uuid"
	testHost      = "test-host"
	testName      = "test-name"
	testNamespace = "test-namespace"
	testAPIURL    = "https://" + testHost + "/e/" + testUUID + "/api"
	testMessage   = "test-message"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestGetDynakubeOrCleanup(t *testing.T) {
	ctx := t.Context()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "dynakube-test", Namespace: "dynatrace"},
	}

	t.Run("dynakube doesn't exist => unmap namespace", func(t *testing.T) {
		markedNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "app-namespace",
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: request.Name,
				},
			},
		}
		fakeClient := fake.NewClientWithIndex(markedNamespace, createCRD(t))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dk, err := controller.getDynakubeOrCleanup(ctx, request.Name, request.Namespace)
		require.NoError(t, err)
		assert.Nil(t, dk)

		unmarkedNamespace := &corev1.Namespace{}
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: markedNamespace.Name}, unmarkedNamespace)
		require.NoError(t, err)
		assert.Empty(t, unmarkedNamespace.Labels)
	})

	t.Run("dynakube exists => return dynakube", func(t *testing.T) {
		expectedDynakube := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Spec: dynakube.DynaKubeSpec{APIURL: "this-is-an-api-url"},
		}
		fakeClient := fake.NewClientWithIndex(expectedDynakube, createCRD(t))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dk, err := controller.getDynakubeOrCleanup(ctx, request.Name, request.Namespace)
		require.NoError(t, err)
		assert.Equal(t, expectedDynakube.Name, dk.Name)
		assert.Equal(t, expectedDynakube.Namespace, dk.Namespace)
		assert.Equal(t, expectedDynakube.APIURL(), dk.APIURL())
	})
}

func TestMinimalRequest(t *testing.T) {
	t.Run("Create works with minimal setup", func(t *testing.T) {
		controller := &Controller{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(createCRD(t)),
		}
		result, err := controller.Reconcile(t.Context(), reconcile.Request{})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("reconcile fails with faulty client", func(t *testing.T) {
		controller := &Controller{
			client:    errorClient{},
			apiReader: errorClient{},
		}

		result, err := controller.Reconcile(t.Context(), reconcile.Request{})

		require.Error(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandleError(t *testing.T) {
	ctx := t.Context()
	dynakubeBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{APIURL: "this-is-an-api-url"},
	}

	t.Run("no error => update status", func(t *testing.T) {
		oldDynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(oldDynakube)
		controller := &Controller{
			client:       fakeClient,
			apiReader:    fakeClient,
			requeueAfter: 12345 * time.Second,
		}
		expectedDynakube := dynakubeBase.DeepCopy()
		expectedDynakube.Status = dynakube.DynaKubeStatus{
			Phase: status.Running,
		}

		result, err := controller.handleError(ctx, oldDynakube, nil, oldDynakube.Status)

		require.NoError(t, err)
		assert.Equal(t, controller.requeueAfter, result.RequeueAfter)

		dk := &dynakube.DynaKube{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: expectedDynakube.Name, Namespace: expectedDynakube.Namespace}, dk)
		require.NoError(t, err)
		assert.Equal(t, expectedDynakube.Status.Phase, dk.Status.Phase)
	})
	t.Run("no error => fail update status => error", func(t *testing.T) {
		oldDynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		result, err := controller.handleError(ctx, oldDynakube, nil, oldDynakube.Status)
		require.Error(t, err)
		assert.Empty(t, result)
	})
	t.Run("dynatrace server error => no error and fast update interval", func(t *testing.T) {
		oldDynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		serverError := dtclient.ServerError{Code: http.StatusTooManyRequests}

		result, err := controller.handleError(ctx, oldDynakube, serverError, oldDynakube.Status)
		require.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
	t.Run("random error => error, set error-phase", func(t *testing.T) {
		oldDynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(oldDynakube, createCRD(t))
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		randomError := errors.New("BOOM")

		result, err := controller.handleError(ctx, oldDynakube, randomError, oldDynakube.Status)
		assert.Empty(t, result)
		require.Error(t, err)

		dk := &dynakube.DynaKube{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: oldDynakube.Name, Namespace: oldDynakube.Namespace}, dk)
		require.NoError(t, err)
		assert.Equal(t, status.Error, dk.Status.Phase)
	})
}

func TestSetupTokensAndClient(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{APIURL: "https://test123.dev.dynatracelabs.com/api"},
	}

	t.Run("no tokens => error + condition", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dk)
		require.Error(t, err)
		assert.Nil(t, dtc)
		assertTokenCondition(t, dk, true)
	})

	t.Run("client builder error => error + condition", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		tokens := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken: []byte("this is a token"),
			},
		}
		fakeClient := fake.NewClientWithIndex(dk, tokens)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDtcBuilder.EXPECT().SetDynakube(mock.AnythingOfType("dynakube.DynaKube")).Return(mockDtcBuilder).Once()
		mockDtcBuilder.EXPECT().SetTokens(mock.AnythingOfType("token.Tokens")).Return(mockDtcBuilder).Once()
		mockDtcBuilder.EXPECT().Build(anyCtx).Return(nil, errors.New("BOOM")).Once()

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dk)
		require.Error(t, err)
		assert.Nil(t, dtc)
		assertTokenCondition(t, dk, true)
	})
	t.Run("tokens + dtclient ok => no error", func(t *testing.T) {
		// There is also a pull-secret created here, however testing it here is a bit counterintuitive.
		// TODO: Make the pull-secret reconciler mockable, so we can improve this test.
		dk := dkBase.DeepCopy()
		dk.Spec.CustomPullSecret = "custom"
		tokens := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken: []byte("this is a token"),
			},
		}
		fakeClient := fake.NewClientWithIndex(dk, tokens)

		mockedDtc := dtclientmock.NewClient(t)
		mockedDtc.EXPECT().GetTokenScopes(anyCtx, "this is a token").Return(dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil).Once()

		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDynatraceClientBuild(mockDtcBuilder, mockedDtc)

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dk)
		require.NoError(t, err)
		assert.NotNil(t, dtc)
		assertTokenCondition(t, dk, false)
	})
}

func assertTokenCondition(t *testing.T, dk *dynakube.DynaKube, hasError bool) {
	condition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.TokenConditionType)
	assert.NotNil(t, condition)

	if hasError {
		assert.Equal(t, dynakube.ReasonTokenError, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	} else {
		assert.Equal(t, dynakube.ReasonTokenReady, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	}
}

func TestReconcileComponents(t *testing.T) {
	ctx := t.Context()
	dkBaser := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:     "this-is-an-api-url",
			OneAgent:   oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{}},
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
	}

	expectReconcileError := func(t *testing.T, reconciler any, reconcileError *error, args ...any) {
		t.Helper()
		uniqueError := fmt.Errorf("BOOM %T", reconciler)

		switch reconciler := reconciler.(type) {
		case *controllermock.Reconciler:
			reconciler.EXPECT().Reconcile(anyCtx).Return(uniqueError).Once()
		case *dynakubemock.K8sEntityReconciler:
			reconciler.EXPECT().Reconcile(anyCtx, args[0], args[1]).Return(uniqueError).Once()
		default:
			return
		}

		t.Cleanup(func() {
			assert.ErrorIs(t, *reconcileError, uniqueError)
		})
	}

	t.Run("all components reconciled, even in case of errors", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)

		mockOneAgentReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockInjectionReconciler := controllermock.NewReconciler(t)
		mockLogMonitoringReconciler := controllermock.NewReconciler(t)
		mockExtensionReconciler := controllermock.NewReconciler(t)
		mockOtelcReconciler := controllermock.NewReconciler(t)
		mockKSPMReconciler := controllermock.NewReconciler(t)
		k8sEntityReconciler := dynakubemock.NewK8sEntityReconciler(t)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,

			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			injectionReconcilerBuilder:     createInjectionReconcilerBuilder(mockInjectionReconciler),
			oneAgentReconcilerBuilder:      createOneAgentReconcilerBuilder(mockOneAgentReconciler),
			logMonitoringReconcilerBuilder: createLogMonitoringReconcilerBuilder(mockLogMonitoringReconciler),
			extensionReconcilerBuilder:     createExtensionReconcilerBuilder(mockExtensionReconciler),
			otelcReconcilerBuilder:         createOtelcReconcilerBuilder(mockOtelcReconciler),
			kspmReconcilerBuilder:          createKSPMReconcilerBuilder(mockKSPMReconciler),
			k8sEntityReconciler:            k8sEntityReconciler,
		}
		mockedDtc := dtclientmock.NewClient(t)
		mockedDtc.EXPECT().AsV2().Return(&dtclient.V2Client{Settings: &settings.Client{}}).Once()

		var err error
		expectReconcileError(t, mockOneAgentReconciler, &err)
		expectReconcileError(t, mockActiveGateReconciler, &err)
		expectReconcileError(t, mockInjectionReconciler, &err)
		expectReconcileError(t, mockLogMonitoringReconciler, &err)
		expectReconcileError(t, mockExtensionReconciler, &err)
		expectReconcileError(t, mockOtelcReconciler, &err)
		expectReconcileError(t, mockKSPMReconciler, &err)
		expectReconcileError(t, k8sEntityReconciler, &err, &settings.Client{}, dk)

		err = controller.reconcileComponents(ctx, mockedDtc, nil, dk)
		require.Error(t, err)
	})

	t.Run("exit early in case of no oneagent conncection info", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockExtensionReconciler := controllermock.NewReconciler(t)
		mockOtelcReconciler := controllermock.NewReconciler(t)
		k8sEntityReconciler := dynakubemock.NewK8sEntityReconciler(t)

		mockLogMonitoringReconciler := controllermock.NewReconciler(t)
		mockLogMonitoringReconciler.EXPECT().Reconcile(anyCtx).Return(oaconnectioninfo.NoOneAgentCommunicationEndpointsError).Once()

		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			logMonitoringReconcilerBuilder: createLogMonitoringReconcilerBuilder(mockLogMonitoringReconciler),
			extensionReconcilerBuilder:     createExtensionReconcilerBuilder(mockExtensionReconciler),
			otelcReconcilerBuilder:         createOtelcReconcilerBuilder(mockOtelcReconciler),
			k8sEntityReconciler:            k8sEntityReconciler,
		}
		mockedDtc := dtclientmock.NewClient(t)
		mockedDtc.EXPECT().AsV2().Return(&dtclient.V2Client{Settings: &settings.Client{}}).Once()

		var err error
		expectReconcileError(t, mockActiveGateReconciler, &err)
		expectReconcileError(t, mockExtensionReconciler, &err)
		expectReconcileError(t, mockOtelcReconciler, &err)
		expectReconcileError(t, k8sEntityReconciler, &err, &settings.Client{}, dk)

		err = controller.reconcileComponents(ctx, mockedDtc, nil, dk)
		require.Error(t, err)
	})
}

func TestReconcileDynaKube(t *testing.T) {
	ctx := t.Context()
	baseDk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}

	fakeClient := fake.NewClient(baseDk, createCRD(t), createAPISecret())
	mockClient := dtclientmock.NewClient(t)
	mockClient.EXPECT().GetTokenScopes(anyCtx, testAPIToken).Return(dtclient.TokenScopes{
		dtclient.TokenScopeDataExport,
		dtclient.TokenScopeSettingsRead,
		dtclient.TokenScopeSettingsWrite,
		dtclient.TokenScopeInstallerDownload,
		dtclient.TokenScopeActiveGateTokenCreate,
	}, nil)
	mockClient.EXPECT().AsV2().Return(&dtclient.V2Client{Settings: &settings.Client{}})

	mockDtcBuilder := dtbuildermock.NewBuilder(t)
	mockDynatraceClientBuild(mockDtcBuilder, mockClient)

	mockDeploymentMetadataReconciler := controllermock.NewReconciler(t)
	mockDeploymentMetadataReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockProxyReconciler := controllermock.NewReconciler(t)
	mockProxyReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockOneAgentReconciler := controllermock.NewReconciler(t)
	mockOneAgentReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockActiveGateReconciler := controllermock.NewReconciler(t)
	mockActiveGateReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockInjectionReconciler := controllermock.NewReconciler(t)
	mockInjectionReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockLogMonitoringReconciler := controllermock.NewReconciler(t)
	mockLogMonitoringReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockExtensionReconciler := controllermock.NewReconciler(t)
	mockExtensionReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockOtelcReconciler := controllermock.NewReconciler(t)
	mockOtelcReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockKSPMReconciler := controllermock.NewReconciler(t)
	mockKSPMReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	mockK8sEntityReconciler := dynakubemock.NewK8sEntityReconciler(t)
	mockK8sEntityReconciler.EXPECT().Reconcile(anyCtx, &settings.Client{}, mock.MatchedBy(func(*dynakube.DynaKube) bool { return true })).Return(nil)

	fakeIstio := fakeistio.NewSimpleClientset()

	baseController := &Controller{
		apiReader:                           fakeClient,
		client:                              fakeClient,
		istioClientBuilder:                  fakeIstioClientBuilder(t, fakeIstio, true),
		activeGateReconcilerBuilder:         createActivegateReconcilerBuilder(mockActiveGateReconciler),
		deploymentMetadataReconcilerBuilder: createDeploymentMetadataReconcilerBuilder(mockDeploymentMetadataReconciler),
		dynatraceClientBuilder:              mockDtcBuilder,
		extensionReconcilerBuilder:          createExtensionReconcilerBuilder(mockExtensionReconciler),
		injectionReconcilerBuilder:          createInjectionReconcilerBuilder(mockInjectionReconciler),
		istioReconcilerBuilder:              istio.NewReconciler,
		kspmReconcilerBuilder:               createKSPMReconcilerBuilder(mockKSPMReconciler),
		logMonitoringReconcilerBuilder:      createLogMonitoringReconcilerBuilder(mockLogMonitoringReconciler),
		oneAgentReconcilerBuilder:           createOneAgentReconcilerBuilder(mockOneAgentReconciler),
		otelcReconcilerBuilder:              createOtelcReconcilerBuilder(mockOtelcReconciler),
		proxyReconcilerBuilder:              createProxyReconcilerBuilder(mockProxyReconciler),
		k8sEntityReconciler:                 mockK8sEntityReconciler,
	}

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: testName, Namespace: testNamespace},
	}

	t.Run("reconcile the controller and its sub controllers", func(t *testing.T) {
		controller := baseController

		result, err := controller.Reconcile(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("reconcile the controller with istio enabled", func(t *testing.T) {
		dk := baseDk.DeepCopy()
		dk.Spec.APIURL = testAPIURL
		dk.Spec.EnableIstio = true

		fakeClientWithIstio := fake.NewClientWithIndex(dk, createCRD(t), createAPISecret())

		controller := baseController
		controller.client = fakeClientWithIstio
		controller.apiReader = fakeClientWithIstio

		result, err := controller.Reconcile(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("reconciling the controller with istio enabled (but without valid API URL) should fail", func(t *testing.T) {
		dk := baseDk.DeepCopy()
		dk.Spec.EnableIstio = true

		fakeClientWithIstio := fake.NewClientWithIndex(dk, createAPISecret())

		controller := baseController
		controller.client = fakeClientWithIstio
		controller.apiReader = fakeClientWithIstio

		result, err := controller.Reconcile(ctx, request)
		require.Error(t, err)
		assert.NotNil(t, result)
	})
}

func createActivegateReconcilerBuilder(reconciler controllers.Reconciler) ag.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube, _ dtclient.Client, _ *istio.Client, _ token.Tokens) controllers.Reconciler {
		return reconciler
	}
}

func createOneAgentReconcilerBuilder(reconciler controllers.Reconciler) oneagentcontroller.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ dtclient.Client, _ *dynakube.DynaKube, _ token.Tokens, _ string) controllers.Reconciler {
		return reconciler
	}
}

func createLogMonitoringReconcilerBuilder(reconciler controllers.Reconciler) logmonitoring.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ dtclient.Client, _ *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

func createExtensionReconcilerBuilder(reconciler controllers.Reconciler) extension.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

func createOtelcReconcilerBuilder(reconciler controllers.Reconciler) otelc.ReconcilerBuilder {
	return func(_ dtclient.Client, _ client.Client, _ client.Reader, _ *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

func createInjectionReconcilerBuilder(reconciler controllers.Reconciler) injection.ReconcilerBuilder {
	return func(client client.Client, apiReader client.Reader, dynatraceClient dtclient.Client, istioClient *istio.Client, dk *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

func createKSPMReconcilerBuilder(reconciler controllers.Reconciler) kspm.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

func createAPIMonitoringReconcilerBuilder(reconciler controllers.Reconciler) apimonitoring.ReconcilerBuilder {
	return func(_ settings.APIClient, _ *dynakube.DynaKube, _ string) controllers.Reconciler {
		return reconciler
	}
}

func createDeploymentMetadataReconcilerBuilder(reconciler controllers.Reconciler) deploymentmetadata.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ dynakube.DynaKube, _ string) controllers.Reconciler {
		return reconciler
	}
}

func createProxyReconcilerBuilder(reconciler controllers.Reconciler) proxy.ReconcilerBuilder {
	return func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube) controllers.Reconciler {
		return reconciler
	}
}

type errorClient struct {
	client.Client
}

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return errors.New("fake error")
}

func (clt errorClient) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return errors.New("fake error")
}

func TestGetDynakube(t *testing.T) {
	t.Run("get dynakube", func(t *testing.T) {
		fakeClient := fake.NewClient(&dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		})
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		ctx := t.Context()
		dynakube, err := controller.getDynakubeOrCleanup(ctx, testName, testNamespace)

		assert.NotNil(t, dynakube)
		require.NoError(t, err)

		assert.Equal(t, testName, dynakube.Name)
		assert.Equal(t, testNamespace, dynakube.Namespace)
		assert.NotNil(t, dynakube.Spec.OneAgent.CloudNativeFullStack)
	})
	t.Run("unmap if not not found", func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespace,
				Labels: map[string]string{dtwebhook.InjectionInstanceLabel: testName},
			},
		}
		fakeClient := fake.NewClient(namespace)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		ctx := t.Context()
		dynakube, err := controller.getDynakubeOrCleanup(ctx, testName, testNamespace)

		assert.Nil(t, dynakube)
		require.NoError(t, err)

		err = fakeClient.Get(ctx, client.ObjectKey{Name: testNamespace}, namespace)
		require.NoError(t, err)
		assert.NotContains(t, namespace.Labels, dtwebhook.InjectionInstanceLabel)
	})
	t.Run("return unknown error", func(t *testing.T) {
		controller := &Controller{
			client:    errorClient{},
			apiReader: errorClient{},
		}

		ctx := t.Context()
		dk, err := controller.getDynakubeOrCleanup(ctx, testName, testNamespace)

		assert.Nil(t, dk)
		require.EqualError(t, err, "fake error")
	})
}

func TestTokenConditions(t *testing.T) {
	ctx := t.Context()

	t.Run("token condition error is set if token are invalid", func(t *testing.T) {
		fakeClient := fake.NewClient()
		dk := &dynakube.DynaKube{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		_, err := controller.setupTokensAndClient(ctx, dk)

		require.Error(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, "secrets \"\" not found")
		assert.Empty(t, dk.Status.DynatraceAPI.LastTokenScopeRequest, "LastTokenProbeTimestamp should be Nil if token retrieval did not work.")
	})
	t.Run("token condition is set if token are valid", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}
		fakeClient := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken: []byte(testAPIToken),
			},
		})
		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().GetTokenScopes(anyCtx, testAPIToken).Return(dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDynatraceClientBuild(mockDtcBuilder, mockClient)

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}

		_, err := controller.setupTokensAndClient(ctx, dk)

		require.NoError(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionTrue, dynakube.ReasonTokenReady, TokenWithoutDataIngestConditionMessage)

		fakeClient = fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.APIToken:        []byte(testAPIToken),
				dtclient.DataIngestToken: []byte(testAPIToken),
			},
		})
		controller = &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}
		_, err = controller.setupTokensAndClient(ctx, dk)

		require.NoError(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionTrue, dynakube.ReasonTokenReady, TokenReadyConditionMessage)
	})
	t.Run("token status condition remains unchanged unless new condition doesn't match", func(t *testing.T) {
		transitionTime := metav1.NewTime(time.Now())
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{
						Type:               dynakube.TokenConditionType,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: transitionTime,
						Reason:             dynakube.ReasonTokenReady,
						Message:            testMessage,
					},
				},
			},
		}

		newCondition := metav1.Condition{
			Type:               dynakube.TokenConditionType,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: transitionTime,
			Reason:             dynakube.ReasonTokenReady,
			Message:            testMessage,
		}

		controller := &Controller{}
		controller.setAndLogCondition(dk, newCondition)

		assertCondition(t, dk, newCondition.Type, newCondition.Status, newCondition.Reason, newCondition.Message)
	})
	t.Run("deprecated conditions types are removed", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{
						Type: dynakube.PaaSTokenConditionType,
					},
					{
						Type: dynakube.APITokenConditionType,
					},
					{
						Type: dynakube.DataIngestTokenConditionType,
					},
				},
			},
		}

		controller := &Controller{}
		controller.removeDeprecatedConditionTypes(dk)
		assert.Empty(t, dk.Status.Conditions)
	})
}

func TestSetupIstio(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:      testAPIURL,
			EnableIstio: true,
		},
	}

	t.Run("no istio installed + EnableIstio: true => error", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := false
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
		}
		istioClient, err := controller.setupIstioClient(dk)
		require.Error(t, err)
		assert.Nil(t, istioClient)
	})
	t.Run("success", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := true
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
		}
		istioClient, err := controller.setupIstioClient(dk)
		require.NoError(t, err)
		require.NotNil(t, istioClient)

		istioReconciler := istio.NewReconciler(istioClient)
		require.NotNil(t, istioClient)

		err = istioReconciler.ReconcileAPIUrl(ctx, dk)

		require.NoError(t, err)

		expectedName := istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OperatorComponent)
		serviceEntry, err := fakeIstio.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		virtualService, err := fakeIstio.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
}

func fakeIstioClientBuilder(t *testing.T, fakeIstio *fakeistio.Clientset, isIstioInstalled bool) istio.ClientBuilder {
	return func(_ *rest.Config, owner metav1.Object) (*istio.Client, error) {
		if isIstioInstalled == true {
			fakeDiscovery, ok := fakeIstio.Discovery().(*fakediscovery.FakeDiscovery)
			fakeDiscovery.Resources = []*metav1.APIResourceList{{GroupVersion: istio.IstioGVR}}

			if !ok {
				t.Fatal("couldn't convert Discovery() to *FakeDiscovery")
			}
		}

		return &istio.Client{
			IstioClientset: fakeIstio,
			Owner:          owner,
		}, nil
	}
}

func assertCondition(t *testing.T, dk *dynakube.DynaKube, expectedConditionType string, expectedConditionStatus metav1.ConditionStatus, expectedReason string, expectedMessage string) { //nolint:revive // argument-limit
	t.Helper()

	actualCondition := meta.FindStatusCondition(dk.Status.Conditions, expectedConditionType)
	require.NotNil(t, actualCondition)
	assert.Equal(t, expectedConditionStatus, actualCondition.Status)
	assert.Equal(t, expectedReason, actualCondition.Reason)
	assert.Equal(t, expectedMessage, actualCondition.Message)
}

func TestTokenConditionsOptionalScopes(t *testing.T) {
	t.Run("conditions not set", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		fakeClient := fake.NewClient()

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.Error(t, err)

		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, "secrets \""+testName+"\" not found")
		assert.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead))
		assert.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite))
	})
	t.Run("no missing scopes", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		cond = meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
	})
	t.Run("one optional scopes missing", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		cond = meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
	})
	t.Run("all optional scopes missing", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		cond = meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
	})
}

func TestLastErrorFromCondition(t *testing.T) {
	t.Run("status nil => nil returned", func(t *testing.T) {
		dkStatus := &dynakube.DynaKubeStatus{}
		err := lastErrorFromCondition(dkStatus)
		assert.NoError(t, err)
	})
	t.Run("status with token error => error returned", func(t *testing.T) {
		dkStatus := &dynakube.DynaKubeStatus{
			Conditions: []metav1.Condition{
				{
					Type:    dynakube.TokenConditionType,
					Status:  metav1.ConditionTrue,
					Reason:  dynakube.ReasonTokenError,
					Message: testMessage,
				},
			},
		}

		err := lastErrorFromCondition(dkStatus)
		require.Error(t, err)
	})
}

func createFakeControllerAndClients(t *testing.T, tokenScopes dtclient.TokenScopes) *Controller {
	fakeClient := fake.NewClient(createAPISecret())

	fakeDtClient := dtclientmock.NewClient(t)
	fakeDtClient.EXPECT().GetTokenScopes(anyCtx, testAPIToken).Return(tokenScopes, nil)

	fakeBuilder := dtbuildermock.NewBuilder(t)
	mockDynatraceClientBuild(fakeBuilder, fakeDtClient)

	return &Controller{
		client:                 fakeClient,
		apiReader:              fakeClient,
		dynatraceClientBuilder: fakeBuilder,
	}
}

func mockDynatraceClientBuild(builder *dtbuildermock.Builder, client *dtclientmock.Client) {
	builder.EXPECT().SetDynakube(mock.AnythingOfType("dynakube.DynaKube")).Return(builder)
	builder.EXPECT().SetTokens(mock.AnythingOfType("token.Tokens")).Return(builder)
	builder.EXPECT().Build(anyCtx).Return(client, nil)
}

func createDynakubeWithK8SMonitoring() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"feature.dynatrace.com/automatic-kubernetes-api-monitoring": "true",
			},
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		},
	}
}

func createAPISecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			dtclient.APIToken: []byte(testAPIToken),
		},
	}
}

func createCRD(t *testing.T) *apiextensionsv1.CustomResourceDefinition {
	t.Setenv(k8senv.AppVersion, "1.0.0")

	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8scrd.DynaKubeName,
			Labels: map[string]string{
				k8slabel.AppVersionLabel: "1.0.0",
			},
		},
	}
}
