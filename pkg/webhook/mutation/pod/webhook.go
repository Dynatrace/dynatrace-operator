package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptrace"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/internal/otel"
	dtwebhookutil "github.com/Dynatrace/dynatrace-operator/pkg/webhook/util"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel/attribute"
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

func AddWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv(env.PodName)
	if podName == "" {
		log.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
		return err
	}

	registerLivezEndpoint(mgr)

	return nil
}

type webhook struct {
	decoder  admission.Decoder
	recorder eventRecorder

	apiReader      client.Reader
	requestCounter metric.Int64Counter

	webhookImage     string
	webhookNamespace string
	clusterID        string

	mutators       []dtwebhook.PodMutator
	apmExists      bool
	deployedViaOLM bool
}

func (wh *webhook) Handle(ctx context.Context, request admission.Request) admission.Response {
	ctx, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

	emptyPatch := admission.Patched("")
	mutationRequest, err := wh.createMutationRequestBase(ctx, request)

	if err != nil {
		emptyPatch.Result.Message = fmt.Sprintf("unable to inject into pod (err=%s)", err.Error())
		log.Error(err, "building mutation request base encountered an error")
		span.RecordError(err)

		return emptyPatch
	}

	if mutationRequest == nil {
		emptyPatch.Result.Message = "injection into pod not required"

		return emptyPatch
	}

	podName := mutationRequest.PodName()

	// add podname as attribute (aka. dimension) to metric and span
	countHandleMutationRequest(ctx, podName)
	span.SetAttributes(attribute.String(mutatedPodNameKey, podName))

	if !mutationRequired(mutationRequest) || wh.isOcDebugPod(mutationRequest.Pod) {
		return emptyPatch
	}

	wh.setupEventRecorder(ctx, mutationRequest)

	if wh.isInjected(ctx, mutationRequest) {
		if wh.handlePodReinvocation(ctx, mutationRequest) {
			log.Info("reinvocation policy applied", "podName", podName)
			wh.recorder.sendPodUpdateEvent()

			return createResponseForPod(ctx, mutationRequest.Pod, request)
		}

		log.Info("no change, all containers already injected", "podName", podName)

		return emptyPatch
	}

	if err := wh.handlePodMutation(ctx, mutationRequest); err != nil {
		return silentErrorResponse(ctx, mutationRequest.Pod, err)
	}

	log.Info("injection finished for pod", "podName", podName, "namespace", request.Namespace)

	return createResponseForPod(ctx, mutationRequest.Pod, request)
}

func mutationRequired(mutationRequest *dtwebhook.MutationRequest) bool {
	if mutationRequest == nil {
		return false
	}

	return maputils.GetFieldBool(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, true)
}

func (wh *webhook) setupEventRecorder(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) {
	_, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	wh.recorder.dk = &mutationRequest.DynaKube
	wh.recorder.pod = mutationRequest.Pod
}

func (wh *webhook) isInjected(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) bool {
	_, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	for _, mutator := range wh.mutators {
		if mutator.Injected(mutationRequest.BaseRequest) {
			return true
		}
	}

	installContainer := container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)
	if installContainer != nil {
		log.Info("Dynatrace init-container already present, skipping mutation, doing reinvocation", "containerName", dtwebhook.InstallContainerName)

		return true
	}

	return false
}

func (wh *webhook) isOcDebugPod(pod *corev1.Pod) bool {
	annotations := []string{ocDebugAnnotationsContainer, ocDebugAnnotationsResource}

	for _, annotation := range annotations {
		if _, ok := pod.Annotations[annotation]; !ok {
			return false
		}
	}

	return true
}

func podNeedsInjection(mutationRequest *dtwebhook.MutationRequest) bool {
	needsInjection := false
	for _, container := range mutationRequest.Pod.Spec.Containers {
		needsInjection = needsInjection || !dtwebhookutil.IsContainerExcludedFromInjection(mutationRequest.BaseRequest, container.Name)
	}

	return needsInjection
}

func (wh *webhook) handlePodMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	ctx, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	if !podNeedsInjection(mutationRequest) {
		log.Info("no mutation is needed, all containers are excluded from injection.")

		return nil
	}

	mutationRequest.InstallContainer = createInstallInitContainerBase(wh.webhookImage, wh.clusterID, mutationRequest.Pod, mutationRequest.DynaKube)

	isMutated := false

	for _, mutator := range wh.mutators {
		if !mutator.Enabled(mutationRequest.BaseRequest) {
			continue
		}

		if err := mutator.Mutate(ctx, mutationRequest); err != nil {
			return err
		}

		isMutated = true
	}

	if !isMutated {
		log.Info("no mutation is enabled")

		return nil
	}

	addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
	wh.recorder.sendPodInjectEvent()
	setDynatraceInjectedAnnotation(mutationRequest)

	return nil
}

func (wh *webhook) handlePodReinvocation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) bool {
	_, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	var needsUpdate bool

	reinvocationRequest := mutationRequest.ToReinvocationRequest()

	for _, mutator := range wh.mutators {
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
	ctx, span := dtotel.StartSpan(ctx, webhookotel.Tracer(), spanOptions()...)
	defer span.End()

	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return silentErrorResponse(ctx, pod, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func silentErrorResponse(ctx context.Context, pod *corev1.Pod, err error) admission.Response {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)

	rsp := admission.Patched("")
	podName := k8spod.GetName(*pod)
	log.Error(err, "failed to inject into pod", "podName", podName)
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())

	return rsp
}
