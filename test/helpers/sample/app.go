//go:build e2e

package sample

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sevent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sreplicaset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	defaultNameTemplate        = "sample-%s"
	podTemplatePath            = filepath.Join(project.TestDataDir(), "sample-app/pod-base.yaml")
	serviceAccountTemplatePath = filepath.Join(project.TestDataDir(), "sample-app/serviceaccount.yaml")
	clusterRolePath            = filepath.Join(project.TestDataDir(), "sample-app/clusterrole.yaml")
	bindingPath                = filepath.Join(project.TestDataDir(), "sample-app/binding.yaml")
)

type App struct {
	t         *testing.T
	base      *corev1.Pod
	scBase    *corev1.ServiceAccount
	owner     metav1.Object
	namespace corev1.Namespace

	installedNamespace bool
	isDeployment       bool
	withoutClusterRole bool
	canInitError       bool
}

type Option func(*App)

func NewApp(t *testing.T, owner metav1.Object, options ...Option) *App {
	base := manifests.ObjectFromFile[*corev1.Pod](t, podTemplatePath)
	base.Name = fmt.Sprintf(defaultNameTemplate, owner.GetName())
	base.Namespace = base.Name

	sc := manifests.ObjectFromFile[*corev1.ServiceAccount](t, serviceAccountTemplatePath)
	sc.Namespace = base.Name

	app := &App{
		t:         t,
		owner:     owner,
		base:      base,
		scBase:    sc,
		namespace: *k8snamespace.New(base.Namespace),
	}

	defaultOptions := []Option{
		WithFailurePolicy(true),
	}

	for _, opt := range append(defaultOptions, options...) {
		opt(app)
	}

	return app
}

func WithName(name string) Option {
	return func(app *App) {
		if app.base.Namespace == app.base.Name {
			app.base.Namespace = name
			app.namespace = *k8snamespace.New(name)
			app.scBase.Namespace = name
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
		app.scBase.Namespace = namespace.Name
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

func WithFailurePolicy(fail bool) Option {
	return func(app *App) {
		if app.base.Annotations == nil {
			app.base.Annotations = map[string]string{}
		}
		if fail {
			app.base.Annotations[mutator.AnnotationFailurePolicy] = "fail"
		} else {
			delete(app.base.Annotations, mutator.AnnotationFailurePolicy)
		}

		app.canInitError = fail
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

func WithPodSecurityContext(securityContext corev1.PodSecurityContext) Option {
	return func(app *App) {
		app.base.Spec.SecurityContext = &securityContext
	}
}

func WithContainerSecurityContext(securityContext corev1.SecurityContext) Option {
	return func(app *App) {
		app.base.Spec.Containers[0].SecurityContext = &securityContext
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

	return k8snamespace.Create(app.namespace)
}

func (app *App) CanInitError() bool {
	return app.canInitError
}

func WithoutClusterRole() Option {
	return func(app *App) {
		app.withoutClusterRole = true
	}
}

func (app *App) Install() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		resource := c.Client().Resources()
		if !app.installedNamespace {
			ctx = app.InstallNamespace()(ctx, t, c)
		}

		if !app.withoutClusterRole {
			ctx = app.installClusterRole(ctx, t, c)
		}

		require.NoError(t, resource.Create(ctx, app.scBase))

		object := app.build()
		require.NoError(t, resource.Create(ctx, object))

		if dep, ok := object.(*appsv1.Deployment); ok {
			err := k8sdeployment.WaitUntilReady(resource, dep)
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

func (app *App) InstallFail() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		resource := c.Client().Resources()
		if !app.installedNamespace {
			ctx = app.InstallNamespace()(ctx, t, c)
		}

		if !app.withoutClusterRole {
			ctx = app.installClusterRole(ctx, t, c)
		}

		require.NoError(t, resource.Create(ctx, app.scBase))

		object := app.build()
		require.NoError(t, resource.Create(ctx, object))

		if dep, ok := object.(*appsv1.Deployment); ok {
			err := k8sdeployment.WaitUntilFailedCreate(resource, dep)
			if err != nil {
				printEventList(t, ctx, resource, app.Namespace())
			}
			require.NoError(t, err)
		}

		return ctx
	}
}

func (app *App) installClusterRole(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		ctx = helpers.ToFeatureFunc(manifests.InstallFromFile(clusterRolePath), true)(ctx, t, c)
		binding := app.createBinding(t)

		err := c.Client().Resources().Create(ctx, binding)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			require.NoError(t, err)
		}
	}

	return ctx
}

func (app *App) createBinding(t *testing.T) *rbacv1.RoleBinding {
	binding := manifests.ObjectFromFile[*rbacv1.RoleBinding](t, bindingPath)
	binding.Namespace = app.Namespace()
	require.Len(t, binding.Subjects, 1)
	binding.Subjects[0].Namespace = app.Namespace()

	return binding
}

func (app *App) Uninstall() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		defer func() {
			app.installedNamespace = false
			ctx = app.uninstallClusterRole(ctx, t, c)
		}()
		resource := c.Client().Resources()
		object := app.build()

		require.NoError(t, resource.Delete(ctx, object))
		require.NoError(t, wait.For(conditions.New(resource).ResourceDeleted(object), wait.WithTimeout(2*time.Minute)))
		if dep, ok := object.(*appsv1.Deployment); ok {
			ctx = k8spod.WaitForDeletionWithOwner(dep.Name, dep.Namespace)(ctx, t, c)
		}

		return k8snamespace.Delete(app.Namespace())(ctx, t, c)
	}
}

func (app *App) UninstallFail() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		defer func() {
			app.installedNamespace = false
			ctx = app.uninstallClusterRole(ctx, t, c)
		}()
		resource := c.Client().Resources()
		object := app.build()

		require.NoError(t, resource.Delete(ctx, object))
		require.NoError(t, wait.For(conditions.New(resource).ResourceDeleted(object), wait.WithTimeout(2*time.Minute)))

		return k8snamespace.Delete(app.Namespace())(ctx, t, c)
	}
}

func (app *App) uninstallClusterRole(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		ctx = helpers.ToFeatureFunc(manifests.UninstallFromFile(clusterRolePath), true)(ctx, t, c)
		binding := app.createBinding(t)

		err := c.Client().Resources().Delete(ctx, binding)
		if err != nil && !k8serrors.IsNotFound(err) {
			require.NoError(t, err)
		}
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
	if app.isDeployment {
		replica := k8sreplicaset.GetReplicaSetsForOwner(ctx, t, resource, app.Name(), app.Namespace())
		require.NotNil(t, replica)

		return k8spod.ListForOwner(ctx, t, resource, replica.Name, app.Namespace())
	}

	var pod corev1.Pod
	require.NoError(t, resource.Get(ctx, app.Name(), app.Namespace(), &pod))

	return corev1.PodList{Items: []corev1.Pod{pod}}
}

func (app *App) Restart() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pods := app.GetPods(ctx, t, resource)

		deletePods(t, ctx, pods, resource)

		if app.isDeployment {
			require.NoError(t, k8sdeployment.WaitUntilReady(resource, app.build().(*appsv1.Deployment)))
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
	events := k8sevent.List(t, ctx, resource, namespace, optFunc)
	t.Log("events", events)
}
