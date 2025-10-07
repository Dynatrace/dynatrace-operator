package resourceattributes

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

const (
	otlpResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
)

type Mutator struct {
	apiReader client.Reader
}

func New(apiReader client.Reader) dtwebhook.Mutator {
	return &Mutator{apiReader: apiReader}
}

func (Mutator) IsEnabled(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env var injection is enabled")

	return false
}

func (Mutator) IsInjected(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env vars have already been injected")

	return false
}

func (m *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	_, err := m.mutate(request.Context, request.BaseRequest)

	return err
}

func (m *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP resource attribute mutator")

	mutated, err := m.mutate(context.Background(), request.BaseRequest)

	if err != nil {
		log.Error(err, "failed to reinvoke OTLP resource attribute mutator", "pod", request.Pod.Name, "namespace", request.Pod.Namespace)
		return false
	}

	return mutated
}

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) (bool, error) {
	mutated := false
	log.Debug("injecting OTLP resource attributes")

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
		ensureEnvVarSourcesSet(c)

		var b strings.Builder
		var existing *corev1.EnvVar
		if ev := env.FindEnvVar(c.Env, otlpResourceAttributesEnvVar); ev != nil {
			existing = ev
			if ev.Value != "" {
				b.WriteString(ev.Value)
			}
		}

		attributesToAdd := map[string]string{
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

		for key, value := range attributesToAdd {
			if appendAttribute(&b, existing, key, value) {
				mutated = true
			}
		}

		// Workload attributes (API lookup for ReplicaSet -> Deployment chain)
		if wkKind, wkName := getWorkloadInfo(ctx, m.apiReader, request.Pod); wkKind != "" && wkName != "" {

			workloadAttributesToAdd := map[string]string{
				"k8s.workload.kind": wkKind,
				"k8s.workload.name": wkName,
			}

			for key, value := range workloadAttributesToAdd {
				if appendAttribute(&b, existing, key, value) {
					mutated = true
				}
			}
		}

		// add attributes from annotations
		for k, v := range request.Pod.Annotations {
			metadataAnnotationPrefix := fmt.Sprintf("%s/", metadataenrichment.Annotation)

			if strings.HasPrefix(k, metadataAnnotationPrefix) {
				attrKey := strings.TrimPrefix(k, metadataAnnotationPrefix)
				appendAttribute(&b, existing, attrKey, v)
			}
		}

		finalValue := b.String()
		if existing != nil {
			existing.Value = finalValue
		} else if finalValue != "" {
			c.Env = append(c.Env, corev1.EnvVar{Name: otlpResourceAttributesEnvVar, Value: finalValue})
		}
	}

	return mutated, nil
}

func shouldSkipContainer(request dtwebhook.BaseRequest, c corev1.Container) bool {
	if dtwebhook.IsContainerExcludedFromInjection(
		request.DynaKube.Annotations,
		request.Pod.Annotations,
		c.Name,
	) {
		return true
	}

	return false
}

func appendAttribute(b *strings.Builder, existing *corev1.EnvVar, key, value string) bool {
	if value == "" {
		return false
	}
	if b.Len() > 0 {
		b.WriteString(",")
	}

	if existing != nil && strings.Contains(existing.Value, key) {
		// do not override existing value
		return false
	}

	b.WriteString(key)
	b.WriteString("=")
	b.WriteString(value)

	return true
}

func ensureEnvVarSourcesSet(c *corev1.Container) {
	if !env.IsIn(c.Env, "K8S_PODNAME") {
		c.Env = append(
			c.Env,
			corev1.EnvVar{
				Name:      "K8S_PODNAME",
				ValueFrom: env.NewEnvVarSourceForField("metadata.name"),
			},
		)
	}

	if !env.IsIn(c.Env, "K8S_PODUID") {
		c.Env = append(
			c.Env,
			corev1.EnvVar{
				Name:      "K8S_PODUID",
				ValueFrom: env.NewEnvVarSourceForField("metadata.uid"),
			},
		)
	}

	if !env.IsIn(c.Env, "K8S_NODE_NAME") {
		c.Env = append(
			c.Env,
			corev1.EnvVar{
				Name:      "K8S_NODE_NAME",
				ValueFrom: env.NewEnvVarSourceForField("spec.nodeName"),
			},
		)
	}
}

// getWorkloadInfo performs live lookups (using reader) to resolve the top-level workload for a pod.
// If the immediate controller owner is a ReplicaSet, it tries to fetch that ReplicaSet and inspect its controller owner (Deployment).
// Returns empty strings if the workload cannot be determined.
func getWorkloadInfo(ctx context.Context, reader client.Reader, pod *corev1.Pod) (kind, name string) {
	if pod == nil || reader == nil {
		return "", ""
	}

	for _, owner := range pod.OwnerReferences {
		if owner.Controller == nil || !*owner.Controller {
			continue
		}

		switch owner.Kind {
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
			return owner.Kind, owner.Name
		case "ReplicaSet":
			// lookup ReplicaSet to see if owned by a Deployment
			rs := &appsv1.ReplicaSet{}
			err := reader.Get(ctx, types.NamespacedName{Name: owner.Name, Namespace: pod.Namespace}, rs)
			if err != nil {
				// fallback to ReplicaSet itself if cannot fetch
				return owner.Kind, owner.Name
			}
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Controller != nil && *rsOwner.Controller && rsOwner.Kind == "Deployment" {
					return rsOwner.Kind, rsOwner.Name
				}
			}
			return owner.Kind, owner.Name
		}
	}

	return "", ""
}
