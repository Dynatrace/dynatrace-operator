//go:build e2e

package oneagent

import (
	"bytes"
	"context"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	uninstallOneAgentDaemonSetName = "uninstall-oneagent"
)

var (
	uninstallOneAgentDaemonSetPath = path.Join(project.TestDataDir(), "oneagent/uninstall-oneagent.yaml")
)

func RunClassicUninstall(builder *features.FeatureBuilder, level features.Level, testDynakube dynatracev1beta1.DynaKube) {
	builder.WithStep("clean up OneAgent files from nodes", level, createUninstallDaemonSet(testDynakube))
	builder.WithStep("wait for daemonset", level, waitForUninstallDaemonset(testDynakube.Namespace))
	builder.WithStep("OneAgent files removed from nodes", level, executeUninstall(testDynakube.Namespace))
	builder.WithStep("clean up removed", level, removeUninstallDaemonset(testDynakube.Namespace))
}

func createUninstallDaemonSet(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		uninstallDaemonSet := manifests.ObjectFromFile[*appsv1.DaemonSet](t, uninstallOneAgentDaemonSetPath)
		uninstallDaemonSet.Namespace = dynakube.Namespace
		uninstallDaemonSet.Spec.Template.Spec.Tolerations = dynakube.Spec.OneAgent.ClassicFullStack.Tolerations
		resource := envConfig.Client().Resources()
		require.NoError(t, resource.Create(ctx, uninstallDaemonSet))

		return ctx
	}
}

func waitForUninstallDaemonset(namespace string) features.Func {
	return helpers.ToFeatureFunc(daemonset.WaitFor(uninstallOneAgentDaemonSetName, namespace), false)
}

func executeUninstall(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		require.NoError(t, daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      uninstallOneAgentDaemonSetName,
			Namespace: namespace,
		}).ForEachPod(cleanUpNodeConsumer(ctx, t, resource)))

		return ctx
	}
}

func removeUninstallDaemonset(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		require.NoError(t, daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      uninstallOneAgentDaemonSetName,
			Namespace: namespace,
		}).Delete())

		return ctx
	}
}

func cleanUpNodeConsumer(ctx context.Context, t *testing.T, resource *resources.Resources) daemonset.PodConsumer {
	return func(pod corev1.Pod) {
		stdOut := bytes.NewBuffer([]byte{})
		stdErr := bytes.NewBuffer([]byte{})
		if err := resource.ExecInPod(ctx, pod.Namespace, pod.Name, "uninstall-oneagent", []string{"/bin/sh", "-c", "chroot /mnt/root /opt/dynatrace/oneagent/agent/uninstall.sh"}, stdOut, stdErr); err != nil {
			t.Logf("uninstall.sh script failed for pod:'%s' (node:'%s') because of the error:'%s'", pod.Name, pod.Spec.NodeName, err)
			t.Logf("stdout: %s", stdOut.String())
			t.Logf("stderr: %s", stdErr.String())
		}
	}
}
