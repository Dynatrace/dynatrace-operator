package dynakube

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability/kubemon"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability/routing"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testUID       = "test-uid"
	testPaasToken = "test-paas-token"
	testAPIToken  = "test-api-token"
	testVersion   = "1.0.0"
)

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
}

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		r := &ReconcileDynaKube{
			client: fakeClient,
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Reconcile works with minimal setup and interface`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)
		instance := &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(instance,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"apiToken":  []byte(testAPIToken),
					"paasToken": []byte(testPaasToken),
				},
			}).Build()
		r := &ReconcileDynaKube{
			client:    fakeClient,
			apiReader: fakeClient,
			scheme:    scheme.Scheme,
			dtcBuildFunc: func(_ client.Client, _ *v1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run(`Reconcile reconciles Kubernetes Monitoring if enabled`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		instance := &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					CapabilityProperties: v1alpha1.CapabilityProperties{
						Enabled: true,
					},
				}}}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(instance,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					dtclient.DynatracePaasToken: []byte(testPaasToken),
					dtclient.DynatraceApiToken:  []byte(testAPIToken),
				}},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: kubesystem.Namespace,
					UID:  testUID,
				}}).Build()
		r := &ReconcileDynaKube{
			client:    fakeClient,
			apiReader: fakeClient,
			scheme:    scheme.Scheme,
			dtcBuildFunc: func(_ client.Client, _ *v1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
			enableUpdates: false,
		}

		mockClient.
			On("GetConnectionInfo").
			Return(dtclient.ConnectionInfo{
				TenantUUID: testName,
			}, nil)
		mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		result, err := r.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		var statefulSet appsv1.StatefulSet
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: instance.Name + kubemon.StatefulSetSuffix, Namespace: testNamespace}, &statefulSet)

		assert.NoError(t, err)
		assert.NotNil(t, statefulSet)
	})
}

func TestReconcile_RemoveRoutingIfDisabled(t *testing.T) {
	mockClient := &dtclient.MockDynatraceClient{}
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.DynaKubeSpec{
			RoutingSpec: v1alpha1.RoutingSpec{
				CapabilityProperties: v1alpha1.CapabilityProperties{
					Enabled: true,
				},
			}}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(instance,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dtclient.DynatracePaasToken: []byte(testPaasToken),
				dtclient.DynatraceApiToken:  []byte(testAPIToken),
			}},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			}}).Build()
	r := &ReconcileDynaKube{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		dtcBuildFunc: func(_ client.Client, _ *v1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
			return mockClient, nil
		},
		enableUpdates: false,
	}
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
	}
	mockClient.
		On("GetConnectionInfo").
		Return(dtclient.ConnectionInfo{
			TenantUUID: testName,
		}, nil)
	mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

	_, err := r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	routingSts := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + routing.StatefulSetSuffix,
	}, routingSts)
	assert.NoError(t, err)
	assert.NotNil(t, routingSts)

	err = r.client.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
	require.NoError(t, err)

	instance.Spec.RoutingSpec.Enabled = false
	err = r.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testName + routing.StatefulSetSuffix,
	}, routingSts)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func TestReconcile_UpdateImageVersion(t *testing.T) {
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &ReconcileDynaKube{
		client:        fakeClient,
		enableUpdates: false,
		logger:        logger.NewDTLogger(),
	}
	rec := &utils.Reconciliation{
		Instance: instance,
	}

	updated, err := r.updateImageVersion(rec, nil)
	assert.NoError(t, err)
	assert.False(t, updated)

	r.enableUpdates = true
	updated, err = r.updateImageVersion(rec, nil)
	assert.Error(t, err)
	assert.False(t, updated)

	data, err := buildTestDockerAuth(t)
	require.NoError(t, err)

	err = createTestPullSecret(t, r.client, rec, data)
	require.NoError(t, err)

	imgVersionProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{
			Version: testVersion,
			Hash:    testValue,
		}, nil
	}

	updated, err = r.updateImageVersion(rec, imgVersionProvider)
	assert.NoError(t, err)
	assert.True(t, updated)

	rec.Instance.Status.ActiveGate.ImageVersion = testVersion
	rec.Instance.Status.ActiveGate.ImageHash = testValue

	updated, err = r.updateImageVersion(rec, imgVersionProvider)
	assert.NoError(t, err)
	assert.False(t, updated)
}

// Adding *testing.T parameter to prevent usage in production code
func createTestPullSecret(_ *testing.T, clt client.Client, rec *utils.Reconciliation, data []byte) error {
	return clt.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rec.Instance.Namespace,
			Name:      rec.Instance.Name + dtpullsecret.PullSecretSuffix,
		},
		Data: map[string][]byte{
			".dockerconfigjson": data,
		},
	})
}

// Adding *testing.T parameter to prevent usage in production code
func buildTestDockerAuth(_ *testing.T) ([]byte, error) {
	dockerConf := struct {
		Auths map[string]dtversion.DockerAuth `json:"auths"`
	}{
		Auths: map[string]dtversion.DockerAuth{
			testKey: {
				Username: testName,
				Password: testValue,
			},
		},
	}
	return json.Marshal(dockerConf)
}
