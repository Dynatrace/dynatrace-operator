package edgeconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	edgeconnectmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/edgeconnect"
	registrymock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/oci/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName                       = "test-name-edgeconnect"
	testNamespace                  = "test-namespace"
	testOauthClientId              = "client-id"
	testOauthClientSecret          = "client-secret"
	testOauthClientResource        = "client-resource"
	testCreatedOauthClientId       = "created-client-id"
	testCreatedOauthClientSecret   = "created-client-secret"
	testCreatedOauthClientResource = "created-client-resource"
	testCreatedId                  = "id"
	testRecreatedInvalidId         = "id-somehow-different"
	testCAConfigMapName            = "test-ca-name"
	testK8sAutomationHostPattern   = "test-name-edgeconnect.test-namespace.1-2-3-4.kubernetes-automation"

	testClusterIP = "1.2.3.4"
	testUID       = "1-2-3-4"

	kubeSystemNamespaceName = "kube-system"
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
	testObjectId = "my:default"

	testEnvironmentSetting = edgeconnectClient.EnvironmentSetting{
		ObjectId: &testObjectId,
		SchemaId: edgeconnectClient.KubernetesConnectionSchemaID,
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
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
		}
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
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
			Status: edgeconnect.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
				Version: status.VersionStatus{
					LastProbeTimestamp: &now,
					ImageID:            "docker.io/dynatrace/edgeconnectClient:latest",
				},
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
	t.Run(`Reconciles phase change correctly`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
		}
		controller := createFakeClientAndReconciler(t, ec,
			createClientSecret(testOauthClientSecret, ec.Namespace),
			createKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.False(t, result.Requeue)

		var edgeConnectDeployment edgeconnect.EdgeConnect

		require.NoError(t,
			controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &edgeConnectDeployment))
		require.NoError(t, controller.client.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, ec))
		assert.Equal(t, status.Running, ec.Status.DeploymentPhase)
	})
	t.Run(`Reconciles doesn't fail if edgeconnectClient not found`, func(t *testing.T) {
		controller := createFakeClientAndReconciler(t, nil)

		_, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})
	t.Run(`Reconciles custom CA provided`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
				CaCertsRef: testCAConfigMapName,
			},
		}

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

	t.Run(`SecretConfigConditionType is set SecretCreated`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
		}
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
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, ec.Name+"-"+consts.EdgeConnectSecretSuffix+" created", condition.Message)
	})

	t.Run(`SecretConfigConditionType is set SecretGenFailed failed to get clientSecret`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
		}

		controller := createFakeClientAndReconciler(t, ec,
			createKubeSystemNamespace(),
		)

		err := controller.reconcileEdgeConnectRegular(context.Background(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret: failed to get clientSecret")
	})

	t.Run(`SecretConfigConditionType is set SecretGenFailed failed`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testOauthClientSecret,
					Provisioner:  false,
				},
			},
		}

		controller := createFakeClientAndReconciler(t, ec,
			createKubeSystemNamespace(),
		)

		boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("BOOM")
			},
		})
		controller.apiReader = boomClient

		err := controller.reconcileEdgeConnectRegular(context.Background(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret: BOOM")
	})
}

func TestReconcileProvisionerCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("create EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateConnectionSetting", mock.Anything).Return(nil)

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
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", edgeconnectClient.NewRequest(testName, testHostPatterns, testHostMappings, ""))
	})
}

func TestReconcileProvisionerRecreate(t *testing.T) {
	ctx := context.Background()

	t.Run("recreate EdgeConnect due to missing client secret", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateConnectionSetting", mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testCreatedId),
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
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", edgeconnectClient.NewRequest(testName, testHostPatterns, testHostMappings, ""))
	})

	t.Run("recreate EdgeConnect due to invalid id", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateConnectionSetting", mock.Anything).Return(nil)

		controller := createFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			mockNewEdgeConnectClientRecreate(edgeConnectClient, testRecreatedInvalidId),
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
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testRecreatedInvalidId)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", edgeconnectClient.NewRequest(testName, testHostPatterns, testHostMappings, ""))
	})
}

