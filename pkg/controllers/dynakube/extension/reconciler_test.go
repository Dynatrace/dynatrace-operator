package extension

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	testutil "github.com/Dynatrace/dynatrace-operator/pkg/util/testing"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Prometheus disabled`, func(t *testing.T) {
		instance := makeTestDynakube()
		instance.Spec.Extensions.Prometheus.Enabled = false

		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, instance)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is not generated
		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.True(t, k8serrors.IsNotFound(err))

		// assert conditions are empty
		require.Empty(t, instance.Conditions())
	})
	t.Run(`Prometheus enabled`, func(t *testing.T) {
		instance := makeTestDynakube()

		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, instance)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is generated
		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.NoError(t, err)

		// assert extensions token condition is added
		require.NotEmpty(t, instance.Conditions())
		var expectedConditions []metav1.Condition
		setSecretCreatedSuccessCondition(&expectedConditions)
		testutil.PartialEqual(t, &expectedConditions, instance.Conditions(), cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"))
	})

	t.Run(`Secret creation failed`, func(t *testing.T) {
		instance := makeTestDynakube()

		misconfiguredReader, _ := client.New(&rest.Config{}, client.Options{})
		r := NewReconciler(fake.NewClient(), misconfiguredReader, instance)
		err := r.Reconcile(context.Background())
		require.Error(t, err)

		// assert extensions token condition is added
		require.NotEmpty(t, instance.Conditions())
		var expectedConditions []metav1.Condition
		setSecretCreatedFailureCondition(&expectedConditions, err)
		testutil.PartialEqual(t, &expectedConditions, instance.Conditions(), cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"))
	})
}

func makeTestDynakube() *dynatracev1beta3.DynaKube {
	return &dynatracev1beta3.DynaKube{
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
}
