//go:build e2e

package resourceattributes

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// LogmonOnly verifies that spec.resourceAttributes are propagated
// as -p key=value args into the LogMonitoring DaemonSet init container.
// No user-pod injection happens in the standalone log module scenario, so no sample app is needed.
func LogmonOnly(t *testing.T) features.Feature {
	builder := features.New("static-resource-logmon-only")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithLogMonitoring(),
		dynakubeComponents.WithLogMonitoringImageRef(t),
		dynakubeComponents.WithResourceAttributes(globalAttrs),
		devRegistryOptions(),
	)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	builder.Assess("LogMonitoring DaemonSet is ready", k8sdaemonset.IsReady(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("LogMonitoring init container args contain global resource attributes", assessLogMonitoringInitArgs(testDynakube, globalAttrs))

	return builder.Feature()
}

func assessLogMonitoringInitArgs(dk dynakube.DynaKube, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ds, err := k8sdaemonset.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      dk.LogMonitoring().GetDaemonSetName(),
			Namespace: dk.Namespace,
		}).Get()
		require.NoError(t, err)
		require.NotEmpty(t, ds.Spec.Template.Spec.InitContainers, "no init containers in LogMonitoring DaemonSet")

		var allArgs []string
		for _, c := range ds.Spec.Template.Spec.InitContainers {
			allArgs = append(allArgs, c.Args...)
		}
		joinedArgs := strings.Join(allArgs, " ")

		for k, v := range expected {
			assert.Containsf(t, joinedArgs, "-p "+k+"="+v, "LogMonitoring init args missing -p %s=%s", k, v)
		}

		return ctx
	}
}
