//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/interface"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type SampleDeployment struct {
	sampleApp
}

func NewSampleDeployment(t *testing.T, testDynakube dynatracev1beta1.DynaKube) sampleapps.SampleApp {
	return &SampleDeployment{
		sampleApp: *newSampleApp(t, testDynakube),
	}
}

func (app SampleDeployment) Name() string {
	if app.name == "" {
		return fmt.Sprintf("%s-sample-deployment", app.testDynakube.Name)
	}
	return app.name
}

func (app SampleDeployment) Build() client.Object {
	replicas := int32(2)
	selectorKey := "app"
	selectorValue := app.Name()
	objectMeta := app.createObjectMeta()
	objectMeta.Labels[selectorKey] = selectorValue

	result := &appsv1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorKey: selectorValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objectMeta,
				Spec:       app.base.Spec,
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
	}
	result.Spec.Template.Spec.Containers[0].Name = app.ContainerName()
	return result
}

func (app SampleDeployment) Install() features.Func {
	return app.install(app.Build())
}

func (app SampleDeployment) Uninstall() features.Func {
	return app.uninstallObject(app.Build())
}

func (app SampleDeployment) Get(ctx context.Context, t *testing.T, resource *resources.Resources) client.Object {
	var deployment appsv1.Deployment

	require.NoError(t, resource.Get(ctx, app.Name(), app.Namespace().Name, &deployment))
	return &deployment
}

func (app SampleDeployment) GetPods(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList {
	return pod.GetPodsForOwner(ctx, t, resource, app.Name(), app.Namespace().Name)
}

func (app SampleDeployment) RestartHalf(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	return app.doRestart(ctx, t, config, restartHalf)
}

func (app SampleDeployment) Restart(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	return app.doRestart(ctx, t, config, restart)
}

func (app SampleDeployment) doRestart(ctx context.Context, t *testing.T, config *envconf.Config, restartFunc restartFunc) context.Context {
	resource := config.Client().Resources()
	pods := app.GetPods(ctx, t, resource)

	restartFunc(t, ctx, pods, resource)

	deployment.WaitUntilReady(resource, app.Build().(*appsv1.Deployment))

	return ctx
}
