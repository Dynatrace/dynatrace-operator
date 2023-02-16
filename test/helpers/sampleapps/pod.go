//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type SamplePod struct {
	sampleApp
}

func NewSamplePod(t *testing.T, testDynakube dynatracev1beta1.DynaKube) SampleApp {
	return &SamplePod{
		sampleApp: *newSampleApp(t, testDynakube),
	}
}

func (app SamplePod) Name() string {
	if app.name == "" {
		return fmt.Sprintf("%s-sample-pod", app.testDynakube.Name)
	}
	return app.name
}

func (app SamplePod) Build() client.Object {
	result := &corev1.Pod{
		ObjectMeta: app.createObjectMeta(),
		Spec:       app.base.Spec,
	}
	result.Spec.Containers[0].Name = app.ContainerName()
	return result
}

func (app SamplePod) Install() features.Func {
	return app.install(app.Build())
}

func (app SamplePod) Uninstall() features.Func {
	return app.uninstallObject(app.Build())
}

func (app SamplePod) Get(ctx context.Context, t *testing.T, resource *resources.Resources) client.Object {
	var pod corev1.Pod
	require.NoError(t, resource.Get(ctx, app.Name(), app.Namespace().Name, &pod))
	return &pod
}

func (app SamplePod) GetPods(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList {
	pod := app.Get(ctx, t, resource).(*corev1.Pod)
	return corev1.PodList{Items: []corev1.Pod{*pod}}
}

func (app SamplePod) RestartHalf(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	return app.doRestart(ctx, t, config, restartHalf)
}

func (app SamplePod) Restart(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	return app.doRestart(ctx, t, config, restart)
}

func (app SamplePod) doRestart(ctx context.Context, t *testing.T, config *envconf.Config, restartFunc restartFunc) context.Context {
	resource := config.Client().Resources()
	restartFunc(t, ctx, app.GetPods(ctx, t, resource), resource)

	pod := app.Build().(*corev1.Pod)
	app.install(pod)

	return ctx
}
