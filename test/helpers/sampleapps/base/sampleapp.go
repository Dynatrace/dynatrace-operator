//go:build e2e

package base

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type App interface {
	Name() string
	ContainerName() string
	Namespace() *corev1.Namespace

	WithName(name string)
	WithNamespace(namespace corev1.Namespace)
	WithAnnotations(annotations map[string]string)
	WithLabels(labels map[string]string)
	WithEnvs(envs []corev1.EnvVar)
	WithSecurityContext(securityContext corev1.PodSecurityContext)
	Build() client.Object

	Install() features.Func
	Uninstall() features.Func
	Restart(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context
	RestartHalf(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context
	InstallNamespace() features.Func
	UninstallNamespace() features.Func

	Get(ctx context.Context, t *testing.T, resource *resources.Resources) client.Object
	GetPods(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList
}
