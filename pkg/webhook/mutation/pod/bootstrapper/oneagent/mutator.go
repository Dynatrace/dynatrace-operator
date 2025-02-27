package oneagent

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	client    client.Client
	apiReader client.Reader
	image     string
	clusterID string
}

var _ dtwebhook.PodMutator = &Mutator{}

func NewMutator(image, clusterID string, client client.Client, apiReader client.Reader) *Mutator {
	return &Mutator{
		image:     image,
		clusterID: clusterID,
		client:    client,
		apiReader: apiReader,
	}
}

func (mut *Mutator) Enabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInject, request.DynaKube.FeatureAutomaticInjection())
	enabledOnDynakube := request.DynaKube.OneAgent().GetNamespaceSelector() != nil

	matchesNamespaceSelector := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.OneAgent().GetNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.OneAgent().GetNamespaceSelector())

		matchesNamespaceSelector = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespaceSelector && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) Injected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInjected, false)
}

func (mut *Mutator) Mutate(ctx context.Context, request *dtwebhook.MutationRequest) error {
	if ok, reason := mut.isInjectionPossible(request); !ok {
		setNotInjectedAnnotations(request.Pod, reason)

		return nil
	}

	log.Info("injecting OneAgent into pod", "podName", request.PodName())

	mut.addVolumes(request.Pod)
	addInitVolumeMounts(request.InstallContainer)
	addInitArgs(*request.Pod, request.InstallContainer, request.DynaKube)
	mut.mutateUserContainers(request)
	setInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mut.Injected(request.BaseRequest) {
		return false
	}

	log.Info("reinvoking", "podName", request.PodName())

	return mut.reinvokeUserContainers(request)
}

func (mut *Mutator) isInjectionPossible(request *dtwebhook.MutationRequest) (bool, string) {
	var reasons []string

	dk := request.DynaKube

	_, err := dk.TenantUUID()
	if err != nil {
		log.Info("tenant UUID is not available, OneAgent cannot be injected", "pod", request.PodName())

		reasons = append(reasons, emptyTenantUUIDReason)
	}

	if !dk.OneAgent().IsCommunicationRouteClear() {
		log.Info("OneAgent communication route is not clear, OneAgent cannot be injected", "pod", request.PodName())

		reasons = append(reasons, emptyConnectionInfoReason)
	}

	if len(reasons) > 0 {
		return false, strings.Join(reasons, ", ")
	}

	return true, ""
}

func ContainerIsInjected(container corev1.Container) bool {
	return env.IsIn(container.Env, preloadEnv) &&
		volumes.IsIn(container.VolumeMounts, oneAgentCodeModulesVolumeName) &&
		volumes.IsIn(container.VolumeMounts, oneAgentCodeModulesConfigVolumeName)
}
