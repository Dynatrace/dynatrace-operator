package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	activegatev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/activegate"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconciler_GetActiveGate(t *testing.T) {
	t.Run("ActiveGate should be created", func(t *testing.T) {
		dynakube := createBaseDynakube()
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
			dynatracev1beta1.MetricsIngestCapability.DisplayName,
		}
		fakeClient := fake.NewClientWithIndex(dynakube)
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube)

		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)

		var activeGate activegatev1alpha1.ActiveGate
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-activegate", Namespace: "dynatrace"}, &activeGate)
		require.NoError(t, err)
		assert.Contains(t, activeGate.Spec.Capabilities, activegatev1alpha1.RoutingCapability.DisplayName)
		assert.Contains(t, activeGate.Spec.Capabilities, activegatev1alpha1.MetricsIngestCapability.DisplayName)
	})
	t.Run("ActiveGate should be deleted", func(t *testing.T) {
		baseDynakube := createBaseDynakube()
		baseActiveGate := createBaseActiveGate()
		fakeClient := fake.NewClientWithIndex(baseDynakube, baseActiveGate)
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, baseDynakube)

		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)

		var activeGate activegatev1alpha1.ActiveGate
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-activegate", Namespace: "dynatrace"}, &activeGate)
		assert.Truef(t, k8serrors.IsNotFound(err), "ActiveGate should be deleted")
	})
	t.Run("ActiveGate should be updated", func(t *testing.T) {
		dynakube := createBaseDynakube()
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
		}
		activeGate := createBaseActiveGate()
		fakeClient := fake.NewClientWithIndex(dynakube, activeGate)
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube)

		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)

		var updatedActiveGate activegatev1alpha1.ActiveGate
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-activegate", Namespace: "dynatrace"}, &updatedActiveGate)
		require.NoError(t, err)
		assert.Contains(t, updatedActiveGate.Spec.Capabilities, activegatev1alpha1.RoutingCapability.DisplayName)
		assert.Len(t, updatedActiveGate.Spec.Capabilities, 1)
	})
	t.Run("ActiveGate should be updated if feature flag changes", func(t *testing.T) {
		dynakube := createBaseDynakube()
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
		}
		dynakube.Annotations = make(map[string]string)
		dynakube.Annotations[activegatev1alpha1.AnnotationFeatureActiveGateUpdates] = "true"
		activeGate := createBaseActiveGate()
		activeGate.Spec.SpecificSpec.Capabilities = []activegatev1alpha1.CapabilityDisplayName{
			activegatev1alpha1.RoutingCapability.DisplayName,
		}
		fakeClient := fake.NewClientWithIndex(dynakube, activeGate)
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube)

		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)

		var updatedActiveGate activegatev1alpha1.ActiveGate
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-activegate", Namespace: "dynatrace"}, &updatedActiveGate)
		require.NoError(t, err)

		annotationValue, ok := updatedActiveGate.Annotations[activegatev1alpha1.AnnotationFeatureActiveGateUpdates]
		assert.True(t, ok)
		assert.Equal(t, "true", annotationValue)
	})
	t.Run("ActiveGate should not be updated", func(t *testing.T) {
		dynakube := createBaseDynakube()
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
		}
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("BOOM")
			},
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				ag := obj.(*activegatev1alpha1.ActiveGate)
				ag.Spec.SpecificSpec.Capabilities = []activegatev1alpha1.CapabilityDisplayName{
					activegatev1alpha1.RoutingCapability.DisplayName,
				}
				ag.Spec.APIURL = "https://test123.dev.dynatracelabs.com/api"

				return nil
			},
		})
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube)

		err := reconciler.Reconcile(context.Background())
		assert.NoError(t, err)
	})
	t.Run("ActiveGate should not be created", func(t *testing.T) {
		dynakube := createBaseDynakube()
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				return errors.New("BOOM")
			},
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("BOOM")
			},
		})
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube)

		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)

		var activeGate activegatev1alpha1.ActiveGate
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-activegate", Namespace: "dynatrace"}, &activeGate)
		assert.Truef(t, k8serrors.IsNotFound(err), "ActiveGate should not be created")
	})
}

func createBaseDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "dynatrace",
			UID:       "test-uid",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{APIURL: "https://test123.dev.dynatracelabs.com/api"},
	}
}

func createBaseActiveGate() *activegatev1alpha1.ActiveGate {
	tr := true

	return &activegatev1alpha1.ActiveGate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-activegate",
			Namespace: "dynatrace",
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:        "test-uid",
					Controller: &tr,
				},
			},
		},
	}
}
