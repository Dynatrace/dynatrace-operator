package attributes

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pod struct {

	// built from customer defined metadata enrichment rules
	rules          map[string]string
	rulesPropagate map[string]string

	// read from metadata.dynatrace.com annotations on the namespace
	namespaceAnnotations map[string]string
	// read from metadata.dynatrace.com annotations on the pod
	podAnnotations map[string]string

	// .spec.resourceAttributes + .spec.(oneagent.*|otlpExporterConfiguration).additionalResourceAttributes
	dynakube map[string]string

	// custom attributes, e.g. OTEL_RESOURCE_ATTRIBUTES env var
	custom map[string]string

	// read from the workload that owns the injected pod
	workloadInfo map[string]string

	// read from K8s cluster
	clusterInfo map[string]string

	// read from the injected pod manifest
	podInfo map[string]string

	// dt.kubernetes.* attributes that are deprecated and will be removed
	deprecated map[string]string

	// include deprecated attributes in combined results
	useDeprecated bool

	// env vars that are referenced by attributes
	podEnvVars []corev1.EnvVar
}

func NewPodAttributes(ctx context.Context, request mutator.BaseRequest, client client.Client) (*Pod, error) {
	attrs := &Pod{
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

func (attrs *Pod) SetCustomAttributes(custom map[string]string) {
	attrs.custom = custom
}

func (attrs *Pod) SetDynakubeAttributes(ctx context.Context, dkAttrs map[string]string) {
	attrs.dynakube = sanitizeKeys(ctx, dkAttrs)
}

func (attrs *Pod) GetPodEnvVars() []corev1.EnvVar {
	return attrs.podEnvVars
}

func (attrs *Pod) Convert(c convertFunc, containerAttrs ...Container) []string {
	combined := attrs.combineAll(containerAttrs...)

	return convert(combined, c)
}

func ToArg(key, value string) string {
	return fmt.Sprintf("--%s=%s=%s", pod.Flag, key, value)
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
