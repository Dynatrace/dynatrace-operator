package dynakube

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	mockedclient "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	mockcontroller "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	mockconnectioninfo "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/connectioninfo"
	dtClientMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/dynatraceclient"
	mockinjection "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/injection"
	mockversion "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/oci/registry"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	fakecontainer "github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID              = "test-uid"
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

	testApiUrl = "https://" + testHost + "/e/" + testUUID + "/api"
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
		expectedDynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{APIURL: "this-is-an-api-url"},
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
		assert.Equal(t, expectedDynakube.ApiUrl(), dk.ApiUrl())
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
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{APIURL: "this-is-an-api-url"},
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
		expectedDynakube.Status = dynatracev1beta1.DynaKubeStatus{
			Phase: status.Running,
		}

		result, err := controller.handleError(ctx, oldDynakube, nil, oldDynakube.Status)

		require.NoError(t, err)
		assert.Equal(t, controller.requeueAfter, result.RequeueAfter)

		dynakube := &dynatracev1beta1.DynaKube{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: expectedDynakube.Name, Namespace: expectedDynakube.Namespace}, dynakube)
		require.NoError(t, err)
		assert.Equal(t, expectedDynakube.Status.Phase, dynakube.Status.Phase)
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

		dynakube := &dynatracev1beta1.DynaKube{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: oldDynakube.Name, Namespace: oldDynakube.Namespace}, dynakube)
		require.NoError(t, err)
		assert.Equal(t, status.Error, dynakube.Status.Phase)
	})
}

func TestSetupTokensAndClient(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{APIURL: "https://test123.dev.dynatracelabs.com/api"},
	}

	t.Run("no tokens => error + condition", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dynakube)
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dynakube)
		require.Error(t, err)
		assert.Nil(t, dtc)
		assertTokenCondition(t, dynakube, true)
	})

	t.Run("client builder error => error + condition", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		tokens := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakube.Tokens(),
				Namespace: dynakube.Namespace,
			},
			Data: map[string][]byte{
				dtclient.ApiToken: []byte("this is a token"),
			},
		}
		fakeClient := fake.NewClientWithIndex(dynakube, tokens)

		mockDtcBuilder := dtClientMock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("BuildWithTokenVerification", mock.Anything).Return(nil, errors.New("BOOM"))

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dynakube)
		require.Error(t, err)
		assert.Nil(t, dtc)
		assertTokenCondition(t, dynakube, true)
	})
	t.Run("tokens + dtclient ok => no error", func(t *testing.T) {
		// There is also a pull-secret created here, however testing it here is a bit counterintuitive.
		// TODO: Make the pull-secret reconciler mockable, so we can improve this test.
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.CustomPullSecret = "custom"
		tokens := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakube.Tokens(),
				Namespace: dynakube.Namespace,
			},
			Data: map[string][]byte{
				dtclient.ApiToken: []byte("this is a token"),
			},
		}
		fakeClient := fake.NewClientWithIndex(dynakube, tokens)

		mockedDtc := mockedclient.NewClient(t)

		mockDtcBuilder := dtClientMock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("BuildWithTokenVerification", mock.Anything).Return(mockedDtc, nil)

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
		}

		dtc, err := controller.setupTokensAndClient(ctx, dynakube)
		require.NoError(t, err)
		assert.NotNil(t, dtc)
		assertTokenCondition(t, dynakube, false)
	})
}

