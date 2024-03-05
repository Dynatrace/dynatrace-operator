//go:build e2e

package publicregistry

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	agPublicECR = "public.ecr.aws/dynatrace/dynatrace-activegate"
	oaPublicECR = "public.ecr.aws/dynatrace/dynatrace-oneagent"
	cmPublicECR = "public.ecr.aws/dynatrace/dynatrace-codemodules"
)

// Feature defines the e2e test to verify that public-registry images can be deployed by the operator and that they function
// This includes:
//   - ActiveGate StatefulSet gets ready
//   - CodeModules can be downloaded and mounted
//   - OneAgent DaemonSet gets ready
//
// It determines the latest version of each image using the registry.
func Feature(t *testing.T) features.Feature {
	builder := features.New("Public registry images")
	// Register operator install
	builder.WithLabel("name", "public-registry-images")
	secretConfig := tenant.GetSingleTenantSecret(t)

	oaSpec := cloudnative.DefaultCloudNativeSpec()
	oaSpec.Image = getLatestImageURI(t, oaPublicECR)
	oaSpec.CodeModulesImage = getLatestImageURI(t, cmPublicECR)

	options := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(oaSpec),
		dynakube.WithActiveGate(),
		dynakube.WithCustomActiveGateImage(getLatestImageURI(t, agPublicECR)),
	}
	testDynakube := *dynakube.New(options...)

	// Register sample app install
	sampleNamespace := *namespace.New("public-registry-sample")
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install - will verify OneAgent DaemonSet startup
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	// Install Sample apps - will check if CodeModule could be downloaded and mounted
	builder.Assess("install sample app", sampleApp.Install())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	// Check if the ActiveGate could start up
	builder.Assess("ActiveGate started", activegate.WaitForStatefulSet(&testDynakube, "activegate"))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func getLatestImageURI(t *testing.T, repoURI string) string {
	repo, err := name.NewRepository(repoURI)
	require.NoError(t, err)

	tags, err := remote.List(repo)
	slices.SortFunc(tags, func(a, b string) int {
		if strings.HasPrefix(a, "sha") {
			return -1
		}
		if strings.HasPrefix(b, "sha") {
			return 1
		}
		semverA, _ := dtversion.ToSemver(a)
		semverB, _ := dtversion.ToSemver(b)

		return semver.Compare(semverA, semverB)
	})
	require.NoError(t, err)

	return fmt.Sprintf("%s:%s", repoURI, tags[len(tags)-1])
}
