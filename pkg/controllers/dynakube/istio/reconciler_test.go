package istio

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
)

func TestSplitCommunicationHost(t *testing.T) {
	t.Run("empty => no fail", func(t *testing.T) {
		ipHosts, fqdnHosts := splitCommunicationHost([]dtclient.CommunicationHost{})
		require.Nil(t, ipHosts)
		require.Nil(t, fqdnHosts)
	})
	t.Run("nil => no fail", func(t *testing.T) {
		ipHosts, fqdnHosts := splitCommunicationHost(nil)
		require.Nil(t, ipHosts)
		require.Nil(t, fqdnHosts)
	})
	t.Run("success", func(t *testing.T) {
		comHosts := []dtclient.CommunicationHost{
			createTestIPCommunicationHost(),
			createTestFQDNCommunicationHost(),
			createTestIPCommunicationHost(),
			createTestFQDNCommunicationHost(),
			createTestIPCommunicationHost(),
			createTestFQDNCommunicationHost(),
		}

		ipHosts, fqdnHosts := splitCommunicationHost(comHosts)
		require.NotNil(t, ipHosts)
		require.NotNil(t, fqdnHosts)
		assert.Len(t, ipHosts, 3)
		assert.Len(t, fqdnHosts, 3)
	})
}

func TestReconcileIPServiceEntry(t *testing.T) {
	ctx := context.Background()
	component := "best-component"
	dynakube := createTestDynaKube()

	t.Run("empty communication host => delete if previously created", func(t *testing.T) {
		serviceEntry := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dynakube.Name, component),
				Namespace: dynakube.Namespace,
			},
		}
		fakeClient := fakeistio.NewSimpleClientset(serviceEntry)
		istioClient := newTestingClient(fakeClient, dynakube.Namespace)
		reconciler := NewReconciler(istioClient).(*reconciler)

		err := reconciler.reconcileIPServiceEntry(ctx, nil, component)
		require.NoError(t, err)
		_, err = fakeClient.NetworkingV1beta1().ServiceEntries(serviceEntry.Namespace).Get(ctx, serviceEntry.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dynakube.Namespace)
		reconciler := NewReconciler(istioClient).(*reconciler)
		commHosts := []dtclient.CommunicationHost{
			createTestIPCommunicationHost(),
		}

		err := reconciler.reconcileIPServiceEntry(ctx, commHosts, component)
		require.NoError(t, err)

		expectedName := BuildNameForIPServiceEntry(dynakube.Name, component)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dynakube.Namespace).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dynakube.Namespace)
		reconciler := NewReconciler(istioClient).(*reconciler)
		commHosts := []dtclient.CommunicationHost{
			createTestIPCommunicationHost(),
		}

		err := reconciler.reconcileIPServiceEntry(ctx, commHosts, component)
		require.Error(t, err)
	})
}

func TestReconcileFQDNServiceEntry(t *testing.T) {
	ctx := context.Background()
	component := "best-component"
	owner := createTestDynaKube()

	t.Run("empty communication host => delete if previously created", func(t *testing.T) {
		serviceEntry := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(owner.GetName(), component),
				Namespace: owner.GetNamespace(),
			},
		}
		virtualService := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(owner.GetName(), component),
				Namespace: owner.GetNamespace(),
			},
		}
		fakeClient := fakeistio.NewSimpleClientset(serviceEntry, virtualService)
		istioClient := newTestingClient(fakeClient, owner.GetNamespace())
		reconciler := NewReconciler(istioClient).(*reconciler)

		err := reconciler.reconcileFQDNServiceEntry(ctx, nil, component)
		require.NoError(t, err)
		_, err = fakeClient.NetworkingV1beta1().ServiceEntries(serviceEntry.Namespace).Get(ctx, serviceEntry.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
		_, err = fakeClient.NetworkingV1beta1().VirtualServices(serviceEntry.Namespace).Get(ctx, virtualService.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, owner.GetNamespace())
		reconciler := NewReconciler(istioClient).(*reconciler)
		commHosts := []dtclient.CommunicationHost{
			createTestFQDNCommunicationHost(),
		}

		err := reconciler.reconcileFQDNServiceEntry(ctx, commHosts, component)
		require.NoError(t, err)

		expectedName := BuildNameForFQDNServiceEntry(owner.GetName(), component)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(owner.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(owner.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, owner.GetNamespace())
		reconciler := NewReconciler(istioClient).(*reconciler)
		commHosts := []dtclient.CommunicationHost{
			createTestFQDNCommunicationHost(),
		}

		err := reconciler.reconcileFQDNServiceEntry(ctx, commHosts, component)
		require.Error(t, err)
	})
}

