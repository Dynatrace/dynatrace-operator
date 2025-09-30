package metadata

import (
	"maps"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	metaClient client.Client
}

func NewMutator(metaClient client.Client) dtwebhook.Mutator {
	return &Mutator{
		metaClient: metaClient,
	}
}

func (mut *Mutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	if oneagent.IsEnabled(request) && oneagent.IsSelfExtractingImage(request) {
		return true
	}

	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject,
		request.DynaKube.FF().IsAutomaticInjection())
	enabledOnDynakube := request.DynaKube.MetadataEnrichment().IsEnabled()

	matchesNamespace := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.MetadataEnrichment().GetNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.MetadataEnrichment().GetNamespaceSelector())

		matchesNamespace = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespace && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (mut *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	workloadInfo, err := retrieveWorkload(mut.metaClient, request)
	if err != nil {
		return err
	}

	attributes := podattr.Attributes{}
	attributes.WorkloadInfo = podattr.WorkloadInfo{
		WorkloadKind: workloadInfo.Kind,
		WorkloadName: workloadInfo.Name,
	}
	addMetadataToInitArgs(request, &attributes)
	setInjectedAnnotation(request.Pod)
	setWorkloadAnnotations(request.Pod, workloadInfo)

	args, err := podattr.ToArgs(attributes)
	if err != nil {
		return err
	}

	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

	turnOnMetadataEnrichment(request)

	return nil
}

func turnOnMetadataEnrichment(request *dtwebhook.MutationRequest) {
	request.InstallContainer.Args = append(request.InstallContainer.Args, arg.ConvertArgsToStrings([]arg.Arg{{Name: bootstrapper.MetadataEnrichmentFlag}})...)
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	return false
}

func addMetadataToInitArgs(request *dtwebhook.MutationRequest, attributes *podattr.Attributes) {
	copiedMetadataAnnotations := copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
	if copiedMetadataAnnotations == nil {
		log.Info("copied metadata annotations from namespace is empty, propagation is not necessary")

		return
	}

	if attributes.UserDefined == nil {
		attributes.UserDefined = make(map[string]string)
	}

	maps.Copy(attributes.UserDefined, copiedMetadataAnnotations)
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "true"
	delete(pod.Annotations, AnnotationReason)
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[AnnotationInjected] = "false"
		pod.Annotations[AnnotationReason] = reason
	}
}

func setWorkloadAnnotations(pod *corev1.Pod, workload *workloadInfo) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationWorkloadKind] = workload.Kind
	pod.Annotations[AnnotationWorkloadName] = workload.Name
}
