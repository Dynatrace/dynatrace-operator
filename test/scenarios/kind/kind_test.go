//go:build e2e

package kind

import (
	"github.com/Dynatrace/dynatrace-operator/test/features/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"os"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
	"testing"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv, _ = env.NewFromFlags()
	kindClusterName := "andrii"

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), kindClusterName),
		helpers.SetScheme,
		operator.InstallViaMake(false),
	)

	testenv.Finish(
		operator.UninstallViaMake(false),
		envfuncs.DestroyCluster(kindClusterName),
	)
	os.Exit(testenv.Run(m))
}


func TestNoCSI_edgeconnect_install(t *testing.T) {
	testenv.Test(t, edgeconnect.NormalModeFeature(t))
}
