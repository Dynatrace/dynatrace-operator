package attributes

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodAttributes struct {

	// non-user given
	rules                map[string]string
	rulesPropagate       map[string]string
	namespaceAnnotations map[string]string
	podAnnotations       map[string]string
	custom               map[string]string

	workloadInfo map[string]string
	clusterInfo  map[string]string
	podInfo      map[string]string
	deprecated   map[string]string
	podEnvVars   []corev1.EnvVar

	useDeprecated bool
}

func NewPodAttributes(ctx context.Context, request mutator.BaseRequest, client client.Client) (*PodAttributes, error) {
	attrs := &PodAttributes{
		podAnnotations:       make(map[string]string),
		namespaceAnnotations: make(map[string]string),
		rulesPropagate:       make(map[string]string),
		rules:                make(map[string]string),
		workloadInfo:         make(map[string]string),
		podInfo:              make(map[string]string),
		podEnvVars:           []corev1.EnvVar{},
		clusterInfo:          make(map[string]string),
		deprecated:           make(map[string]string),
		custom:               make(map[string]string),

		useDeprecated: request.DynaKube.FF().EnableAttributesDTKubernetes(),
	}

	err := attrs.readWorkloadInfoAttributes(ctx, request, client)
	if err != nil {
		return nil, err
	}

	attrs.readMetadataAnnotations(request)
	attrs.readPodAttributes(request)

	if attrs.useDeprecated {
		attrs.applyDeprecatedAttributes()
	}

	return attrs, nil
}

func (attrs *PodAttributes) SetCustomAttributes(custom map[string]string) {
	attrs.custom = custom
}

func (attrs *PodAttributes) GetPodEnvVars() []corev1.EnvVar {
	return attrs.podEnvVars
}

func (attrs *PodAttributes) Convert(c convertFunc, containerAttrs ...ContainerAttributes) []string {
	combined := attrs.combineAll(containerAttrs...)

	return convert(combined, c)
}

func (attrs *PodAttributes) combineAll(containerAttrs ...ContainerAttributes) map[string]string {
	combined := make(map[string]string)

	// precedence from low -> high
	if attrs.useDeprecated {
		maps.Copy(combined, attrs.deprecated)
	}

	maps.Copy(combined, attrs.workloadInfo)
	maps.Copy(combined, attrs.podInfo)
	maps.Copy(combined, attrs.clusterInfo)

	for _, cAttr := range containerAttrs {
		maps.Copy(combined, cAttr.ToMap())
	}

	maps.Copy(combined, attrs.namespaceAnnotations)

	maps.Copy(combined, attrs.rules)
	maps.Copy(combined, attrs.rulesPropagate)

	maps.Copy(combined, attrs.podAnnotations)

	maps.Copy(combined, attrs.custom)

	return combined
}

func (attrs *PodAttributes) combineForMetadataAnnotations() map[string]string {
	combined := make(map[string]string)

	// make sure we use the same precedence as in combine()
	maps.Copy(combined, attrs.workloadInfo)
	maps.Copy(combined, attrs.namespaceAnnotations)
	maps.Copy(combined, attrs.rulesPropagate)

	return combined
}

func (attrs *PodAttributes) combineForJSONAnnotation() (string, error) {
	combined := make(map[string]string)

	// make sure we use the same precedence as in combineAll()
	maps.Copy(combined, attrs.namespaceAnnotations)
	maps.Copy(combined, attrs.rules)
	maps.Copy(combined, attrs.rulesPropagate)
	maps.Copy(combined, attrs.podAnnotations)

	marshaledAnnotations, err := json.Marshal(combined)
	if err != nil {
		return "", errors.WithMessage(errors.WithStack(err), "could not marshal metadata annotations to JSON")
	}

	return string(marshaledAnnotations), nil
}

type convertFunc func(string, string) string

func convert(attributes map[string]string, c convertFunc) []string {
	converted := make([]string, 0, len(attributes))
	for key, value := range attributes {
		if result := c(key, value); result != "" {
			converted = append(converted, result)
		}
	}

	return converted
}
