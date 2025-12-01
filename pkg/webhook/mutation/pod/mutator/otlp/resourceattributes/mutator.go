package resourceattributes

import (
	"context"
	"maps"
	"slices"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
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

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) (bool, error) {
	mutated := false

	log.Debug("injecting OTLP resource Attributes")

	// fetch workload information once per pod
	workloadInfoAttributes, err := metamutator.GetWorkloadInfoAttributes(ctx, *request, m.kubeClient)
	if err != nil {
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(metamutator.OwnerLookupFailedReason),
		}
	}

	podAttributes, envs := injection.GetPodAttributes(*request)
	namespaceAttributes := metamutator.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		containerAttributes := injection.GetContainerAttributes(*c)

		if m.addResourceAttributes(c, namespaceAttributes, workloadInfoAttributes, podAttributes, containerAttributes, envs) {
			mutated = true
		}
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(c *corev1.Container, namespaceAttributes map[string]string, workloadInfoAttributes podattr.Attributes, podAttributes podattr.Attributes, containerAttributes container.Attributes, envs []corev1.EnvVar) bool {
	preconfiguredAttributes, preconfigureEnvVarFound := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)

	mutated := preconfiguredAttributes.Merge(SanitizeMap(namespaceAttributes))

	podAttributesMap, err := getAttributesMap(podAttributes)
	if err != nil {
		log.Error(err, "failed to convert pod attributes to map")

		return false
	}

	mutated = preconfiguredAttributes.Merge(podAttributesMap) || mutated

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	mutated = applyEnvVars(c, envs) || mutated

	workloadInfoAttributesMap, err := getAttributesMap(workloadInfoAttributes)
	if err != nil {
		log.Error(err, "failed to convert workload info attributes to map")

		return false
	}

	mutated = preconfiguredAttributes.Merge(workloadInfoAttributesMap) || mutated

	if _, found := preconfiguredAttributes["k8s.container.name"]; !found && containerAttributes.ContainerName != "" {
		preconfiguredAttributes["k8s.container.name"] = containerAttributes.ContainerName
		mutated = true
	}

	finalValue := preconfiguredAttributes.String()

	if finalValue != "" {
		if preconfigureEnvVarFound {
			// remove existing OTEL_RESOURCE_ATTRIBUTES env var and readd to make sure it's at the end of the list
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

	// need to sanitize and copy user-defined attributes separately
	maps.Copy(attrsMap, SanitizeMap(attrs.UserDefined))

	return attrsMap, nil
}
