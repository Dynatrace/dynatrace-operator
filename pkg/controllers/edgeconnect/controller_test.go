package edgeconnect

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry/mocks"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	mocksedgeconnect "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/edgeconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName                       = "test-name-edgeconnectv1alpha1"
	testNamespace                  = "test-namespace"
	testOauthClientId              = "client-id"
	testOauthClientSecret          = "client-secret"
	testOauthClientResource        = "client-resource"
	testCreatedOauthClientId       = "created-client-id"
	testCreatedOauthClientSecret   = "created-client-secret"
	testCreatedOauthClientResource = "created-client-resource"
	testCreatedId                  = "id"
	testRecreatedInvalidId         = "id-somehow-different"
)

var (
	testHostPatterns  = []string{"*.internal.org"}
	testHostPatterns2 = []string{"*.external.org"}
)

func TestReconcile(t *testing.T) {
	t.Run("Create works with minimal setup", func(t *testing.T) {
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
		}
		controller := createFakeClientAndReconciler(instance)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("Timestamp update in EdgeConnect status works", func(t *testing.T) {
		now := metav1.Now()
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
				Version: status.VersionStatus{
					LastProbeTimestamp: &now,
					ImageID:            "docker.io/dynatrace/edgeconnect:latest",
				},
			},
		}

		controller := createFakeClientAndReconciler(instance)
		controller.timeProvider.Freeze()

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		err = controller.apiReader.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
		require.NoError(t, err)
		// Fake client drops seconds, so we have to do the same
		expectedTimestamp := controller.timeProvider.Now().Truncate(time.Second)
		assert.Equal(t, expectedTimestamp, instance.Status.UpdatedTimestamp.Time)
	})
	t.Run(`Reconciles phase change correctly`, func(t *testing.T) {
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
		}
		controller := createFakeClientAndReconciler(instance)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.False(t, result.Requeue)

		var edgeConnectDeployment edgeconnectv1alpha1.EdgeConnect

		require.NoError(t,
			controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &edgeConnectDeployment))
		require.NoError(t, controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, instance))
		assert.Equal(t, status.Running, instance.Status.DeploymentPhase)
	})
	t.Run(`Reconciles doesn't fail if edgeconnect not found`, func(t *testing.T) {
		controller := createFakeClientAndReconciler(nil)

		_, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})
}

func TestReconcileProvisionerCreate(t *testing.T) {
	t.Run("create EdgeConnect", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientCreate(edgeConnectClient),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", testName, testHostPatterns, "")
	})
}

func TestReconcileProvisionerRecreate(t *testing.T) {
	t.Run("recreate EdgeConnect due to missing client secret", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testCreatedId),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", testName, testHostPatterns, "")
	})

	t.Run("recreate EdgeConnect due to invalid id", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testRecreatedInvalidId),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
			createClientSecret(instance.Name, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(controller.apiReader, types.NamespacedName{Name: edgeConnectClientSecretName(instance.Name), Namespace: instance.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testRecreatedInvalidId)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", testName, testHostPatterns, "")
	})
}

func TestReconcileProvisionerDelete(t *testing.T) {
	t.Run("delete EdgeConnect", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientDelete(edgeConnectClient),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
			createClientSecret(instance.Name, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
	})

	t.Run("delete EdgeConnect - missing client secret", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientDelete(edgeConnectClient),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
	})

	t.Run("delete EdgeConnect - missing EdgeConnect on the tenant", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientDeleteNotFoundOnTenant(edgeConnectClient),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
			createClientSecret(instance.Name, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, instance.Name, instance.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertNotCalled(t, "DeleteEdgeConnect", testCreatedId)
	})
}

func TestReconcileProvisionerUpdate(t *testing.T) {
	t.Run("update EdgeConnect", func(t *testing.T) {
		instance := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)

		edgeConnectClient := mocksedgeconnect.NewClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			instance,
			mockNewEdgeConnectClientUpdate(edgeConnectClient),
			createOauthSecret(instance.Spec.OAuth.ClientSecret, instance.Namespace),
			createClientSecret(instance.Name, instance.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "GetEdgeConnect", testCreatedId)
		edgeConnectClient.AssertCalled(t, "UpdateEdgeConnect", testCreatedId, testName, testHostPatterns2, testCreatedOauthClientId)
	})
}

func createOauthSecret(name string, namespace string) *corev1.Secret {
	return newSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectOauthClientID:     testOauthClientId,
		consts.KeyEdgeConnectOauthClientSecret: testOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testOauthClientResource,
	})
}

func createClientSecret(name string, namespace string) *corev1.Secret {
	return newSecret(edgeConnectClientSecretName(name), namespace, map[string]string{
		consts.KeyEdgeConnectId:                testCreatedId,
		consts.KeyEdgeConnectOauthClientID:     testCreatedOauthClientId,
		consts.KeyEdgeConnectOauthClientSecret: testCreatedOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testCreatedOauthClientResource,
	})
}

func newSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}

	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func getEdgeConnectCR(apiReader client.Reader, name string, namespace string) (edgeconnectv1alpha1.EdgeConnect, error) {
	var edgeConnectCR edgeconnectv1alpha1.EdgeConnect
	err := apiReader.Get(
		context.Background(),
		client.ObjectKey{
			Name:      name,
			Namespace: namespace,
		},
		&edgeConnectCR,
	)

	return edgeConnectCR, err
}

