// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	apitypes "github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newTestPM(name, namespace string) *prometheusmonitoring.PrometheusMonitoring {
	return &prometheusmonitoring.PrometheusMonitoring{Name: name, Namespace: namespace, UID: types.UID("pm-uid")}
}

func newTestScope(pm *prometheusmonitoring.PrometheusMonitoring) *reconcileScope {
	return newTestScopeWithDynaKube(pm, &dynakube.DynaKube{})
}

func newTestScopeWithDynaKube(pm *prometheusmonitoring.PrometheusMonitoring, dk *dynakube.DynaKube) *reconcileScope {
	return &reconcileScope{
		Owner:     pm,
		DynaKube:  dk,
		Spec:      pm.Gateway(),
		AppLabels: k8slabel.New(k8slabel.OTelGatewayAppName, pm.Name, ""),
	}
}

// The condition branch logic is covered by the condition package. What matters here is
// that the gateway wires itself to the right condition type, component name and rollout check.
func TestReconcileCondition(t *testing.T) {
	t.Run("freshly created statefulset with no ready replicas -> pending", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		pm.Spec.Gateway.Image = "registry.example.com/gateway:1.2.3"
		pm.Spec.Gateway.Replicas = new(int32(2))

		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"

		r := &Reconciler{Client: fake.NewClient()}
		require.NoError(t, r.Reconcile(t.Context(), pm, dk, nil))

		condition := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.GatewayAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonReconciling, condition.Reason)
		assert.Equal(t, "gateway is pending", condition.Message)
	})

	t.Run("reconcile error -> error, with unwrapped message", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		pm.Spec.Gateway.Image = "registry.example.com/gateway:1.2.3"

		boom := errors.New("boom")
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
				return boom
			},
		})

		r := &Reconciler{Client: clt}
		require.Error(t, r.Reconcile(t.Context(), pm, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(pm.Status.Conditions, prometheusmonitoring.GatewayAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonError, condition.Reason)
		assert.Equal(t, boom.Error(), condition.Message)
	})
}

func TestBuildGatewayConfigData(t *testing.T) {
	t.Run("resource attributes are read from the DynaKube", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		dk.Spec.ResourceAttributes = map[string]string{"region": "us-east"}

		data := buildGatewayConfigData(dk)

		assert.Equal(t, map[string]string{"region": "us-east"}, data.ResourceAttributes)
	})

	t.Run("no resource attributes set on the DynaKube", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"

		data := buildGatewayConfigData(dk)

		assert.Empty(t, data.ResourceAttributes)
	})
}

