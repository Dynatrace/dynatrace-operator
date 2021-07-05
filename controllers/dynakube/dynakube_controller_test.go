package dynakube

import (
	"context"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/capability"
	rcap "github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/capability"
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
	testVersion   = "1.217-12345-678910"

	testUUID = "test-uuid"

	testHost     = "test-host"
	testPort     = uint32(1234)
	testProtocol = "test-protocol"

	testAnotherHost     = "test-another-host"
	testAnotherPort     = uint32(5678)
	testAnotherProtocol = "test-another-protocol"
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

		mockClient.On("GetCommunicationHostForClient").Return(&dtclient.CommunicationHost{
			Protocol: testProtocol,
			Host:     testHost,
			Port:     testPort,
		}, nil)
		mockClient.On("GetAgentTenantInfo").
			Return(&dtclient.TenantInfo{
				ConnectionInfo: dtclient.ConnectionInfo{
					CommunicationHosts: []*dtclient.CommunicationHost{
						{
							Protocol: testProtocol,
							Host:     testHost,
							Port:     testPort,
						},
						{
							Protocol: testAnotherProtocol,
							Host:     testAnotherHost,
							Port:     testAnotherPort,
						},
					},
					TenantUUID: testUUID,
				},
			}, nil)
		mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
		mockClient.On("GetAgentTenantInfo").Return(&dtclient.TenantInfo{
			ConnectionInfo: dtclient.ConnectionInfo{TenantUUID: "abc123456"},
		}, nil)
		mockClient.On("GetAGTenantInfo").Return(&dtclient.TenantInfo{}, nil)
		mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(testVersion, nil)
		mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypePaaS).Return(testVersion, nil)

		instance := &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(instance,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: kubesystem.Namespace,
					UID:  testUID,
				},
			},
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

		mockClient.On("GetCommunicationHostForClient").Return(&dtclient.CommunicationHost{
			Protocol: testProtocol,
			Host:     testHost,
			Port:     testPort,
		}, nil)
		mockClient.On("GetAgentTenantInfo").Return(&dtclient.TenantInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				CommunicationHosts: []*dtclient.CommunicationHost{
					{
						Protocol: testProtocol,
						Host:     testHost,
						Port:     testPort,
					},
					{
						Protocol: testAnotherProtocol,
						Host:     testAnotherHost,
						Port:     testAnotherPort,
					},
				},
				TenantUUID: testUUID,
			},
		}, nil)
		mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
		mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(testVersion, nil)
		mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypePaaS).Return(testVersion, nil)
		mockClient.On("GetAGTenantInfo").
			Return(&dtclient.TenantInfo{
				ConnectionInfo: dtclient.ConnectionInfo{
					TenantUUID: "123",
				},
				Token:                 "asdf",
				Endpoints:             nil,
				CommunicationEndpoint: nil,
			}, nil)

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

	mockClient.On("GetCommunicationHostForClient").Return(&dtclient.CommunicationHost{
		Protocol: testProtocol,
		Host:     testHost,
		Port:     testPort,
	}, nil)

	mockClient.On("GetAgentTenantInfo").
		Return(&dtclient.TenantInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				CommunicationHosts: []*dtclient.CommunicationHost{
					{
						Protocol: testProtocol,
						Host:     testHost,
						Port:     testPort,
					},
					{
						Protocol: testAnotherProtocol,
						Host:     testAnotherHost,
						Port:     testAnotherPort,
					},
				},
				TenantUUID: testUUID,
			},
		}, nil)
	mockClient.On("GetTokenScopes", testPaasToken).Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	mockClient.On("GetTokenScopes", testAPIToken).Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)
	mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return(testVersion, nil)
	mockClient.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypePaaS).Return(testVersion, nil)

	url1, _ := url.Parse("https://aaa")
	url2, _ := url.Parse("https://bbb")
	url3, _ := url.Parse("http://ccc")

	mockClient.On("GetAGTenantInfo").
		Return(&dtclient.TenantInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID: "123",
			},
			Token:                 "asdf",
			Endpoints:             []*url.URL{url1, url2, url3},
			CommunicationEndpoint: url1,
		},
			nil)

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
		Name:      rcap.BuildServiceName(testName, routingCapability.GetModuleName()),
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
		Name:      rcap.BuildServiceName(testName, routingCapability.GetModuleName()),
	}, routingSvc)
	assert.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}
