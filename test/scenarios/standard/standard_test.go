//go:build e2e

package standard

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/applicationmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/classic"
	classicToCloud "github.com/Dynatrace/dynatrace-operator/test/features/classic/switch_modes"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	cloudnativeDefault "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/default"
	disabledAutoInjection "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/disabled_auto_injection"
	cloudToClassic "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/switch_modes"
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/features/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/features/publicregistry"
	supportArchive "github.com/Dynatrace/dynatrace-operator/test/features/support_archive"
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
		activegate.Feature(t, nil),
		cloudnativeDefault.Feature(t, false),
		applicationmonitoring.MetadataEnrichment(t),
		applicationmonitoring.LabelVersionDetection(t),
		applicationmonitoring.ReadOnlyCSIVolume(t),
		applicationmonitoring.WithoutCSI(t),
		codemodules.InstallFromImage(t),
		publicregistry.Feature(t),
		extensions.Feature(t),
		disabledAutoInjection.Feature(t),
		supportArchive.Feature(t),
		edgeconnect.NormalModeFeature(t),
		edgeconnect.ProvisionerModeFeature(t),
		edgeconnect.AutomationModeFeature(t),
		classic.Feature(t),
		classicToCloud.Feature(t),
		cloudToClassic.Feature(t),
	}

	testEnv.Test(t, scenarios.FilterFeatures(*cfg, feats)...)
}
