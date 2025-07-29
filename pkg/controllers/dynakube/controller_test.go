package dynakube

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	ag "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm"
	logmon "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring"
	oneagentcontroller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	dtbuildermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID              = "test-uid"
	testMEID             = "KUBERNETES_CLUSTER-0E30FE4BF2007587"
	testPaasToken        = "test-paas-token"
	testAPIToken         = "test-api-token"
	testVersion          = "1.217.1.12345-678910"
	testComponentVersion = "test-component-version"

	testUUID     = "test-uuid"
	testObjectID = "test-object-id"

	testHost     = "test-host"
	testPort     = uint32(1234)
	testProtocol = "test-protocol"

	testAnotherHost     = "test-another-host"
	testAnotherPort     = uint32(5678)
	testAnotherProtocol = "test-another-protocol"

	testName      = "test-name"
	testNamespace = "test-namespace"

	testAPIURL = "https://" + testHost + "/e/" + testUUID + "/api"
)

func TestGetDynakubeOrCleanup(t *testing.T) {
	ctx := context.Background()
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
		fakeClient := fake.NewClientWithIndex(markedNamespace)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dk, err := controller.getDynakubeOrCleanup(ctx, request.Name, request.Namespace)
		require.NoError(t, err)
		assert.Nil(t, dk)

		unmarkedNamespace := &corev1.Namespace{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: markedNamespace.Name}, unmarkedNamespace)
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
		fakeClient := fake.NewClientWithIndex(expectedDynakube)
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
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		controller := &Controller{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := controller.Reconcile(context.Background(), reconcile.Request{})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandleError(t *testing.T) {
	ctx := context.Background()
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
		fakeClient := fake.NewClientWithIndex(oldDynakube)
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

func TestAPIError(t *testing.T) {
	mockClient := createDTMockClient(t, dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, dtclient.TokenScopes{dtclient.TokenScopeActiveGateTokenCreate})
	mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testVersion, nil)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:   testAPIURL,
			OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{HostInjectSpec: oneagent.HostInjectSpec{}}},
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
			LogMonitoring: &logmonitoring.Spec{},
		},
		Status: *getTestDynkubeStatus(),
	}

	t.Run("should return error result on 503", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusServiceUnavailable, Message: "Service unavailable"})
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
	t.Run("should return error result on 429", func(t *testing.T) {
		mockClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, dtclient.ServerError{Code: http.StatusTooManyRequests, Message: "Too many requests"})
		controller := createFakeClientAndReconciler(t, mockClient, dk, testPaasToken, testAPIToken)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, fastUpdateInterval, result.RequeueAfter)
	})
}

func TestSetupTokensAndClient(t *testing.T) {
	ctx := context.Background()
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
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build", mock.Anything).Return(nil, errors.New("BOOM"))

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
		mockedDtc.On("GetTokenScopes", mock.Anything, "this is a token").Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build", mock.Anything).Return(mockedDtc, nil)

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
	ctx := context.Background()
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

	t.Run("all components reconciled, even in case of errors", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)
		// ReconcileCodeModuleCommunicationHosts
		mockOneAgentReconciler := controllermock.NewReconciler(t)
		mockOneAgentReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockInjectionReconciler := controllermock.NewReconciler(t)
		mockInjectionReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockLogMonitoringReconciler := controllermock.NewReconciler(t)
		mockLogMonitoringReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockExtensionReconciler := controllermock.NewReconciler(t)
		mockExtensionReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockOtelcReconciler := controllermock.NewReconciler(t)
		mockOtelcReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockKSPMReconciler := controllermock.NewReconciler(t)
		mockKSPMReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
			fs:        afero.Afero{Fs: afero.NewMemMapFs()},

			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			injectionReconcilerBuilder:     createInjectionReconcilerBuilder(mockInjectionReconciler),
			oneAgentReconcilerBuilder:      createOneAgentReconcilerBuilder(mockOneAgentReconciler),
			logMonitoringReconcilerBuilder: createLogMonitoringReconcilerBuilder(mockLogMonitoringReconciler),
			extensionReconcilerBuilder:     createExtensionReconcilerBuilder(mockExtensionReconciler),
			otelcReconcilerBuilder:         createOtelcReconcilerBuilder(mockOtelcReconciler),
			kspmReconcilerBuilder:          createKSPMReconcilerBuilder(mockKSPMReconciler),
		}
		mockedDtc := dtclientmock.NewClient(t)

		err := controller.reconcileComponents(ctx, mockedDtc, nil, dk)

		require.Error(t, err)
		// goerrors.Join concats errors with \n
		assert.Len(t, strings.Split(err.Error(), "\n"), 7) // ActiveGate, Extension, OtelC, OneAgent LogMonitoring, and Injection reconcilers
	})

	t.Run("exit early in case of no oneagent conncection info", func(t *testing.T) {
		dk := dkBaser.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockExtensionReconciler := controllermock.NewReconciler(t)
		mockExtensionReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockOtelcReconciler := controllermock.NewReconciler(t)
		mockOtelcReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		mockLogMonitoringReconciler := controllermock.NewReconciler(t)
		mockLogMonitoringReconciler.On("Reconcile", mock.Anything).Return(oaconnectioninfo.NoOneAgentCommunicationHostsError)

		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			fs:                             afero.Afero{Fs: afero.NewMemMapFs()},
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			logMonitoringReconcilerBuilder: createLogMonitoringReconcilerBuilder(mockLogMonitoringReconciler),
			extensionReconcilerBuilder:     createExtensionReconcilerBuilder(mockExtensionReconciler),
			otelcReconcilerBuilder:         createOtelcReconcilerBuilder(mockOtelcReconciler),
		}
		mockedDtc := dtclientmock.NewClient(t)

		err := controller.reconcileComponents(ctx, mockedDtc, nil, dk)

		require.Error(t, err)
		// goerrors.Join concats errors with \n
		assert.Len(t, strings.Split(err.Error(), "\n"), 3) // ActiveGate, Extension, OtelC, no OneAgent connection info is not an error
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

func createLogMonitoringReconcilerBuilder(reconciler controllers.Reconciler) logmon.ReconcilerBuilder {
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
	return func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube) controllers.Reconciler {
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
	return func(_ dtclient.Client, _ *dynakube.DynaKube, _ string) controllers.Reconciler {
		return reconciler
	}
}

type errorClient struct {
	client.Client
}

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
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
		ctx := context.Background()
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
		ctx := context.Background()
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

		ctx := context.Background()
		dk, err := controller.getDynakubeOrCleanup(ctx, testName, testNamespace)

		assert.Nil(t, dk)
		require.EqualError(t, err, "fake error")
	})
}