func assertTokenCondition(t *testing.T, dynakube *dynatracev1beta1.DynaKube, hasError bool) {
	condition := dynakube.Status.Conditions[0]
	assert.Equal(t, dynatracev1beta1.TokenConditionType, condition.Type)

	if hasError {
		assert.Equal(t, dynatracev1beta1.ReasonTokenError, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	} else {
		assert.Equal(t, dynatracev1beta1.ReasonTokenReady, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	}
}

func TestReconcileComponents(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:     "this-is-an-api-url",
			OneAgent:   dynatracev1beta1.OneAgentSpec{CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{}},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("all components reconciled, even in case of errors", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dynakube)
		// ReconcileCodeModuleCommunicationHosts
		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)
		mockConnectionInfoReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(errors.New("BOOM"))
		mockConnectionInfoReconciler.On("ReconcileOneAgent", mock.Anything, dynakube).Return(nil)

		mockVersionReconciler := mockversion.NewReconciler(t)
		mockVersionReconciler.On("ReconcileOneAgent", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)

		mockInjectionReconciler := mockinjection.NewReconciler(t)
		mockInjectionReconciler.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

		controller := &Controller{
			client:                fakeClient,
			apiReader:             fakeClient,
			scheme:                scheme.Scheme,
			fs:                    afero.Afero{Fs: afero.NewMemMapFs()},
			registryClientBuilder: createFakeRegistryClientBuilder(t),

			versionReconcilerBuilder:        createVersionReconcilerBuilder(mockVersionReconciler),
			connectionInfoReconcilerBuilder: createConnectionInfoReconcilerBuilder(mockConnectionInfoReconciler),
			activeGateReconcilerBuilder:     createActivegateReconcilerBuilder(mockActiveGateReconciler),
			injectionReconcilerBuilder:      createInjectionReconcilerBuilder(mockInjectionReconciler),
		}
		mockedDtc := mockedclient.NewClient(t)
		err := controller.reconcileComponents(ctx, mockedDtc, nil, dynakube)

		require.Error(t, err)
		// goerrors.Join concats errors with \n
		assert.Len(t, strings.Split(err.Error(), "\n"), 3) // ActiveGate, OneAgentInjection, EnrichmentInjection, OneAgent
	})

	t.Run("exit early in case of no oneagent conncection info", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dynakube)

		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)
		mockConnectionInfoReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)
		mockConnectionInfoReconciler.On("ReconcileOneAgent", mock.Anything, dynakube).Return(connectioninfo.NoOneAgentCommunicationHostsError)

		mockVersionReconciler := mockversion.NewReconciler(t)
		mockVersionReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

		mockInjectionReconciler := mockinjection.NewReconciler(t)

		controller := &Controller{
			client:                          fakeClient,
			apiReader:                       fakeClient,
			scheme:                          scheme.Scheme,
			fs:                              afero.Afero{Fs: afero.NewMemMapFs()},
			registryClientBuilder:           createFakeRegistryClientBuilder(t),
			versionReconcilerBuilder:        createVersionReconcilerBuilder(mockVersionReconciler),
			connectionInfoReconcilerBuilder: createConnectionInfoReconcilerBuilder(mockConnectionInfoReconciler),
			activeGateReconcilerBuilder:     createActivegateReconcilerBuilder(mockActiveGateReconciler),
			injectionReconcilerBuilder:      createInjectionReconcilerBuilder(mockInjectionReconciler),
		}
		mockedDtc := mockedclient.NewClient(t)
		err := controller.reconcileComponents(ctx, mockedDtc, nil, dynakube)

		require.Error(t, err)
		// goerrors.Join concats errors with \n
		assert.Len(t, strings.Split(err.Error(), "\n"), 1) // ActiveGate, no OneAgent connection info is not an error
	})
	t.Run("exit early in case of error for oneagent connection info", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeClient := fake.NewClientWithIndex(dynakube)

		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)
		mockConnectionInfoReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

		mockVersionReconciler := mockversion.NewReconciler(t)

		mockConnectionInfoReconciler.On("ReconcileOneAgent", mock.Anything, dynakube).Return(errors.New("BOOM"))

		mockInjectionReconciler := mockinjection.NewReconciler(t)

		controller := &Controller{
			client:                          fakeClient,
			apiReader:                       fakeClient,
			scheme:                          scheme.Scheme,
			fs:                              afero.Afero{Fs: afero.NewMemMapFs()},
			registryClientBuilder:           createFakeRegistryClientBuilder(t),
			versionReconcilerBuilder:        createVersionReconcilerBuilder(mockVersionReconciler),
			connectionInfoReconcilerBuilder: createConnectionInfoReconcilerBuilder(mockConnectionInfoReconciler),
			injectionReconcilerBuilder:      createInjectionReconcilerBuilder(mockInjectionReconciler),
		}
		mockedDtc := mockedclient.NewClient(t)
		err := controller.reconcileComponents(ctx, mockedDtc, nil, dynakube)

		require.Error(t, err)
		// goerrors.Join concats errors with \n
		assert.Len(t, strings.Split(err.Error(), "\n"), 2) // ActiveGate, OneAgent connection info error
	})
}

func createActivegateReconcilerBuilder(reconciler controllers.Reconciler) activegate.ReconcilerBuilder {
	return func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) controllers.Reconciler {
		return reconciler
	}
}

func createConnectionInfoReconcilerBuilder(reconciler *mockconnectioninfo.Reconciler) connectioninfo.ReconcilerBuilder {
	return func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client) connectioninfo.Reconciler {
		return reconciler
	}
}

func createVersionReconcilerBuilder(reconciler *mockversion.Reconciler) version.ReconcilerBuilder {
	return func(apiReader client.Reader, dtClient dtclient.Client, fs afero.Afero, timeProvider *timeprovider.Provider) version.Reconciler {
		return reconciler
	}
}

