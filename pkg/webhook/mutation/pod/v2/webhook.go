package v2

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Injector struct {
	recorder events.EventRecorder

	apiReader  client.Reader
	metaClient client.Client
}

func NewInjector(apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder) Injector {
	return Injector{
		apiReader:  apiReader,
		metaClient: metaClient,
		recorder:   recorder,
	}
}

func (wh *Injector) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	return nil
}

// TODO: use in oneagent.mutator
// if dk.FeatureBootstrapperInjection() {
// 	var initSecret corev1.Secret

// 	secretObjectKey := client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: request.Namespace.Name}
// 	if err := mut.apiReader.Get(request.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
// 		log.Info("dynatrace-bootstrapper-config is not available, OneAgent cannot be injected", "pod", request.PodName())

// 		reasons = append(reasons, NoBootstrapperConfigReason)
// 	}
// }
