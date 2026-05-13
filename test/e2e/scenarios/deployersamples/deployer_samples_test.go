//go:build e2e

package deployersamples

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/deployersamples"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/environment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/logs"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)

	testEnv.Setup(
		helpers.SetScheme,
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			return deployersamples.SharedSAFile().Install(ctx, c)
		},
	)

	if !cfg.FailFast() {
		testEnv.Finish(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			return deployersamples.SharedSAFile().Uninstall(ctx, c)
		})
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
			logs.WriteOperatorLog(ctx, c, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestDeployerSamples(t *testing.T) {
	for _, feat := range deployersamples.AllFeatures(t) {
		testEnv.Test(t, feat)
	}
}

func TestDeployerSamplesNegative(t *testing.T) {
	testEnv.Test(t, deployersamples.NegativeFeature(t))
}
