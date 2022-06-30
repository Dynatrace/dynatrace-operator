package config

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Provider interface {
	GetConfig() (*rest.Config, error)
}

type kubeConfigProvider struct {
}

func NewKubeConfigProvider() Provider {
	return kubeConfigProvider{}
}

func (provider kubeConfigProvider) GetConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()
	return cfg, errors.WithStack(err)
}
