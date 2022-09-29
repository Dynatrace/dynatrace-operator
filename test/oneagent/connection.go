package oneagent

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/test/log"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func OSAgentCanConnect() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())

		require.NoError(t, err)
		require.NoError(t, ForEachPod(ctx, resource, func(pod v1.Pod) {
			logStream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{}).Stream(ctx)

			require.NoError(t, err)

			log.AssertLogContains(t, logStream, "[oneagentos] [PingReceiver] Ping received: Healthy(0)")
		}))

		return ctx
	}
}
