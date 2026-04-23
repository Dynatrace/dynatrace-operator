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
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	tokenclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testAPIToken  = "test-api-token"
	testUUID      = "test-uuid"
	testHost      = "test-host"
	testName      = "test-name"
	testNamespace = "test-namespace"
	testAPIURL    = "https://" + testHost + "/e/" + testUUID + "/api"
	testMessage   = "test-message"
)

var (
	anyCtx      = mock.MatchedBy(func(context.Context) bool { return true })
	anyDynaKube = mock.MatchedBy(func(*dynakube.DynaKube) bool { return true })
)

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
		serverError := &core.HTTPError{StatusCode: http.StatusTooManyRequests}

		result, err := controller.handleError(ctx, oldDynakube, serverError, oldDynakube.Status)
		require.NoError(t, err)
		assert.Equal(t, fastRequeueInterval, result.RequeueAfter)
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

	const (
		tokenValue = "this-is-a-token"
	)

	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{APIURL: "https://test123.dev.dynatracelabs.com/api"},
	}
	tokenSecretBase := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkBase.Tokens(),
			Namespace: dkBase.Namespace,
		},
		Data: map[string][]byte{
			token.APIKey: []byte(tokenValue),
		},
	}

	t.Run("no tokens => error + condition", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dtClient, err := controller.setupTokensAndClient(ctx, dk)
		require.Error(t, err)
		assert.Nil(t, dtClient)
		assertTokenCondition(t, dk, true)
	})

	t.Run("dtClient Factory error => error + condition", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		tokenSecret := tokenSecretBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk, tokenSecret)

		controller := &Controller{
			client:          fakeClient,
			apiReader:       fakeClient,
			dtClientFactory: newErrorClientFactory(errors.New("BOOM")),
		}

		dtClient, err := controller.setupTokensAndClient(ctx, dk)
		require.Error(t, err)
		assert.Nil(t, dtClient)
		assertTokenCondition(t, dk, true)
	})
	t.Run("tokens + dtClient ok => no error", func(t *testing.T) {
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
				token.APIKey: []byte("this is a token"),
			},
		}
		fakeClient := fake.NewClientWithIndex(dk, tokens)

		mockedTokenClient := tokenclientmock.NewClient(t)
		mockedTokenClient.EXPECT().GetScopes(anyCtx, "this is a token").Return([]string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeSettingsRead,
			tokenclient.ScopeSettingsWrite,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		}, nil).Once()

		controller := &Controller{
			client:          fakeClient,
			apiReader:       fakeClient,
			dtClientFactory: newClientFactory(&dynatrace.Client{Token: mockedTokenClient}),
		}

		dtClient, err := controller.setupTokensAndClient(ctx, dk)
		require.NoError(t, err)
		assert.NotNil(t, dtClient)
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

	type mockReconciler interface {
		On(methodName string, arguments ...any) *mock.Call
	}

	expectReconcileError := func(t *testing.T, reconciler mockReconciler, reconcileError *error, args ...any) {
		t.Helper()
		uniqueError := fmt.Errorf("BOOM %T", reconciler)
		reconciler.On("Reconcile", append([]any{anyCtx}, args...)...).Return(uniqueError).Once()
		t.Cleanup(func() {
			assert.ErrorIs(t, *reconcileError, uniqueError)
		})
	}

	t.Run("all components reconciled, even in case of errors", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)

		mockOneAgentReconciler := newMockOneAgentReconciler(t)
		mockActiveGateReconciler := newMockActiveGateReconciler(t)
		mockInjectionReconciler := newMockInjectionReconciler(t)
		mockLogMonitoringReconciler := newMockLogMonitoringReconciler(t)

		mockExtensionReconciler := newMockDynakubeReconciler(t)
		mockKSPMReconciler := newMockDtSettingReconciler(t)
		mockK8sEntityReconciler := newMockDtSettingReconciler(t)
		mockOtelcReconciler := newMockDynakubeReconciler(t)
		mockIstioReconciler := newMockIstioReconciler(t)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,

			logMonitoringReconciler: mockLogMonitoringReconciler,
			extensionReconciler:     mockExtensionReconciler,
			istioReconciler:         mockIstioReconciler,
			otelcReconciler:         mockOtelcReconciler,
			kspmReconciler:          mockKSPMReconciler,
			k8sEntityReconciler:     mockK8sEntityReconciler,
			oneAgentReconciler:      mockOneAgentReconciler,
			activeGateReconciler:    mockActiveGateReconciler,
			injectionReconciler:     mockInjectionReconciler,
		}
		dtClient := &dynatrace.Client{Settings: settings.NewClient(nil)}

		var err error

		expectReconcileError(t, mockOneAgentReconciler, &err, dk, dtClient, token.Tokens(nil))
		expectReconcileError(t, mockActiveGateReconciler, &err, dk, dtClient, token.Tokens(nil))
		expectReconcileError(t, mockInjectionReconciler, &err, dtClient, dk)
		expectReconcileError(t, mockLogMonitoringReconciler, &err, dtClient, dk)
		expectReconcileError(t, mockExtensionReconciler, &err, dk)
		expectReconcileError(t, mockOtelcReconciler, &err, dk)
		expectReconcileError(t, mockKSPMReconciler, &err, settings.NewClient(nil), dk)
		expectReconcileError(t, mockK8sEntityReconciler, &err, settings.NewClient(nil), dk)

		err = controller.reconcileComponents(ctx, dtClient, dk)
		require.Error(t, err)
	})

	t.Run("exit early in case of no oneagent conncection info", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := newMockActiveGateReconciler(t)
		mockExtensionReconciler := newMockDynakubeReconciler(t)
		mockOtelcReconciler := newMockDynakubeReconciler(t)
		k8sEntityReconciler := newMockDtSettingReconciler(t)
		mockIstioReconciler := newMockIstioReconciler(t)
		mockKSPMReconciler := newMockKspmReconciler(t)

		dtClient := &dynatrace.Client{Settings: settings.NewClient(nil)}

		mockLogMonitoringReconciler := newMockLogMonitoringReconciler(t)
		mockLogMonitoringReconciler.EXPECT().Reconcile(anyCtx, dtClient, mock.Anything).Return(oaconnectioninfo.NoOneAgentCommunicationEndpointsError).Once()

		controller := &Controller{
			client:                  fakeClient,
			apiReader:               fakeClient,
			activeGateReconciler:    mockActiveGateReconciler,
			logMonitoringReconciler: mockLogMonitoringReconciler,
			extensionReconciler:     mockExtensionReconciler,
			otelcReconciler:         mockOtelcReconciler,
			k8sEntityReconciler:     k8sEntityReconciler,
			istioReconciler:         mockIstioReconciler,
			kspmReconciler:          mockKSPMReconciler,
		}

		var err error
		expectReconcileError(t, mockActiveGateReconciler, &err, dk, dtClient, token.Tokens(nil))
		expectReconcileError(t, mockExtensionReconciler, &err, dk)
		expectReconcileError(t, mockOtelcReconciler, &err, dk)
		expectReconcileError(t, k8sEntityReconciler, &err, settings.NewClient(nil), dk)
		expectReconcileError(t, mockKSPMReconciler, &err, settings.NewClient(nil), dk)

		err = controller.reconcileComponents(ctx, dtClient, dk)
		require.Error(t, err)
	})
}

