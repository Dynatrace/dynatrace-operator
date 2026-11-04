// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testName      = "dk"
	testNamespace = "dynatrace"
	testImage     = "registry.example.com/otel-collector:1.2.3"
	testObjName   = testName + "-self-monitoring"
)

func newTestDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		Name: testName, Namespace: testNamespace, UID: "dk-uid",
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
			SelfMonitoring: &dynakube.SelfMonitoringSpec{
				Image: testImage,
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("500Mi")},
				},
			},
		},
	}
}

func newTokenSecret(withDataIngest bool) *corev1.Secret {
	secret := &corev1.Secret{
		Name:      testName,
		Namespace: testNamespace,
		Data:      map[string][]byte{token.APIKey: []byte("api-token")},
	}
	if withDataIngest {
		secret.Data[token.DataIngestKey] = []byte("data-ingest-token")
	}

	return secret
}

func newTestReconciler(objs ...client.Object) (*Reconciler, client.Client) {
	c := fake.NewClient(objs...)

	return NewReconciler(c, c), c
}

func newTestScope(dk *dynakube.DynaKube) *reconcileScope {
	return &reconcileScope{
		DynaKube:  dk,
		Spec:      dk.Spec.SelfMonitoring,
		AppLabels: newAppLabels(dk),
	}
}

func getObject[T client.Object](t *testing.T, c client.Client, obj T) T {
	t.Helper()
	require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: testObjName, Namespace: testNamespace}, obj))

	return obj
}

func assertNotFound(t *testing.T, c client.Client, obj client.Object) {
	t.Helper()

	err := c.Get(t.Context(), client.ObjectKey{Name: testObjName, Namespace: testNamespace}, obj)
	assert.True(t, k8serrors.IsNotFound(err), "expected %T to be absent, got %v", obj, err)
}

func TestReconcile(t *testing.T) {
	t.Run("creates all managed objects owned by the DynaKube", func(t *testing.T) {
		dk := newTestDynaKube()
		r, c := newTestReconciler(newTokenSecret(true))

		require.NoError(t, r.Reconcile(t.Context(), nil, dk))

		for _, obj := range []client.Object{&appsv1.Deployment{}, &corev1.Service{}, &corev1.ConfigMap{}} {
			obj = getObject(t, c, obj)
			require.Len(t, obj.GetOwnerReferences(), 1, "%T", obj)
			assert.Equal(t, dk.UID, obj.GetOwnerReferences()[0].UID, "%T", obj)
		}

		assert.Equal(t, testImage, dk.Status.SelfMonitoring.Image)

		condition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, status.ReasonReconciling, condition.Reason)
	})

	t.Run("service selects the collector pods", func(t *testing.T) {
		dk := newTestDynaKube()
		r, c := newTestReconciler(newTokenSecret(true))

		require.NoError(t, r.Reconcile(t.Context(), nil, dk))

		deploy := getObject(t, c, &appsv1.Deployment{})
		svc := getObject(t, c, &corev1.Service{})

		podLabels := deploy.Spec.Template.Labels
		for k, v := range svc.Spec.Selector {
			assert.Equal(t, v, podLabels[k], "service selector %s", k)
		}

		assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
		require.Len(t, svc.Spec.Ports, 1)
		assert.Equal(t, int32(otlpHTTPPort), svc.Spec.Ports[0].Port)
	})

	t.Run("missing data ingest token surfaces token condition and deploys nothing", func(t *testing.T) {
		dk := newTestDynaKube()
		r, c := newTestReconciler(newTokenSecret(false))

		require.NoError(t, r.Reconcile(t.Context(), nil, dk))

		tokenCondition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.TokenConditionType)
		require.NotNil(t, tokenCondition)
		assert.Equal(t, metav1.ConditionFalse, tokenCondition.Status)
		assert.Equal(t, k8sconditions.DataIngestTokenMissing, tokenCondition.Reason)
		assert.Equal(t, dataIngestTokenMissingMessage, tokenCondition.Message)

		condition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, status.ReasonError, condition.Reason)

		assertNotFound(t, c, &appsv1.Deployment{})
	})

	t.Run("missing token secret is an error", func(t *testing.T) {
		dk := newTestDynaKube()
		r, _ := newTestReconciler()

		require.Error(t, r.Reconcile(t.Context(), nil, dk))

		condition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, status.ReasonError, condition.Reason)
	})

	t.Run("resolves image from the registry when not set in the spec", func(t *testing.T) {
		resolvedImage := "resolved:1.0"
		dk := newTestDynaKube()
		dk.Spec.SelfMonitoring.Image = ""
		imgClient := imagemock.NewClient(t)
		imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, dtimage.OTelCollector, mock.Anything).Return(&dtimage.Info{URI: resolvedImage}, nil)
		r, c := newTestReconciler(newTokenSecret(true))

		require.NoError(t, r.Reconcile(t.Context(), imgClient, dk))

		assert.Equal(t, resolvedImage, dk.Status.SelfMonitoring.Image)
		assert.Equal(t, resolvedImage, getObject(t, c, &appsv1.Deployment{}).Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("kube API error is propagated to the condition", func(t *testing.T) {
		dk := newTestDynaKube()
		boom := errors.New("boom")
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error { return boom },
		}, newTokenSecret(true))

		require.ErrorIs(t, NewReconciler(c, c).Reconcile(t.Context(), nil, dk), boom)

		condition := meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable)
		require.NotNil(t, condition)
		assert.Equal(t, status.ReasonError, condition.Reason)
	})
}

