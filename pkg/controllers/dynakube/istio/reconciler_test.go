package istio

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestSplitCommunicationHost(t *testing.T) {
	t.Run("empty => no fail", func(t *testing.T) {
		ipHosts, fqdnHosts := splitCommunicationHost([]CommunicationHost{})
		require.Nil(t, ipHosts)
		require.Nil(t, fqdnHosts)
	})
	t.Run("nil => no fail", func(t *testing.T) {
		ipHosts, fqdnHosts := splitCommunicationHost(nil)
		require.Nil(t, ipHosts)
		require.Nil(t, fqdnHosts)
	})
	t.Run("success", func(t *testing.T) {
		comHosts := []CommunicationHost{
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
	component := "best-component"
	t.Run("empty communication host => delete if previously created", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		serviceEntry := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dk.Name, component),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClientWithIndex(serviceEntry)
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.reconcileIPServiceEntry(ctx, nil, dk, component)
		require.NoError(t, err)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceEntry), serviceEntry)
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)
		commHosts := []CommunicationHost{
			createTestIPCommunicationHost(),
		}

		err := reconciler.reconcileIPServiceEntry(ctx, commHosts, dk, component)
		require.NoError(t, err)

		expectedServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dk.Name, component),
				Namespace: dk.Namespace,
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedServiceEntryMeta), expectedServiceEntryMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedServiceEntryMeta)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := createFailK8sClient()

		reconciler := NewReconciler(fakeClient, fakeClient)
		commHosts := []CommunicationHost{
			createTestIPCommunicationHost(),
		}

		err := reconciler.reconcileIPServiceEntry(ctx, commHosts, dk, component)
		require.Error(t, err)
	})
}

func TestReconcileFQDNServiceEntry(t *testing.T) {
	component := "best-component"

	t.Run("empty communication host => delete if previously created", func(t *testing.T) {
		ctx := t.Context()
		owner := createTestDynaKube()
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
		fakeClient := fake.NewClientWithIndex(serviceEntry, virtualService)
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.reconcileFQDNServiceEntry(ctx, nil, owner, component)
		require.NoError(t, err)
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceEntry), serviceEntry)
		require.True(t, k8serrors.IsNotFound(err))
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(virtualService), virtualService)
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		owner := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)
		commHosts := []CommunicationHost{
			createTestFQDNCommunicationHost(),
		}

		err := reconciler.reconcileFQDNServiceEntry(ctx, commHosts, owner, component)
		require.NoError(t, err)

		expectedServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(owner.GetName(), component),
				Namespace: owner.GetNamespace(),
			},
		}
		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(owner.GetName(), component),
				Namespace: owner.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedServiceEntryMeta), expectedServiceEntryMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedServiceEntryMeta)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		ctx := t.Context()
		owner := createTestDynaKube()
		fakeClient := createFailK8sClient()

		reconciler := NewReconciler(fakeClient, fakeClient)
		commHosts := []CommunicationHost{
			createTestFQDNCommunicationHost(),
		}

		err := reconciler.reconcileFQDNServiceEntry(ctx, commHosts, owner, component)
		require.Error(t, err)
	})
}

func TestReconcileAPIUrl(t *testing.T) {
	t.Run("nil => error", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileAPIURL(ctx, nil)
		require.Error(t, err)
	})
	t.Run("malformed api-url => error", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		dk.Spec.APIURL = "something-random"
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileAPIURL(ctx, dk)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileAPIURL(ctx, dk)
		require.NoError(t, err)

		expectedServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), OperatorComponent),
				Namespace: dk.GetNamespace(),
			},
		}
		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), OperatorComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedServiceEntryMeta), expectedServiceEntryMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedServiceEntryMeta)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := createFailK8sClient()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileAPIURL(ctx, dk)
		require.Error(t, err)
	})
}

func TestReconcileOneAgentCommunicationHosts(t *testing.T) {
	t.Run("nil => error", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileCodeModules(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)

		expectedFQDNServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		expectedIPServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}
		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.NoError(t, err)
		assert.Contains(t, fmt.Sprintf("%v", expectedFQDNServiceEntryMeta), "something.test.io")

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedIPServiceEntryMeta), expectedIPServiceEntryMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedIPServiceEntryMeta)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForOneAgent")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForOneAgentChanged", statusCondition.Reason)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := createFailK8sClient()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileCodeModules(ctx, dk)
		require.Error(t, err)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForOneAgent")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForOneAgentFailed", statusCondition.Reason)
	})
	t.Run("remove and cleanup if AppInjection is disabled", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)

		expectedFQDNServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		expectedIPServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForIPServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}
		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), CodeModuleComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.NoError(t, err)
		assert.Contains(t, fmt.Sprintf("%v", expectedFQDNServiceEntryMeta), "something.test.io")

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedIPServiceEntryMeta), expectedIPServiceEntryMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedIPServiceEntryMeta)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForOneAgent")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForOneAgentChanged", statusCondition.Reason)

		dk.Spec.OneAgent.CloudNativeFullStack = nil
		dk.Spec.OneAgent.HostMonitoring = &oneagent.HostInjectSpec{}

		err = reconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)

		statusCondition = meta.FindStatusCondition(*dk.Conditions(), "IstioForOneAgent")
		require.Nil(t, statusCondition)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.Error(t, err)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.Error(t, err)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedIPServiceEntryMeta), expectedIPServiceEntryMeta)
		require.Error(t, err)
	})
}

