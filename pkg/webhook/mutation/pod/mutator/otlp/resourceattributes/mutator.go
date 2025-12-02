package resourceattributes

import (
	"context"
	"maps"
	"slices"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
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
	//_, err := m.mutate(request.Context, request.BaseRequest)
	_, err := m.mutate(request.Context, request.BaseRequest)

	return err
}

func (m *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)

	//mutated, _ := m.mutate(context.Background(), request.BaseRequest)
	mutated, _ := m.mutate(context.Background(), request.BaseRequest)

	return mutated
}

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) (bool, error) {
	mutated := false

	log.Debug("injecting OTLP resource Attributes")

	// get the workload info and set workload annotations
	workloadInfoAttributes, err := metamutator.GetWorkloadInfoAttributes(ctx, *request, m.kubeClient)
	if err != nil {
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(metamutator.OwnerLookupFailedReason),
		}
	}

	workloadInfoNamespaceAttributesMap, err := getAttributesMap(workloadInfoAttributes)
	if err != nil {
		log.Error(err, "failed to convert workload info attributes to map, unable to add resource attributes to container", "pod", request.Pod.Name, "namespace", request.Pod.Namespace)
	}

	// get attributes from namespace according to enrichment rules, propagate metadata annotations from pod to namespace and
	// collect metadata annotations from the pod
	workloadInfoNamespaceAttributesMap.Merge(SanitizeMap(metamutator.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)))

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		if m.addResourceAttributes(request, c, workloadInfoNamespaceAttributesMap) {
			mutated = true
		}
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(request *dtwebhook.BaseRequest, c *corev1.Container, workloadInfoNamespaceAttributesMap Attributes) bool {
	var mutated bool

	existingAttributes, ok := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)
	if ok {
		// delete existing env var to add again as last step (to ensure it is at the end of the list because of referenced env vars)
		c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
			return e.Name == OTELResourceAttributesEnv
		})
	}

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	envVarSourcesAdded := ensureEnvVarSourcesSet(c)

	attributesToAdd := Attributes{
		"k8s.namespace.name":           request.Pod.Namespace,
		"k8s.cluster.uid":              request.DynaKube.Status.KubeSystemUUID,
		"dt.kubernetes.cluster.id":     request.DynaKube.Status.KubeSystemUUID,
		"k8s.cluster.name":             request.DynaKube.Status.KubernetesClusterName,
		"dt.entity.kubernetes_cluster": request.DynaKube.Status.KubernetesClusterMEID,
		"k8s.container.name":           c.Name,
		"k8s.pod.name":                 "$(K8S_PODNAME)",
		"k8s.pod.uid":                  "$(K8S_PODUID)",
		"k8s.node.name":                "$(K8S_NODE_NAME)",
	}

	// add workload Attributes (only once fetched per pod, but appended per container to env var if not already present)
	_ = attributesToAdd.Merge(workloadInfoNamespaceAttributesMap)

	mutated = existingAttributes.Merge(attributesToAdd)

	finalValue := existingAttributes.String()

	if finalValue != "" {
		c.Env = append(c.Env, corev1.EnvVar{Name: OTELResourceAttributesEnv, Value: finalValue})
	}

	return envVarSourcesAdded || mutated
}

func shouldSkipContainer(request dtwebhook.BaseRequest, c corev1.Container) bool {
	return dtwebhook.IsContainerExcludedFromInjection(
		request.DynaKube.Annotations,
		request.Pod.Annotations,
		c.Name,
	)
}

func ensureEnvVarSourcesSet(c *corev1.Container) bool {
	mutated := false

	if envs, added := k8senv.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_PODNAME",
		ValueFrom: k8senv.NewSourceForField("metadata.name"),
	}); added {
		c.Env = envs
		mutated = true
	}

	if envs, added := k8senv.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_PODUID",
		ValueFrom: k8senv.NewSourceForField("metadata.uid"),
	}); added {
		c.Env = envs
		mutated = true
	}

	if envs, added := k8senv.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_NODE_NAME",
		ValueFrom: k8senv.NewSourceForField("spec.nodeName"),
	}); added {
		c.Env = envs
		mutated = true
	}

	return mutated
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

func getAttributesMap(attrs podattr.Attributes) (Attributes, error) {
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
