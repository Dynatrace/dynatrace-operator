package metadata

import (
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	metaClient client.Client
}

func NewMutator(metaClient client.Client) mutator.Mutator {
	return &Mutator{
		metaClient: metaClient,
	}
}

func (mut *Mutator) IsEnabled(request *mutator.BaseRequest) bool {
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

func (mut *Mutator) IsInjected(request *mutator.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (mut *Mutator) Mutate(request *mutator.MutationRequest) error {
	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	attrs := podattr.Attributes{}

	attrs, err := attributes.GetWorkloadInfoAttributes(attrs, request.Context, request.BaseRequest, mut.metaClient)
	if err != nil {
		return mutator.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(OwnerLookupFailedReason),
		}
	}

	attrs = attributes.GetMetadataAnnotations(attrs, request.BaseRequest)

	log.Debug("collected metadata attributes for pod", "attributes", attrs)

	setInjectedAnnotation(request.Pod)

	args, err := podattr.ToArgs(attrs)
	if err != nil {
		return err
	}

	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

	turnOnMetadataEnrichment(request)

	return nil
}

func turnOnMetadataEnrichment(request *mutator.MutationRequest) {
	request.InstallContainer.Args = append(request.InstallContainer.Args, arg.ConvertArgsToStrings([]arg.Arg{{Name: bootstrapper.MetadataEnrichmentFlag}})...)
}

func (mut *Mutator) Reinvoke(request *mutator.ReinvocationRequest) bool {
	return false
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
