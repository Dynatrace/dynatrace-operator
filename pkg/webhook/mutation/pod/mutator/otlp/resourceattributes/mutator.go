package resourceattributes

import (
	"context"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (Mutator) IsEnabled(_ context.Context, _ *dtwebhook.BaseRequest) bool {
	// always return true, as this mutator is only called if OTLP exporter mutator is enabled
	return true
}

func (Mutator) IsInjected(_ context.Context, _ *dtwebhook.BaseRequest) bool {
	// always return false, as this mutator is only called if OTLP exporter mutator is enabled
	return false
}

func (m *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	ctx, _ := logd.NewFromContext(request.Context, "otlp-exporter-pod-mutation")
	_, err := m.mutate(ctx, request.BaseRequest)

	return err
}

func (m *Mutator) Reinvoke(ctx context.Context, request *dtwebhook.ReinvocationRequest) bool {
	ctx, log := logd.NewFromContext(ctx, "otlp-exporter-pod-mutation")
	log.Debug("reinvocation of OTLP resource attribute mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)

	mutated, _ := m.mutate(ctx, request.BaseRequest)

	return mutated
}

func (m *Mutator) mutate(ctx context.Context, request *dtwebhook.BaseRequest) (bool, error) {
	log := logd.FromContext(ctx)

	log.Debug("injecting OTLP resource PodAttributes")

	attrs, err := attributes.NewPodAttributes(ctx, *request, m.kubeClient)

	if err != nil {
		log.Error(err, "failed to get workload info", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false, dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(metadata.OwnerLookupFailedReason),
		}
	}

	err = attrs.ApplyAnnotationsToPod(request.Pod)
	if err != nil {
		log.Error(err, "failed to propagate metadata annotations", "pod", request.Pod, "namespace", request.Namespace)
	}

	mutated := false
	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c) {
			continue
		}

		mutated = m.addResourceAttributes(attrs, request, c) || mutated
	}

	return mutated, nil
}

func (m *Mutator) addResourceAttributes(podAttrs *attributes.PodAttributes, request *dtwebhook.BaseRequest, c *corev1.Container) bool {
	// order of precedence as in metadata webhook (lowest to highest):
	// 1. workload
	// 2. namespace
	// 3. container
	// 4. pod
	// 5. exsting OTEL_RESOURCE_ATTRIBUTES env var

	// existing existingResourceAttrs have the highes precedence, they are the base
	existingResourceAttrs, ok := NewAttributesFromEnv(c.Env, OTELResourceAttributesEnv)
	if ok {
		// delete existing env var to add again as last step (to ensure it is at the end of the list because of referenced env vars)
		c.Env = slices.DeleteFunc(c.Env, func(e corev1.EnvVar) bool {
			return e.Name == OTELResourceAttributesEnv
		})
	}

	podAttrs.AddCustomAttributes(existingResourceAttrs)

	containerAttrs := attributes.NewContainerAttributes(*c)

	// TODO: detect mutation

	mutated := false
	// podAttrs also contains custom attributes (annotations, rules, ..) which take precedence
	kvPairs := podAttrs.Convert(func(k, v string) string {
		v = sanitizeValue(v)

		// resource attributes will be changed (==mutated) if new entries are added
		if _, found := existingResourceAttrs[k]; !found {
			mutated = true
		}
		return k + "=" + v
	}, *containerAttrs)

	// ensure the container env vars for POD_NAME, POD_UID, and NODE_NAME are set
	mutated = ensureEnvVarSourcesSet(podAttrs, c) || mutated

	if len(kvPairs) != 0 {
		c.Env = append(c.Env, corev1.EnvVar{Name: OTELResourceAttributesEnv, Value: strings.Join(kvPairs, ",")})
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

func ensureEnvVarSourcesSet(attrs *attributes.PodAttributes, c *corev1.Container) bool {
	mutated := false

	podEnvs := attrs.GetPodEnvVars()

	for _, env := range podEnvs {
		if envs, added := k8senv.Append(c.Env, env); added {
			c.Env = envs
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
