//go:build e2e

package registry

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

const (
	agPublicECR = "public.ecr.aws/dynatrace/dynatrace-activegate"
	oaPublicECR = "public.ecr.aws/dynatrace/dynatrace-oneagent"
	cmPublicECR = "public.ecr.aws/dynatrace/dynatrace-codemodules"
)

// repoEnvVars maps ECR repository URIs to env var overrides for GetLatestImageURI.
var repoEnvVars = map[string]string{
	agPublicECR: "E2E_AG_IMAGE",
	oaPublicECR: "E2E_OA_IMAGE",
	cmPublicECR: "E2E_ECR_CODEMODULES_IMAGE",
}

var latestImageURIs = map[string]string{}

// GetLatestImageURI returns the latest image URI for the given repository.
// If an env var override is registered for the repo, it is returned directly.
// Results are cached per repo for the lifetime of the test binary.
func GetLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	if envVar, ok := repoEnvVars[repoURI]; ok {
		val := os.Getenv(envVar)
		if val != "" {
			t.Logf("using image from env %s: %s", envVar, val)

			return val
		}
	}

	if uri, ok := latestImageURIs[repoURI]; ok {
		t.Logf("using cached resolved newest image: %s", uri)

		return uri
	}

	uri := getLatestImageURI(t, repoURI)
	latestImageURIs[repoURI] = uri
	t.Logf("resolved newest image: %s", uri)

	return uri
}

func GetLatestActiveGateImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, agPublicECR)
}

func GetLatestOneAgentImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, oaPublicECR)
}

func GetLatestCodeModulesImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, cmPublicECR)
}

func getLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	repo, err := name.NewRepository(repoURI)
	require.NoError(t, err)

	var tags []string
	for attempt := range 3 {
		tags, err = remote.List(repo)
		if err == nil {
			break
		}

		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusTooManyRequests {
			wait := time.Duration(5*(attempt+1)) * time.Second
			t.Logf("rate limited listing tags for %s, retrying in %s (attempt %d/3)", repoURI, wait, attempt+1)
			time.Sleep(wait)

			continue
		}

		break
	}

	require.NoError(t, err)

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

	return fmt.Sprintf("%s:%s", repoURI, filteredTags[len(filteredTags)-1])
}
