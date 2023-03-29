//go:build e2e

package sampleapps

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type SampleApp interface {
	Name() string
	ContainerName() string
	Namespace() *corev1.Namespace

	WithName(name string)
	WithNamespace(namespace corev1.Namespace)
	WithAnnotations(annotations map[string]string)
	WithLabels(labels map[string]string)
	WithEnvs(envs []corev1.EnvVar)
	Build() client.Object

	Install() features.Func
	Uninstall() features.Func
	Restart(ctx context.Context, t *testing.T, config *envconf.Config) context.Context
	RestartHalf(ctx context.Context, t *testing.T, config *envconf.Config) context.Context
	UninstallNamespace() features.Func

	Get(ctx context.Context, t *testing.T, resource *resources.Resources) client.Object
	GetPods(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList
}

