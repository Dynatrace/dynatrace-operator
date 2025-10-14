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
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
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
			WithObjects(getMatchingDeployment(dk)).
			WithInterceptorFuncs(interceptor.Funcs{
				Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
					return k8serrors.NewInternalError(errors.New("bad"))
				},
			})

		// change ID to trigger deletion
		dk.Spec.Extensions.Databases[0].ID = "foo"

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
			Name:        testDynakubeName + "-" + rand.String(6),
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

func getMatchingDeployment(dk *dynakube.DynaKube) *appsv1.Deployment {
	db := dk.Spec.Extensions.Databases[0]

	labels, matchLabels, templateLabels := buildAllLabels(dk, db)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.Name + "-database-datasource-" + db.ID,
			Namespace: testNamespaceName,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: db.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: templateLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						buildContainer(dk, db),
					},
					Volumes: buildVolumes(dk, db),
				},
			},
		},
	}
}
