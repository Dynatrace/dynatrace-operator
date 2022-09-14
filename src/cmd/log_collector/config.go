package log_collector

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"
)

type logCollectorContext struct {
	ctx           context.Context
	clientSet     *kubernetes.Clientset
	namespaceName string // the default namespace ("dynatrace") or provided in the command line
	stream        bool

	wg sync.WaitGroup
}

var (
	log = newLogCollectorLogger("[log collector]")
)
