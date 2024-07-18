package extension

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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
	t.Run(`Extension secret not generated when Prometheus is disabled`, func(t *testing.T) {
		instance := makeTestDynakube(false)

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
	t.Run(`Extension secret gets deleted when Prometheus is disabled`, func(t *testing.T) {
		instance := makeTestDynakube(false)

		// mock SecretCreated condition
		conditions.SetSecretCreated(instance.Conditions(), secretConditionType, instance.Name+secretSuffix)

		// mock secret
		secretToken, _ := dttoken.New(eecTokenSecretValuePrefix)
		secretData := map[string][]byte{
			eecTokenSecretKey: []byte(secretToken.String()),
		}
		secretMock, _ := k8ssecret.Create(instance, k8ssecret.NewNameModifier(testName+"-extensions-token"), k8ssecret.NewNamespaceModifier(instance.GetNamespace()), k8ssecret.NewDataModifier(secretData))

		fakeClient := fake.NewClient()
		fakeClient.Create(context.Background(), secretMock)
		r := NewReconciler(fakeClient, fakeClient, instance)

		// assert extensions token is there before reconciliation
		var secretFound corev1.Secret
		err := fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.False(t, k8serrors.IsNotFound(err))

		// assert conditions are not empty
		require.NotEmpty(t, instance.Conditions())

		// reconcile
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is deleted after reconciliation
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.True(t, k8serrors.IsNotFound(err))

		// assert conditions are empty
		require.Empty(t, instance.Conditions())
	})
	t.Run(`Extension secret is generated when Prometheus is enabled`, func(t *testing.T) {
		instance := makeTestDynakube(true)

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

		conditions.SetSecretCreated(&expectedConditions, secretConditionType, instance.Name+secretSuffix)
		testutil.PartialEqual(t, &expectedConditions, instance.Conditions(), cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"))
	})
	t.Run(`Extension SecretCreated failure condition is set when error`, func(t *testing.T) {
		instance := makeTestDynakube(true)

		misconfiguredReader, _ := client.New(&rest.Config{}, client.Options{})
		r := NewReconciler(fake.NewClient(), misconfiguredReader, instance)
		err := r.Reconcile(context.Background())
		require.Error(t, err)

		// assert extensions token condition is added
		require.NotEmpty(t, instance.Conditions())

		var expectedConditions []metav1.Condition

		conditions.SetSecretCreatedFailed(&expectedConditions, secretConditionType, fmt.Sprintf(secretCreatedMessageFailure, err))
		testutil.PartialEqual(t, &expectedConditions, instance.Conditions(), cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"))
	})
}

func makeTestDynakube(prometheusEnabled bool) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: dynakube.ExtensionsSpec{
				Prometheus: dynakube.PrometheusSpec{
					Enabled: prometheusEnabled,
				},
			},
		},
	}
}
