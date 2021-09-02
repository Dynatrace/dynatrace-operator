package kubesystem

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

type KubernetesVersionProvider interface {
	Major() (string, error)
	Minor() (string, error)
}

type discoveryVersionProvider struct {
	config      *rest.Config
	versionInfo *version.Info
}

func NewVersionProvider(config *rest.Config) KubernetesVersionProvider {
	return &discoveryVersionProvider{
		config: config,
	}
}

func (versionProvider *discoveryVersionProvider) Major() (string, error) {
	versionInfo, err := versionProvider.getVersionInfo()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return versionInfo.Major, nil
}

func (versionProvider *discoveryVersionProvider) Minor() (string, error) {
	versionInfo, err := versionProvider.getVersionInfo()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return versionInfo.Minor, nil
}

func (versionProvider *discoveryVersionProvider) getVersionInfo() (*version.Info, error) {
	if versionProvider.versionInfo != nil {
		return versionProvider.versionInfo, nil
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(versionProvider.config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionProvider.versionInfo = versionInfo
	return versionInfo, nil
}
