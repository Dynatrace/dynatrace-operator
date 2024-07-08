package extension

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Prometheus disabled`, func(t *testing.T) {
		instance := &dynatracev1beta3.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, instance)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`Prometheus enabled`, func(t *testing.T) {
		instance := &dynatracev1beta3.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta3.DynaKubeSpec{
				Extensions: dynatracev1beta3.ExtensionsSpec{
					Prometheus: dynatracev1beta3.PrometheusSpec{
						Enabled: true,
					},
				},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, instance)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.NoError(t, err)
	})
}
