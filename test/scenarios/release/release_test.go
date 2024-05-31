//go:build e2e

package release

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnv env.Environment
var thresholdVersion, _ = dtversion.ToSemver("1.2.0")

const releaseTag = "1.1.0"

func TestMain(m *testing.M) {
	cfg := environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(
		tenant.CreateOtelSecret(operator.DefaultNamespace),
		operator.InstallViaHelm(releaseTag, true, operator.DefaultNamespace),
	)
	// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
	if !cfg.FailFast() {
		testEnv.Finish(operator.UninstallViaMake(true))
	}
	testEnv.Run(m)
}

func TestRelease(t *testing.T) {
	usesOldVersion := false
	usedSemVer, _ := dtversion.ToSemver(releaseTag)
	if semver.Compare(thresholdVersion, usedSemVer) >= 1 {
		usesOldVersion = true
	}
	feats := []features.Feature{
		upgrade.Feature(t, usesOldVersion),
	}
	testEnv.Test(t, feats...)
}
