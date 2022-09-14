package log_collector

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

type logCollectorContext struct {
	ctx           context.Context
	clientSet     *kubernetes.Clientset
	namespaceName string // the default namespace ("dynatrace") or provided in the command line
}

var (
	log = newLogCollectorLogger("[log collector]")
)
