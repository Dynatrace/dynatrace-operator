package manager

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Provider interface {
	CreateManager(namespace string, config *rest.Config) (manager.Manager, error)
}