func TestReconcileConfigMap(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		s := newTestScopeWithDynaKube(pm, dk)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, cm))

		helpers.AssertGolden(t, filepath.Join("testdata", "configmap.yaml"), cm)

		sum := sha256.Sum256([]byte(cm.Data[gatewayConfigKey]))
		assert.Equal(t, hex.EncodeToString(sum[:]), s.ConfigMapHash)
	})

	t.Run("resource attributes are rendered into the configmap", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		dk.Spec.ResourceAttributes = map[string]string{"favorite.coffee": "espresso", "deploy.mood": "yolo"}
		s := newTestScopeWithDynaKube(pm, dk)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, cm))
		helpers.AssertGolden(t, filepath.Join("testdata", "configmap_with_resource_attributes.yaml"), cm)
	})

	t.Run("self-monitoring telemetry is rendered into the configmap", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		dk := &dynakube.DynaKube{Name: "dk"}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		dk.Spec.SelfMonitoring = &dynakube.SelfMonitoringSpec{}
		s := newTestScopeWithDynaKube(pm, dk)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, cm))
		helpers.AssertGolden(t, filepath.Join("testdata", "configmap_with_self_monitoring.yaml"), cm)
	})

	t.Run("changing resource attributes changes the config hash, triggering a rollout", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		dk := &dynakube.DynaKube{}
		dk.Spec.APIURL = "https://abc12345.live.dynatrace.com/api"
		s := newTestScopeWithDynaKube(pm, dk)
		r := &Reconciler{Client: fake.NewClient()}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))
		hashBefore := s.ConfigMapHash

		dk.Spec.ResourceAttributes = map[string]string{"region": "us-east"}
		require.NoError(t, r.reconcileConfigMap(t.Context(), s))
		hashAfter := s.ConfigMapHash

		assert.NotEqual(t, hashBefore, hashAfter)
	})

	t.Run("merge labels", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		s := newTestScope(pm)
		existing := &corev1.ConfigMap{
			Name:      s.Spec.GetStatefulSetName(),
			Namespace: pm.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, cm))
		assert.Equal(t, "value", cm.Labels["custom"])
		assert.Equal(t, pm.Name, cm.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileConfigMap(t.Context(), newTestScope(newTestPM("pm", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconcileService(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		s := newTestScope(pm)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, svc))

		helpers.AssertGolden(t, filepath.Join("testdata", "service.yaml"), svc)
	})

	t.Run("merge labels", func(t *testing.T) {
		pm := newTestPM("pm", "dynatrace")
		s := newTestScope(pm)
		existing := &corev1.Service{
			Name:      s.Spec.GetStatefulSetName(),
			Namespace: pm.Namespace,
			Labels:    map[string]string{"custom": "value", k8slabel.AppInstanceLabel: "override"}}
		c := fake.NewClient(existing)
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileService(t.Context(), s))

		svc := &corev1.Service{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetStatefulSetName(), Namespace: pm.Namespace}, svc))
		assert.Equal(t, "value", svc.Labels["custom"])
		assert.Equal(t, pm.Name, svc.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileService(t.Context(), newTestScope(newTestPM("pm", "dynatrace")))
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

func TestMutateStatefulSetKeepsExistingTemplateMetadata(t *testing.T) {
	obj := &appsv1.StatefulSet{}
	obj.Spec.Template.Annotations = map[string]string{"kubectl.kubernetes.io/restartedAt": "now"}
	obj.Spec.Template.Labels = map[string]string{"user": "label"}

	s := newTestScope(newTestPM("pm", "dynatrace"))
	s.Spec.Annotations = map[string]string{"from": "spec"}
	s.Spec.Labels = map[string]string{"spec": "label"}

	mutateStatefulSet(obj, s)

	assert.Equal(t, "now", obj.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	assert.Equal(t, "spec", obj.Spec.Template.Annotations["from"])
	assert.Equal(t, "label", obj.Spec.Template.Labels["user"])
	assert.Equal(t, "label", obj.Spec.Template.Labels["spec"])
}

func TestMutateStatefulSetCustomEnvVolumes(t *testing.T) {
	newScope := func(spec func(*prometheusmonitoring.PodSpec)) *reconcileScope {
		s := newTestScope(newTestPM("pm", "dynatrace"))
		spec(&s.Spec.PodSpec)

		return s
	}

	t.Run("empty -> only operator-generated entries", func(t *testing.T) {
		obj := &appsv1.StatefulSet{}
		require.NoError(t, mutateStatefulSet(obj, newScope(func(*prometheusmonitoring.PodSpec) {})))

		container := obj.Spec.Template.Spec.Containers[0]
		require.NotEmpty(t, container.Env)
		require.NotEmpty(t, container.VolumeMounts)
		require.NotEmpty(t, obj.Spec.Template.Spec.Volumes)
		assert.NotContains(t, envNames(container.Env), "CUSTOM")
		assert.NotContains(t, mountNames(container.VolumeMounts), "custom")
	})

	t.Run("custom entries are appended", func(t *testing.T) {
		obj := &appsv1.StatefulSet{}
		s := newScope(func(p *prometheusmonitoring.PodSpec) {
			p.Env = []corev1.EnvVar{{Name: "CUSTOM", Value: "1"}}
			p.Volumes = apitypes.Volumes{{Name: "custom", AllowedVolumeSource: apitypes.AllowedVolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
			p.VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: "/custom"}}
		})
		require.NoError(t, mutateStatefulSet(obj, s))

		container := obj.Spec.Template.Spec.Containers[0]
		assert.Equal(t, "CUSTOM", container.Env[len(container.Env)-1].Name)
		assert.Equal(t, "custom", container.VolumeMounts[len(container.VolumeMounts)-1].Name)
		assert.Equal(t, "custom", obj.Spec.Template.Spec.Volumes[len(obj.Spec.Template.Spec.Volumes)-1].Name)
	})

	t.Run("custom env var with an operator-generated name is appended last", func(t *testing.T) {
		base := &appsv1.StatefulSet{}
		require.NoError(t, mutateStatefulSet(base, newScope(func(*prometheusmonitoring.PodSpec) {})))
		baseEnv := base.Spec.Template.Spec.Containers[0].Env

		obj := &appsv1.StatefulSet{}
		s := newScope(func(p *prometheusmonitoring.PodSpec) {
			p.Env = []corev1.EnvVar{{Name: baseEnv[0].Name, Value: "override"}}
		})
		require.NoError(t, mutateStatefulSet(obj, s))

		env := obj.Spec.Template.Spec.Containers[0].Env
		assert.Len(t, env, len(baseEnv)+1)
		assert.Equal(t, baseEnv, env[:len(baseEnv)])
		assert.Equal(t, corev1.EnvVar{Name: baseEnv[0].Name, Value: "override"}, env[len(env)-1])
	})

	t.Run("template is stable across reconciles", func(t *testing.T) {
		s := newScope(func(p *prometheusmonitoring.PodSpec) {
			p.Env = []corev1.EnvVar{{Name: "CUSTOM", Value: "1"}}
			p.VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: "/custom"}}
		})
		obj := &appsv1.StatefulSet{}
		require.NoError(t, mutateStatefulSet(obj, s))
		first := obj.DeepCopy()

		require.NoError(t, mutateStatefulSet(obj, s))

		assert.Equal(t, first, obj)
	})
}

