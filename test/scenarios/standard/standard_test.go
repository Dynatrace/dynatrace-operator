//go:build e2e

package standard

import (
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
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
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
		operator.InstallViaMake(true),
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
	}
	testEnv.Run(m)
}

func TestStandard(t *testing.T) {
	feats := []features.Feature{
		cloudnativeStandard.Feature(t, false, true),
		codemodules.InstallFromImage(t),
		publicregistry.Feature(t),
		noInjection.Feature(t),
		supportArchive.Feature(t),
		classicToCloud.Feature(t),
		cloudToClassic.Feature(t),
		bootstrapper.InstallWithCSI(t),
	}

	testEnv.Test(t, scenarios.FilterFeatures(*cfg, feats)...)
}