func TestReconcileAPIUrl(t *testing.T) {
	ctx := context.Background()
	dynakube := createTestDynaKube()

	t.Run("nil => error", func(t *testing.T) {
		istioClient := newTestingClient(nil, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, nil)
		require.Error(t, err)
	})
	t.Run("malformed api-url => error", func(t *testing.T) {
		dynakube := createTestDynaKube()
		dynakube.Spec.APIURL = "something-random"
		istioClient := newTestingClient(nil, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dynakube)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dynakube)
		require.NoError(t, err)

		expectedName := BuildNameForFQDNServiceEntry(dynakube.GetName(), OperatorComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dynakube.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dynakube)
		require.Error(t, err)
	})
}

func TestReconcileOneAgentCommunicationHosts(t *testing.T) {
	ctx := context.Background()
	dynakube := createTestDynaKube()

	t.Run("nil => error", func(t *testing.T) {
		istioClient := newTestingClient(nil, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, dynakube)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dynakube.GetName(), OneAgentComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "something.test.io")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dynakube.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		expectedIPName := BuildNameForIPServiceEntry(dynakube.GetName(), OneAgentComponent)
		serviceEntry, err = fakeClient.NetworkingV1beta1().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedIPName, metav1.GetOptions{})

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, dynakube)
		require.Error(t, err)
	})
}

func TestReconcileActiveGateCommunicationHosts(t *testing.T) {
	ctx := context.Background()
	dynakube := createTestDynaKube()

	t.Run("nil => error", func(t *testing.T) {
		istioClient := newTestingClient(nil, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, dynakube)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dynakube.GetName(), ActiveGateComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dynakube.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "abcd123.some.activegate.endpointurl.com")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dynakube.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dynakube.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, dynakube)
		require.Error(t, err)
	})
}

func createTestIPCommunicationHost() dtclient.CommunicationHost {
	return dtclient.CommunicationHost{
		Protocol: "http",
		Host:     "42.42.42.42",
		Port:     620,
	}
}

func createTestFQDNCommunicationHost() dtclient.CommunicationHost {
	return dtclient.CommunicationHost{
		Protocol: "http",
		Host:     "something.test.io",
		Port:     620,
	}
}

func createTestDynaKube() *dynatracev1beta1.DynaKube {
	fqdnHost := createTestFQDNCommunicationHost()
	ipHost := createTestIPCommunicationHost()
	endpoints := "https://abcd123.some.activegate.endpointurl.com:443"

	return &dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind: "DynaKube",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "owner",
			Namespace: "test",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://test.dev.dynatracelabs.com/api",
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
					CommunicationHosts: []dynatracev1beta1.CommunicationHostStatus{
						{
							Protocol: fqdnHost.Protocol,
							Host:     fqdnHost.Host,
							Port:     fqdnHost.Port,
						},
						{
							Protocol: ipHost.Protocol,
							Host:     ipHost.Host,
							Port:     ipHost.Port,
						},
					},
				},
			},
			ActiveGate: dynatracev1beta1.ActiveGateStatus{
				ConnectionInfoStatus: dynatracev1beta1.ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
						TenantUUID:  "test-tenant",
						Endpoints:   endpoints,
						LastRequest: metav1.Time{},
					},
				},
			},
		},
	}
}

func TestIstio(t *testing.T) {
	type test struct {
		name    string
		input   []*metav1.APIResourceList
		wantErr error
		want    bool
	}

	tests := []test{
		{name: "enabled", input: []*metav1.APIResourceList{{GroupVersion: IstioGVR}}, wantErr: nil, want: true},
		{name: "disabled", input: []*metav1.APIResourceList{}, wantErr: nil, want: false},
	}

	ist := fakeistio.NewSimpleClientset()

	fakeDiscovery, ok := ist.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeDiscovery.Resources = tc.input
			istioClient := newTestingClient(ist, "")
			isInstalled, err := istioClient.CheckIstioInstalled()
			assert.Equal(t, tc.want, isInstalled)
			require.ErrorIs(t, tc.wantErr, err)
		})
	}
}
