package tls

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace    = "test-namespace"
	testDynakubeName = "test-dynakube"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`ActiveGate disabled`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testDynakubeName,
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		agTLSSecret := corev1.Secret{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: r.dk.Namespace}, &agTLSSecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run(`custom ActiveGate TLS secret exists`, func(t *testing.T) {
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
					TlsSecretName: "test",
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		agTLSSecret := corev1.Secret{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: r.dk.Namespace}, &agTLSSecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run(`automatic-tls-certificate feature disabled`, func(t *testing.T) {
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
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		agTLSSecret := corev1.Secret{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: r.dk.Namespace}, &agTLSSecret)

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run(`secret created`, func(t *testing.T) {
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
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		agTLSSecret := corev1.Secret{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: r.dk.Namespace}, &agTLSSecret)

		require.NoError(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, conditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, fmt.Sprintf("%s created", agTLSSecret.Name), condition.Message)
	})
}
