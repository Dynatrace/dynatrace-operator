//go:build e2e

package oneagent

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func OSAgentCanConnect(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())

		require.NoError(t, err)
		require.NoError(t, ForEachPod(ctx, resource, dynakube, func(pod corev1.Pod) {
			logStream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(ctx)

			require.NoError(t, err)

			logs.AssertContains(t, logStream, "[oneagentos] [PingReceiver] Ping received: Healthy(0)")
		}))

		return ctx
	}
}
