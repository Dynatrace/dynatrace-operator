//go:build e2e

package nocsi

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/applicationmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/test/features/classic"
	cloudnativeStandard "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/standard"
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/features/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/features/extensions/dbexecutor"
	"github.com/Dynatrace/dynatrace-operator/test/features/hostmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/kspm"
	"github.com/Dynatrace/dynatrace-operator/test/features/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/telemetryingest"
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

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})
	testEnv.Run(m)
}

func TestNoCSI_activegate(t *testing.T) {
	testEnv.Test(t, activegate.Feature(t, nil))
}

func TestNoCSI_metadata_enrichment(t *testing.T) {
	testEnv.Test(t, applicationmonitoring.MetadataEnrichment(t))
}

func TestNoCSI_otlp_exporter_configuration(t *testing.T) {
	testEnv.Test(t, applicationmonitoring.OTLPExporterConfiguration(t))
}

func TestNoCSI_labelversion(t *testing.T) {
	testEnv.Test(t, applicationmonitoring.LabelVersionDetection(t))
}

func TestNoCSI_app_monitoring_without_csi(t *testing.T) {
	testEnv.Test(t, applicationmonitoring.WithoutCSI(t))
}

func TestNoCSI_extensions(t *testing.T) {
	testEnv.Test(t, extensions.Feature(t))
}

func TestNoCSI_edgeconnect_install(t *testing.T) {
	testEnv.Test(t, edgeconnect.NormalModeFeature(t))
}

func TestNoCSI_edgeconnect_install_provisioner(t *testing.T) {
	testEnv.Test(t, edgeconnect.ProvisionerModeFeature(t))
}

func TestNoCSI_edgeconnect_install_proxy_http(t *testing.T) {
	testEnv.Test(t, edgeconnect.WithHTTPProxy(t))
}

func TestNoCSI_edgeconnect_install_proxy_https(t *testing.T) {
	testEnv.Test(t, edgeconnect.WithHTTPSProxy(t))
}

func TestNoCSI_custom_edgeconnect(t *testing.T) {
	testEnv.Test(t, edgeconnect.AutomationModeFeature(t))
}

func TestNoCSI_classic(t *testing.T) {
	testEnv.Test(t, classic.Feature(t))
}

func TestNoCSI_node_image_pull_with_no_csi(t *testing.T) {
	testEnv.Test(t, bootstrapper.NoCSI(t))
}

func TestNoCSI_logmonitoring(t *testing.T) {
	testEnv.Test(t, logmonitoring.Feature(t))
}

func TestNoCSI_logmonitoring_with_optional_scopes(t *testing.T) {
	testEnv.Test(t, logmonitoring.WithOptionalScopes(t))
}

func TestNoCSI_host_monitoring_without_csi(t *testing.T) {
	testEnv.Test(t, hostmonitoring.WithoutCSI(t))
}

func TestNoCSI_cloudnative(t *testing.T) {
	const istioEnabled, withCSI = false, false
	testEnv.Test(t, cloudnativeStandard.Feature(t, istioEnabled, withCSI))
}

func TestNoCSI_telemetryingest_w_local_ag_and_cleanup_after(t *testing.T) {
	testEnv.Test(t, telemetryingest.WithLocalActiveGateAndCleanup(t))
}

func TestNoCSI_telemetryingest_w_public_ag(t *testing.T) {
	testEnv.Test(t, telemetryingest.WithPublicActiveGate(t))
}

func TestNoCSI_telemetryingest_w_otel_collector_endpoint_tls(t *testing.T) {
	testEnv.Test(t, telemetryingest.WithTelemetryIngestEndpointTLS(t))
}

func TestNoCSI_telemetryingest_configuration_update(t *testing.T) {
	testEnv.Test(t, telemetryingest.OtelCollectorConfigUpdate(t))
}

func TestNoCSI_kspm(t *testing.T) {
	testEnv.Test(t, kspm.Feature(t))
}

func TestNoCSI_extensions_db_executor(t *testing.T) {
	testEnv.Test(t, dbexecutor.Feature(t))
}
