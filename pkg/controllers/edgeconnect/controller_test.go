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

func Test_Controller_Reconcile(t *testing.T) {
	t.Run("create works with minimal setup", func(t *testing.T) {
		ec := testEdgeConnectRegularCR()

		controller := testFakeClientAndReconciler(t, ec,
			testClientSecret(testOauthClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("timestamp update in EdgeConnect status works", func(t *testing.T) {
		now := metav1.Now()
		ec := testEdgeConnectRegularCR()
		ec.Status = edgeconnect.EdgeConnectStatus{
			UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			Version: status.VersionStatus{
				LastProbeTimestamp: &now,
				ImageID:            "docker.io/dynatrace/edgeconnectClient:latest",
			},
		}

		controller := testFakeClientAndReconciler(t, ec,
			testClientSecret(testOauthClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)
		controller.timeProvider.Freeze()

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		err = controller.apiReader.Get(t.Context(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, ec)
		require.NoError(t, err)
		// Fake client drops seconds, so we have to do the same
		expectedTimestamp := controller.timeProvider.Now().Truncate(time.Second)
		assert.Equal(t, expectedTimestamp, ec.Status.UpdatedTimestamp.Time)
	})

	t.Run("reconciles phase change correctly", func(t *testing.T) {
		ec := testEdgeConnectRegularCR()

		controller := testFakeClientAndReconciler(t, ec,
			testClientSecret(testOauthClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.Equal(t, defaultRequeueInterval, result.RequeueAfter)

		var edgeConnectDeployment edgeconnect.EdgeConnect

		require.NoError(t,
			controller.client.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &edgeConnectDeployment))
		require.NoError(t, controller.client.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, ec))
		assert.Equal(t, status.Running, ec.Status.DeploymentPhase)
	})

	t.Run("reconciles doesn't fail if edgeconnect not found", func(t *testing.T) {
		controller := testFakeClientNoVersionCheck(t, nil)

		_, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})

	t.Run("reconciles custom CA provided", func(t *testing.T) {
		const testCAConfigMapName = "test-ca-name"

		ec := testEdgeConnectRegularCR()
		ec.Spec.CaCertsRef = testCAConfigMapName

		data := make(map[string]string)
		data[consts.EdgeConnectCAConfigMapKey] = "dummy"
		customCA := testConfigMap(testCAConfigMapName, ec.Namespace, data)
		clientSecret := testClientSecret(testOauthClientSecret, ec.Namespace)

		controller := testFakeClientAndReconciler(t, ec, clientSecret, customCA, testKubeSystemNamespace())

		_, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
	})

	t.Run("SecretConfigConditionType is set SecretCreated", func(t *testing.T) {
		ec := testEdgeConnectRegularCR()

		controller := testFakeClientAndReconciler(t, ec,
			testClientSecret(testOauthClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		_, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)

		err = controller.apiReader.Get(t.Context(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, ec)
		require.NoError(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, ec.Name+"-"+consts.EdgeConnectSecretSuffix+" created", condition.Message)
	})

	t.Run("SecretConfigConditionType is set SecretGenFailed failed to get clientSecret", func(t *testing.T) {
		ec := testEdgeConnectRegularCR()

		controller := testFakeClientNoVersionCheck(t, ec,
			testKubeSystemNamespace(),
		)

		err := controller.reconcileEdgeConnectRegular(t.Context(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret")
	})

	t.Run("SecretConfigConditionType is set SecretGenFailed failed", func(t *testing.T) {
		ec := testEdgeConnectRegularCR()

		controller := testFakeClientNoVersionCheck(t, ec,
			testKubeSystemNamespace(),
		)

		boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("BOOM")
			},
		})
		controller.apiReader = boomClient
		controller.secrets = k8ssecret.Query(controller.client, controller.apiReader, log)

		err := controller.reconcileEdgeConnectRegular(t.Context(), ec)
		require.Error(t, err)
		require.NotEmpty(t, ec.Conditions())

		condition := meta.FindStatusCondition(*ec.Conditions(), consts.SecretConfigConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.SecretGenerationFailed, condition.Reason)
		assert.Contains(t, condition.Message, "Failed to generate secret")
	})
}

func Test_Controller_Reconcile_replicas(t *testing.T) {
	testEdgeConnect := func(provisioner bool, replicas *int32) *edgeconnect.EdgeConnect {
		ec := testEdgeConnectRegularCR()
		if provisioner {
			ec = testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		}

		ec.Spec.Replicas = replicas

		return ec
	}

	testController := func(t *testing.T, ec *edgeconnect.EdgeConnect, provisioner bool, objs ...client.Object) *Controller {
		if !provisioner {
			return testFakeClientAndReconciler(t, ec, objs...)
		}

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		return testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientCreate(ecClient, testHostPatterns),
			objs...,
		)
	}

	testObjects := func(ec *edgeconnect.EdgeConnect, provisioner bool, existingReplicas *int32) []client.Object {
		objs := []client.Object{
			testKubeSystemNamespace(),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
		}

		if provisioner {
			objs = append(objs, testClientSecret(ec.ClientSecretName(), ec.Namespace))
		}

		if existingReplicas != nil {
			existing := deployment.New(ec)
			existing.Spec.Replicas = existingReplicas
			objs = append(objs, existing)
		}

		return objs
	}

	testAssertDeploymentReplicas := func(t *testing.T, apiReader client.Reader, ec *edgeconnect.EdgeConnect, expectedReplicas int32) {
		t.Helper()

		d := &appsv1.Deployment{}
		err := apiReader.Get(t.Context(), client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, d)
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
					ec := testEdgeConnect(mode.provisioner, tc.specReplicas)

					objs := testObjects(ec, mode.provisioner, tc.existingReplicas)

					controller := testController(t, ec, mode.provisioner, objs...)

					_, err := controller.Reconcile(t.Context(), reconcile.Request{
						NamespacedName: types.NamespacedName{Namespace: ec.Namespace, Name: ec.Name},
					})
					require.NoError(t, err)

					testAssertDeploymentReplicas(t, controller.apiReader, ec, tc.expectedReplicas)
				})
			}
		})
	}
}

func Test_Controller_Reconcile_provisioner(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientCreate(ecClient, testHostPatterns),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		ecOauthClientID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, ecOauthClientID)

		ecOauthClientSecret, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, ecOauthClientSecret)

		ecOauthResource, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, ecOauthResource)

		ecID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, ecID)

		var ecDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			t.Context(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&ecDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", ecDeployment.Spec.Template.Spec.Containers[0].Name)
	})

	t.Run("recreate due to missing client secret", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientRecreate(ecClient, testCreatedID),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		ecOauthClientID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, ecOauthClientID)

		ecOauthClientSecret, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, ecOauthClientSecret)

		ecOauthResource, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, ecOauthResource)

		ecID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, ecID)

		var ecDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			t.Context(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&ecDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", ecDeployment.Spec.Template.Spec.Containers[0].Name)
	})

	t.Run("recreate due to invalid id", func(t *testing.T) {
		const testRecreatedInvalidID = "id-somehow-different"

		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientRecreate(ecClient, testRecreatedInvalidID),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testClientSecret(ec.ClientSecretName(), ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		ecOauthClientID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, ecOauthClientID)

		ecOauthClientSecret, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, ecOauthClientSecret)

		ecOauthResource, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, ecOauthResource)

		ecID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, ecID)

		var ecDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			t.Context(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&ecDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", ecDeployment.Spec.Template.Spec.Containers[0].Name)
	})

	t.Run("delete", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().DeleteEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientForDeletion(
			t,
			ec,
			testNewEdgeConnectClientDelete(ecClient),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testClientSecret(ec.ClientSecretName(), ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("delete - missing client secret", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().DeleteEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientForDeletion(
			t,
			ec,
			testNewEdgeConnectClientDelete(ecClient),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("delete - missing EdgeConnect on the tenant", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{finalizerName}, &metav1.Time{Time: time.Now()}, testHostPatterns)

		ecClient := edgeconnectmock.NewClient(t)

		controller := testFakeClientForDeletion(
			t,
			ec,
			testNewEdgeConnectClientDeleteNotFoundOnTenant(ecClient),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testClientSecret(ec.ClientSecretName(), ec.Namespace),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("update", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)

		ecClient := edgeconnectmock.NewClient(t)

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientUpdate(ecClient, testHostPatterns, testHostPatterns2),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testClientSecret(ec.ClientSecretName(), ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("k8s automation create", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientCreate(ecClient, testHostPatterns),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)

		edgeConnectCR, err := testGetEdgeConnectCR(t, controller.apiReader, ec.Name, ec.Namespace)
		require.NoError(t, err)
		require.NotEmpty(t, edgeConnectCR.Finalizers)

		ecOauthClientID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientID, ecOauthClientID)

		ecOauthClientSecret, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthClientSecret, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientSecret, ecOauthClientSecret)

		ecOauthResource, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectOauthResource, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedOauthClientResource, ecOauthResource)

		ecID, err := k8ssecret.GetDataFromSecretName(t.Context(), controller.apiReader, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace}, consts.KeyEdgeConnectID, log)
		require.NoError(t, err)
		assert.Equal(t, testCreatedID, ecID)

		var ecDeployment appsv1.Deployment
		err = controller.apiReader.Get(
			t.Context(),
			client.ObjectKey{
				Name:      ec.Name,
				Namespace: ec.Namespace,
			},
			&ecDeployment,
		)
		require.NoError(t, err)
		assert.Equal(t, "edge-connect", ecDeployment.Spec.Template.Spec.Containers[0].Name)
	})

	t.Run("k8s automation update", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns2)
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		}

		ecClient := edgeconnectmock.NewClient(t)

		controller := testFakeClientAndReconcilerForProvisioner(
			t,
			ec,
			testNewEdgeConnectClientUpdate(ecClient, testHostPatterns, testHostPatterns2),
			testOauthSecret(ec.Spec.OAuth.ClientSecret, ec.Namespace),
			testClientSecret(ec.ClientSecretName(), ec.Namespace),
			testKubeSystemNamespace(),
		)

		result, err := controller.Reconcile(t.Context(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func Test_Controller_createOrUpdateConnectionSetting(t *testing.T) {
	t.Run("create connection setting object", func(t *testing.T) {
		controller := testNewController()
		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{}, nil).Once()
		ecClient.EXPECT().CreateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()
		err := controller.createOrUpdateConnectionSetting(t.Context(), ecClient, testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})

	t.Run("existing connection setting object", func(t *testing.T) {
		controller := testNewController()
		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
		err := controller.createOrUpdateConnectionSetting(t.Context(), ecClient, testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})

	t.Run("existing object with same cluster ID but different name", func(t *testing.T) {
		controller := testNewController()
		differentEnvironmentSetting := testEnvironmentSetting
		differentEnvironmentSetting.Value.Name = "different-name"
		differentEnvironmentSetting.Value.Namespace = "different-namespace"

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{differentEnvironmentSetting}, nil).Once()
		ecClient.EXPECT().CreateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()
		err := controller.createOrUpdateConnectionSetting(t.Context(), ecClient, testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.NoError(t, err)
	})

	t.Run("server fails", func(t *testing.T) {
		controller := testNewController()

		ecClient := edgeconnectmock.NewClient(t)
		ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return(nil, errors.New("something went wrong")).Once()
		err := controller.createOrUpdateConnectionSetting(t.Context(), ecClient, testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns), "")
		require.Error(t, err)
	})
}

func Test_newEdgeConnectClient(t *testing.T) {
	t.Run("new EdgeConnect client with scopes including k8s automation extra scopes", func(t *testing.T) {
		ec := testEdgeConnectProvisionerCR([]string{}, nil, testHostPatterns)
		ecClientBuilder := newEdgeConnectClient()
		require.NotNil(t, ecClientBuilder)
		actualClient, err := ecClientBuilder(t.Context(), ec, oauthCredentialsType{clientID: "fake", clientSecret: "fake"}, nil)
		require.NoError(t, err)
		require.NotNil(t, actualClient)
	})

	t.Run("new EdgeConnect client with min scopes and without k8s automation", func(t *testing.T) {
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
		ecClientBuilder := newEdgeConnectClient()
		require.NotNil(t, ecClientBuilder)
		actualClient, err := ecClientBuilder(t.Context(), ec, oauthCredentialsType{clientID: "fake", clientSecret: "fake"}, nil)
		require.NoError(t, err)
		require.NotNil(t, actualClient)
	})
}

func Test_buildOAuthScopes(t *testing.T) {
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

func testNewController() *Controller {
	return &Controller{
		client:                   fake.NewClient(),
		apiReader:                fake.NewClient(),
		registryClientBuilder:    registry.NewClient,
		config:                   &rest.Config{},
		timeProvider:             timeprovider.New(),
		edgeConnectClientBuilder: newEdgeConnectClient(),
	}
}

func testEdgeConnectRegularCR() *edgeconnect.EdgeConnect {
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

func testOauthSecret(name string, namespace string) *corev1.Secret {
	return testSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectOauthClientID:     testOauthClientID,
		consts.KeyEdgeConnectOauthClientSecret: testOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testOauthClientResource,
	})
}

func testClientSecret(name string, namespace string) *corev1.Secret {
	return testSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectID:                testCreatedID,
		consts.KeyEdgeConnectOauthClientID:     testCreatedOauthClientID,
		consts.KeyEdgeConnectOauthClientSecret: testCreatedOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testCreatedOauthClientResource,
	})
}

func testSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}

	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func testConfigMap(name, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}

func testGetEdgeConnectCR(t *testing.T, apiReader client.Reader, name string, namespace string) (edgeconnect.EdgeConnect, error) {
	var ec edgeconnect.EdgeConnect
	err := apiReader.Get(
		t.Context(),
		client.ObjectKey{
			Name:      name,
			Namespace: namespace,
		},
		&ec,
	)

	return ec, err
}

// testFakeClientNoVersionCheck builds a controller without a GetImageVersion mock expectation.
// Use this for tests that do not go through the full Reconcile path (e.g. direct calls to
// reconcileEdgeConnectRegular, or Reconcile with a missing EC that returns early).
func testFakeClientNoVersionCheck(t *testing.T, ec *edgeconnect.EdgeConnect, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(testCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, testCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	mockEdgeConnectClient := edgeconnectmock.NewClient(t)

	mockEdgeConnectClientBuilder := func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return mockEdgeConnectClient, nil
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    registry.NewClient,
		edgeConnectClientBuilder: mockEdgeConnectClientBuilder,
		secrets:                  k8ssecret.Query(fakeClient, fakeClient, log),
	}

	return controller
}

func testFakeClientAndReconciler(t *testing.T, ec *edgeconnect.EdgeConnect, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(testCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, testCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}

	mockImageGetter := registrymock.NewImageGetter(t)
	mockImageGetter.EXPECT().GetImageVersion(mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Maybe()

	mockRegistryClientBuilder := func(options ...func(*registry.Client)) (registry.ImageGetter, error) {
		return mockImageGetter, nil
	}

	mockEdgeConnectClient := edgeconnectmock.NewClient(t)

	mockEdgeConnectClientBuilder := func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
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

func testFakeClientAndReconcilerForProvisioner(t *testing.T, ec *edgeconnect.EdgeConnect, builder edgeConnectClientBuilderType, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(testCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, testCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	const fakeDigest = "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}

	mockImageGetter := registrymock.NewImageGetter(t)
	mockImageGetter.EXPECT().GetImageVersion(mock.Anything, mock.Anything).Return(fakeImageVersion, nil).Maybe()

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

// testFakeClientForDeletion builds a controller without a version-check registry mock.
// The deletion reconcile path skips updateVersionInfo, so GetImageVersion is never called.
func testFakeClientForDeletion(t *testing.T, ec *edgeconnect.EdgeConnect, builder edgeConnectClientBuilderType, objects ...client.Object) *Controller {
	fakeClient := fake.NewClientWithIndex(testCRD(t))

	if ec != nil {
		objs := slices.Concat([]client.Object{ec, testCRD(t)}, objects)
		fakeClient = fake.NewClientWithIndex(objs...)
	}

	controller := &Controller{
		client:                   fakeClient,
		apiReader:                fakeClient,
		timeProvider:             timeprovider.New(),
		registryClientBuilder:    registry.NewClient,
		edgeConnectClientBuilder: builder,
		secrets:                  k8ssecret.Query(fakeClient, fakeClient, log),
	}

	return controller
}

func testNewEdgeConnectClientCreate(ecClient *edgeconnectmock.Client, hostPatterns []string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	ecClient.EXPECT().ListEdgeConnects(mock.Anything, testName).Return(
		[]edgeconnectClient.APIResponse{},
		nil,
	).Once()

	ecClient.EXPECT().CreateEdgeConnect(mock.Anything, edgeconnectClient.NewCreateRequest(testName, hostPatterns, testHostMappings)).Return(
		edgeconnectClient.APIResponse{
			ID:                  testCreatedID,
			Name:                testName,
			HostPatterns:        hostPatterns,
			OauthClientID:       testCreatedOauthClientID,
			OauthClientSecret:   testCreatedOauthClientSecret,
			OauthClientResource: testCreatedOauthClientResource,
		},
		nil,
	).Once()

	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return ecClient, nil
	}
}

func testNewEdgeConnectClientRecreate(ecClient *edgeconnectmock.Client, id string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	ecClient.EXPECT().ListEdgeConnects(mock.Anything, testName).Return(
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
	).Once()

	ecClient.EXPECT().DeleteEdgeConnect(mock.Anything, id).Return(nil).Once()
	ecClient.EXPECT().CreateEdgeConnect(mock.Anything, edgeconnectClient.NewCreateRequest(testName, testHostPatterns, testHostMappings)).Return(
		edgeconnectClient.APIResponse{
			ID:                  testCreatedID,
			Name:                testName,
			HostPatterns:        testHostPatterns,
			OauthClientID:       testCreatedOauthClientID,
			OauthClientSecret:   testCreatedOauthClientSecret,
			OauthClientResource: testCreatedOauthClientResource,
		},
		nil,
	).Once()

	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return ecClient, nil
	}
}

func testNewEdgeConnectClientDelete(ecClient *edgeconnectmock.Client) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	ecClient.EXPECT().ListEdgeConnects(mock.Anything, testName).Return(
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
	).Once()
	ecClient.EXPECT().DeleteEdgeConnect(mock.Anything, testCreatedID).Return(nil).Once()

	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return ecClient, nil
	}
}

