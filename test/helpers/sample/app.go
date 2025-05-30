//go:build e2e

package sample

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/event"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/replicaset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/klient/k8s"
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

type App struct {
	t         *testing.T
	base      *corev1.Pod
	owner     metav1.Object
	namespace corev1.Namespace

	installedNamespace bool
	isDeployment       bool
	withSCC            bool
}

type Option func(*App)

func NewApp(t *testing.T, owner metav1.Object, options ...Option) *App {
	base := manifests.ObjectFromFile[*corev1.Pod](t, podTemplatePath)
	base.Name = fmt.Sprintf(defaultNameTemplate, owner.GetName())
	base.Namespace = base.Name
	app := &App{
		t:         t,
		owner:     owner,
		base:      base,
		namespace: *namespace.New(base.Namespace),
		withSCC:   true,
	}
	for _, opt := range options {
		opt(app)
	}

	return app
}

func WithName(name string) Option {
	return func(app *App) {
		if app.base.Namespace == app.base.Name {
			app.base.Namespace = name
			app.namespace = *namespace.New(name)
		}
		app.base.Name = name
	}
}

func AsDeployment() Option {
	return func(app *App) {
		app.isDeployment = true
	}
}

func WithNamespace(namespace corev1.Namespace) Option {
	return func(app *App) {
		app.namespace = namespace
		app.base.Namespace = namespace.Name
	}
}

func WithNamespaceLabels(labels map[string]string) Option {
	return func(app *App) {
		app.namespace.Labels = labels
	}
}

func WithAnnotations(annotations map[string]string) Option {
	return func(app *App) {
		app.base.Annotations = annotations
	}
}

func WithLabels(labels map[string]string) Option {
	return func(app *App) {
		app.base.Labels = labels
	}
}

func WithEnvs(envs []corev1.EnvVar) Option {
	return func(app *App) {
		app.base.Spec.Containers[0].Env = envs
	}
}

func WithSecurityContext(securityContext corev1.PodSecurityContext) Option {
	return func(app *App) {
		app.base.Spec.SecurityContext = &securityContext
	}
}

func WithoutSCC() Option {
	return func(app *App) {
		app.withSCC = false
	}
}

func (app *App) Name() string {
	return app.base.Name
}

func (app *App) ContainerName() string {
	return app.base.Spec.Containers[0].Name
}

func (app *App) Namespace() string {
	return app.base.Namespace
}

func (app *App) InstallNamespace() features.Func {
	app.installedNamespace = true

	return namespace.Create(app.namespace)
}

func (app *App) Install() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		resource := c.Client().Resources()
		if !app.installedNamespace {
			ctx = app.InstallNamespace()(ctx, t, c)
		}

		if app.withSCC {
			ctx = app.installSCC(ctx, t, c)
		}

		object := app.build()
		require.NoError(t, resource.Create(ctx, object))

		if dep, ok := object.(*appsv1.Deployment); ok {
			err := deployment.WaitUntilReady(resource, dep)
			if err != nil {
				printEventList(t, ctx, resource, app.Namespace())
			}
			require.NoError(t, err)
		} else if p, ok := object.(*corev1.Pod); ok {
			require.NoError(t, wait.For(conditions.New(resource).PodReady(p), wait.WithTimeout(5*time.Minute)))
		}

		return ctx
	}
}

func (app *App) installSCC(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		ctx = helpers.ToFeatureFunc(manifests.InstallFromFile(sccPath), true)(ctx, t, c)
	}

	return ctx
}

func (app *App) Uninstall() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		defer func() {
			app.installedNamespace = false
		}()
		resource := c.Client().Resources()
		object := app.build()

		require.NoError(t, resource.Delete(ctx, object))
		ctx = app.uninstallSCC(ctx, t, c)
		require.NoError(t, wait.For(conditions.New(resource).ResourceDeleted(object), wait.WithTimeout(2*time.Minute)))
		if dep, ok := object.(*appsv1.Deployment); ok {
			ctx = pod.WaitForPodsDeletionWithOwner(dep.Name, dep.Namespace)(ctx, t, c)
		}

		return namespace.Delete(app.Namespace())(ctx, t, c)
	}
}

func (app *App) uninstallSCC(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		ctx = helpers.ToFeatureFunc(manifests.UninstallFromFile(sccPath), true)(ctx, t, c)
	}

	return ctx
}

func (app *App) build() k8s.Object {
	if app.isDeployment {
		return app.asDeployment()
	}

	return app.base
}

func (app *App) asDeployment() *appsv1.Deployment {
	selectorKey := "app"
	selectorValue := app.Name()
	if app.base.Labels == nil {
		app.base.Labels = map[string]string{}
	}
	app.base.Labels[selectorKey] = selectorValue

	return &appsv1.Deployment{
		ObjectMeta: app.base.ObjectMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(2)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorKey: selectorValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: app.base.ObjectMeta,
				Spec:       app.base.Spec,
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
	}
}

func (app *App) GetPods(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList {
	var pods corev1.PodList
	if app.isDeployment {
		replica := replicaset.GetReplicaSetsForOwner(ctx, t, resource, app.Name(), app.Namespace())
		require.NotNil(t, replica)
		pods = pod.GetPodsForOwner(ctx, t, resource, replica.Name, app.Namespace())
	} else {
		var p corev1.Pod
		require.NoError(t, resource.Get(ctx, app.Name(), app.Namespace(), &p))
		pods = corev1.PodList{Items: []corev1.Pod{p}}
	}

	return pods
}

func (app *App) Restart() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pods := app.GetPods(ctx, t, resource)

		deletePods(t, ctx, pods, resource)

		if app.isDeployment {
			require.NoError(t, deployment.WaitUntilReady(resource, app.build().(*appsv1.Deployment)))
		} else {
			ctx = app.Install()(ctx, t, envConfig)
		}

		return ctx
	}
}

func deletePods(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources) {
	for _, podItem := range pods.Items {
		require.NoError(t, resource.Delete(ctx, &podItem))
		require.NoError(t, wait.For(
			conditions.New(resource).ResourceDeleted(&podItem)), wait.WithTimeout(1*time.Minute))
	}
}

func printEventList(t *testing.T, ctx context.Context, resource *resources.Resources, namespace string) {
	optFunc := func(options *metav1.ListOptions) {
		options.Limit = int64(300)
		options.FieldSelector = fmt.Sprint(fields.OneTermEqualSelector("type", corev1.EventTypeWarning))
	}
	events := event.List(t, ctx, resource, namespace, optFunc)
	t.Log("events", events)
}