func createFakeClientAndReconciler(instance *edgeconnectv1alpha1.EdgeConnect) *Controller {
	fakeClient := fake.NewClientWithIndex()
	if instance != nil {
		fakeClient = fake.NewClientWithIndex(instance)
	}

	mockImageGetter := &mocks.MockImageGetter{}

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	controller := &Controller{
		client:                fakeClient,
		apiReader:             fakeClient,
		scheme:                scheme.Scheme,
		timeProvider:          timeprovider.New(),
		registryClientBuilder: mockRegistryClientBuilder,
	}

	return controller
}

func createFakeClientAndReconcilerForProvisioner(instance *edgeconnectv1alpha1.EdgeConnect, builder edgeConnectClientBuilderType, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex()

	if instance != nil {
		objs := []client.Object{instance}
		objs = append(objs, objects...)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	mockImageGetter := &mocks.MockImageGetter{}

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		scheme:                   scheme.Scheme,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    mockRegistryClientBuilder,
		edgeConnectClientBuilder: builder,
	}

	return controller
}

func mockNewEdgeConnectClientCreate(edgeConnectClient *mocksedgeconnect.Client) func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnect.ListResponse{
				TotalCount: 0,
			},
			nil,
		)

		// CreateEdgeConnect creates edge connect
		edgeConnectClient.On("CreateEdgeConnect", testName, testHostPatterns, "").Return(
			edgeconnect.CreateResponse{
				ID:                  testCreatedId,
				Name:                testName,
				HostPatterns:        testHostPatterns,
				OauthClientId:       testCreatedOauthClientId,
				OauthClientSecret:   testCreatedOauthClientSecret,
				OauthClientResource: testCreatedOauthClientResource,
			},
			nil,
		)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientRecreate(edgeConnectClient *mocksedgeconnect.Client, id string) func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnect.ListResponse{
				EdgeConnects: []edgeconnect.GetResponse{
					{
						ID:                         id,
						Name:                       testName,
						HostPatterns:               testHostPatterns,
						OauthClientId:              testOauthClientId,
						ManagedByDynatraceOperator: true,
					},
				},
				TotalCount: 1,
			},
			nil,
		)

		edgeConnectClient.On("DeleteEdgeConnect", id).Return(nil)
		// CreateEdgeConnect creates edge connect
		edgeConnectClient.On("CreateEdgeConnect", testName, testHostPatterns, "").Return(
			edgeconnect.CreateResponse{
				ID:                  testCreatedId,
				Name:                testName,
				HostPatterns:        testHostPatterns,
				OauthClientId:       testCreatedOauthClientId,
				OauthClientSecret:   testCreatedOauthClientSecret,
				OauthClientResource: testCreatedOauthClientResource,
			},
			nil,
		)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientDelete(edgeConnectClient *mocksedgeconnect.Client) func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnect.ListResponse{
				EdgeConnects: []edgeconnect.GetResponse{
					{
						ID:                         testCreatedId,
						Name:                       testName,
						HostPatterns:               testHostPatterns,
						OauthClientId:              testOauthClientId,
						ManagedByDynatraceOperator: true,
					},
				},
				TotalCount: 1,
			},
			nil,
		)
		edgeConnectClient.On("DeleteEdgeConnect", testCreatedId).Return(nil)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientDeleteNotFoundOnTenant(edgeConnectClient *mocksedgeconnect.Client) func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnect.ListResponse{
				TotalCount: 0,
			},
			nil,
		)
		edgeConnectClient.On("DeleteEdgeConnect", testCreatedId).Return(nil).Maybe()

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientUpdate(edgeConnectClient *mocksedgeconnect.Client) func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnect.ListResponse{
				EdgeConnects: []edgeconnect.GetResponse{
					{
						ID:                         testCreatedId,
						Name:                       testName,
						HostPatterns:               testHostPatterns,
						OauthClientId:              testOauthClientId,
						ManagedByDynatraceOperator: true,
					},
				},
				TotalCount: 1,
			},
			nil,
		)

		edgeConnectClient.On("GetEdgeConnect", testCreatedId).Return(
			edgeconnect.GetResponse{
				ID:            testCreatedId,
				Name:          testName,
				HostPatterns:  testHostPatterns,
				OauthClientId: testOauthClientId,
			},
			nil,
		)

		// CreateEdgeConnect creates edge connect
		edgeConnectClient.On("UpdateEdgeConnect", testCreatedId, testName, testHostPatterns2, testCreatedOauthClientId).Return(nil)

		return edgeConnectClient, nil
	}
}

func createEdgeConnectProvisionerCR(finalizers []string, deletionTimestamp *metav1.Time, hostPatterns []string) *edgeconnectv1alpha1.EdgeConnect {
	return &edgeconnectv1alpha1.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testName,
			Namespace:         testNamespace,
			Finalizers:        finalizers,
			DeletionTimestamp: deletionTimestamp,
		},
		Spec: edgeconnectv1alpha1.EdgeConnectSpec{
			ApiServer: "abc12345.dynatrace.com",
			OAuth: edgeconnectv1alpha1.OAuthSpec{
				ClientSecret: "client-secret",
				Provisioner:  true,
			},
			HostPatterns: hostPatterns,
		},
	}
}
