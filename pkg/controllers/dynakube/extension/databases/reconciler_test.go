package databases

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testDynakubeName            = "dynakube"
	testNamespaceName           = "dynatrace"
	testPullSecret              = "pull-secret"
	testExecutorImageRepository = "repo/dynatrace-executor"
	testExecutorImageTag        = "1.123.0"
)

func TestReconcileErrors(t *testing.T) {
	t.Run("failed delete", func(t *testing.T) {
		dk := getTestDynakube()

		builder := fake.NewClientBuilder().
			WithObjects(&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDynakubeName + "-database-datasource-foo",
					Namespace: testNamespaceName,
					Labels: func() map[string]string {
						labels, _, _ := buildAllLabels(dk, extensions.DatabaseSpec{})

						return labels
					}(),
				},
			}).
			WithInterceptorFuncs(interceptor.Funcs{
				Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
					return k8serrors.NewInternalError(errors.New("bad"))
				},
			})

		requireReconcileFails(t, dk, builder)
	})

	t.Run("failed create", func(t *testing.T) {
		dk := getTestDynakube()

		builder := fake.NewClientBuilder().
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					return k8serrors.NewInternalError(errors.New("bad"))
				},
			})

		requireReconcileFails(t, dk, builder)
	})

	t.Run("failed replica lookup", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].Replicas = nil

		builder := fake.NewClientBuilder().
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return k8serrors.NewInternalError(errors.New("bad"))
				},
			})

		requireReconcileFails(t, dk, builder)
	})
}

func requireReconcileFails(t *testing.T, dk *dynakube.DynaKube, builder *fake.ClientBuilder) {
	t.Helper()

	mockK8sClient := builder.
		WithScheme(scheme.Scheme).
		WithObjects(dk).
		WithStatusSubresource(dk).
		Build()
	reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)

	err := reconciler.Reconcile(t.Context())
	require.Error(t, err)
	require.True(t, k8serrors.IsInternalError(err))
	require.True(t, meta.IsStatusConditionFalse(dk.Status.Conditions, conditionType), meta.FindStatusCondition(dk.Status.Conditions, conditionType))
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{
				Databases: []extensions.DatabaseSpec{
					{
						ID:                 "test",
						Replicas:           ptr.To(int32(1)),
						ServiceAccountName: "test",
						Labels:             map[string]string{"foo": "bar"},
						Annotations:        map[string]string{"foo": "bar"},
					},
				},
			},
			Templates: dynakube.TemplatesSpec{
				DatabaseExecutor: extensions.DatabaseExecutorSpec{
					ImageRef: image.Ref{
						Repository: testExecutorImageRepository,
						Tag:        testExecutorImageTag,
					},
				},
			},
			CustomPullSecret: testPullSecret,
		},
	}
}
