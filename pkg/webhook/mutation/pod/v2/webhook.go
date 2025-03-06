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

var _ dtwebhook.PodInjector = &Injector{}

func NewInjector(apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder) *Injector {
	return &Injector{
		apiReader:  apiReader,
		metaClient: metaClient,
		recorder:   recorder,
	}
}

func (wh *Injector) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	return nil
}