func TestTokenConditions(t *testing.T) {
	ctx := context.Background()

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
		mockClient.On("GetTokenScopes", mock.Anything, testAPIToken).Return(dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		mockDtcBuilder := dtbuildermock.NewBuilder(t)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("Build", mock.Anything).Return(mockClient, nil)

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
}

func TestSetupIstio(t *testing.T) {
	ctx := context.Background()
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

func getTestDynkubeStatus() *dynakube.DynaKubeStatus {
	return &dynakube.DynaKubeStatus{
		ActiveGate: activegate.Status{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testUUID,
				Endpoints:  "endpoint",
			},
		},
		OneAgent: oneagent.Status{
			ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testUUID,
					Endpoints:  "endpoint",
				},
				CommunicationHosts: []oneagent.CommunicationHostStatus{
					{
						Protocol: "http",
						Host:     "localhost",
						Port:     9999,
					},
				},
			},
		},
		KubeSystemUUID: testUID,
		Conditions: []metav1.Condition{
			{
				Type:   dtclient.ConditionTypeAPITokenSettingsRead,
				Status: metav1.ConditionTrue,
			},
		},
	}
}

func createTenantSecrets(dk *dynakube.DynaKube) []client.Object {
	return []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.OneAgent().GetTenantSecret(),
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				connectioninfo.TenantTokenKey: []byte("test-token"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.ActiveGate().GetTenantSecretName(),
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				connectioninfo.TenantTokenKey: []byte("test-token"),
			},
		},
	}
}

func TestTokenConditionsOptionalScopes(t *testing.T) {
	ctx := context.Background()

	t.Run("conditions not set", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		fakeClient := fake.NewClient()

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		_, err := controller.setupTokensAndClient(ctx, dk)
		require.Error(t, err)

		assertCondition(t, dk, dynakube.TokenConditionType, metav1.ConditionFalse, dynakube.ReasonTokenError, "secrets \""+testName+"\" not found")
		assert.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead))
		assert.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite))
	})
	t.Run("no missing scopes", func(t *testing.T) {
		dk := createDynakubeWithK8SMonitoring()

		controller := createFakeControllerAndClients(t, dtclient.TokenScopes{
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(ctx, dk)
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
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(ctx, dk)
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
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		})

		_, err := controller.setupTokensAndClient(ctx, dk)
		require.NoError(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		cond = meta.FindStatusCondition(dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
	})
}

func createFakeControllerAndClients(t *testing.T, tokenScopes dtclient.TokenScopes) *Controller {
	fakeClient := fake.NewClient(createAPISecret())

	fakeDtClient := dtclientmock.NewClient(t)
	fakeDtClient.On("GetTokenScopes", mock.Anything, testAPIToken).Return(tokenScopes, nil)

	fakeBuilder := dtbuildermock.NewBuilder(t)
	fakeBuilder.On("Build", mock.Anything).Return(fakeDtClient, nil)
	fakeBuilder.On("SetDynakube", mock.Anything).Return(fakeBuilder, nil)
	fakeBuilder.On("SetTokens", mock.Anything).Return(fakeBuilder, nil)

	return &Controller{
		client:                 fakeClient,
		apiReader:              fakeClient,
		dynatraceClientBuilder: fakeBuilder,
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
			dtclient.APIToken: []byte(testAPIToken),
		},
	}
}
