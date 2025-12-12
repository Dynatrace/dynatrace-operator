//go:build e2e

package release

import (
	"context"
	"testing"

	cloudnativeupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/upgrade"
	extensionsupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/extensions/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

const releaseTag = "1.7.1"

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(
		helpers.SetScheme,
		operator.Install(releaseTag, true), // TODO: add logic to get releaseTag in a dynamic way instead of hard coding it
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.Uninstall(true))
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestRelease_cloudnative_upgrade(t *testing.T) {
	testEnv.Test(t, cloudnativeupgrade.Feature(t))
}

func TestRelease_extensions_upgrade(t *testing.T) {
	testEnv.Test(t, extensionsupgrade.Feature(t))
}
