//go:build e2e

package standard

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/bootstrapper"
	classicToCloud "github.com/Dynatrace/dynatrace-operator/test/features/classic/switchmodes"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	noInjection "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/noinjection"
	cloudnativeStandard "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/standard"
	cloudToClassic "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/switchmodes"
	"github.com/Dynatrace/dynatrace-operator/test/features/publicregistry"
	supportArchive "github.com/Dynatrace/dynatrace-operator/test/features/supportarchive"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios"
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

	if scenarios.InstallViaHelm() {
		testEnv.Setup(
			helpers.SetScheme,
			operator.InstallViaHelm(operator.QuayRegistryURL, os.Getenv(scenarios.HelmChartTagEnvVar), true, operator.DefaultNamespace),
		)
	} else {
		testEnv.Setup(
			helpers.SetScheme,
			operator.InstallViaMake(true),
		)
	}

	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestStandard_cloudnative(t *testing.T) {
	testEnv.Test(t, cloudnativeStandard.Feature(t, false, true))
}

func TestStandard_cloudnative_codemodules_image(t *testing.T) {
	testEnv.Test(t, codemodules.InstallFromImage(t))
}

func TestStandard_public_registry_images(t *testing.T) {
	testEnv.Test(t, publicregistry.Feature(t))
}

func TestStandard_cloudnative_disabled_auto_inject(t *testing.T) {
	testEnv.Test(t, noInjection.Feature(t))
}

func TestStandard_support_archive(t *testing.T) {
	testEnv.Test(t, supportArchive.Feature(t))
}

func TestStandard_classic_to_cloudnative(t *testing.T) {
	testEnv.Test(t, classicToCloud.Feature(t))
}

func TestStandard_cloudnative_to_classic(t *testing.T) {
	testEnv.Test(t, cloudToClassic.Feature(t))
}

func TestStandard_node_image_pull_with_csi(t *testing.T) {
	testEnv.Test(t, bootstrapper.InstallWithCSI(t))
}