func TestReconcileProvisionerDelete(t *testing.T) {
	t.Run("delete EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("DeleteConnectionSetting", mock.Anything).Return(nil)

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

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
	})

	t.Run("delete EdgeConnect - missing client secret", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("DeleteConnectionSetting", mock.Anything).Return(nil)

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

		edgeConnectClient.AssertCalled(t, "DeleteEdgeConnect", testCreatedId)
	})

	t.Run("delete EdgeConnect - missing EdgeConnect on the tenant", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		edgeConnectClient := edgeconnectmock.NewClient(t)

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

		edgeConnectClient.AssertNotCalled(t, "DeleteEdgeConnect", testCreatedId)
	})
}

func TestReconcileProvisionerUpdate(t *testing.T) {
	t.Run("update EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)

		edgeConnectClient := edgeconnectmock.NewClient(t)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "GetEdgeConnect", testCreatedId)
		edgeConnectClient.AssertCalled(t, "UpdateEdgeConnect", testCreatedId, edgeconnectClient.NewRequest(testName, testHostPatterns2, testHostMappings, testCreatedOauthClientId))
	})
}

func TestReconcileProvisionerWithK8sAutomationsCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("create EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateConnectionSetting", mock.Anything).Return(nil)

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
		assert.Equal(t, testCreatedOauthClientId, edgeConnectOauthClientID)

		edgeConnectOauthClientSecret, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, edgeConnectOauthClientSecret)

		edgeConnectOauthResource, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, edgeConnectOauthResource)

		edgeConnectId, err := k8ssecret.GetDataFromSecretName(ctx, controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectId, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedId, edgeConnectId)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "CreateEdgeConnect", edgeconnectClient.NewRequest(testName, testHostPatterns, testHostMappings, ""))
	})
}

