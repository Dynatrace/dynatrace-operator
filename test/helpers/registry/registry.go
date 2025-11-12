//go:build e2e

package registry

import (
	"fmt"
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
	agPublicECR = "public.ecr.aws/dynatrace/dynatrace-activegate"
	oaPublicECR = "public.ecr.aws/dynatrace/dynatrace-oneagent"
	cmPublicECR = "public.ecr.aws/dynatrace/dynatrace-codemodules"
)

var (
	latestActiveGateURI string
	latestOneAgentURI   string
	latestCodeModuleURI string
)

func GetLatestActiveGateImageURI(t *testing.T) string {
	t.Helper()

	if latestActiveGateURI == "" {
		latestActiveGateURI = getLatestImageURI(t, agPublicECR)
	}

	return latestActiveGateURI
}

func GetLatestOneAgentImageURI(t *testing.T) string {
	t.Helper()

	if latestOneAgentURI == "" {
		latestOneAgentURI = getLatestImageURI(t, oaPublicECR)
	}

	return latestOneAgentURI
}

func GetLatestCodeModulesImageURI(t *testing.T) string {
	t.Helper()

	if latestCodeModuleURI == "" {
		latestCodeModuleURI = getLatestImageURI(t, cmPublicECR)
	}

	return latestCodeModuleURI
}

func getLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	repo, err := name.NewRepository(repoURI)
	require.NoError(t, err)

	tags, err := remote.List(repo)

	// We should skip tags that are technology-specific or sha digests,
	// e.g., "latest", "1.327.30.20251107-111521-python", "sha256:abcd1234..."
	// and find maximum among the remaining tags.
	endsWithTech, _ := regexp.Compile("[a-z-]+$")
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

	return fmt.Sprintf("%s:%s", repoURI, filteredTags[len(tags)-1])
}
