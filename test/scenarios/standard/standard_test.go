//go:build e2e

package standard

import (
	"github.com/Dynatrace/dynatrace-operator/test/features/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/features/applicationmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/features/classic"
	classicToCloud "github.com/Dynatrace/dynatrace-operator/test/features/classic/switch_modes"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	cloudnativeDefault "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/default"
	disabledAutoInjection "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/disabled_auto_injection"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/network_zones"
	publicRegistry "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/public_registry"
	cloudToClassic "github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/switch_modes"
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	supportArchive "github.com/Dynatrace/dynatrace-operator/test/features/support_archive"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnv env.Environment

func TestMain(m *testing.M) {
	testEnv = environment.GetStandardKubeClusterEnvironment()
	testEnv.Setup(operator.InstallViaMake(true))
	testEnv.Finish(operator.UninstallViaMake(true))
	testEnv.Run(m)
}

func TestStandard(t *testing.T) {
	feats := []features.Feature{
		activegate.Feature(t, nil),
		cloudnativeDefault.Feature(t, false),
		applicationmonitoring.DataIngest(t),
		applicationmonitoring.LabelVersionDetection(t),
		applicationmonitoring.ReadOnlyCSIVolume(t),
		applicationmonitoring.WithoutCSI(t),
		codemodules.InstallFromImage(t),
		disabledAutoInjection.Feature(t),
		supportArchive.Feature(t),
		edgeconnect.Feature(t),
		classic.Feature(t),
		classicToCloud.Feature(t),
		publicRegistry.Feature(t),
		cloudToClassic.Feature(t),
		network_zones.Feature(t),
	}
	testEnv.Test(t, feats...)
}
