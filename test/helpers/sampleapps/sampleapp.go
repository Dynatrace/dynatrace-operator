//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"path"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	defaultNameTemplate = "sample-%s"
	podTemplatePath     = path.Join(project.TestDataDir(), "sample-app/pod-base.yaml")
	sccPath             = path.Join(project.TestDataDir(), "sample-app/restricted-csi.yaml")
)

type restartFunc func(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources)

type sampleApp struct {
	name      string
	namespace *corev1.Namespace

	t            *testing.T
	base         *corev1.Pod
	testDynakube dynatracev1.DynaKube

	installedNamespace bool
}

func newSampleApp(t *testing.T, testDynakube dynatracev1.DynaKube) *sampleApp {
	return &sampleApp{
		name:         fmt.Sprintf(defaultNameTemplate, testDynakube.Name),
		t:            t,
		base:         manifests.ObjectFromFile[*corev1.Pod](t, podTemplatePath),
		testDynakube: testDynakube,
	}
}

func (app sampleApp) Name() string {
	if app.name == "" {
		return fmt.Sprintf("%s-sample", app.testDynakube.Name)
	}
	return app.name
}

func (app sampleApp) ContainerName() string {
	return app.Name()
}

func (app sampleApp) Namespace() *corev1.Namespace {
	if app.namespace == nil {
		defaultNamespace := namespace.NewBuilder(app.Name()).Build()
		return &defaultNamespace
	}
	return app.namespace
}

func (app *sampleApp) WithName(name string) {
	app.name = name
}

func (app *sampleApp) WithNamespace(namespace corev1.Namespace) {
	app.namespace = &namespace
}

func (app *sampleApp) WithAnnotations(annotations map[string]string) {
	app.base.Annotations = annotations
}

func (app *sampleApp) WithLabels(labels map[string]string) {
	app.base.Labels = labels
}

func (app *sampleApp) WithEnvs(envs []corev1.EnvVar) {
	app.base.Spec.Containers[0].Env = envs
}

func (app *sampleApp) WithSecurityContext(securityContext corev1.PodSecurityContext) {
	app.base.Spec.SecurityContext = &securityContext
}

func (app sampleApp) install(object client.Object) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		resource := c.Client().Resources()
		if !app.installedNamespace {
			ctx = namespace.Create(*app.Namespace())(ctx, t, c)
		}
		ctx = app.installSCC(ctx, t, c)
		require.NoError(t, resource.Create(ctx, object))
		if dep, ok := object.(*appsv1.Deployment); ok {
			require.NoError(t, deployment.WaitUntilReady(resource, dep))
		} else if pod, ok := object.(*corev1.Pod); ok {
			require.NoError(t, wait.For(conditions.New(resource).PodReady(pod)))
		}
		return ctx
	}
}

func (app sampleApp) installSCC(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	platform := kubeobjects.ResolvePlatformFromEnv()
	if platform == kubeobjects.Openshift {
		ctx = manifests.InstallFromFile(sccPath)(ctx, t, c)
	}
	return ctx
}

func (app *sampleApp) InstallNamespace() features.Func {
	app.installedNamespace = true
	return namespace.Create(*app.Namespace())
}

func (app sampleApp) UninstallNamespace() features.Func {
	return namespace.Delete(app.Namespace().Name)
}

func (app sampleApp) uninstallObject(object client.Object) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		resource := c.Client().Resources()
		require.NoError(t, resource.Delete(ctx, object))
		ctx = app.uninstallSCC(ctx, t, c)
		require.NoError(t, wait.For(conditions.New(resource).ResourceDeleted(object)))
		if dep, ok := object.(*appsv1.Deployment); ok {
			ctx = pod.WaitForPodsDeletionWithOwner(dep.Name, dep.Namespace)(ctx, t, c)
		}
		return ctx
	}
}

func (app sampleApp) uninstallSCC(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	platform := kubeobjects.ResolvePlatformFromEnv()
	if platform == kubeobjects.Openshift {
		ctx = manifests.UninstallFromFile(sccPath)(ctx, t, c)
	}
	return ctx
}

func (app sampleApp) createObjectMeta() metav1.ObjectMeta {
	objectMeta := app.base.ObjectMeta
	objectMeta.Name = app.Name()
	objectMeta.Namespace = app.Namespace().Name
	return objectMeta
}

func restartHalf(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources) {
	for i, podItem := range pods.Items {
		if i%2 == 1 {
			continue // skip odd-indexed pods
		}
		require.NoError(t, resource.Delete(ctx, &podItem)) // nolint:gosec
	}
}

func restart(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources) {
	for _, podItem := range pods.Items {
		require.NoError(t, resource.Delete(ctx, &podItem)) // nolint:gosec
		require.NoError(t, wait.For(
			conditions.New(resource).ResourceDeleted(&podItem))) // nolint:gosec
	}
}
