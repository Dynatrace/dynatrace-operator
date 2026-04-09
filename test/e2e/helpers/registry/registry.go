//go:build e2e

package registry

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

const (
	AgPublicECR     = "public.ecr.aws/dynatrace/dynatrace-activegate"
	OAPublicECR     = "public.ecr.aws/dynatrace/dynatrace-oneagent"
	CMPublicECR     = "public.ecr.aws/dynatrace/dynatrace-codemodules"
	EECPublicECR    = "public.ecr.aws/dynatrace/dynatrace-eec"
	LogMonPublicECR = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	KSPMPublicECR   = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	OTelPublicECR   = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	DBExecPublicECR = "public.ecr.aws/dynatrace/dynatrace-database-datasource-executor"
)

// repoEnvVars maps ECR repository URIs to env var overrides for GetLatestImageURI.
var repoEnvVars = map[string]string{
	AgPublicECR: "E2E_AG_IMAGE",
	OAPublicECR: "E2E_OA_IMAGE",
	CMPublicECR: "E2E_ECR_CODEMODULES_IMAGE",
}

var latestImageURIs = map[string]string{}

// GetLatestImageURI returns the latest image URI for the given repository.
// If an env var override is registered for the repo, it is returned directly.
// Results are cached per repo for the lifetime of the test binary.
func GetLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	envVar, ok := repoEnvVars[repoURI]
	if ok {
		if val := os.Getenv(envVar); val != "" {
			t.Logf("using image from env %s: %s", envVar, val)

			return val
		}
	}

	if uri, ok := latestImageURIs[repoURI]; ok {
		return uri
	}

	uri := getLatestImageURI(t, repoURI)
	latestImageURIs[repoURI] = uri
	t.Logf("resolved newest image for %s: %s", repoURI, uri)

	return uri
}

func GetLatestActiveGateImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, AgPublicECR)
}

func GetLatestOneAgentImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, OAPublicECR)
}

func GetLatestCodeModulesImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, CMPublicECR)
}

func getLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	repo, err := name.NewRepository(repoURI)
	require.NoError(t, err)

	tags, err := remote.List(repo)

	// We should skip tags that are technology-specific or sha digests,
	// e.g., "latest", "1.327.30.20251107-111521-python", "sha256:abcd1234..."
	// and find maximum among the remaining tags.
	endsWithTech := regexp.MustCompile("[a-z-]+$")
	filteredTags := []string{}
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "sha") && !endsWithTech.MatchString(tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	slices.SortFunc(filteredTags, func(a, b string) int {
		semverA, _ := dtversion.ToSemver(a)
		semverB, _ := dtversion.ToSemver(b)

		return semver.Compare(semverA, semverB)
	})
	require.NoError(t, err)

	return fmt.Sprintf("%s:%s", repoURI, filteredTags[len(filteredTags)-1])
}
