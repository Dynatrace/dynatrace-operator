//go:build e2e

package hostmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
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
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("one agent started", daemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))

	// Register sample, dynakube and operator uninstall
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
