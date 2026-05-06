package resourceattributes

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
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

func sanitizeValue(value string) string {
	// apply percent encoding to prevent errors when passing attribute values with special characters to the OTEL SDKs
	// see https://opentelemetry.io/docs/specs/otel/resource/sdk/#specifying-resource-information-via-an-environment-variable
	if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
		return value
	} else {
		return url.QueryEscape(value)
	}
}
