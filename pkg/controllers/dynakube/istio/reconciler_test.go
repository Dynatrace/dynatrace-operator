package istio

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	dk := createTestDynaKube()

	t.Run("empty communication host => delete if previously created", func(t *testing.T) {
		serviceEntry := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dk.Name, component),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fakeistio.NewSimpleClientset(serviceEntry)
		istioClient := newTestingClient(fakeClient, dk.Namespace)
		reconciler := NewReconciler(istioClient).(*reconciler)

		err := reconciler.reconcileIPServiceEntry(ctx, nil, component)
		require.NoError(t, err)
		_, err = fakeClient.NetworkingV1beta1().ServiceEntries(serviceEntry.Namespace).Get(ctx, serviceEntry.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.Namespace)
		reconciler := NewReconciler(istioClient).(*reconciler)
		commHosts := []dtclient.CommunicationHost{
			createTestIPCommunicationHost(),
		}

		err := reconciler.reconcileIPServiceEntry(ctx, commHosts, component)
		require.NoError(t, err)

		expectedName := BuildNameForIPServiceEntry(dk.Name, component)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.Namespace).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dk.Namespace)
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
	dk := createTestDynaKube()

	t.Run("nil => error", func(t *testing.T) {
		istioClient := newTestingClient(nil, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, nil)
		require.Error(t, err)
	})
	t.Run("malformed api-url => error", func(t *testing.T) {
		dk := createTestDynaKube()
		dk.Spec.APIURL = "something-random"
		istioClient := newTestingClient(nil, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dk)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dk)
		require.NoError(t, err)

		expectedName := BuildNameForFQDNServiceEntry(dk.GetName(), OperatorComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileAPIUrl(ctx, dk)
		require.Error(t, err)
	})
}

func TestReconcileOneAgentCommunicationHosts(t *testing.T) {
	ctx := context.Background()

	t.Run("nil => error", func(t *testing.T) {
		dk := createTestDynaKube()
		istioClient := newTestingClient(nil, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dk.GetName(), OneAgentComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "something.test.io")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		expectedIPName := BuildNameForIPServiceEntry(dk.GetName(), OneAgentComponent)
		serviceEntry, err = fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedIPName, metav1.GetOptions{})

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForCodeModule")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForCodeModuleChanged", statusCondition.Reason)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileCodeModuleCommunicationHosts(ctx, dk)
		require.Error(t, err)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForCodeModule")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForCodeModuleFailed", statusCondition.Reason)
	})
	t.Run("remove and cleanup if AppInjection is disabled", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		r := NewReconciler(istioClient)

		err := r.ReconcileCodeModuleCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dk.GetName(), OneAgentComponent)
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "something.test.io")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		expectedIPName := BuildNameForIPServiceEntry(dk.GetName(), OneAgentComponent)
		serviceEntry, err = fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedIPName, metav1.GetOptions{})

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForCodeModule")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForCodeModuleChanged", statusCondition.Reason)

		dk.Spec.OneAgent.CloudNativeFullStack = nil
		dk.Spec.OneAgent.HostMonitoring = &dynakube.HostInjectSpec{}

		err = r.ReconcileCodeModuleCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		statusCondition = meta.FindStatusCondition(*dk.Conditions(), "IstioForCodeModule")
		require.Nil(t, statusCondition)

		_, err = fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.Error(t, err)

		_, err = fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.Error(t, err)

		_, err = fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedIPName, metav1.GetOptions{})

		require.Error(t, err)
	})
}

func TestReconcileActiveGateCommunicationHosts(t *testing.T) {
	ctx := context.Background()

	t.Run("nil => error", func(t *testing.T) {
		dk := createTestDynaKube()
		istioClient := newTestingClient(nil, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dk.GetName(), strings.ToLower(ActiveGateComponent))
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "abcd123.some.activegate.endpointurl.com")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateChanged", statusCondition.Reason)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)

		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.Error(t, err)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateFailed", statusCondition.Reason)
	})
	t.Run("verify removal of conditions", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		r := NewReconciler(istioClient)
		rec := r.(*reconciler)
		rec.timeProvider.Freeze()

		err := r.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dk.GetName(), strings.ToLower(ActiveGateComponent))
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "abcd123.some.activegate.endpointurl.com")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateChanged", statusCondition.Reason)

		// disable endpoints, make request within api threshold
		dk.Status.ActiveGate.ConnectionInfo.Endpoints = ""

		err = r.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		statusCondition2 := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition2)

		// advance time to be outside api threshold
		rec2 := r.(*reconciler)
		time := rec2.timeProvider.Now().Add(dk.ApiRequestThreshold() * 2)
		rec2.timeProvider.Set(time)
		err = rec2.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		statusCondition3 := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.Nil(t, statusCondition3)
	})
	t.Run("verify removal of conditions when ActiveGate disabled", func(t *testing.T) {
		dk := createTestDynaKube()
		fakeClient := fakeistio.NewSimpleClientset()
		istioClient := newTestingClient(fakeClient, dk.GetNamespace())
		reconciler := NewReconciler(istioClient)

		err := reconciler.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		expectedFQDNName := BuildNameForFQDNServiceEntry(dk.GetName(), strings.ToLower(ActiveGateComponent))
		serviceEntry, err := fakeClient.NetworkingV1beta1().ServiceEntries(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)
		assert.Contains(t, fmt.Sprintf("%v", serviceEntry), "abcd123.some.activegate.endpointurl.com")

		virtualService, err := fakeClient.NetworkingV1beta1().VirtualServices(dk.GetNamespace()).Get(ctx, expectedFQDNName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, virtualService)

		require.NoError(t, err)
		assert.NotNil(t, serviceEntry)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateChanged", statusCondition.Reason)

		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{}
		err = reconciler.ReconcileActiveGateCommunicationHosts(ctx, dk)
		require.NoError(t, err)

		statusCondition2 := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.Nil(t, statusCondition2)
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

func createTestDynaKube() *dynakube.DynaKube {
	fqdnHost := createTestFQDNCommunicationHost()
	ipHost := createTestIPCommunicationHost()
	endpoints := "https://abcd123.some.activegate.endpointurl.com:443"

	return &dynakube.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind: "DynaKube",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "owner",
			Namespace: "test",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://test.dev.dynatracelabs.com/api",
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.RoutingCapability.DisplayName,
				},
			},
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
			DynatraceApiRequestThreshold: 15,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
				ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
					CommunicationHosts: []dynakube.CommunicationHostStatus{
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
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: "test-tenant",
					Endpoints:  endpoints,
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
