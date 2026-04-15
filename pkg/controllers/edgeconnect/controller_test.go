package edgeconnect

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/deployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	edgeconnectmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/edgeconnect"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/util/oci/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName                       = "test-name-edgeconnect"
	testNamespace                  = "test-namespace"
	testOauthClientID              = "client-id"
	testOauthClientSecret          = "client-secret"
	testOauthClientResource        = "client-resource"
	testCreatedOauthClientID       = "created-client-id"
	testCreatedOauthClientSecret   = "created-client-secret"
	testCreatedOauthClientResource = "created-client-resource"
	testCreatedID                  = "id"
	testRecreatedInvalidID         = "id-somehow-different"
	testCAConfigMapName            = "test-ca-name"
	testK8sAutomationHostPattern   = "test-name-edgeconnect.test-namespace.1-2-3-4.kubernetes-automation"

	testUID = "1-2-3-4"
)

var (
	testHostPatterns  = []string{"*.internal.org", testK8sAutomationHostPattern}
	testHostPatterns2 = []string{"*.external.org", testK8sAutomationHostPattern}
	testHostMappings  = []edgeconnect.HostMapping{
		{
			From: testK8sAutomationHostPattern,
			To:   edgeconnect.KubernetesDefaultDNS,
		},
	}
	testObjectID = "my:default"

	testEnvironmentSetting = edgeconnectClient.EnvironmentSetting{
		ObjectID: testObjectID,
		SchemaID: edgeconnectClient.KubernetesConnectionSchemaID,
		Scope:    edgeconnectClient.KubernetesConnectionScope,
		Value: edgeconnectClient.EnvironmentSettingValue{
			Name:      testName,
			Namespace: testNamespace,
			UID:       testUID,
		},
	}
)

func TestReconcile(t *testing.T) {
	t.Run("Create works with minimal setup", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()

		controller := createFakeClientAndReconciler(t, ec,
			createClientSecret(testOauthClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("Timestamp update in EdgeConnect status works", func(t *testing.T) {
		now := metav1.Now()
		ec := createEdgeConnectRegularCR()
		ec.Status = edgeconnect.EdgeConnectStatus{
			UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			Version: status.VersionStatus{
				LastProbeTimestamp: &now,
				ImageID:            "docker.io/dynatrace/edgeconnectClient:latest",
			},
		}

		controller := createFakeClientAndReconciler(t, ec,
			createClientSecret(testOauthClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)
		controller.timeProvider.Freeze()

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		err = controller.apiReader.Get(context.TODO(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, ec)
		require.NoError(t, err)
		// Fake client drops seconds, so we have to do the same
		expectedTimestamp := controller.timeProvider.Now().Truncate(time.Second)
		assert.Equal(t, expectedTimestamp, ec.Status.UpdatedTimestamp.Time)
	})
	t.Run("Reconciles phase change correctly", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()

		controller := createFakeClientAndReconciler(t, ec,
			createClientSecret(testOauthClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, defaultRequeueInterval, result.RequeueAfter)

		var edgeConnectDeployment edgeconnect.EdgeConnect

		require.NoError(t,
			controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &edgeConnectDeployment))
		require.NoError(t, controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, ec))
		assert.Equal(t, status.Running, ec.Status.DeploymentPhase)
	})
	t.Run("Reconciles doesn't fail if edgeconnectClient not found", func(t *testing.T) {
		controller := createFakeClientAndReconciler(t, nil)

		_, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})
	t.Run("Reconciles custom CA provided", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()
		ec.Spec.CaCertsRef = testCAConfigMapName

		data := make(map[string]string)
		data[consts.EdgeConnectCAConfigMapKey] = "dummy"
		customCA := newConfigMap(testCAConfigMapName, ec.Namespace, data)
		clientSecret := createClientSecret(testOauthClientSecret, ec.Namespace)

		controller := createFakeClientAndReconciler(t, ec, clientSecret, customCA, createKubeSystemNamespace())

		_, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})

	t.Run("SecretConfigConditionType is set SecretCreated", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()

		controller := createFakeClientAndReconciler(t, ec,
			createClientSecret(testOauthClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		_, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)

		err = controller.apiReader.Get(context.TODO(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, ec)
		require.NoError(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, ec.Name+"-"+consts.EdgeConnectSecretSuffix+" created", condition.Message)
	})

	t.Run("SecretConfigConditionType is set SecretGenFailed failed to get clientSecret", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()

		controller := createFakeClientAndReconciler(t, ec,
			createKubeSystemNamespace(),
		)

		err := controller.reconcileEdgeConnectRegular(context.Background(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret")
	})

	t.Run("SecretConfigConditionType is set SecretGenFailed failed", func(t *testing.T) {
		ec := createEdgeConnectRegularCR()

		controller := createFakeClientAndReconciler(t, ec,
			createKubeSystemNamespace(),
		)

		boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("BOOM")
			},
		})
		controller.apiReader = boomClient
		controller.secrets = k8ssecret.Query(controller.client, controller.apiReader, log)

		err := controller.reconcileEdgeConnectRegular(context.Background(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret")
	})
}

func TestReconcileProvisionerCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("create EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientCreate(edgeConnectClient, testHostPatterns),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, edgeConnectID)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings))
	})
}

func TestReconcileProvisionerRecreate(t *testing.T) {
	ctx := context.Background()

	t.Run("recreate EdgeConnect due to missing client secret", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testCreatedID),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, edgeConnectID)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", mock.Anything, testCreatedID)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings))
	})

	t.Run("recreate EdgeConnect due to invalid id", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testRecreatedInvalidID),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createClientSecret(ec.ClientSecretName(), ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, edgeConnectID)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", mock.Anything, testRecreatedInvalidID)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings))
	})
}

