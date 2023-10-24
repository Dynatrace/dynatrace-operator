//go:build e2e

package release

import (
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/upgrade"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnv env.Environment

func TestMain(m *testing.M) {
	testEnv = environment.GetStandardKubeClusterEnvironment()
	testEnv.Setup(operator.InstallViaHelm("0.11.1", true, "dynatrace"))
	testEnv.Finish(operator.UninstallViaMake(true))
	testEnv.Run(m)
}

func TestRelease(t *testing.T) {
	feats := []features.Feature{
		upgrade.Feature(t),
	}
	testEnv.Test(t, feats...)
}