func testNewEdgeConnectClientDeleteNotFoundOnTenant(ecClient *edgeconnectmock.Client) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	ecClient.EXPECT().ListEdgeConnects(mock.Anything, testName).Return(
		[]edgeconnectClient.APIResponse{},
		nil,
	).Once()

	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return ecClient, nil
	}
}

func testNewEdgeConnectClientUpdate(ecClient *edgeconnectmock.Client, fromHostPatterns []string, toHostPatterns []string) func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	ecClient.EXPECT().ListEdgeConnects(mock.Anything, testName).Return(
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
	).Once()

	ecClient.EXPECT().GetEdgeConnect(mock.Anything, testCreatedID).Return(
		edgeconnectClient.APIResponse{
			ID:            testCreatedID,
			Name:          testName,
			HostPatterns:  fromHostPatterns,
			OauthClientID: testOauthClientID,
		},
		nil,
	).Once()

	ecClient.EXPECT().UpdateEdgeConnect(mock.Anything, testCreatedID, edgeconnectClient.NewUpdateRequest(testName, toHostPatterns, testHostMappings, testCreatedOauthClientID)).Return(nil).Once()

	ecClient.EXPECT().ListEnvironmentSettings(mock.Anything).Return([]edgeconnectClient.EnvironmentSetting{testEnvironmentSetting}, nil).Once()
	ecClient.EXPECT().UpdateEnvironmentSetting(mock.Anything, mock.Anything).Return(nil).Once()

	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, _ []byte) (edgeconnectClient.Client, error) {
		return ecClient, nil
	}
}

func testEdgeConnectProvisionerCR(finalizers []string, deletionTimestamp *metav1.Time, hostPatterns []string) *edgeconnect.EdgeConnect {
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

func testKubeSystemNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metav1.NamespaceSystem,
			Namespace: "",
			UID:       testUID,
		},
	}
}

func testDeployment(namespace, name string, replicas, readyReplicas int32) *appsv1.Deployment {
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

func testCRD(t *testing.T) *apiextensionsv1.CustomResourceDefinition {
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
