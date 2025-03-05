package oneagent

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	client           client.Client
	apiReader        client.Reader
	clusterID        string
	webhookNamespace string
}

var _ dtwebhook.PodMutator = &Mutator{}

func NewMutator(clusterID, webhookNamespace string, client client.Client, apiReader client.Reader) *Mutator {
	return &Mutator{
		clusterID:        clusterID,
		webhookNamespace: webhookNamespace,
		client:           client,
		apiReader:        apiReader,
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

	// TODO

	oacommon.SetInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mut.Injected(request.BaseRequest) {
		return false
	}

	// TODO

	return true
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

	if dk.OneAgent().GetCodeModulesVersion() == "" && dk.OneAgent().GetCodeModulesImage() == "" {
		log.Info("information about the codemodules (version or image) is not available, OneAgent cannot be injected", "pod", request.PodName())

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
	return true // TODO
}
