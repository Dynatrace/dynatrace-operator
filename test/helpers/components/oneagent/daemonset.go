//go:build e2e

package oneagent

import (
	"bytes"
	"context"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
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

func WaitForDaemonset(dynakube dynatracev1beta1.DynaKube) features.Func {
	return daemonset.WaitFor(dynakube.OneAgentDaemonsetName(), dynakube.Namespace)
}

func WaitForDaemonSetPodsDeletion(dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitForPodsDeletionWithOwner(dynakube.OneAgentDaemonsetName(), dynakube.Namespace)
}

func Get(ctx context.Context, resource *resources.Resources, dynakube dynatracev1beta1.DynaKube) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dynakube.OneAgentDaemonsetName(),
		Namespace: dynakube.Namespace,
	}).Get()
}

func CreateUninstallDaemonSet(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		oaDaemonSet := manifests.ObjectFromFile[*appsv1.DaemonSet](t, uninstallOneAgentDaemonSetPath)
		oaDaemonSet.Namespace = namespace
		resource := environmentConfig.Client().Resources()
		require.NoError(t, resource.Create(ctx, oaDaemonSet))
		return ctx
	}
}

func WaitForUninstallOneAgentDaemonset(namespace string) features.Func {
	return daemonset.WaitFor(uninstallOneAgentDaemonSetName, namespace)
}

func CleanUpEachNode(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		require.NoError(t, daemonset.ForEachPod(ctx, resource, uninstallOneAgentDaemonSetName, namespace, cleanUpNodeConsumer(ctx, t, resource)))
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
