//go:build e2e

package no_csi

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/applicationmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/test/features/classic"
	cloudnativeDefault "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/default"
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/features/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/features/hostmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/telemetryingest"
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
		operator.InstallViaMake(false),
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(false))
	}
	testEnv.Run(m)
}

func TestNoCSI(t *testing.T) {
	feats := []features.Feature{
		activegate.Feature(t, nil),
		applicationmonitoring.MetadataEnrichment(t),
		applicationmonitoring.LabelVersionDetection(t),
		applicationmonitoring.WithoutCSI(t),
		extensions.Feature(t),
		edgeconnect.NormalModeFeature(t),
		edgeconnect.ProvisionerModeFeature(t),
		edgeconnect.AutomationModeFeature(t),
		classic.Feature(t),
		bootstrapper.NoCSI(t),
		logmonitoring.Feature(t),
		hostmonitoring.WithoutCSI(t),
		cloudnativeDefault.Feature(t, false, false),
		telemetryingest.WithLocalActiveGateAndCleanup(t),
		telemetryingest.WithPublicActiveGate(t),
		telemetryingest.WithTelemetryIngestEndpointTLS(t),
		telemetryingest.OtelCollectorConfigUpdate(t),
	}

	testEnv.Test(t, scenarios.FilterFeatures(*cfg, feats)...)
}
