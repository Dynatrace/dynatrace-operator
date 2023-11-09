package query

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeQuery struct {
	KubeClient client.Client
	KubeReader client.Reader
	Ctx        context.Context
	Log        logr.Logger
}

func New(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) KubeQuery {
	return KubeQuery{
		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Ctx:        ctx,
		Log:        log,
	}
}
