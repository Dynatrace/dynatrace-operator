//go:build e2e

package standard

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/bootstrapper"
	classicToCloud "github.com/Dynatrace/dynatrace-operator/test/e2e/features/classic/switchmodes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/csimigration"
	noInjection "github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/noinjection"
	cloudnativeStandard "github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/standard"
	cloudToClassic "github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/switchmodes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/hostmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/publicregistry"
	supportArchive "github.com/Dynatrace/dynatrace-operator/test/e2e/features/supportarchive"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/usepublicregistry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
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
		operator.InstallLocal(true),
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.Uninstall(true))
	}

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
			logs.WriteOperatorLogToFile(ctx, c, t)
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

func TestStandard_cloudnative_codemodules_migrate_to_image(t *testing.T) {
	testEnv.Test(t, codemodules.MigrateToImage(t))
}

func TestStandard_cloudnative_codemodules_migrate_to_node_image_pull(t *testing.T) {
	testEnv.Test(t, codemodules.MigrateToNodeImagePull(t))
}

func TestStandard_public_registry_images(t *testing.T) {
	testEnv.Test(t, publicregistry.Feature(t))
}

func TestStandard_use_public_registry_oneagent(t *testing.T) {
	testEnv.Test(t, usepublicregistry.OneAgent(t))
}

func TestStandard_use_public_registry_oneagent_with_override(t *testing.T) {
	testEnv.Test(t, usepublicregistry.OneAgentWithOverride(t))
}

func TestStandard_use_public_registry_activegate(t *testing.T) {
	testEnv.Test(t, usepublicregistry.ActiveGate(t))
}

func TestStandard_use_public_registry_activegate_with_override(t *testing.T) {
	testEnv.Test(t, usepublicregistry.ActiveGateWithOverride(t))
}

func TestStandard_use_public_registry_codemodules_with_csi(t *testing.T) {
	testEnv.Test(t, usepublicregistry.CodeModulesWithCSI(t))
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

func TestStandard_pgc_with_fullstack(t *testing.T) {
	testEnv.Test(t, bootstrapper.PGCWithCloudNativeFullStack(t))
}

func TestStandard_host_agent_pgc_host_monitoring(t *testing.T) {
	testEnv.Test(t, hostmonitoring.HostAgentPGC(t))
}

func TestStandard_host_agent_pgc_cloudnative(t *testing.T) {
	testEnv.Test(t, cloudnative.HostAgentPGC(t))
}

func TestStandard_cloudnative_csi_migration(t *testing.T) {
	testEnv.Test(t, csimigration.Feature(t))
}
