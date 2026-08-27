// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/stretchr/testify/assert"
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

const testImage = "registry.example.com/scraper:1.2.3"

func newTestDTP(name, namespace string) *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("dtp-uid")},
		Spec: dtprometheus.DTPrometheusSpec{
			Scraper: dtprometheus.ScraperSpec{
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
				TargetsPollInterval: metav1.Duration{Duration: 60 * time.Second},
			},
		},
	}
}

func newTestScope(dtp *dtprometheus.DTPrometheus) *reconcileScope {
	return newTestScopeWithDynaKube(dtp, &dynakube.DynaKube{})
}

func newTestScopeWithDynaKube(dtp *dtprometheus.DTPrometheus, dk *dynakube.DynaKube) *reconcileScope {
	return &reconcileScope{
		Owner:     dtp,
		DynaKube:  dk,
		Spec:      dtp.Scraper(),
		AppLabels: k8slabel.OTelScraper(),
	}
}

// failingImageClient stands in for the fleet management image API when resolution
// is expected to fail.
type failingImageClient struct {
	err error
}

func (c failingImageClient) GetComponentLatestInfo(context.Context, image.ComponentType, string) (*image.Info, error) {
	return nil, c.err
}

func createErrorClient(createErr error) client.Client {
	return fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
			return createErr
		},
	})
}

// The condition branch logic is covered by the condition package. What matters here is
// that the scraper wires itself to the right condition type, component name and rollout check.
func TestReconcileCondition(t *testing.T) {
	t.Run("freshly created deployment with no ready replicas -> pending", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Image = testImage
		dtp.Spec.Scraper.Replicas = new(int32(2))

		r := &Reconciler{Client: fake.NewClient()}
		require.NoError(t, r.Reconcile(t.Context(), dtp, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.ScraperAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonReconciling, condition.Reason)
		assert.Equal(t, "scraper is pending", condition.Message)
	})

	t.Run("reconcile error -> error, with unwrapped message", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Image = testImage

		boom := errors.New("boom")
		r := &Reconciler{Client: createErrorClient(boom)}
		require.Error(t, r.Reconcile(t.Context(), dtp, &dynakube.DynaKube{}, nil))

		condition := meta.FindStatusCondition(dtp.Status.Conditions, dtprometheus.ScraperAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonError, condition.Reason)
		assert.Equal(t, boom.Error(), condition.Message)
	})
}

