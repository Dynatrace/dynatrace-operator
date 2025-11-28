package resourceattributes

import (
	"context"
	"maps"
	"slices"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	kubeClient client.Client
}

func New(apiReader client.Client) mutator.Mutator {
	return &Mutator{kubeClient: apiReader}
}

func (*Mutator) IsEnabled(_ *mutator.BaseRequest) bool {
	// always return true, as this mutator is only called if OTLP exporter mutator is enabled
	return true
}

func (*Mutator) IsInjected(_ *mutator.BaseRequest) bool {
	// always return false, as this mutator is only called if OTLP exporter mutator is enabled
	return false
}

func (m *Mutator) Mutate(request *mutator.MutationRequest) error {
	_, err := m.mutate(request.Context, request.BaseRequest)

	return err
}

func (m *Mutator) Reinvoke(request *mutator.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)

	mutated, err := m.mutate(context.Background(), request.BaseRequest)
	if err != nil {
		log.Error(err, "reinvoke of OTLP resource attribute mutator failed")
	}

	return mutated
}

func (m *Mutator) mutate(ctx context.Context, request *mutator.BaseRequest) (bool, error) {
	var err error

	mutated := false

	log.Debug("injecting OTLP resource Attributes")

	podAttributes := pod.Attributes{}

	podAttributes, err = attributes.GetWorkloadInfoAttributes(podAttributes, ctx, request, m.kubeClient)
	if err != nil {
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		// TODO: should we abort here or go on with rest of attributes?
		/*
			return false, mutator.MutatorError{
				Err:      errors.WithStack(err),
				Annotate: setNotInjectedAnnotationFunc(mutator.OwnerLookupFailedReason),
			}*/
	}

	var envs []corev1.EnvVar

	podAttributes, envs = attributes.GetPodAttributes(podAttributes, request)

	podAttributes = attributes.GetMetadataAnnotations(podAttributes, request)

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		containerAttributes := attributes.GetContainerAttributes(c)

		mutated = m.addResourceAttributes(c, podAttributes, containerAttributes, envs) || mutated
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(c *corev1.Container, podAttributes pod.Attributes, containerAttributes container.Attributes, envs []corev1.EnvVar) bool {
	existingAttributes, ok := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)
	if ok {
		// delete existing env var to add again as last step (to ensure it is at the end of the list because of referenced env vars)
		c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
			return e.Name == OTELResourceAttributesEnv
		})
	}

	podAttributesMap, err := podAttributes.ToMap()
	if err != nil {
		log.Error(err, "failed to convert pod attributes to map")

		return false
	}

	// need to sanitize and copy user-defined attributes separately
	maps.Copy(podAttributesMap, SanitizeMap(podAttributes.UserDefined))

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	mutated := ensureEnvVarSourcesSet(c, envs)

	mutated = existingAttributes.Merge(podAttributesMap) || mutated

	if _, found := existingAttributes["k8s.container.name"]; !found && containerAttributes.ContainerName != "" {
		existingAttributes["k8s.container.name"] = containerAttributes.ContainerName
		mutated = true
	}

	// TODO: what attributes need sanitization? All or just user-defined ones?
	//	finalValue := SanitizeMap(existingAttributes).String()
	finalValue := existingAttributes.String()

	if finalValue != "" {
		c.Env = append(c.Env, corev1.EnvVar{Name: OTELResourceAttributesEnv, Value: finalValue})
	}

	return mutated
}

func shouldSkipContainer(request mutator.BaseRequest, c corev1.Container) bool {
	return mutator.IsContainerExcludedFromInjection(
		request.DynaKube.Annotations,
		request.Pod.Annotations,
		c.Name,
	)
}

func ensureEnvVarSourcesSet(c *corev1.Container, envs []corev1.EnvVar) bool {
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

/*
func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[mutator.AnnotationOTLPInjected] = "false"
		pod.Annotations[mutator.AnnotationOTLPReason] = reason
	}
}
*/