func createInjectionReconcilerBuilder(reconciler *mockinjection.Reconciler) injection.ReconcilerBuilder {
	return func(client client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, istioReconciler istio.Reconciler, versionReconciler version.Reconciler) injection.Reconciler {
		return reconciler
	}
}

func TestRemoveOneAgentDaemonset(t *testing.T) {
	t.Run(`Create validates apiToken correctly if apiToken with "InstallerDownload"-scope is provided`, func(t *testing.T) {
		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{
			dtclient.TokenScopeDataExport,
			dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate})

		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
			Status: *getTestDynkubeStatus(),
		}
		data := map[string][]byte{
			dtclient.ApiToken: []byte(testAPIToken),
		}

		objects := []client.Object{
			dynakube,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: data},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: kubesystem.Namespace,
					UID:  testUID,
				},
			},
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneAgentDaemonsetName(),
					Namespace: testNamespace,
				},
			},
		}
		objects = append(objects, createTenantSecrets(dynakube)...)

		fakeClient := fake.NewClient(objects...)

		mockDtcBuilder := dtClientMock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("BuildWithTokenVerification", mock.Anything).Return(mockClient, nil)

		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)

		mockVersionReconciler := mockversion.NewReconciler(t)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		mockReconciler := mockcontroller.NewReconciler(t)
		mockReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		mockInjectionReconciler := mockinjection.NewReconciler(t)
		mockInjectionReconciler.On("Reconcile", mock.Anything).Return(nil)

		controller := &Controller{
			client:                              fakeClient,
			apiReader:                           fakeClient,
			scheme:                              scheme.Scheme,
			dynatraceClientBuilder:              mockDtcBuilder,
			registryClientBuilder:               createFakeRegistryClientBuilder(t),
			deploymentMetadataReconcilerBuilder: createDeploymentMetadataReconcilerBuilder(mockReconciler),
			versionReconcilerBuilder:            createVersionReconcilerBuilder(mockVersionReconciler),
			connectionInfoReconcilerBuilder:     createConnectionInfoReconcilerBuilder(mockConnectionInfoReconciler),
			activeGateReconcilerBuilder:         createActivegateReconcilerBuilder(mockActiveGateReconciler),
			injectionReconcilerBuilder:          createInjectionReconcilerBuilder(mockInjectionReconciler),
		}

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		var daemonSet appsv1.DaemonSet

		err = controller.client.Get(context.Background(), client.ObjectKey{Name: dynakube.OneAgentDaemonsetName(), Namespace: testNamespace}, &daemonSet)

		require.Error(t, err)
	})
}

func createDeploymentMetadataReconcilerBuilder(mockReconciler *mockcontroller.Reconciler) func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube dynatracev1beta1.DynaKube, clusterID string) controllers.Reconciler {
	return func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube dynatracev1beta1.DynaKube, clusterID string) controllers.Reconciler {
		return mockReconciler
	}
}

func createFakeRegistryClientBuilder(t *testing.T) func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
	fakeRegistryClient := registrymock.NewImageGetter(t)
	fakeImage := &fakecontainer.FakeImage{}
	fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
		return &containerv1.ConfigFile{}, nil
	}
	_, _ = fakeImage.ConfigFile()
	image := containerv1.Image(fakeImage)

	fakeRegistryClient.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: "1.2.3.4-5"}, nil).Maybe()
	fakeRegistryClient.On("PullImageInfo", mock.Anything, mock.Anything).Return(&image, nil).Maybe()

	return func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return fakeRegistryClient, nil
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
		fakeClient := fake.NewClient(&dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
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
		dynakube, err := controller.getDynakubeOrCleanup(ctx, testName, testNamespace)

		assert.Nil(t, dynakube)
		require.EqualError(t, err, "fake error")
	})
}

func TestTokenConditions(t *testing.T) {
	t.Run("token condition error is set if token are invalid", func(t *testing.T) {
		fakeClient := fake.NewClient()
		dynakube := &dynatracev1beta1.DynaKube{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}

		err := controller.reconcileDynaKube(context.Background(), dynakube)

		require.Error(t, err)
		assertCondition(t, dynakube, dynatracev1beta1.TokenConditionType, metav1.ConditionFalse, dynatracev1beta1.ReasonTokenError, "secrets \"\" not found")
		assert.Nil(t, dynakube.Status.LastTokenProbeTimestamp, "LastTokenProbeTimestamp should be Nil if token retrieval did not work.")
	})
	t.Run("token condition is set if token are valid", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
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
				dtclient.ApiToken: []byte(testAPIToken),
			},
		})
		mockClient := mockedclient.NewClient(t)
		mockDtcBuilder := dtClientMock.NewBuilder(t)
		mockDtcBuilder.On("SetContext", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetDynakube", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("SetTokens", mock.Anything).Return(mockDtcBuilder)
		mockDtcBuilder.On("BuildWithTokenVerification", mock.Anything).Return(mockClient, nil)

		controller := &Controller{
			client:                 fakeClient,
			apiReader:              fakeClient,
			dynatraceClientBuilder: mockDtcBuilder,
			registryClientBuilder:  createFakeRegistryClientBuilder(t),
		}

		err := controller.reconcileDynaKube(context.TODO(), dynakube)

		require.Error(t, err, "status update will fail")
		assertCondition(t, dynakube, dynatracev1beta1.TokenConditionType, metav1.ConditionTrue, dynatracev1beta1.ReasonTokenReady, "")
	})
}

