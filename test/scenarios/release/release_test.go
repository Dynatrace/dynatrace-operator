//go:build e2e

package release

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

const releaseTag = "1.5.1"

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(
		helpers.SetScheme,
		operator.InstallViaHelm(releaseTag, true, operator.DefaultNamespace), // TODO: add logic to get releaseTag in a dynamic way instead of hard coding it
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			// Log events if the test failed
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestRelease(t *testing.T) {
	testEnv.Test(t, upgrade.Feature(t))
}
