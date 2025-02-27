//go:build e2e

package codemodules

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	RuxitAgentProcFile = "ruxitagentproc.conf"
)

func CheckRuxitAgentProcFileHasNoConnInfo(testDynakube dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		var dk dynakube.DynaKube
		require.NoError(t, e.Client().Resources().Get(ctx, testDynakube.Name, testDynakube.Namespace, &dk))

		err := daemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: testDynakube.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			// /data/codemodules/1.273.0.20230719-145632/agent/conf/ruxitagentproc.conf
			dir := filepath.Join("/data", "codemodules", dk.OneAgent().GetCodeModulesVersion(), "agent", "conf", RuxitAgentProcFile)
			readFileCommand := shell.ReadFile(dir)
			result, err := pod.Exec(ctx, resources, podItem, "provisioner", readFileCommand...)
			require.NoError(t, err)
			assert.NotContains(t, result.StdOut.String(), "tenant")
			assert.NotContains(t, result.StdOut.String(), "tenantToken")
		})

		require.NoError(t, err)

		return ctx
	}
}