func TestSetupIstio(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:      testApiUrl,
			EnableIstio: true,
		},
	}

	t.Run("EnableIstio: false => do nothing + nil", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.EnableIstio = false
		controller := &Controller{}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.NoError(t, err)
		assert.Nil(t, istioReconciler)
	})
	t.Run("no istio installed + EnableIstio: true => error", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := false
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
			scheme:             scheme.Scheme,
		}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.Error(t, err)
		assert.Nil(t, istioReconciler)
	})
	t.Run("success", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		fakeIstio := fakeistio.NewSimpleClientset()
		isIstioInstalled := true
		controller := &Controller{
			istioClientBuilder: fakeIstioClientBuilder(t, fakeIstio, isIstioInstalled),
			scheme:             scheme.Scheme,
		}
		istioReconciler, err := controller.setupIstio(ctx, dynakube)
		require.NoError(t, err)
		assert.NotNil(t, istioReconciler)

		expectedName := istio.BuildNameForFQDNServiceEntry(dynakube.GetName(), istio.OperatorComponent)
		serviceEntry, err := fakeIstio.NetworkingV1beta1().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		virtualService, err := fakeIstio.NetworkingV1beta1().VirtualServices(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
}

func fakeIstioClientBuilder(t *testing.T, fakeIstio *fakeistio.Clientset, isIstioInstalled bool) istio.ClientBuilder {
	return func(_ *rest.Config, scheme *runtime.Scheme, owner metav1.Object) (*istio.Client, error) {
		if isIstioInstalled == true {
			fakeDiscovery, ok := fakeIstio.Discovery().(*fakediscovery.FakeDiscovery)
			fakeDiscovery.Resources = []*metav1.APIResourceList{{GroupVersion: istio.IstioGVR}}

			if !ok {
				t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
			}
		}

		return &istio.Client{
			IstioClientset: fakeIstio,
			Scheme:         scheme,
			Owner:          owner,
		}, nil
	}
}

func assertCondition(t *testing.T, dk *dynatracev1beta1.DynaKube, expectedConditionType string, expectedConditionStatus metav1.ConditionStatus, expectedReason string, expectedMessage string) { //nolint:revive // argument-limit
	t.Helper()

	actualCondition := meta.FindStatusCondition(dk.Status.Conditions, expectedConditionType)
	require.NotNil(t, actualCondition)
	assert.Equal(t, expectedConditionStatus, actualCondition.Status)
	assert.Equal(t, expectedReason, actualCondition.Reason)
	assert.Equal(t, expectedMessage, actualCondition.Message)
}

func getTestDynkubeStatus() *dynatracev1beta1.DynaKubeStatus {
	return &dynatracev1beta1.DynaKubeStatus{
		ActiveGate: dynatracev1beta1.ActiveGateStatus{
			ConnectionInfoStatus: dynatracev1beta1.ActiveGateConnectionInfoStatus{
				ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID:  testUUID,
					Endpoints:   "endpoint",
					LastRequest: metav1.NewTime(time.Now()),
				},
			},
		},
		OneAgent: dynatracev1beta1.OneAgentStatus{
			ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
				ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
					TenantUUID:  testUUID,
					Endpoints:   "endpoint",
					LastRequest: metav1.NewTime(time.Now()),
				},
				CommunicationHosts: []dynatracev1beta1.CommunicationHostStatus{
					{
						Protocol: "http",
						Host:     "localhost",
						Port:     9999,
					},
				},
			},
		},
	}
}

func createTenantSecrets(dynakube *dynatracev1beta1.DynaKube) []client.Object {
	return []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakube.OneagentTenantSecret(),
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				connectioninfo.TenantTokenName: []byte("test-token"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakube.ActivegateTenantSecret(),
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				connectioninfo.TenantTokenName: []byte("test-token"),
			},
		},
	}
}
