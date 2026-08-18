// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

func newTestDTP(name, namespace string) *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("dtp-uid")}}
}

// assertGolden compares obj, marshaled to YAML, against testdata/name. The fake
// client is the only source of non-determinism (it stamps resourceVersion), so
// that field is cleared before comparing.
func assertGolden(t *testing.T, name string, obj client.Object) {
	t.Helper()

	obj.SetResourceVersion("")
	// Ensure object has GVK so that TypeMeta is included
	kinds, _, _ := scheme.Scheme.ObjectKinds(obj)
	require.Len(t, kinds, 1)
	obj.GetObjectKind().SetGroupVersionKind(kinds[0])

	got, err := yaml.Marshal(obj)
	require.NoError(t, err)

	want, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	assert.YAMLEq(t, string(want), string(got))
}

func newTestScope(dtp *dtprometheus.DTPrometheus) *reconcileScope {
	return &reconcileScope{
		Owner:     dtp,
		Spec:      dtp.TargetAllocator(),
		AppLabels: k8slabel.OTelTargetAllocator(),
	}
}

func TestReconcileCondition(t *testing.T) {
	boom := fmt.Errorf("wrap: %w", errors.New("boom"))

	completeDeployment := &appsv1.Deployment{
		Spec:   appsv1.DeploymentSpec{Replicas: new(int32(2))},
		Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
	}

	tests := []struct {
		name        string
		err         error
		deployment  *appsv1.Deployment
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{"deployment not rolled out -> reconciling", nil, nil, metav1.ConditionFalse, status.ReasonReconciling, "target allocator is pending"},
		{"rollout complete -> available", nil, completeDeployment, metav1.ConditionTrue, status.ReasonAvailable, "target allocator is ready"},
		{"error -> error", boom, nil, metav1.ConditionFalse, status.ReasonError, "boom"},
		{"error takes precedence over complete rollout", boom, completeDeployment, metav1.ConditionFalse, status.ReasonError, "boom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dtp := newTestDTP("dtp", "dynatrace")
			s := &reconcileScope{Owner: dtp, Deployment: test.deployment}
			r := &Reconciler{}

			r.reconcileCondition(s, test.err)

			condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.TargetAllocatorAvailable)
			require.NotNil(t, condition)
			assert.Equal(t, test.wantStatus, condition.Status)
			assert.Equal(t, test.wantReason, condition.Reason)
			assert.Equal(t, test.wantMessage, condition.Message)
		})
	}
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

		assertGolden(t, "configmap.yaml", cm)

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
	t.Run("missing image", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		err := r.reconcileDeployment(t.Context(), s)

		require.Error(t, err)
		assert.Nil(t, s.Deployment)

		getErr := c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, &appsv1.Deployment{})
		assert.True(t, k8serrors.IsNotFound(getErr))
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

		assertGolden(t, "deployment.yaml", deploy)
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

		assertGolden(t, "service.yaml", svc)
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