func TestReconcileProvisionerDelete(t *testing.T) {
	t.Run("delete EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("DeleteEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientDelete(edgeConnectClient),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createClientSecret(ec.ClientSecretName(), ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", mock.Anything, testCreatedID)
	})

	t.Run("delete EdgeConnect - missing client secret", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("DeleteEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientDelete(edgeConnectClient),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", mock.Anything, testCreatedID)
	})

	t.Run("delete EdgeConnect - missing EdgeConnect on the tenant", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientDeleteNotFoundOnTenant(edgeConnectClient),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createClientSecret(ec.ClientSecretName(), ec.Namespace),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))

		edgeConnectClient.AssertNotCalled(t, "DeleteEdgeConnect", mock.Anything, testCreatedID)
	})
}

func TestReconcileProvisionerUpdate(t *testing.T) {
	t.Run("update EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientUpdate(edgeConnectClient, testHostPatterns, testHostPatterns2),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createClientSecret(ec.ClientSecretName(), ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "GetEdgeConnect", mock.Anything, testCreatedID)
		edgeConnectClient.AssertCalled(t, "UpdateEdgeConnect", mock.Anything, testCreatedID, edgeconnectClient.NewUpdateRequest(testName, testHostPatterns2, testHostMappings, testCreatedOauthClientID))
	})
}

func TestReconcileProvisionerWithK8sAutomationsCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("create EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientCreate(edgeConnectClient, testHostPatterns),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := getEdgeConnectCR(controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		edgeConnectOauthClientID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectID, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, edgeConnectID)

		var edgeConnectDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			context.Background(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&edgeConnectDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", edgeConnectDeployment.Spec.Template.Spec.Containers[0].Name)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings))
	})
}

func TestReconcileProvisionerWithK8sAutomationsUpdate(t *testing.T) {
	t.Run("update EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientUpdate(edgeConnectClient, testHostPatterns, testHostPatterns2),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			createClientSecret(ec.ClientSecretName(), ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectClient.AssertCalled(t, "ListEdgeConnects", mock.Anything, testName)
		edgeConnectClient.AssertCalled(t, "GetEdgeConnect", mock.Anything, testCreatedID)
		edgeConnectClient.AssertCalled(t, "UpdateEdgeConnect", mock.Anything, testCreatedID, edgeconnectClient.NewUpdateRequest(testName, testHostPatterns2, testHostMappings, testCreatedOauthClientID))
	})
}

