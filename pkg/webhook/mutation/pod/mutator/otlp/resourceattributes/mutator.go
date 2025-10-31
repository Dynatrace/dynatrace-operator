package resourceattributes

import (
	"context"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

const (
	otlpResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
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
	_ = m.mutate(request.Context, request.BaseRequest)

	return nil
}

func (m *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)

	mutated := m.mutate(context.Background(), request.BaseRequest)

	return mutated
}

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) bool {
	mutated := false

	log.Debug("injecting OTLP resource attributes")

	// fetch workload information once per pod
	ownerInfo, err := workload.FindRootOwnerOfPod(ctx, m.kubeClient, *request, log)
	if err != nil {
		// log error but continue (best effort)
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)
	}

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		if m.addResourceAttributes(request, c, ownerInfo) {
			mutated = true
		}
	}

	return mutated
}

func (m *Mutator) addResourceAttributes(request *dtwebhook.BaseRequest, c *corev1.Container, ownerInfo *workload.Info) bool {
	var mutated bool

	existingAttributes, ok := newAttributesFromEnv(c.Env, otlpResourceAttributesEnvVar)
	if ok {
		// delete existing env var to add again as last step (to ensure it is at the end of the list because of referenced env vars)
		c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
			return e.Name == otlpResourceAttributesEnvVar
		})
	}

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	envVarSourcesAdded := ensureEnvVarSourcesSet(c)

	attributesToAdd := attributes{
		"k8s.namespace.name":         request.Pod.Namespace,
		"k8s.cluster.uid":            request.DynaKube.Status.KubeSystemUUID,
		"dt.kubernetes.cluster.id":   request.DynaKube.Status.KubeSystemUUID,
		"k8s.cluster.name":           request.DynaKube.Status.KubeSystemUUID,
		"dt.kubernetes.cluster.name": request.DynaKube.Status.KubernetesClusterName,
		"k8s.container.name":         c.Name,
		"k8s.pod.name":               "$(K8S_PODNAME)",
		"k8s.pod.uid":                "$(K8S_PODUID)",
		"k8s.node.name":              "$(K8S_NODE_NAME)",
	}

	// add workload attributes (only once fetched per pod, but appended per container to env var if not already present)
	if ownerInfo != nil {
		workloadAttributesToAdd := attributes{
			"k8s.workload.kind": ownerInfo.Kind,
			"k8s.workload.name": ownerInfo.Name,
		}
		_ = attributesToAdd.merge(workloadAttributesToAdd)
	}
	// add attributes from annotations - these have the highest precedence, i.e. users can potentially overwrite the above attributes
	attributesFromAnnotations := newAttributesFromMap(request.Pod.Annotations)
	_ = attributesFromAnnotations.merge(attributesToAdd)

	mutated = existingAttributes.merge(attributesFromAnnotations)

	finalValue := existingAttributes.toString()

	if finalValue != "" {
		c.Env = append(c.Env, corev1.EnvVar{Name: otlpResourceAttributesEnvVar, Value: finalValue})
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

	if envs, added := env.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_PODNAME",
		ValueFrom: env.NewEnvVarSourceForField("metadata.name"),
	}); added {
		c.Env = envs
		mutated = true
	}

	if envs, added := env.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_PODUID",
		ValueFrom: env.NewEnvVarSourceForField("metadata.uid"),
	}); added {
		c.Env = envs
		mutated = true
	}

	if envs, added := env.Append(c.Env, corev1.EnvVar{
		Name:      "K8S_NODE_NAME",
		ValueFrom: env.NewEnvVarSourceForField("spec.nodeName"),
	}); added {
		c.Env = envs
		mutated = true
	}

	return mutated
}