func TestCleanup(t *testing.T) {
	assertOff := func(t *testing.T, dk *dynakube.DynaKube) {
		t.Helper()

		cond := meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, status.ReasonOff, cond.Reason)
		assert.Equal(t, selfMonitoringOffMessage, cond.Message)
	}

	t.Run("only sets the off condition when never deployed", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.SelfMonitoring = nil
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				t.Fatal("no delete expected")

				return nil
			},
		})

		require.NoError(t, NewReconciler(c, c).Reconcile(t.Context(), nil, dk))
		assertOff(t, dk)
	})

	t.Run("no-op when already cleaned up", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.SelfMonitoring = nil
		setOffCondition(dk)
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				t.Fatal("no delete expected")

				return nil
			},
		})

		require.NoError(t, NewReconciler(c, c).Reconcile(t.Context(), nil, dk))
		assertOff(t, dk)
	})

	t.Run("removes all managed objects and the status once disabled", func(t *testing.T) {
		dk := newTestDynaKube()
		r, c := newTestReconciler(newTokenSecret(true))
		require.NoError(t, r.Reconcile(t.Context(), nil, dk))
		require.NotNil(t, meta.FindStatusCondition(dk.Status.Conditions, dynakube.SelfMonitoringAvailable))

		dk.Spec.SelfMonitoring = nil
		require.NoError(t, r.Reconcile(t.Context(), nil, dk))

		for _, obj := range []client.Object{&appsv1.Deployment{}, &corev1.Service{}, &corev1.ConfigMap{}} {
			assertNotFound(t, c, obj)
		}

		assertOff(t, dk)
		assert.Empty(t, dk.Status.SelfMonitoring.Image)
	})

	t.Run("keeps the condition when deletion fails so cleanup is retried", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.SelfMonitoring = nil
		meta.SetStatusCondition(&dk.Status.Conditions, metav1.Condition{Type: dynakube.SelfMonitoringAvailable, Status: metav1.ConditionTrue, Reason: status.ReasonAvailable})
		boom := errors.New("boom")
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error { return boom },
		})

		require.ErrorIs(t, NewReconciler(c, c).Reconcile(t.Context(), nil, dk), boom)
		assert.True(t, meta.IsStatusConditionTrue(dk.Status.Conditions, dynakube.SelfMonitoringAvailable))
	})
}

