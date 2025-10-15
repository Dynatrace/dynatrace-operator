package databases

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
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

		// Change ID to trigger deletion
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

func TestReconcileSpec(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		dk := getTestDynakube()
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		db := dk.Extensions().Databases[0]
		assert.Equal(t, db.Replicas, deploy.Spec.Replicas)
		assert.Equal(t, defaultServiceAccount, deploy.Spec.Template.Spec.ServiceAccountName)
		assert.Contains(t, deploy.Spec.Template.Labels, executorLabelKey)
		assert.Contains(t, deploy.Spec.Template.Labels, datasourceLabelKey)
		assert.NotNil(t, deploy.Spec.Template.Spec.SecurityContext)
		assert.Len(t, deploy.Spec.Template.Spec.Volumes, 3)

		container := deploy.Spec.Template.Spec.Containers[0]
		assert.NotNil(t, container.LivenessProbe)
		assert.NotNil(t, container.ReadinessProbe)
		assert.NotNil(t, container.SecurityContext)
		assert.Equal(t, dk.Spec.Templates.DatabaseExecutor.ImageRef.String(), container.Image)
		assert.Equal(t, corev1.PullIfNotPresent, container.ImagePullPolicy)
		assert.Len(t, container.Args, 3)
		assert.Len(t, container.Env, 1)
		assert.Len(t, container.VolumeMounts, 3)
		assert.NotEmpty(t, container.Resources.Requests)
		assert.NotEmpty(t, container.Resources.Limits)
	})

	t.Run("override image pull policy", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.DatabaseExecutor.ImageRef.Tag = "latest"
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Equal(t, corev1.PullAlways, deploy.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	})

	t.Run("override labels", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].Labels = map[string]string{"foo": "bar"}
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Subset(t, deploy.Spec.Template.Labels, dk.Spec.Extensions.Databases[0].Labels)
	})

	t.Run("override annotations", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].Annotations = map[string]string{"foo": "bar"}
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Subset(t, deploy.Spec.Template.Annotations, dk.Spec.Extensions.Databases[0].Annotations)
	})

	t.Run("override service account", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].ServiceAccountName = "custom"
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Equal(t, "custom", deploy.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("override resources", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].Resources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1256Mi"),
				corev1.ResourceCPU:    resource.MustParse("1250m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1500m"),
			},
		}
		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Equal(t, *dk.Spec.Extensions.Databases[0].Resources, deploy.Spec.Template.Spec.Containers[0].Resources)
	})

	t.Run("extra volumes", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Databases[0].Volumes = []corev1.Volume{
			{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
		dk.Spec.Extensions.Databases[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "test", MountPath: "/tmp"},
		}

		deploy := getReconciledDeployment(t, fakeClient(), dk)
		assert.Len(t, deploy.Spec.Template.Spec.Volumes, 4)
		assert.Len(t, deploy.Spec.Template.Spec.Containers[0].VolumeMounts, 4)
	})
}

func fakeClient() client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()
}

func requireReconcileFails(t *testing.T, dk *dynakube.DynaKube, builder *fake.ClientBuilder) {
	t.Helper()

	mockK8sClient := builder.
		WithScheme(scheme.Scheme).
		WithObjects(dk).
		Build()
	reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)

	err := reconciler.Reconcile(t.Context())
	require.Error(t, err)
	require.True(t, k8serrors.IsInternalError(err))
	require.True(t, meta.IsStatusConditionFalse(dk.Status.Conditions, conditionType), meta.FindStatusCondition(dk.Status.Conditions, conditionType))
}

func getReconciledDeployment(t *testing.T, clt client.Client, dk *dynakube.DynaKube) *appsv1.Deployment {
	t.Helper()
	require.NoError(t, NewReconciler(clt, clt, dk).Reconcile(t.Context()))
	deployments := &appsv1.DeploymentList{}
	require.NoError(t, clt.List(t.Context(), deployments))
	if len(deployments.Items) == 0 {
		return nil
	}
	require.Len(t, deployments.Items, 1)
	deployment := &deployments.Items[0]
	require.NoError(t, clt.Delete(t.Context(), deployment.DeepCopy()))

	return deployment
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
						ID:       "test",
						Replicas: ptr.To(int32(1)),
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
