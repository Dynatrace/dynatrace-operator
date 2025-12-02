package resourceattributes

import (
	"context"
	"maps"
	"slices"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/injection"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metamutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

const (
	OTELResourceAttributesEnv = "OTEL_RESOURCE_ATTRIBUTES"
)

type Mutator struct {
	kubeClient client.Client
}

func New(apiReader client.Client) dtwebhook.Mutator {
	return &Mutator{kubeClient: apiReader}
}

func (Mutator) IsEnabled(_ *dtwebhook.BaseRequest) bool {
	// always return true, as this mutator is only called if OTLP exporter mutator is enabled
	return true
}

func (Mutator) IsInjected(_ *dtwebhook.BaseRequest) bool {
	// always return false, as this mutator is only called if OTLP exporter mutator is enabled
	return false
}

func (m *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	_, err := m.mutate(request.Context, request.BaseRequest)

	return err
}

func (m *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)

	mutated, _ := m.mutate(context.Background(), request.BaseRequest)

	return mutated
}

func getWorkloadInfoAttributesMap(ctx context.Context, request dtwebhook.BaseRequest, clt client.Client) (Attributes, error) {
	workloadInfoAttributes, err := metamutator.GetWorkloadInfoAttributes(ctx, request, clt)
	if err != nil {
		return Attributes{}, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(metamutator.OwnerLookupFailedReason),
		}
	}

	attrsMap, err := getAttributesMap(workloadInfoAttributes)
	if err != nil {
		return Attributes{}, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(AttributesConversionFailedReason),
		}
	}

	return attrsMap, nil
}

func getPodAttributesMap(request dtwebhook.BaseRequest) (Attributes, []corev1.EnvVar, error) {
	podAttributes, envs := injection.GetPodAttributes(request)

	podAttributesMap, err := getAttributesMap(podAttributes)
	if err != nil {
		return Attributes{}, nil, &dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(AttributesConversionFailedReason),
		}
	}

	return podAttributesMap, envs, nil
}

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) (bool, error) {
	mutated := false

	log.Debug("injecting OTLP resource Attributes")

	// fetch workload information once per pod
	workloadInfoAttributes, mutationError := getWorkloadInfoAttributesMap(ctx, *request, m.kubeClient)
	if mutationError != nil {
		log.Error(mutationError, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, mutationError
	}

	podAttributes, envs, mutationError := getPodAttributesMap(*request)
	if mutationError != nil {
		log.Error(mutationError, "failed to get pod attributes info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, mutationError
	}

	namespaceAttributes := SanitizeMap(metamutator.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube))

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		if m.addResourceAttributes(c, workloadInfoAttributes, podAttributes, namespaceAttributes, envs) {
			mutated = true
		}
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(c *corev1.Container, workloadInfoAttributes Attributes, podAttributes Attributes, namespaceAttributes Attributes, envs []corev1.EnvVar) bool {
	// order of precedence as in metadata webhook (lowest to highest):
	// 1. workload
	// 2. namespace
	// 2. container
	// 3. pod
	preconfiguredAttributes, preconfigureEnvVarFound := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)

	mutated := preconfiguredAttributes.Merge(workloadInfoAttributes)
	mutated = preconfiguredAttributes.Merge(namespaceAttributes) || mutated

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	mutated = applyEnvVars(c, envs) || mutated

	containerAttributes := injection.GetContainerAttributes(*c)
	if _, found := preconfiguredAttributes["k8s.container.name"]; !found && containerAttributes.ContainerName != "" {
		preconfiguredAttributes["k8s.container.name"] = containerAttributes.ContainerName
		mutated = true
	}

	mutated = preconfiguredAttributes.Merge(podAttributes) || mutated

	finalValue := preconfiguredAttributes.String()

	if finalValue != "" {
		if preconfigureEnvVarFound {
			// remove existing OTEL_RESOURCE_ATTRIBUTES env var and re-add to make sure it's at the end of the list
			// because of referenced env vars in attribute values
			c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
				return e.Name == OTELResourceAttributesEnv
			})
		}

		c.Env = append(c.Env, corev1.EnvVar{Name: OTELResourceAttributesEnv, Value: finalValue})
	}

	return mutated
}

func shouldSkipContainer(request dtwebhook.BaseRequest, c corev1.Container) bool {
	return dtwebhook.IsContainerExcludedFromInjection(
		request.DynaKube.Annotations,
		request.Pod.Annotations,
		c.Name,
	)
}

func applyEnvVars(c *corev1.Container, envs []corev1.EnvVar) bool {
	mutated := false

	for env := range envs {
		var added bool

		c.Env, added = k8senv.Append(c.Env, envs[env])
		if added {
			mutated = true
		}
	}

	return mutated
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[dtwebhook.AnnotationOTLPInjected] = "false"
		pod.Annotations[dtwebhook.AnnotationOTLPReason] = reason
	}
}

func getAttributesMap(attrs podattr.Attributes) (map[string]string, error) {
	attrsMap, err := attrs.ToMap()
	if err != nil {
		return nil, err
	}

	if attrs.UserDefined != nil {
		// need to sanitize and copy user-defined attributes separately
		maps.Copy(attrsMap, SanitizeMap(attrs.UserDefined))
	}

	return attrsMap, nil
}
