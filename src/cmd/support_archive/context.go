package support_archive

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type supportArchiveContext struct {
	ctx           context.Context
	clientSet     kubernetes.Interface // client set is mandatory to  get access to logs, client.Reader doesn't support log retrieval
	apiReader     client.Reader        // supports unstructured (=generic) object retrieval, which clientSets don't have
	namespaceName string
	toStdout      bool
	targetDir     string
}