func TestReconcileReplicas(t *testing.T) {
	createEdgeConnect := func(provisioner bool, replicas *int32) *edgeconnect.EdgeConnect {
		ec := createEdgeConnectRegularCR()
		if provisioner {
			ec = createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		}

		ec.Spec.Replicas = replicas

		return ec
	}

	createController := func(t *testing.T, ec *edgeconnect.EdgeConnect, provisioner bool, objs ...client.Object) *Controller {
		t.Helper()

		if !provisioner {
			return createFakeClientAndReconciler(t, ec, objs...)
		}

		edgeClient := edgeconnectmock.NewAPIClient(t)
		edgeClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Maybe()
		edgeClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil).Maybe()

		return createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientCreate(edgeClient, testHostPatterns),
			objs...,
		)
	}

	buildObjects := func(ec *edgeconnect.EdgeConnect, provisioner bool, existingReplicas *int32) []client.Object {
		objs := []client.Object{
			createKubeSystemNamespace(),
			createOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
		}

		if provisioner {
			objs = append(objs, createClientSecret(ec.ClientSecretName(), ec.Namespace))
		}

		if existingReplicas != nil {
			existing := deployment.New(ec)
			existing.Spec.Replicas = existingReplicas
			objs = append(objs, existing)
		}

		return objs
	}

	assertDeploymentReplicas := func(t *testing.T, apiReader client.Reader, ec *edgeconnect.EdgeConnect, expectedReplicas int32) {
		t.Helper()

		d := &appsv1.Deployment{}
		err := apiReader.Get(context.Background(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, d)
		require.NoError(t, err)
		require.NotNil(t, d.Spec.Replicas)
		assert.Equal(t, expectedReplicas, *d.Spec.Replicas)
	}

	modes := []struct {
		name        string
		provisioner bool
	}{
		{name: "regular", provisioner: false},
		{name: "provisioner", provisioner: true},
	}

	tests := []struct {
		name             string
		specReplicas     *int32
		existingReplicas *int32
		expectedReplicas int32
	}{
		{
			name:             "uses explicit spec replicas over existing deployment",
			specReplicas:     ptr.To(int32(2)),
			existingReplicas: ptr.To(int32(3)),
			expectedReplicas: int32(2),
		},
		{
			name:             "uses existing deployment replicas when spec replicas are nil",
			specReplicas:     nil,
			existingReplicas: ptr.To(int32(2)),
			expectedReplicas: int32(2),
		},
		{
			name:             "uses default replicas when spec replicas are nil and deployment does not exist",
			specReplicas:     nil,
			existingReplicas: nil,
			expectedReplicas: int32(1),
		},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					ec := createEdgeConnect(mode.provisioner, tc.specReplicas)

					objs := buildObjects(ec, mode.provisioner, tc.existingReplicas)

					controller := createController(t, ec, mode.provisioner, objs...)

					_, err := controller.Reconcile(context.Background(), reconcile.Request{
						NamespacedName: types.NamespacedName{Namespace: ec.Namespace, Name: ec.Name},
					})
					require.NoError(t, err)

					assertDeploymentReplicas(t, controller.apiReader, ec, tc.expectedReplicas)
				})
			}
		})
	}
}

func createEdgeConnectRegularCR() *edgeconnect.EdgeConnect {
	return &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: edgeconnect.EdgeConnectSpec{
			APIServer: "abc12345.dynatrace.com",
			OAuth: edgeconnect.OAuthSpec{
				Endpoint:     "https://test.com/sso/oauth2/token",
				Resource:     "urn:dtenvironment:test12345",
				ClientSecret: testOauthClientSecret,
				Provisioner:  false,
			},
		},
	}
}

func createOauthSecret(name string, namespace string) *corev1.Secret {
	return newSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectOauthClientID:     testOauthClientID,
		consts.KeyEdgeConnectOauthClientSecret: testOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testOauthClientResource,
	})
}

