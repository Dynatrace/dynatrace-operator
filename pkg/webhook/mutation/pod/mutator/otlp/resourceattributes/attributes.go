package resourceattributes

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

type attributes map[string]string

func newAttributesFromEnv(envs []corev1.EnvVar, name string) (attributes, bool) {
	res := make(map[string]string)
	found := false

	if ev := env.FindEnvVar(envs, name); ev != nil {
		found = true

		split := strings.Split(ev.Value, ",")
		for _, pair := range split {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				res[key] = value
			}
		}
	}

	return res, found
}

func newAttributesFromMap(input map[string]string) attributes {
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

func (a attributes) merge(other attributes) bool {
	mutated := false

	for key, value := range other {
		if _, exists := a[key]; !exists {
			a[key] = value
			mutated = true
		}
	}

	return mutated
}

func (a attributes) toString() string {
	result := ""

	first := true
	for key, value := range a {
		if !first {
			result += ","
		}

		result += key + "=" + value
		first = false
	}

	return result
}