func TestReconcileConfigMap(t *testing.T) {
	t.Run("apply spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		c := fake.NewClient()
		r := &Reconciler{Client: c}

		require.NoError(t, r.reconcileConfigMap(t.Context(), s))

		cm := &corev1.ConfigMap{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: s.Spec.GetDeploymentName(), Namespace: dtp.Namespace}, cm))

		helpers.AssertGolden(t, "testdata/configmap.yaml", cm)

		sum := sha256.Sum256([]byte(cm.Data[scraperConfigKey]))
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
		assert.Equal(t, k8slabel.OTelScraper().Instance, cm.Labels[k8slabel.AppInstanceLabel])
	})

	t.Run("propagate error", func(t *testing.T) {
		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileConfigMap(t.Context(), newTestScope(newTestDTP("dtp", "dynatrace")))
		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconcileDeployment(t *testing.T) {
	t.Run("unresolvable image", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScope(dtp)
		// No explicit image on the spec, so the reconciler falls back to the registry.
		s.ImageClient = failingImageClient{err: errors.New("no such component")}
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
		dtp.Spec.Scraper.Image = testImage
		dtp.Spec.Scraper.ImagePullPolicy = corev1.PullAlways
		dtp.Spec.Scraper.Replicas = new(int32(3))
		dtp.Spec.Scraper.NodeSelector = map[string]string{"disk": "ssd"}
		dtp.Spec.Scraper.PriorityClassName = "high-priority"
		dtp.Spec.Scraper.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		dtp.Spec.Scraper.Annotations = map[string]string{"custom": "annotation"}
		dtp.Spec.Scraper.Labels = map[string]string{"custom": "label"}
		dtp.Spec.Scraper.Args = []string{"--foo=bar"}
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

	t.Run("records the resolved image in the status", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Image = testImage
		s := newTestScope(dtp)
		r := &Reconciler{Client: fake.NewClient()}

		require.NoError(t, r.reconcileDeployment(t.Context(), s))

		assert.Equal(t, testImage, dtp.Status.Scraper.ResolvedImage)
	})

	t.Run("propagate error", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Image = testImage

		expectErr := errors.New("boom")
		err := (&Reconciler{Client: createErrorClient(expectErr)}).reconcileDeployment(t.Context(), newTestScope(dtp))
		require.ErrorIs(t, err, expectErr)
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("config flag only when no user args", func(t *testing.T) {
		s := newTestScope(newTestDTP("dtp", "dynatrace"))

		assert.Equal(t, []string{"--config=/conf/scraper.yaml"}, buildArgs(s))
	})

	t.Run("user args are appended after the config flag", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Args = []string{"--feature-gates=foo", "--set=bar"}
		s := newTestScope(dtp)

		assert.Equal(t, []string{"--config=/conf/scraper.yaml", "--feature-gates=foo", "--set=bar"}, buildArgs(s))
	})

	t.Run("user args are sanitized", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.Args = []string{"--ok=1\nrm -rf /"}
		s := newTestScope(dtp)

		args := buildArgs(s)
		require.Len(t, args, 2)
		assert.NotContains(t, args[1], "\n")
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("no proxy env without a proxy on the dynakube", func(t *testing.T) {
		envs := buildEnv(newTestScope(newTestDTP("dtp", "dynatrace")))

		for _, env := range envs {
			assert.NotContains(t, env.Name, "PROXY")
		}
	})

	t.Run("pod name and ip are exposed for the collector config", func(t *testing.T) {
		envs := buildEnv(newTestScope(newTestDTP("dtp", "dynatrace")))

		assert.Equal(t, "metadata.name", findEnv(t, envs, "MY_POD_NAME").ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, "status.podIP", findEnv(t, envs, "MY_POD_IP").ValueFrom.FieldRef.FieldPath)
	})

	t.Run("proxy value is taken from the dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.Proxy = &value.Source{Value: "http://proxy:3128"}
		s := newTestScopeWithDynaKube(newTestDTP("dtp", "dynatrace"), dk)

		envs := buildEnv(s)

		assert.Equal(t, "http://proxy:3128", findEnv(t, envs, "HTTPS_PROXY").Value)
		assert.Equal(t, "http://proxy:3128", findEnv(t, envs, "HTTP_PROXY").Value)
	})

	t.Run("proxy secret reference is passed through", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.Proxy = &value.Source{ValueFrom: "proxy-secret"}
		s := newTestScopeWithDynaKube(newTestDTP("dtp", "dynatrace"), dk)

		envs := buildEnv(s)

		ref := findEnv(t, envs, "HTTPS_PROXY").ValueFrom.SecretKeyRef
		assert.Equal(t, "proxy-secret", ref.Name)
		assert.Equal(t, dynakube.ProxyKey, ref.Key)
	})

	t.Run("no_proxy excludes the api server, target allocator and gateway", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.Proxy = &value.Source{Value: "http://proxy:3128"}
		dtp := newTestDTP("dtp", "dynatrace")
		s := newTestScopeWithDynaKube(dtp, dk)

		noProxy := findEnv(t, buildEnv(s), "NO_PROXY").Value

		assert.Contains(t, noProxy, "$(KUBERNETES_SERVICE_HOST)")
		assert.Contains(t, noProxy, "kubernetes.default")
		assert.Contains(t, noProxy, "dtp-prometheus-allocator.dynatrace.svc.cluster.local")
		assert.Contains(t, noProxy, "dtp-gateway.dynatrace.svc.cluster.local")
	})

	t.Run("no_proxy appends the user-configured entries", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.Proxy = &value.Source{Value: "http://proxy:3128"}
		dk.Annotations = map[string]string{"feature.dynatrace.com/no-proxy": "10.42.0.0/16, *.internal"}
		s := newTestScopeWithDynaKube(newTestDTP("dtp", "dynatrace"), dk)

		noProxy := findEnv(t, buildEnv(s), "NO_PROXY").Value

		assert.Contains(t, noProxy, "10.42.0.0/16")
		assert.Contains(t, noProxy, "*.internal")
	})
}

func findEnv(t *testing.T, envs []corev1.EnvVar, name string) corev1.EnvVar {
	t.Helper()

	for _, env := range envs {
		if env.Name == name {
			return env
		}
	}

	t.Fatalf("env var %s not found", name)

	return corev1.EnvVar{}
}

func TestBuildVolumes(t *testing.T) {
	t.Run("config volume only without trusted CAs", func(t *testing.T) {
		s := newTestScope(newTestDTP("dtp", "dynatrace"))

		volumes := buildVolumes(s)
		mounts := buildVolumeMounts(s)

		require.Len(t, volumes, 1)
		assert.Equal(t, configVolumeName, volumes[0].Name)
		assert.Equal(t, "dtp-scraper", volumes[0].ConfigMap.Name)
		require.Len(t, mounts, 1)
		assert.Equal(t, configMountDir, mounts[0].MountPath)
	})

	t.Run("cacerts volume is added for trusted CAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec.TrustedCAs = "ca-bundle"
		s := newTestScopeWithDynaKube(newTestDTP("dtp", "dynatrace"), dk)

		volumes := buildVolumes(s)
		mounts := buildVolumeMounts(s)

		require.Len(t, volumes, 2)
		assert.Equal(t, cacertsVolumeName, volumes[1].Name)
		assert.Equal(t, "ca-bundle", volumes[1].ConfigMap.Name)
		require.Len(t, mounts, 2)
		assert.Equal(t, trustedCAVolumeMountPath, mounts[1].MountPath)
		assert.True(t, mounts[1].ReadOnly)
	})
}

func TestMutateDeploymentIsIdempotent(t *testing.T) {
	dtp := newTestDTP("dtp", "dynatrace")
	dtp.Spec.Scraper.Image = testImage
	s := newTestScope(dtp)
	s.resolvedImage = testImage

	deploy := &appsv1.Deployment{}
	mutateDeployment(deploy, s)
	first := deploy.DeepCopy()

	mutateDeployment(deploy, s)

	assert.Equal(t, first, deploy)
}
