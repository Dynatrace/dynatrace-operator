package attributes

import (
	"context"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
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

	err := attrs.GetWorkloadInfoAttributes(ctx, request, client)
	if err != nil {
		return nil, err
	}

	attrs.GetMetadataAnnotations(request)
	attrs.readPodAttributes(request)

	if attrs.useDeprecated {
		attrs.applyDeprecatedAttributes()
	}

	return attrs, nil
}

func (attrs *PodAttributes) AddCustomAttribute(key, value string) {
	attrs.custom[key] = value
}

func (attrs *PodAttributes) AddCustomAttributes(custom map[string]string) {
	maps.Copy(attrs.custom, custom)
}

func (attrs *PodAttributes) GetPodEnvVars() []corev1.EnvVar {
	return attrs.podEnvVars
}

func (attrs *PodAttributes) Convert(c convertFunc, containerAttrs ...ContainerAttributes) []string {
	combined := attrs.combine(containerAttrs...)
	return convert(combined, c)
}

// this is the one function that takes care of the order of precedence as in metadata webhook (lowest to highest):
// 1. workload, pod and cluster infos
// 2. container
// 3. enrichment rules
// 4. namespace annotations
// 5. pod annotations
// 6. custom attributes (OTLP env var, DynaKube resource attributes, ...)
func (attrs *PodAttributes) combine(containerAttrs ...ContainerAttributes) map[string]string {
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

	maps.Copy(combined, attrs.rules)
	maps.Copy(combined, attrs.rulesPropagate)

	maps.Copy(combined, attrs.namespaceAnnotations)
	maps.Copy(combined, attrs.podAnnotations)

	maps.Copy(combined, attrs.custom)

	return combined
}
