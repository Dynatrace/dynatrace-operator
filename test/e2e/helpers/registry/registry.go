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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

const (
	agPublicECR = "public.ecr.aws/dynatrace/dynatrace-activegate"
	oaPublicECR = "public.ecr.aws/dynatrace/dynatrace-oneagent"
	cmPublicECR = "public.ecr.aws/dynatrace/dynatrace-codemodules"
)

const (
	agImageEnv = "E2E_AG_IMAGE"
	oaImageEnv = "E2E_OA_IMAGE"
	cmImageEnv = "E2E_ECR_CODEMODULES_IMAGE"
)

var latestImageURIs = map[string]string{}

// GetLatestImageURI returns the image URI for the given repository.
// If the envVar is set, its value is returned directly.
// Otherwise the latest tag is resolved from the registry and cached per repo for the lifetime of the test binary.
func GetLatestImageURI(t *testing.T, repoURI string, envVar string) string {
	t.Helper()

	val := os.Getenv(envVar)
	if val != "" {
		t.Logf("using image from env %s: %s", envVar, val)

		return val
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

	return GetLatestImageURI(t, agPublicECR, agImageEnv)
}

func GetLatestOneAgentImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, oaPublicECR, oaImageEnv)
}

func GetLatestCodeModulesImageURI(t *testing.T) string {
	t.Helper()

	return GetLatestImageURI(t, cmPublicECR, cmImageEnv)
}

func isRateLimited(err error) bool {
	var transportErr *transport.Error

	return errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusTooManyRequests
}

func getLatestImageURI(t *testing.T, repoURI string) string {
	t.Helper()

	repo, err := name.NewRepository(repoURI)
	require.NoError(t, err)

	backoff := wait.Backoff{
		Steps:    3,
		Duration: 5 * time.Second,
		Factor:   2.0,
	}

	var tags []string

	err = retry.OnError(backoff, isRateLimited, func() error {
		var listErr error
		tags, listErr = remote.List(repo)
		if listErr != nil {
			t.Logf("rate limited listing tags for %s, retrying...", repoURI)
		}

		return listErr
	})
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

	require.NotEmpty(t, filteredTags, "no valid semver tags found for %s", repoURI)

	return fmt.Sprintf("%s:%s", repoURI, filteredTags[len(filteredTags)-1])
}