func TestReconcileActiveGateCommunicationHosts(t *testing.T) {
	t.Run("nil => error", func(t *testing.T) {
		ctx := t.Context()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileActiveGate(ctx, nil)
		require.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileActiveGate(ctx, dk)
		require.NoError(t, err)

		expectedFQDNServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), ActiveGateComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), ActiveGateComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.NoError(t, err)
		assert.Contains(t, fmt.Sprintf("%v", expectedFQDNServiceEntryMeta), "abcd123.some.activegate.endpointurl.com")

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateChanged", statusCondition.Reason)
	})
	t.Run("unknown k8s client error => error", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := createFailK8sClient()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileActiveGate(ctx, dk)
		require.Error(t, err)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateFailed", statusCondition.Reason)
	})
	t.Run("remove and cleanup if activeGate is disabled", func(t *testing.T) {
		ctx := t.Context()
		dk := createTestDynaKube()
		fakeClient := fake.NewClientWithIndex()
		reconciler := NewReconciler(fakeClient, fakeClient)

		err := reconciler.ReconcileActiveGate(ctx, dk)
		require.NoError(t, err)

		expectedFQDNServiceEntryMeta := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), ActiveGateComponent),
				Namespace: dk.GetNamespace(),
			},
		}
		expectedVirtualServiceMeta := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BuildNameForFQDNServiceEntry(dk.GetName(), ActiveGateComponent),
				Namespace: dk.GetNamespace(),
			},
		}

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.NoError(t, err)
		assert.Contains(t, fmt.Sprintf("%v", expectedFQDNServiceEntryMeta), "abcd123.some.activegate.endpointurl.com")

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.NoError(t, err)
		assert.NotNil(t, expectedVirtualServiceMeta)

		statusCondition := meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.NotNil(t, statusCondition)
		require.Equal(t, "IstioForActiveGateChanged", statusCondition.Reason)

		dk.Spec.ActiveGate = activegate.Spec{}

		err = reconciler.ReconcileActiveGate(ctx, dk)
		require.NoError(t, err)

		statusCondition = meta.FindStatusCondition(*dk.Conditions(), "IstioForActiveGate")
		require.Nil(t, statusCondition)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedFQDNServiceEntryMeta), expectedFQDNServiceEntryMeta)
		require.Error(t, err)

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedVirtualServiceMeta), expectedVirtualServiceMeta)
		require.Error(t, err)
	})
}

func createTestIPCommunicationHost() CommunicationHost {
	return CommunicationHost{
		Protocol: "http",
		Host:     "42.42.42.42",
		Port:     620,
	}
}

func createTestFQDNCommunicationHost() CommunicationHost {
	return CommunicationHost{
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
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
			DynatraceAPIRequestThreshold: ptr.To(uint16(15)),
			EnableIstio:                  true,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				ConnectionInfo: communication.ConnectionInfo{
					Endpoints: fqdnHost.String() + "," + ipHost.String(),
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

func createFailK8sClient() client.Client {
	boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}

func TestIsInstalled(t *testing.T) {
	createErrorClient := func(istioMissing bool) client.Client {
		errClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				if istioMissing {
					return new(meta.NoResourceMatchError)
				} else {
					return errors.New("BOOM")
				}
			},
		})

		return errClient
	}

	t.Run("istio is installed => returns true", func(t *testing.T) {
		fakeClient := fake.NewClientWithIndex()

		installed := IsInstalled(t.Context(), fakeClient)
		assert.True(t, installed)
	})

	t.Run("istio is not installed => returns false", func(t *testing.T) {
		fakeClient := createErrorClient(true)

		installed := IsInstalled(t.Context(), fakeClient)
		assert.False(t, installed)
	})

	t.Run("unknown client err => returns true (no discovery fail == istio is present)", func(t *testing.T) {
		fakeClient := createErrorClient(false)

		installed := IsInstalled(t.Context(), fakeClient)
		assert.True(t, installed)
	})
}
