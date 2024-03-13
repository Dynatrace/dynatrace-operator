package query

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeQuery struct {
	KubeClient client.Client
	KubeReader client.Reader
	Ctx        context.Context
	Log        logd.Logger
}

func New(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logd.Logger) KubeQuery {
	return KubeQuery{
		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Ctx:        ctx,
		Log:        log,
	}
}
