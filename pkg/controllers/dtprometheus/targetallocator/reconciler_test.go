// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newTestDTP(name, namespace string) *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("dtp-uid")},
		Spec: dtprometheus.DTPrometheusSpec{
			TargetAllocator: dtprometheus.TargetAllocatorSpec{
				PodSpec: dtprometheus.PodSpec{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("250Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("125Mi"),
						},
					},
				},
			},
		},
	}
}

func newTestScope(dtp *dtprometheus.DTPrometheus) *reconcileScope {
	return &reconcileScope{
		Owner:     dtp,
		DynaKube:  &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: "dynatrace"}},
		Spec:      dtp.TargetAllocator(),
		AppLabels: k8slabel.OTelTargetAllocator(),
	}
}

// The condition branch logic is covered by the condition package. What matters here is
// that the target allocator wires itself to the right condition type, component name and
// rollout check.
func TestReconcileCondition(t *testing.T) {
	t.Run("freshly created deployment with no ready replicas -> pending", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.Image = "registry.example.com/target-allocator:1.2.3"
		dtp.Spec.TargetAllocator.Replicas = new(int32(2))

		r := &Reconciler{Client: fake.NewClient()}
		require.NoError(t, r.Reconcile(t.Context(), dtp, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.TargetAllocatorAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonReconciling, condition.Reason)
		assert.Equal(t, "target allocator is pending", condition.Message)
	})

	t.Run("reconcile error -> error, with unwrapped message", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.Image = "registry.example.com/target-allocator:1.2.3"

		boom := errors.New("boom")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
				return boom
			},
		})

		r := &Reconciler{Client: clt}
		require.Error(t, r.Reconcile(t.Context(), dtp, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.TargetAllocatorAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonError, condition.Reason)
		assert.Equal(t, boom.Error(), condition.Message)
	})
}

func TestReconcileConfigMap(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.ScrapeInterval = metav1.Duration{Duration: 5 * time.Minute}
		dtp.Spec.TargetAllocator.ScrapeCRNamespaceSelector = &metav1.LabelSelector{MatchLabels: map[string]string{"bar": "foo"}}
		dtp.Spec.TargetAllocator.ScrapeCRSelector = &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}
		s := newTestScope(dtp)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, cm))

		helpers.AssertGolden(t, "testdata/configmap.yaml", cm)

		sum := sha256.Sum256([]byte(cm.Data[configFile]))
		assert.Equal(t, hex.EncodeToString(sum[:]), s.ConfigMapHash)
	})

	t.Run("merge labels", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      s.Spec.GetDeploymentName(),
			Namespace: dtp.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"},
		}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, cm))
		assert.Equal(t, "value", cm.Labels["custom"])
		assert.Equal(t, k8slabel.OTelTargetAllocator().Instance, cm.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileConfigMap(t.Context(), newTestScope(newTestDTP("dtp", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconcileDeployment(t *testing.T) {
	t.Run("fleet resolve fails when no imageRef set", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		imageClient := imagemock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no image found"))
		s.ImageClient = imageClient
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		err := r.reconcileDeployment(t.Context(), s)

		require.Error(t, err)
		assert.Nil(t, s.Deployment)

		getErr := c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, &appsv1.Deployment{})
		assert.True(t, k8serrors.IsNotFound(getErr))
	})

	t.Run("resolves image from fleet API when no imageRef set", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		imageClient := imagemock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, image.TargetAllocator, "").Return(&image.Info{URI: "registry.example.com/fleet-ta:latest"}, nil)
		s.ImageClient = imageClient
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileDeployment(t.Context(), s))
		require.NotNil(t, s.Deployment)

		deploy := &appsv1.Deployment{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, deploy))
		assert.Equal(t, "registry.example.com/fleet-ta:latest", deploy.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, "registry.example.com/fleet-ta:latest", dtp.Status.TargetAllocator.ResolvedImage)
	})

	t.Run("resolves image from fleet API with publicRegistryOverride", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.PublicRegistryOverride = "custom.registry.example.com"
		s := newTestScope(dtp)
		imageClient := imagemock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, image.TargetAllocator, "custom.registry.example.com").Return(&image.Info{URI: "custom.registry.example.com/fleet-ta:latest"}, nil)
		s.ImageClient = imageClient
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileDeployment(t.Context(), s))
		require.NotNil(t, s.Deployment)

		deploy := &appsv1.Deployment{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, deploy))
		assert.Equal(t, "custom.registry.example.com/fleet-ta:latest", deploy.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, "custom.registry.example.com/fleet-ta:latest", dtp.Status.TargetAllocator.ResolvedImage)
	})

	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.Image = "registry.example.com/target-allocator:1.2.3"
		dtp.Spec.TargetAllocator.ImagePullPolicy = corev1.PullAlways
		dtp.Spec.TargetAllocator.Replicas = new(int32(3))
		dtp.Spec.TargetAllocator.NodeSelector = map[string]string{"disk": "ssd"}
		dtp.Spec.TargetAllocator.PriorityClassName = "high-priority"
		dtp.Spec.TargetAllocator.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		dtp.Spec.TargetAllocator.Annotations = map[string]string{"custom": "annotation"}
		dtp.Spec.TargetAllocator.Labels = map[string]string{"custom": "label"}
		dtp.Spec.TargetAllocator.Args = []string{"--foo=bar"}
		s := newTestScope(dtp)
		s.ConfigMapHash = "deadbeef"
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileDeployment(t.Context(), s))
		require.NotNil(t, s.Deployment)

		deploy := &appsv1.Deployment{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, deploy))

		helpers.AssertGolden(t, "testdata/deployment.yaml", deploy)
	})

	t.Run("preserve existing replicas", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.Image = "img:1"
		s := newTestScope(dtp)
		existing := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace},
			Spec:       appsv1.DeploymentSpec{Replicas: new(int32(5))},
		}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileDeployment(t.Context(), s))

		deploy := &appsv1.Deployment{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, deploy))
		assert.Equal(t, new(int32(5)), deploy.Spec.Replicas)
	})

	t.Run("propagate error", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.TargetAllocator.Image = "img:1"
		expectErr := errors.New("boom")
		r := &Reconciler{Client: createErrorClient(expectErr)}

		err := r.reconcileDeployment(t.Context(), newTestScope(dtp))

		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconcileService(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, svc))

		helpers.AssertGolden(t, "testdata/service.yaml", svc)
	})

	t.Run("merge labels", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		existing := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
			Name:      s.Spec.GetDeploymentName(),
			Namespace: dtp.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"},
		}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, svc))
		assert.Equal(t, "value", svc.Labels["custom"])
		assert.Equal(t, k8slabel.OTelTargetAllocator().Instance, svc.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileService(t.Context(), newTestScope(newTestDTP("dtp", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func createErrorClient(createErr error) client.Client {
	return fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, clt client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return createErr
		},
	})
}
