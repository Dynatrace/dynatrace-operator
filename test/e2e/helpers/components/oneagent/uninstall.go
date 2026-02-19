//go:build e2e

package oneagent

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
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
	uninstallOneAgentDaemonSetPath = filepath.Join(project.TestDataDir(), "oneagent/uninstall-oneagent.yaml")
)

func RunClassicUninstall(builder *features.FeatureBuilder, level features.Level, testDynakube dynakube.DynaKube) {
	builder.WithStep("clean up OneAgent files from nodes", level, createUninstallDaemonSet(testDynakube))
	builder.WithStep("wait for daemonset", level, waitForUninstallDaemonset(testDynakube.Namespace))
	builder.WithStep("OneAgent files removed from nodes", level, executeUninstall(testDynakube.Namespace))
	builder.WithStep("clean up removed", level, removeUninstallDaemonset(testDynakube.Namespace))
}

func createUninstallDaemonSet(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		uninstallDaemonSet := manifests.ObjectFromFile[*appsv1.DaemonSet](t, uninstallOneAgentDaemonSetPath)
		uninstallDaemonSet.Namespace = dk.Namespace
		uninstallDaemonSet.Spec.Template.Spec.Tolerations = dk.Spec.OneAgent.ClassicFullStack.Tolerations
		resource := envConfig.Client().Resources()
		require.NoError(t, resource.Create(ctx, uninstallDaemonSet))

		return ctx
	}
}

func waitForUninstallDaemonset(namespace string) features.Func {
	return helpers.ToFeatureFunc(k8sdaemonset.WaitFor(uninstallOneAgentDaemonSetName, namespace), false)
}

func executeUninstall(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		require.NoError(t, k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      uninstallOneAgentDaemonSetName,
			Namespace: namespace,
		}).ForEachPod(cleanUpNodeConsumer(ctx, t, resource)))

		return ctx
	}
}

func removeUninstallDaemonset(namespace string) features.Func {
	return k8sdaemonset.Delete(uninstallOneAgentDaemonSetName, namespace)
}

func cleanUpNodeConsumer(ctx context.Context, t *testing.T, resource *resources.Resources) k8sdaemonset.PodConsumer {
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
