//go:build e2e

package hostmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ApplicationMonitoring deployment without CSI driver
func WithoutCSI(t *testing.T) features.Feature {
	builder := features.New("host-monitoring-without-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakube.Option{
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
	}
	testDynakube := *dynakube.New(options...)

	// Register dynakube install
	dynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("one agent started", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))

	return builder.Feature()
}
