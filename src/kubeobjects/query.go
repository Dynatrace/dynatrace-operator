package kubeobjects

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type complexKubeQuery struct {
	kubeClient client.Client
	kubeReader client.Reader
	ctx        context.Context
	log        logr.Logger
}

func newComplexKubeQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) complexKubeQuery {
	return complexKubeQuery{
		kubeClient: kubeClient,
		kubeReader: kubeReader,
		ctx:        ctx,
		log:        log,
	}
}
