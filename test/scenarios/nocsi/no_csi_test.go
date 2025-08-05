//go:build e2e

package nocsi

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/applicationmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/test/features/classic"
	cloudnativeStandard "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/standard"
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/features/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/features/hostmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/event"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog/v2"
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
		cloudnativeStandard.Feature(t, false, false),
		telemetryingest.WithLocalActiveGateAndCleanup(t),
		telemetryingest.WithPublicActiveGate(t),
		telemetryingest.WithTelemetryIngestEndpointTLS(t),
		telemetryingest.OtelCollectorConfigUpdate(t),
	}

	testEnv.AfterEachFeature(func(ctx context.Context, c *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		if t.Failed() {
			klog.InfoS("feature failed", "f", f.Name(), "failed", t.Failed())

			resource := c.Client().Resources()

			optFunc := func(options *metav1.ListOptions) {
				options.Limit = int64(300)
				options.FieldSelector = fmt.Sprint(fields.OneTermEqualSelector("type", corev1.EventTypeWarning))
			}

			events := event.List(t, ctx, resource, "dynatrace", optFunc)

			klog.InfoS("Events list", "events total", len(events.Items))
			for _, eventItem := range events.Items {
				klog.InfoS("Event", "name", eventItem.Name, "message", eventItem.Message, "reason", eventItem.Reason, "type", eventItem.Type)
			}

			pods := &corev1.PodList{}
			err := resource.List(ctx, pods)
			if err != nil {
				klog.Error(err)
			}

			for _, pod := range pods.Items {
				klog.InfoS("Pod list", "pod", pod.Name, "status", pod.Status.Phase)
			}
		}

		return ctx, nil
	})

	testEnv.Test(t, scenarios.FilterFeatures(*cfg, feats)...)
}
