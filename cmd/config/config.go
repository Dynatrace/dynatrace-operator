package config

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Provider interface {
	GetConfig() (*rest.Config, error)
}

type KubeConfigProvider struct {
}

func NewKubeConfigProvider() Provider {
	return KubeConfigProvider{}
}

func (provider KubeConfigProvider) GetConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()

	return cfg, errors.WithStack(err)
}
