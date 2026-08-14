// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: "dynatrace"}}
	dk.Status.KubernetesClusterName = "test-cluster"
	dk.Status.KubernetesClusterMEID = "KUBERNETES_CLUSTER-ABC123"

	return dk
}

func TestReconcileStatefulSet(t *testing.T) {
	t.Run("missing image", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScopeWithDynaKube(dtp, newTestDynaKube())
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		err := r.reconcileStatefulset(t.Context(), s)

		require.Error(t, err)
		assert.Nil(t, s.StatefulSet)

		getErr := c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, &appsv1.StatefulSet{})
		assert.True(t, k8serrors.IsNotFound(getErr))
	})

	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Gateway.Image = "registry.example.com/gateway:1.2.3"
		dtp.Spec.Gateway.ImagePullPolicy = corev1.PullAlways
		dtp.Spec.Gateway.Replicas = new(int32(3))
		dtp.Spec.Gateway.NodeSelector = map[string]string{"disk": "ssd"}
		dtp.Spec.Gateway.PriorityClassName = "high-priority"
		dtp.Spec.Gateway.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		dtp.Spec.Gateway.Annotations = map[string]string{"custom": "annotation"}
		dtp.Spec.Gateway.Labels = map[string]string{"custom": "label"}
		dtp.Spec.Gateway.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("500Mi")},
		}
		s := newTestScopeWithDynaKube(dtp, newTestDynaKube())
		s.ConfigMapHash = "deadbeef"
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileStatefulset(t.Context(), s))
		require.NotNil(t, s.StatefulSet)

		sts := &appsv1.StatefulSet{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, sts))

		assertGolden(t, "statefulset.yaml", sts)
	})

	t.Run("preserve existing replicas", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Gateway.Image = "img:1"
		s := newTestScopeWithDynaKube(dtp, newTestDynaKube())
		existing := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace},
			Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(5))},
		}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileStatefulset(t.Context(), s))

		sts := &appsv1.StatefulSet{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: dtp.Namespace}, sts))
		assert.Equal(t, new(int32(5)), sts.Spec.Replicas)
	})

	t.Run("propagate error", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Gateway.Image = "img:1"
		expectErr := errors.New("boom")
		r := &Reconciler{Client: createErrorClient(expectErr)}

		err := r.reconcileStatefulset(t.Context(), newTestScopeWithDynaKube(dtp, newTestDynaKube()))

		require.ErrorIs(t, err, expectErr)
	})
}
