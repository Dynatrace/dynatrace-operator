package resourceattributes

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

type Attributes map[string]string

func NewAttributesFromEnv(envs []corev1.EnvVar, name string) (Attributes, bool) {
	res := make(map[string]string)
	found := false

	if ev := env.FindEnvVar(envs, name); ev != nil {
		found = true

		split := strings.Split(ev.Value, ",")
		for _, pair := range split {
			if key, value, ok := strings.Cut(pair, "="); ok {
				res[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
	}

	return res, found
}

func NewAttributesFromMap(input map[string]string) Attributes {
	metadataAnnotationPrefix := metadataenrichment.Annotation + "/"

	res := make(map[string]string)

	for key, value := range input {
		if strings.HasPrefix(key, metadataAnnotationPrefix) {
			attrKey := strings.TrimPrefix(key, metadataAnnotationPrefix)
			// apply percent encoding to prevent errors when passing attribute values with special characters to the OTEL SDKs
			// see https://opentelemetry.io/docs/specs/otel/resource/sdk/#specifying-resource-information-via-an-environment-variable
			res[attrKey] = url.QueryEscape(value)
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
