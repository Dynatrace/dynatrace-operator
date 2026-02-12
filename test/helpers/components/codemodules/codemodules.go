//go:build e2e

package codemodules

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	RuxitAgentProcFile = "ruxitagentproc.conf"
	interval           = 2 * time.Second
	timeout            = 1 * time.Minute
)

func CheckRuxitAgentProcFileHasNoConnInfo(testDynakube dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		var dk dynakube.DynaKube
		require.NoError(t, e.Client().Resources().Get(ctx, testDynakube.Name, testDynakube.Namespace, &dk))

		err := k8sdaemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: testDynakube.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			// /data/codemodules/1.318.0.20250609-191530/agent/conf/ruxitagentproc.conf
			dir := filepath.Join("/data", "codemodules", dk.OneAgent().GetCodeModulesVersion(), "agent", "conf", RuxitAgentProcFile)
			err := wait.For(func(ctx context.Context) (done bool, err error) {
				result, err := k8spod.Exec(ctx, resources, podItem, "provisioner", shell.ReadFile(dir)...)
				if err != nil {
					if strings.Contains(result.StdErr.String(), "No such file or directory") {
						return false, nil
					}

					return false, err
				}
				assert.NotContains(t, result.StdOut.String(), "tenant")
				assert.NotContains(t, result.StdOut.String(), "tenantToken")

				return true, nil
			}, wait.WithTimeout(timeout), wait.WithInterval(interval))
			require.NoError(t, err)
		})

		require.NoError(t, err)

		return ctx
	}
}
