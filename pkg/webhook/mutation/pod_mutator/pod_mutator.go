package pod_mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	dtotel "github.com/Dynatrace/dynatrace-operator/pkg/util/otel"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ocDebugAnnotationsContainer = "debug.openshift.io/source-container"
	ocDebugAnnotationsResource  = "debug.openshift.io/source-resource"
)

// AddPodMutationWebhookToManager adds the Webhook server to the Manager
func AddPodMutationWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv(kubeobjects.EnvPodName)
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
		return err
	}
	registerLivezEndpoint(mgr)
	return nil
}

// podMutatorWebhook executes mutators on Pods
type podMutatorWebhook struct {
	apiReader client.Reader
	decoder   admission.Decoder
	recorder  podMutatorEventRecorder

	webhookImage     string
	webhookNamespace string
	clusterID        string
	apmExists        bool
	deployedViaOLM   bool

	mutators   []dtwebhook.PodMutator
	spanTracer trace.Tracer
	otelMeter  metric.Meter

	requestCounter metric.Int64Counter
}

func (webhook *podMutatorWebhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	webhook.countHandleMutationRequest(ctx)

	ctx, span := dtotel.StartSpan(ctx, webhook.spanTracer, "podMutatorHandle")
	defer span.End()

	emptyPatch := admission.Patched("")
	mutationRequest, err := webhook.createMutationRequestBase(ctx, request)
	if err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}

	podName := mutationRequest.PodName()
	if !mutationRequired(mutationRequest) || webhook.isOcDebugPod(mutationRequest.Pod) {
		return emptyPatch
	}

	webhook.setupEventRecorder(ctx, mutationRequest)

	if webhook.isInjected(ctx, mutationRequest) {
		if webhook.handlePodReinvocation(ctx, mutationRequest) {
			log.Info("reinvocation policy applied", "podName", podName)
			webhook.recorder.sendPodUpdateEvent()
			return createResponseForPod(ctx, mutationRequest.Pod, request)
		}
		log.Info("no change, all containers already injected", "podName", podName)
		return emptyPatch
	}

	if err := webhook.handlePodMutation(ctx, mutationRequest); err != nil {
		return silentErrorResponse(mutationRequest.Pod, err)
	}
	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(ctx, mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}
	return kubeobjects.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)
}

func (webhook *podMutatorWebhook) setupEventRecorder(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) {
	_, span := dtotel.StartSpan(ctx, webhook.spanTracer, "setupEventRecorder")
	defer span.End()

	webhook.recorder.dynakube = &mutationRequest.DynaKube
	webhook.recorder.pod = mutationRequest.Pod
}

func (webhook *podMutatorWebhook) isInjected(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) bool {
	_, span := dtotel.StartSpan(ctx, webhook.spanTracer, "isInjected")
	defer span.End()

	for _, mutator := range webhook.mutators {
		if mutator.Injected(mutationRequest.BaseRequest) {
			return true
		}
	}
	return false
}

func (webhook *podMutatorWebhook) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}

func (webhook *podMutatorWebhook) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	_, span := dtotel.StartSpan(ctx, webhook.spanTracer, "handlePodMutation")
	defer span.End()

	mutationRequest.InstallContainer = createInstallInitContainerBase(webhook.webhookImage, webhook.clusterID, mutationRequest.Pod, mutationRequest.DynaKube)
	isMutated := false
	for _, mutator := range webhook.mutators {
		if !mutator.Enabled(mutationRequest.BaseRequest) {
			continue
		}
		if err := mutator.Mutate(mutationRequest); err != nil {
			return err
		}
		isMutated = true
	}
	if !isMutated {
		log.Info("no mutation is enabled")
		return nil
	}

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	webhook.recorder.sendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)
	return nil
}

func (webhook *podMutatorWebhook) handlePodReinvocation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) bool {
	_, span := dtotel.StartSpan(ctx, webhook.spanTracer, "handlePodReinvocation")
	defer span.End()

	var needsUpdate bool

	if mutationRequest.DynaKube.FeatureDisableWebhookReinvocationPolicy() {
		return false
	}

	reinvocationRequest := mutationRequest.ToReinvocationRequest()
	for _, mutator := range webhook.mutators {
		if mutator.Enabled(mutationRequest.BaseRequest) {
			if update := mutator.Reinvoke(reinvocationRequest); update {
				needsUpdate = true
			}
		}
	}
	return needsUpdate
}

func setDynatraceInjectedAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}
	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
}

// createResponseForPod tries to format pod as json
func createResponseForPod(ctx context.Context, pod *corev1.Pod, req admission.Request) admission.Response {
	_, span := dtotel.StartSpan(ctx, otelName, "createResponseForPod")
	defer span.End()

	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return silentErrorResponse(pod, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func silentErrorResponse(pod *corev1.Pod, err error) admission.Response {
	rsp := admission.Patched("")
	podName := kubeobjects.GetPodName(*pod)
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())
	return rsp
}