func envNames(envs []corev1.EnvVar) []string {
	names := make([]string, 0, len(envs))
	for _, e := range envs {
		names = append(names, e.Name)
	}

	return names
}

func mountNames(mounts []corev1.VolumeMount) []string {
	names := make([]string, 0, len(mounts))
	for _, m := range mounts {
		names = append(names, m.Name)
	}

	return names
}

func TestNoProxyEnvVar(t *testing.T) {
	t.Run("proxy enabled", func(t *testing.T) {
		// enable AG and make sure that "dynakube-activegate.dynatrace" is not on the NO_PROXY list
		dk := &dynakube.DynaKube{
			Name:      "dynakube",
			Namespace: "dynatrace",
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
				Proxy: &apitypes.ValueSource{
					Value: "http://proxy:8080",
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterName: "my-cluster",
			},
		}

		rs := &reconcileScope{
			Spec: &prometheusmonitoring.Gateway{
				GatewaySpec: &prometheusmonitoring.GatewaySpec{
					Resources: corev1.ResourceRequirements{},
				},
			},
			DynaKube: dk,
		}

		envVars := buildEnv(rs)
		noProxy := k8senv.Find(envVars, "NO_PROXY")
		require.NotNil(t, noProxy)
		assert.Equal(t, "$(KUBERNETES_SERVICE_HOST),kubernetes.default", noProxy.Value)
	})

	t.Run("proxy disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterName: "my-cluster",
			},
		}

		rs := &reconcileScope{
			Spec: &prometheusmonitoring.Gateway{
				GatewaySpec: &prometheusmonitoring.GatewaySpec{
					Resources: corev1.ResourceRequirements{},
				},
			},
			DynaKube: dk,
		}

		envVars := buildEnv(rs)
		noProxy := k8senv.Find(envVars, "NO_PROXY")
		require.Nil(t, noProxy)
	})

	t.Run("proxy enabled with no-proxy ff", func(t *testing.T) {
		const (
			host1 = "host1"
			host2 = "host2:8443"
		)
		dk := &dynakube.DynaKube{
			Annotations: map[string]string{
				// check if spaces are removed
				exp.NoProxyKey: host1 + " , " + host2,
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &apitypes.ValueSource{
					Value: "http://proxy:8080",
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterName: "my-cluster",
			},
		}

		rs := &reconcileScope{
			Spec: &prometheusmonitoring.Gateway{
				GatewaySpec: &prometheusmonitoring.GatewaySpec{
					Resources: corev1.ResourceRequirements{},
				},
			},
			DynaKube: dk,
		}

		envVars := buildEnv(rs)
		noProxy := k8senv.Find(envVars, "NO_PROXY")
		require.NotNil(t, noProxy)
		assert.Equal(t, "$(KUBERNETES_SERVICE_HOST),kubernetes.default,"+host1+","+host2, noProxy.Value)
	})
}
