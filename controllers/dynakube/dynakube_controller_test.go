package dynakube

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
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
	testUID       = "test-uid"
	testPaasToken = "test-paas-token"
	testAPIToken  = "test-api-token"
)

func TestReconcileActiveGate_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		r := &ReconcileDynaKube{
			client: fake.NewClient(),
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
		fakeClient := fake.NewClient(instance,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"apiToken":  []byte(testAPIToken),
					"paasToken": []byte(testPaasToken),
				},
			})
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
		fakeClient := fake.NewClient(instance,
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
				}})
		r := &ReconcileDynaKube{
			client:    fakeClient,
			apiReader: fakeClient,
			scheme:    scheme.Scheme,
			dtcBuildFunc: func(_ client.Client, _ *v1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
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

		kubeMonCapability := capability.NewKubeMonCapability(&instance.Spec.KubernetesMonitoringSpec.CapabilityProperties)
		name := capability.CalculateStatefulSetName(kubeMonCapability, instance.Name)
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

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
	fakeClient := fake.NewClient(instance,
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
			}})
	r := &ReconcileDynaKube{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		dtcBuildFunc: func(_ client.Client, _ *v1alpha1.DynaKube, _ *corev1.Secret) (dtclient.Client, error) {
			return mockClient, nil
		},
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

	// Reconcile twice since routing service is created before the stateful set
	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	routingCapability := capability.NewRoutingCapability(&instance.Spec.RoutingSpec.CapabilityProperties)
	stsName := capability.CalculateStatefulSetName(routingCapability, testName)

	routingSts := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.NoError(t, err)
	assert.NotNil(t, routingSts)

	routingSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      capability.BuildServiceName(testName, routingCapability.GetModuleName()),
	}, routingSvc)
	assert.NoError(t, err)
	assert.NotNil(t, routingSvc)

	err = r.client.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
	require.NoError(t, err)

	instance.Spec.RoutingSpec.Enabled = false
	err = r.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      stsName,
	}, routingSts)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))

	err = r.client.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      capability.BuildServiceName(testName, routingCapability.GetModuleName()),
	}, routingSvc)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}