func TestReconcileProvisionerWithK8sAutomationsUpdate(t *testing.T) {
	t.Run("update EdgeConnect", func(t *testing.T) {
		ec := createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		edgeConnectClient := edgeconnectmock.NewClient(t)

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

		edgeConnectClient.AssertCalled(t, "GetEdgeConnects", testName)
		edgeConnectClient.AssertCalled(t, "GetEdgeConnect", testCreatedId)
		edgeConnectClient.AssertCalled(t, "UpdateEdgeConnect", testCreatedId, edgeconnectClient.NewRequest(testName, testHostPatterns2, testHostMappings, testCreatedOauthClientId))
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
	return newSecret(name, namespace, map[string]string{
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
	fakeClient := fake.NewClientWithIndex()

	if ec != nil {
		objs := []client.Object{ec}
		objs = append(objs, objects...)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	mockImageGetter := registrymock.NewImageGetter(t)

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Maybe()

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	mockEdgeConnectClient := edgeconnectmock.NewClient(t)

	mockEdgeConnectClientBuilder := func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		return mockEdgeConnectClient, nil
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    mockRegistryClientBuilder,
		edgeConnectClientBuilder: mockEdgeConnectClientBuilder,
	}

	return controller
}

func createFakeClientAndReconcilerForProvisioner(t *testing.T, ec *edgeconnect.EdgeConnect, builder edgeConnectClientBuilderType, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex()

	if ec != nil {
		objs := []client.Object{ec}
		objs = append(objs, objects...)
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
	}

	return controller
}

func mockNewEdgeConnectClientCreate(edgeConnectClient *edgeconnectmock.Client, hostPatterns []string) func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnectClient.ListResponse{
				TotalCount: 0,
			},
			nil,
		)

		// CreateEdgeConnect creates edge connect
		edgeConnectClient.On("CreateEdgeConnect", edgeconnectClient.NewRequest(testName, hostPatterns, testHostMappings, "")).Return(
			edgeconnectClient.CreateResponse{
				ID:                  testCreatedId,
				Name:                testName,
				HostPatterns:        hostPatterns,
				OauthClientId:       testCreatedOauthClientId,
				OauthClientSecret:   testCreatedOauthClientSecret,
				OauthClientResource: testCreatedOauthClientResource,
			},
			nil,
		)

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientRecreate(edgeConnectClient *edgeconnectmock.Client, id string) func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnectClient.ListResponse{
				EdgeConnects: []edgeconnectClient.GetResponse{
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
		edgeConnectClient.On("CreateEdgeConnect", edgeconnectClient.NewRequest(testName, testHostPatterns, testHostMappings, "")).Return(
			edgeconnectClient.CreateResponse{
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

func mockNewEdgeConnectClientDelete(edgeConnectClient *edgeconnectmock.Client) func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnectClient.ListResponse{
				EdgeConnects: []edgeconnectClient.GetResponse{
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

func mockNewEdgeConnectClientDeleteNotFoundOnTenant(edgeConnectClient *edgeconnectmock.Client) func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnectClient.ListResponse{
				TotalCount: 0,
			},
			nil,
		)
		edgeConnectClient.On("DeleteEdgeConnect", testCreatedId).Return(nil).Maybe()

		return edgeConnectClient, nil
	}
}

func mockNewEdgeConnectClientUpdate(edgeConnectClient *edgeconnectmock.Client, fromHostPatterns []string, toHostPatterns []string) func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient.On("GetEdgeConnects", testName).Return(
			edgeconnectClient.ListResponse{
				EdgeConnects: []edgeconnectClient.GetResponse{
					{
						ID:                         testCreatedId,
						Name:                       testName,
						HostPatterns:               fromHostPatterns,
						OauthClientId:              testOauthClientId,
						ManagedByDynatraceOperator: true,
					},
				},
				TotalCount: 1,
			},
			nil,
		)

		edgeConnectClient.On("GetEdgeConnect", testCreatedId).Return(
			edgeconnectClient.GetResponse{
				ID:            testCreatedId,
				Name:          testName,
				HostPatterns:  fromHostPatterns,
				OauthClientId: testOauthClientId,
			},
			nil,
		)

		// CreateEdgeConnect creates edge connect
		edgeConnectClient.On("UpdateEdgeConnect", testCreatedId, edgeconnectClient.NewRequest(testName, toHostPatterns, testHostMappings, testCreatedOauthClientId)).Return(nil)

		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		edgeConnectClient.On("UpdateConnectionSetting", mock.Anything).Return(nil)

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
			ApiServer: "abc12345.dynatrace.com",
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
			Name:      kubeSystemNamespaceName,
			Namespace: "",
			UID:       testUID,
		},
	}
}

func TestController_createOrUpdateConnectionSetting(t *testing.T) {
	t.Run("Create Connection Setting object", func(t *testing.T) {
		controller := mockController()
		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{}, nil)
		edgeConnectClient.On("CreateConnectionSetting", mock.Anything).Return(nil)
		err := controller.createOrUpdateConnectionSetting(edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})
	t.Run("Existing Connection Setting object", func(t *testing.T) {
		controller := mockController()
		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil)
		err := controller.createOrUpdateConnectionSetting(edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
		edgeConnectClient.AssertNotCalled(t, "CreateConnectionSetting", mock.Anything)
	})
	t.Run("Existing object with same Cluster ID but different name", func(t *testing.T) {
		controller := mockController()
		differentEnvironmentSetting := testEnvironmentSetting
		differentEnvironmentSetting.Value.Name = "different-name"
		differentEnvironmentSetting.Value.Namespace = "different-namespace"

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return([]edgeconnectClient.EnvironmentSetting{differentEnvironmentSetting}, nil)
		edgeConnectClient.On("CreateConnectionSetting", mock.Anything).Return(nil)
		err := controller.createOrUpdateConnectionSetting(edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})
	t.Run("Server fails", func(t *testing.T) {
		controller := mockController()
		expectedEnvironmentSetting := testEnvironmentSetting
		expectedEnvironmentSetting.Value.Name = "different-name"
		expectedEnvironmentSetting.Value.Namespace = "different-namespace"

		edgeConnectClient := edgeconnectmock.NewClient(t)
		edgeConnectClient.On("GetConnectionSettings").Return(nil, errors.New("something went wrong"))
		err := controller.createOrUpdateConnectionSetting(edgeConnectClient, createEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.Error(t, err)
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