func TestReconcileDynaKube(t *testing.T) {
	ctx := t.Context()
	baseDK := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}

	fakeClient := fake.NewClient(baseDK, createCRD(t), createAPISecret())

	mockedTokenClient := tokenclientmock.NewClient(t)
	mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return([]string{
		tokenclient.ScopeDataExport,
		tokenclient.ScopeSettingsRead,
		tokenclient.ScopeSettingsWrite,
		tokenclient.ScopeInstallerDownload,
		tokenclient.ScopeActiveGateTokenCreate,
	}, nil)

	dtClient := &dynatrace.Client{
		Settings: settings.NewClient(nil),
		Token:    mockedTokenClient,
	}

	mockDeploymentMetadataReconciler := newMockDynakubeReconciler(t)
	mockDeploymentMetadataReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube).Return(nil)

	mockProxyReconciler := newMockDynakubeReconciler(t)
	mockProxyReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube).Return(nil)

	mockOneAgentReconciler := newMockOneAgentReconciler(t)
	mockOneAgentReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube, mock.Anything, mock.Anything).Return(nil)

	mockActiveGateReconciler := newMockActiveGateReconciler(t)
	mockActiveGateReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube, mock.Anything, mock.Anything).Return(nil)

	mockInjectionReconciler := newMockInjectionReconciler(t)
	mockInjectionReconciler.EXPECT().Reconcile(anyCtx, dtClient, anyDynaKube).Return(nil)

	mockLogMonitoringReconciler := newMockLogMonitoringReconciler(t)
	mockLogMonitoringReconciler.EXPECT().Reconcile(anyCtx, dtClient, anyDynaKube).Return(nil)

	mockExtensionReconciler := newMockDynakubeReconciler(t)

	mockExtensionReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube).Return(nil)

	mockOtelcReconciler := newMockDynakubeReconciler(t)
	mockOtelcReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube).Return(nil)

	mockIstioReconciler := newMockIstioReconciler(t)
	mockIstioReconciler.EXPECT().ReconcileAPIURL(anyCtx, anyDynaKube).Return(nil)

	mockKSPMReconciler := newMockDtSettingReconciler(t)
	mockKSPMReconciler.EXPECT().Reconcile(anyCtx, settings.NewClient(nil), anyDynaKube).Return(nil)

	mockK8sEntityReconciler := newMockDtSettingReconciler(t)
	mockK8sEntityReconciler.EXPECT().Reconcile(anyCtx, settings.NewClient(nil), anyDynaKube).Return(nil)

	baseController := &Controller{
		apiReader:                    fakeClient,
		client:                       fakeClient,
		deploymentMetadataReconciler: mockDeploymentMetadataReconciler,
		dtClientFactory:              newClientFactory(dtClient),
		extensionReconciler:          mockExtensionReconciler,
		injectionReconciler:          mockInjectionReconciler,
		istioReconciler:              mockIstioReconciler,
		logMonitoringReconciler:      mockLogMonitoringReconciler,
		otelcReconciler:              mockOtelcReconciler,
		proxyReconciler:              mockProxyReconciler,
		kspmReconciler:               mockKSPMReconciler,
		k8sEntityReconciler:          mockK8sEntityReconciler,
		oneAgentReconciler:           mockOneAgentReconciler,
		activeGateReconciler:         mockActiveGateReconciler,
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
		dk := baseDK.DeepCopy()
		dk.Spec.APIURL = testAPIURL
		dk.Spec.EnableIstio = true

		fakeClient := fake.NewClientWithIndex(dk, createCRD(t), createAPISecret())

		controller := baseController
		controller.client = fakeClient
		controller.apiReader = fakeClient

		result, err := controller.Reconcile(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("reconciling the controller with istio enabled (but without valid API URL) should fail", func(t *testing.T) {
		dk := baseDK.DeepCopy()
		dk.Spec.EnableIstio = true

		fakeClient := fake.NewClientWithIndex(dk, createAPISecret())

		controller := baseController
		controller.client = fakeClient
		controller.apiReader = fakeClient

		result, err := controller.Reconcile(ctx, request)
		require.Error(t, err)
		assert.NotNil(t, result)
	})
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

	t.Run("token condition error is set if token secret is missing", func(t *testing.T) {
		fakeClient := fake.NewClient()
		dk := &dynakube.DynaKube{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		_, err := controller.setupTokensAndClient(ctx, dk)

		require.Error(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, TokenVerificationFailedConditionMessage)
	})
	t.Run("token condition error is set if token verification fails", func(t *testing.T) {
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
				token.APIKey: []byte(testAPIToken),
			},
		})

		mockedTokenClient := tokenclientmock.NewClient(t)
		mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return(nil, &core.HTTPError{
			Message:    "test-error",
			StatusCode: 1234,
		})

		controller := &Controller{
			client:          fakeClient,
			apiReader:       fakeClient,
			dtClientFactory: newClientFactory(&dynatrace.Client{Token: mockedTokenClient}),
		}

		_, err := controller.setupTokensAndClient(ctx, dk)

		require.Error(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, TokenVerificationFailedConditionMessage)
	})
	t.Run("token condition is set if required scopes are missing", func(t *testing.T) {
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
				token.APIKey: []byte(testAPIToken),
			},
		})

		mockedTokenClient := tokenclientmock.NewClient(t)
		mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return([]string{}, nil)

		dtClientFactory := newClientFactory(&dynatrace.Client{Token: mockedTokenClient})

		controller := &Controller{
			client:          fakeClient,
			apiReader:       fakeClient,
			dtClientFactory: dtClientFactory,
		}

		_, err := controller.setupTokensAndClient(ctx, dk)

		require.Error(t, err)
		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, fmt.Sprintf(TokenScopesMissingConditionMessage, "DataExport, InstallerDownload"))
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
		controller.setCondition(dk, newCondition)

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

		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, TokenVerificationFailedConditionMessage)
		assert.False(t, dk.Status.Tenant.APITokenSettingsReadAvailable)
		assert.False(t, dk.Status.Tenant.APITokenSettingsWriteAvailable)
	})
	t.Run("no missing scopes", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, []string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeSettingsRead,
			tokenclient.ScopeSettingsWrite,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		assert.True(t, dk.Status.Tenant.APITokenSettingsReadAvailable)
		assert.True(t, dk.Status.Tenant.APITokenSettingsWriteAvailable)
	})
	t.Run("one optional scopes missing", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, []string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeSettingsRead,
			tokenclient.ScopeSettingsWrite,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		assert.True(t, dk.Status.Tenant.APITokenSettingsReadAvailable)
		assert.True(t, dk.Status.Tenant.APITokenSettingsWriteAvailable)
	})
	t.Run("all optional scopes missing", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, []string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(t.Context(), dk)
		require.NoError(t, err)

		assert.False(t, dk.Status.Tenant.APITokenSettingsReadAvailable)
		assert.False(t, dk.Status.Tenant.APITokenSettingsWriteAvailable)
	})
	t.Run("state of the optional scopes condition", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()
		tokenScopesWithoutSettingsWrite := []string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeSettingsRead,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		}
		tokenScopes := []string{
			tokenclient.ScopeDataExport,
			tokenclient.ScopeSettingsRead,
			tokenclient.ScopeSettingsWrite,
			tokenclient.ScopeInstallerDownload,
			tokenclient.ScopeActiveGateTokenCreate,
		}

		testCases := []struct {
			description          string
			firstCallReturns     []string
			secondCallReturns    []string
			firstCallCondExists  bool
			secondCallCondExists bool
		}{
			{
				description:          "condition not exists",
				firstCallReturns:     tokenScopes,
				secondCallReturns:    tokenScopes,
				firstCallCondExists:  false,
				secondCallCondExists: false,
			},
			{
				description:          "condition exists",
				firstCallReturns:     tokenScopesWithoutSettingsWrite,
				secondCallReturns:    tokenScopesWithoutSettingsWrite,
				firstCallCondExists:  true,
				secondCallCondExists: true,
			},
			{
				description:          "condition set",
				firstCallReturns:     tokenScopes,
				secondCallReturns:    tokenScopesWithoutSettingsWrite,
				firstCallCondExists:  false,
				secondCallCondExists: true,
			},
			{
				description:          "condition deleted",
				firstCallReturns:     tokenScopesWithoutSettingsWrite,
				secondCallReturns:    tokenScopes,
				firstCallCondExists:  true,
				secondCallCondExists: false,
			},
		}

		for _, testCase := range testCases {
			fakeClient := fake.NewClient(createAPISecret())

			mockedTokenClient := tokenclientmock.NewClient(t)
			mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return(testCase.firstCallReturns, nil).Once()
			mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return(testCase.secondCallReturns, nil).Once()

			controller := &Controller{
				client:          fakeClient,
				apiReader:       fakeClient,
				dtClientFactory: newClientFactory(&dynatrace.Client{Token: mockedTokenClient}),
			}

			_, err := controller.setupTokensAndClient(t.Context(), dk)
			require.NoError(t, err, testCase.description)

			condition := meta.FindStatusCondition(*dk.Conditions(), conditionTypeAPITokenOptionalScopes)
			if testCase.firstCallCondExists {
				assert.NotNil(t, condition, testCase.description)
			} else {
				assert.Nil(t, condition, testCase.description)
			}

			_, err = controller.setupTokensAndClient(t.Context(), dk)
			require.NoError(t, err)

			condition = meta.FindStatusCondition(*dk.Conditions(), conditionTypeAPITokenOptionalScopes)
			if testCase.secondCallCondExists {
				assert.NotNil(t, condition, testCase.description)
			} else {
				assert.Nil(t, condition, testCase.description)
			}
		}
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

func createFakeControllerAndClients(t *testing.T, tokenScopes []string) *Controller {
	fakeClient := fake.NewClient(createAPISecret())

	mockedTokenClient := tokenclientmock.NewClient(t)
	mockedTokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return(tokenScopes, nil)

	return &Controller{
		client:          fakeClient,
		apiReader:       fakeClient,
		dtClientFactory: newClientFactory(&dynatrace.Client{Token: mockedTokenClient}),
	}
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
			token.APIKey: []byte(testAPIToken),
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

func newClientFactory(dtClient *dynatrace.Client) dynatrace.ClientFactory {
	return func(_ context.Context, _ client.Reader, _ *dynakube.DynaKube, _, _, _ string) (*dynatrace.Client, error) {
		return dtClient, nil
	}
}

func newErrorClientFactory(err error) dynatrace.ClientFactory {
	return func(_ context.Context, _ client.Reader, _ *dynakube.DynaKube, _, _, _ string) (*dynatrace.Client, error) {
		return nil, err
	}
}
