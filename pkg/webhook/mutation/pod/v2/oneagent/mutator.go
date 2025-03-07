package oneagent

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	apiReader client.Reader
}

var _ dtwebhook.PodMutator = &Mutator{}

func NewMutator(apiReader client.Reader) *Mutator {
	return &Mutator{
		apiReader: apiReader,
	}
}

func (mut *Mutator) Enabled(request *dtwebhook.BaseRequest) bool {
	return oacommon.IsEnabled(request)
}

func (mut *Mutator) Injected(request *dtwebhook.BaseRequest) bool {
	return oacommon.IsInjected(request)
}

func (mut *Mutator) Mutate(ctx context.Context, request *dtwebhook.MutationRequest) error {
	if ok, reason := mut.isInjectionPossible(request); !ok {
		oacommon.SetNotInjectedAnnotations(request.Pod, reason)

		return nil
	}

	log.Info("injecting OneAgent into pod", "podName", request.PodName())

	addInitVolumeMounts(request.InstallContainer)
	addInitArgs(*request.Pod, request.InstallContainer, request.DynaKube)
	addVolumes(request.Pod)

	mut.mutateUserContainers(request)

	oacommon.SetInjectedAnnotation(request.Pod)

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
	reasons := []string{}

	dk := request.DynaKube

	_, err := dk.TenantUUID()
	if err != nil {
		log.Info("tenant UUID is not available, OneAgent cannot be injected", "pod", request.PodName())

		reasons = append(reasons, oacommon.EmptyTenantUUIDReason)
	}

	if !dk.OneAgent().IsCommunicationRouteClear() {
		log.Info("OneAgent communication route is not clear, OneAgent cannot be injected", "pod", request.PodName())

		reasons = append(reasons, oacommon.EmptyConnectionInfoReason)
	}

	if dk.OneAgent().GetCustomCodeModulesImage() == "" {
		log.Info("code modules version not set, OneAgent cannot be injected", "pod", request.PodName())

		reasons = append(reasons, oacommon.UnknownCodeModuleReason)
	}

	if dk.FeatureBootstrapperInjection() {
		var initSecret corev1.Secret

		secretObjectKey := client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: request.Namespace.Name}
		if err := mut.apiReader.Get(request.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
			log.Info("dynatrace-bootstrapper-config is not available, OneAgent cannot be injected", "pod", request.PodName())

			reasons = append(reasons, NoBootstrapperConfigReason)
		}
	}

	if len(reasons) > 0 {
		return false, strings.Join(reasons, ", ")
	}

	return true, ""
}

func ContainerIsInjected(container corev1.Container) bool {
	return env.IsIn(container.Env, oacommon.PreloadEnv) &&
		volumes.IsIn(container.VolumeMounts, oneAgentCodeModulesVolumeName) &&
		volumes.IsIn(container.VolumeMounts, oneAgentCodeModulesConfigVolumeName)
}