func createClientSecret(name string, namespace string) *corev1.Secret {
	return newSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectID:                testCreatedID,
		consts.KeyEdgeConnectOauthClientID:     testCreatedOauthClientID,
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

func newConfigMap(name, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func getEdgeConnectCR(apiReader client.Reader, name string, namespace string) (edgeconnect.EdgeConnect, error) {
	var edgeConnectCR edgeconnect.EdgeConnect
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

func createFakeClientAndReconciler(t *testing.T, ec *edgeconnect.EdgeConnect, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(createCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, createCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	mockImageGetter := registrymock.NewImageGetter(t)

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Maybe()

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	mockEdgeConnectClient := edgeconnectmock.NewAPIClient(t)

	mockEdgeConnectClientBuilder := func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
		return mockEdgeConnectClient, nil
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    mockRegistryClientBuilder,
		edgeConnectClientBuilder: mockEdgeConnectClientBuilder,
		secrets:                  k8ssecret.Query(fakeClient, fakeClient, log),
	}

	return controller
}

func createFakeClientAndReconcilerForProvisioner(t *testing.T, ec *edgeconnect.EdgeConnect, builder edgeConnectClientBuilderType, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(createCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, createCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	mockImageGetter := registrymock.NewImageGetter(t)

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Maybe()

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    mockRegistryClientBuilder,
		edgeConnectClientBuilder: builder,
		secrets:                  k8ssecret.Query(fakeClient, fakeClient, log),
	}

	return controller
}

func mockNewEdgeConnectClientCreate(edgeConnectClient *edgeconnectmock.APIClient, hostPatterns []string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.APIClient, error) {
		edgeConnectClient.On("ListEdgeConnects", mock.Anything, testName).Return(
			[]edgeconnectClient.APIResponse{},
			nil,
		)

		// CreateEdgeConnect creates EdgeConnect
		edgeConnectClient.On("CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, hostPatterns, testHostMappings)).Return(
			edgeconnectClient.APIResponse{
				ID:                  testCreatedID,
				Name:                testName,
				HostPatterns:        hostPatterns,
				OauthClientID:       testCreatedOauthClientID,
				OauthClientSecret:   testCreatedOauthClientSecret,
				OauthClientResource: testCreatedOauthClientResource,
			},
			nil,
		)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientRecreate(edgeConnectClient *edgeconnectmock.APIClient, id string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.APIClient, error) {
		edgeConnectClient.On("ListEdgeConnects", mock.Anything, testName).Return(
			[]edgeconnectClient.APIResponse{
				{
					ID:                         id,
					Name:                       testName,
					HostPatterns:               testHostPatterns,
					OauthClientID:              testOauthClientID,
					ManagedByDynatraceOperator: true,
				},
			},
			nil,
		)

		edgeConnectClient.On("DeleteEdgeConnect", mock.Anything, id).Return(nil)
		// CreateEdgeConnect creates EdgeConnect
		edgeConnectClient.On("CreateEdgeConnect", mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings)).Return(
			edgeconnectClient.APIResponse{
				ID:                  testCreatedID,
				Name:                testName,
				HostPatterns:        testHostPatterns,
				OauthClientID:       testCreatedOauthClientID,
				OauthClientSecret:   testCreatedOauthClientSecret,
				OauthClientResource: testCreatedOauthClientResource,
			},
			nil,
		)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientDelete(edgeConnectClient *edgeconnectmock.APIClient) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.APIClient, error) {
		edgeConnectClient.On("ListEdgeConnects", mock.Anything, testName).Return(
			[]edgeconnectClient.APIResponse{
				{
					ID:                         testCreatedID,
					Name:                       testName,
					HostPatterns:               testHostPatterns,
					OauthClientID:              testOauthClientID,
					ManagedByDynatraceOperator: true,
				},
			},
			nil,
		)
		edgeConnectClient.On("DeleteEdgeConnect", mock.Anything, testCreatedID).Return(nil)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientDeleteNotFoundOnTenant(edgeConnectClient *edgeconnectmock.APIClient) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.APIClient, error) {
		edgeConnectClient.On("ListEdgeConnects", mock.Anything, testName).Return(
			[]edgeconnectClient.APIResponse{},
			nil,
		)
		edgeConnectClient.On("DeleteEdgeConnect", mock.Anything, testCreatedID).Return(nil).Maybe()

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientUpdate(edgeConnectClient *edgeconnectmock.APIClient, fromHostPatterns []string, toHostPatterns []string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.APIClient, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.APIClient, error) {
		edgeConnectClient.On("ListEdgeConnects", mock.Anything, testName).Return(
			[]edgeconnectClient.APIResponse{
				{
					ID:                         testCreatedID,
					Name:                       testName,
					HostPatterns:               fromHostPatterns,
					OauthClientID:              testOauthClientID,
					ManagedByDynatraceOperator: true,
				},
			},
			nil,
		)

		edgeConnectClient.On("GetEdgeConnect", mock.Anything, testCreatedID).Return(
			edgeconnectClient.APIResponse{
				ID:            testCreatedID,
				Name:          testName,
				HostPatterns:  fromHostPatterns,
				OauthClientID: testOauthClientID,
			},
			nil,
		)

		// CreateEdgeConnect creates EdgeConnect
		edgeConnectClient.On("UpdateEdgeConnect", mock.Anything, testCreatedID, edgeconnectClient.NewUpdateRequest(testName, toHostPatterns, testHostMappings, testCreatedOauthClientID)).Return(nil)

		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)

		return edgeConnectClient, nil
	}
}

func createEdgeConnectProvisionerCR(finalizers []string, deletionTimestamp *metav1.Time, hostPatterns []string) *edgeconnect.EdgeConnect {
	return &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testName,
			Namespace:         testNamespace,
			Finalizers:        finalizers,
			DeletionTimestamp: deletionTimestamp,
		},
		Spec: edgeconnect.EdgeConnectSpec{
			APIServer: "abc12345.dynatrace.com",
			OAuth: edgeconnect.OAuthSpec{
				ClientSecret: testName + "client",
				Provisioner:  true,
			},
			HostPatterns:         hostPatterns,
			KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{Enabled: true},
		},
		Status: edgeconnect.EdgeConnectStatus{KubeSystemUID: testUID},
	}
}

func createKubeSystemNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metav1.NamespaceSystem,
			Namespace: "",
			UID:       testUID,
		},
	}
}

func TestController_createOrUpdateConnectionSetting(t *testing.T) {
	t.Run("Create Connection Setting object", func(t *testing.T) {
		controller := mockController()
		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{}, nil)
		edgeConnectClient.On("CreateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)
		err := controller.createOrUpdateConnectionSetting(t.Context(), edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})
	t.Run("Existing Connection Setting object", func(t *testing.T) {
		controller := mockController()
		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		err := controller.createOrUpdateConnectionSetting(t.Context(), edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
		edgeConnectClient.AssertNotCalled(t, "CreateEnvironmentSetting", mock.Anything)
	})
	t.Run("Existing object with same Cluster ID but different name", func(t *testing.T) {
		controller := mockController()
		differentEnvironmentSetting := testEnvironmentSetting
		differentEnvironmentSetting.Value.Name = "different-name"
		differentEnvironmentSetting.Value.Namespace = "different-namespace"

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{differentEnvironmentSetting}, nil)
		edgeConnectClient.On("CreateEnvironmentSetting", mock.Anything, mock.Anything).Return(nil)
		err := controller.createOrUpdateConnectionSetting(t.Context(), edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})
	t.Run("Server fails", func(t *testing.T) {
		controller := mockController()
		expectedEnvironmentSetting := testEnvironmentSetting
		expectedEnvironmentSetting.Value.Name = "different-name"
		expectedEnvironmentSetting.Value.Namespace = "different-namespace"

		edgeConnectClient := edgeconnectmock.NewAPIClient(t)
		edgeConnectClient.On("ListEnvironmentSettings", mock.Anything).Return(nil, errors.New("something went wrong"))
		err := controller.createOrUpdateConnectionSetting(t.Context(), edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.Error(t, err)
	})
}

func TestController_newEdgeConnectClient(t *testing.T) {
	t.Run("New EdgeConnect APIClient with scopes including k8s automation extra scopes", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		ecClient := newEdgeConnectClient()
		require.NotNil(t, ecClient)
		actualClient, err := ecClient(context.Background(), ec, oauthCredentialsType{clientID: "fake", clientSecret: "fake"}, nil)
		require.NoError(t, err)
		require.NotNil(t, actualClient)
	})

	t.Run("New EdgeConnect APIClient with min scopes and without k8s automation", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: testName + "client",
					Provisioner:  true,
				},
				HostPatterns: []string{},
			},
		}
		ecClient := newEdgeConnectClient()
		require.NotNil(t, ecClient)
		actualClient, err := ecClient(context.Background(), ec, oauthCredentialsType{clientID: "fake", clientSecret: "fake"}, nil)
		require.NoError(t, err)
		require.NotNil(t, actualClient)
	})
}

func TestBuildOAuthScopes(t *testing.T) {
	baseScopes := []string{
		"app-engine:edge-connects:read",
		"app-engine:edge-connects:write",
		"app-engine:edge-connects:delete",
		"oauth2:clients:manage",
	}

	t.Run("k8s automation disabled returns only base scopes", func(t *testing.T) {
		scopes := buildOAuthScopes(false)
		assert.Equal(t, baseScopes, scopes)
	})

	t.Run("k8s automation enabled appends settings scopes", func(t *testing.T) {
		scopes := buildOAuthScopes(true)
		expected := slices.Concat(baseScopes, []string{"settings:objects:read", "settings:objects:write"})
		assert.Equal(t, expected, scopes)
	})

	t.Run("k8s automation disabled does not include settings scopes", func(t *testing.T) {
		scopes := buildOAuthScopes(false)
		assert.NotContains(t, scopes, "settings:objects:read")
		assert.NotContains(t, scopes, "settings:objects:write")
	})
}

func mockController() *Controller {
	return &Controller{
		client:                   fake.NewClient(),
		apiReader:                fake.NewClient(),
		registryClientBuilder:    registry.NewClient,
		config:                   &rest.Config{},
		timeProvider:             timeprovider.New(),
		edgeConnectClientBuilder: newEdgeConnectClient(),
	}
}

type errorClient struct {
	client.Client
}

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return errors.New("fake error")
}

func createDeployment(namespace, name string, replicas, readyReplicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      replicas,
			ReadyReplicas: readyReplicas,
		},
	}
}

func createCRD(t *testing.T) *apiextensionsv1.CustomResourceDefinition {
	t.Setenv(k8senv.AppVersion, "1.0.0")

	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8scrd.EdgeConnectName,
			Labels: map[string]string{
				k8slabel.AppVersionLabel: "1.0.0",
			},
		},
	}
}
