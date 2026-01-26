package resourceattributes

import (
	"context"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
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
	ownerInfo, err := workload.FindRootOwnerOfPod(ctx, m.kubeClient, *request, log)
	if err != nil {
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(metadata.OwnerLookupFailedReason),
		}
	}

	annotationAttributes := sanitizeMap(metadata.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube))

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		mutated = m.addResourceAttributes(request, c, ownerInfo, annotationAttributes) || mutated
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(request *dtwebhook.BaseRequest, c *corev1.Container, ownerInfo *workload.Info, annotationAttributes Attributes) bool {
	// order of precedence as in metadata webhook (lowest to highest):
	// 1. workload
	// 2. namespace
	// 3. container
	// 4. pod
	// 5. exsting OTEL_RESOURCE_ATTRIBUTES env var

	// existing attributes have the highes precedence, they are the base
	attributes, ok := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)
	if ok {
		// delete existing env var to add again as last step (to ensure it is at the end of the list because of referenced env vars)
		c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
			return e.Name == OTELResourceAttributesEnv
		})
	}

	// add Attributes from annotations
	mutated := attributes.Merge(annotationAttributes)

	kubernetesMetaDataAttributes := Attributes{
		"k8s.namespace.name":           request.Pod.Namespace,
		"k8s.cluster.uid":              request.DynaKube.Status.KubeSystemUUID,
		"k8s.cluster.name":             request.DynaKube.Status.KubernetesClusterName,
		"dt.entity.kubernetes_cluster": request.DynaKube.Status.KubernetesClusterMEID,
		"k8s.container.name":           c.Name,
		"k8s.pod.name":                 "$(K8S_PODNAME)",
		"k8s.pod.uid":                  "$(K8S_PODUID)",
		"k8s.node.name":                "$(K8S_NODE_NAME)",
	}

	if request.DynaKube.FF().EnableAttributesDtKubernetes() {
		kubernetesMetaDataAttributes.Merge(Attributes{
			"dt.kubernetes.cluster.id": request.DynaKube.Status.KubeSystemUUID,
		})
	}

	kubernetesMetaDataAttributes = sanitizeMap(kubernetesMetaDataAttributes)

	// add workload Attributes (only once fetched per pod, but appended per container to env var if not already present)
	if ownerInfo != nil {
		_ = kubernetesMetaDataAttributes.Merge(Attributes{
			"k8s.workload.kind": ownerInfo.Kind,
			"k8s.workload.name": ownerInfo.Name,
		})
	}

	if ownerInfo != nil && request.DynaKube.FF().EnableAttributesDtKubernetes() {
		_ = kubernetesMetaDataAttributes.Merge(Attributes{
			metadata.DeprecatedWorkloadNameKey: ownerInfo.Name,
			metadata.DeprecatedWorkloadKindKey: ownerInfo.Kind,
		})
	}

	// add standard kubernetes metadata attributes
	mutated = attributes.Merge(kubernetesMetaDataAttributes) || mutated

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	mutated = ensureEnvVarSourcesSet(c) || mutated

	finalValue := attributes.String()

	if mutated {
		metadata.SetWorkloadAnnotations(request.Pod, ownerInfo)
	}

	if finalValue != "" {
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

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[dtwebhook.AnnotationOTLPInjected] = "false"
		pod.Annotations[dtwebhook.AnnotationOTLPReason] = reason
	}
}
