package resourceattributes

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	corev1 "k8s.io/api/core/v1"
)

type Attributes map[string]string

func NewAttributesFromEnv(envs []corev1.EnvVar, name string) (Attributes, bool) {
	res := make(map[string]string)
	found := false

	if ev := k8senv.Find(envs, name); ev != nil {
		found = true

		split := strings.SplitSeq(ev.Value, ",")
		for pair := range split {
			if key, value, ok := strings.Cut(pair, "="); ok {
				res[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
	}

	return res, found
}

func sanitizeMap(input map[string]string) Attributes {
	res := make(map[string]string)

	for key, value := range input {
		// apply percent encoding to prevent errors when passing attribute values with special characters to the OTEL SDKs
		// see https://opentelemetry.io/docs/specs/otel/resource/sdk/#specifying-resource-information-via-an-environment-variable
		if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
			res[key] = value
		} else {
			res[key] = url.QueryEscape(value)
		}
	}

	return res
}

func (a Attributes) Merge(other Attributes) bool {
	mutated := false

	for key, value := range other {
		if _, exists := a[key]; !exists {
			a[key] = value
			mutated = true
		}
	}

	return mutated
}

func (a Attributes) String() string {
	first := true

	var result strings.Builder

	for key, value := range a {
		if key == "" || value == "" {
			// do not add empty values
			continue
		}

		if !first {
			result.WriteString(",")
		}

		result.WriteString(key + "=" + value)

		first = false
	}

	return result.String()
}
