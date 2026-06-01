package resourceattributes

import (
	"net/url"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	corev1 "k8s.io/api/core/v1"
)

type Attributes map[string]string

func NewAttributesFromEnv(envs []corev1.EnvVar, name string) (map[string]string, bool) {
	res := make(map[string]string)
	found := false

	if ev := k8senv.Find(envs, name); ev != nil {
		found = true

		for pair := range strings.SplitSeq(ev.Value, ",") {
			if key, value, ok := strings.Cut(pair, "="); ok {
				res[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
	}

	return res, found
}

func isSafeEnvRef(value string) bool {
	after, found := strings.CutPrefix(value, "$(")
	if !found {
		return false
	}

	before, found := strings.CutSuffix(after, ")")

	return found && slices.Contains(attributes.SafeEnvRefs, before)
}

// sanitizeValue percent-encodes value for use in OTEL_RESOURCE_ATTRIBUTES, except for the
// small set of operator-injected env refs that the kubelet must expand at pod startup.
// See https://opentelemetry.io/docs/specs/otel/resource/sdk/#specifying-resource-information-via-an-environment-variable
func sanitizeValue(value string) string {
	if isSafeEnvRef(value) {
		return value
	}

	// Decode any existing percent-encoding before re-encoding to prevent double-encoding
	// already-sanitized values (e.g. values from a pre-existing OTEL_RESOURCE_ATTRIBUTES env var).
	// url.QueryUnescape decodes both %XX sequences and + as space; if the string contains
	// an invalid percent sequence (e.g. "100%") it returns an error and we fall back to
	// encoding the raw string directly.
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return url.QueryEscape(value)
	}

	return url.QueryEscape(decoded)
}
