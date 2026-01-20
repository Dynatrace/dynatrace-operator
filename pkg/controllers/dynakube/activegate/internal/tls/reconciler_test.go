package tls

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testNamespace    = "test-namespace"
	testDynakubeName = "test-dynakube"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("ActiveGate disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		_, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetTLSSecretName(),
		})

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("custom ActiveGate TLS secret exists", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
					TLSSecretName: "test",
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		_, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetTLSSecretName(),
		})

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("automatic-tls-certificate feature disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
				Annotations: map[string]string{
					exp.AGAutomaticTLSCertificateKey: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		_, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetTLSSecretName(),
		})

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("secret created", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		agTLSSecret, err := r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetTLSSecretName(),
		})

		require.NoError(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, conditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, fmt.Sprintf("%s created", agTLSSecret.Name), condition.Message)
	})

	t.Run("secret deleted", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		_, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAutoTLSSecretName(),
		})
		require.NoError(t, err)

		dk.Annotations = make(map[string]string)
		dk.Annotations[exp.AGAutomaticTLSCertificateKey] = "false"

		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		_, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAutoTLSSecretName(),
		})
		require.Error(t, err)

		assert.True(t, k8serrors.IsNotFound(err))
	})
}
