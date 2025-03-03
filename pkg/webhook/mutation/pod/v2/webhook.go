package pod

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Webhook struct {
	recorder events.EventRecorder

	apiReader  client.Reader
	metaClient client.Client
}

func New(apiReader client.Reader, metaClient client.Client, recorder events.EventRecorder) Webhook {
	return Webhook{
		apiReader:  apiReader,
		metaClient: metaClient,
		recorder:   recorder,
	}
}

func (wh *Webhook) Handle(ctx context.Context, mutationRequest *dtwebhook.MutationRequest) error {
	return nil
}