func TestReconcileDeployment(t *testing.T) {
	t.Run("applies the spec", func(t *testing.T) {
		dk := newTestDynaKube()
		sm := dk.Spec.SelfMonitoring
		sm.ImagePullPolicy = corev1.PullAlways
		sm.Replicas = new(int32(2))
		sm.NodeSelector = map[string]string{"disk": "ssd"}
		sm.PriorityClassName = "high-priority"
		sm.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		sm.Annotations = map[string]string{"custom": "annotation"}
		sm.Labels = map[string]string{"custom": "label"}
		sm.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		s := newTestScope(dk)
		s.ConfigMapHash = "deadbeef"
		r, c := newTestReconciler()

		require.NoError(t, r.reconcileDeployment(t.Context(), s))

		deploy := getObject(t, c, &appsv1.Deployment{})
		podSpec := deploy.Spec.Template.Spec
		assert.Equal(t, int32(2), *deploy.Spec.Replicas)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deploy.Spec.Strategy.Type)
		assert.Equal(t, "deadbeef", deploy.Spec.Template.Annotations[configHashAnnotation])
		assert.Equal(t, "annotation", deploy.Spec.Template.Annotations["custom"])
		assert.Equal(t, "label", deploy.Spec.Template.Labels["custom"])
		assert.Equal(t, serviceAccountName, podSpec.ServiceAccountName)
		assert.Equal(t, sm.NodeSelector, podSpec.NodeSelector)
		assert.Equal(t, sm.PriorityClassName, podSpec.PriorityClassName)
		assert.Equal(t, sm.Tolerations, podSpec.Tolerations)
		assert.Nil(t, podSpec.Affinity)
		assert.Equal(t, int64(otelCollectorNonRootUser), *podSpec.SecurityContext.FSGroup)

		require.Len(t, podSpec.Containers, 1)
		container := podSpec.Containers[0]
		assert.Equal(t, testImage, container.Image)
		assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)
		assert.Equal(t, int64(otelCollectorNonRootUser), *container.SecurityContext.RunAsUser)
		assert.True(t, *container.SecurityContext.ReadOnlyRootFilesystem)
		assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
		assert.Equal(t, []corev1.Capability{"ALL"}, container.SecurityContext.Capabilities.Drop)
	})

	t.Run("is idempotent", func(t *testing.T) {
		s := newTestScope(newTestDynaKube())
		s.DynaKube.Status.SelfMonitoring.Image = testImage
		deploy := &appsv1.Deployment{}

		mutateDeployment(deploy, s)
		first := deploy.DeepCopy()
		mutateDeployment(deploy, s)

		assert.Equal(t, first, deploy)
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("config flag only when no user args", func(t *testing.T) {
		assert.Equal(t, []string{"--config=/conf/self-monitoring.yaml"}, buildArgs(newTestScope(newTestDynaKube())))
	})

	t.Run("user args are appended after the config flag", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.SelfMonitoring.Args = []string{"--config=/custom/override.yaml"}

		assert.Equal(t, []string{"--config=/conf/self-monitoring.yaml", "--config=/custom/override.yaml"}, buildArgs(newTestScope(dk)))
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("pod ip, cluster name and go memory limit", func(t *testing.T) {
		envs := buildEnv(newTestScope(newTestDynaKube()))

		byName := map[string]corev1.EnvVar{}
		for _, e := range envs {
			byName[e.Name] = e
		}

		assert.Equal(t, "status.podIP", byName["MY_POD_IP"].ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, testName, byName["K8S_CLUSTER_NAME"].Value)

		for name, fieldPath := range map[string]string{
			"K8S_NODE_NAME":      "spec.nodeName",
			"K8S_POD_NAME":       "metadata.name",
			"K8S_NAMESPACE_NAME": "metadata.namespace",
			"K8S_POD_UID":        "metadata.uid",
		} {
			require.Contains(t, byName, name)
			require.NotNil(t, byName[name].ValueFrom)
			assert.Equal(t, fieldPath, byName[name].ValueFrom.FieldRef.FieldPath, name)
			assert.Equal(t, "v1", byName[name].ValueFrom.FieldRef.APIVersion, name)
		}
		// 90% of the 500Mi limit
		assert.Equal(t, "471859200", byName["GOMEMLIMIT"].Value)
		assert.NotContains(t, byName, "HTTPS_PROXY")
	})

	t.Run("proxy from secret", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.Proxy = &types.ValueSource{ValueFrom: "proxy-secret"}

		envs := buildEnv(newTestScope(dk))

		byName := map[string]corev1.EnvVar{}
		for _, e := range envs {
			byName[e.Name] = e
		}

		require.Contains(t, byName, "HTTPS_PROXY")
		assert.Equal(t, "proxy-secret", byName["HTTPS_PROXY"].ValueFrom.SecretKeyRef.Name)
		assert.Contains(t, byName["NO_PROXY"].Value, "$(KUBERNETES_SERVICE_HOST)")
	})
}

func TestBuildVolumes(t *testing.T) {
	t.Run("token is mounted as a directory, not via subPath", func(t *testing.T) {
		s := newTestScope(newTestDynaKube())

		volumes := buildVolumes(s)
		mounts := buildVolumeMounts(s)

		require.Len(t, volumes, 2)
		tokenVolume := volumes[1]
		assert.Equal(t, tokenVolumeName, tokenVolume.Name)
		require.NotNil(t, tokenVolume.Projected)
		assert.Equal(t, int32(288), *tokenVolume.Projected.DefaultMode)
		assert.Equal(t, testName, tokenVolume.Projected.Sources[0].Secret.Name)
		assert.Equal(t, []corev1.KeyToPath{{Key: token.DataIngestKey, Path: tokenFileName}}, tokenVolume.Projected.Sources[0].Secret.Items)

		for _, m := range mounts {
			assert.Empty(t, m.SubPath, m.Name)
		}

		assert.Contains(t, mounts, corev1.VolumeMount{Name: tokenVolumeName, MountPath: tokenMountPath, ReadOnly: true})
	})

	t.Run("trusted CAs and user volumes are added", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.TrustedCAs = "my-cas"
		userVolume := corev1.Volume{Name: "override", EmptyDir: &corev1.EmptyDirVolumeSource{}}
		userMount := corev1.VolumeMount{Name: "override", MountPath: "/custom"}
		dk.Spec.SelfMonitoring.Volumes = types.Volumes{{Name: userVolume.Name, EmptyDir: userVolume.EmptyDir}}
		dk.Spec.SelfMonitoring.VolumeMounts = []corev1.VolumeMount{userMount}
		s := newTestScope(dk)

		volumes := buildVolumes(s)
		mounts := buildVolumeMounts(s)

		require.Len(t, volumes, 4)
		assert.Equal(t, cacertsVolumeName, volumes[2].Name)
		assert.Equal(t, userVolume, volumes[3])
		assert.Contains(t, mounts, userMount)
	})
}
