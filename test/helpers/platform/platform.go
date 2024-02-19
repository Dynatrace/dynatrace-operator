package platform

import (
	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
)

const (
	openshiftPlatformEnvValue  = "openshift"
	kubernetesPlatformEnvValue = "kubernetes"
)

const openshiftSecurityGVR = "security.openshift.io/v1"

type Resolver struct {
	discoveryProvider func() (discovery.DiscoveryInterface, error)
}

func NewResolver() *Resolver {
	return &Resolver{
		discoveryProvider: getDiscoveryClient,
	}
}

func (p *Resolver) IsOpenshift() (bool, error) {
	client, err := p.discoveryProvider()
	if err != nil {
		// t.Fatal("failed to detect platform from cluster", err)
		return false, err
	}

	_, err = client.ServerResourcesForGroupVersion(openshiftSecurityGVR)

	return !k8serrors.IsNotFound(err), nil
}

func (p *Resolver) GetPlatform() (string, error) {
	isOpenshift, err := p.IsOpenshift()
	if err != nil {
		return "", err
	}
	if isOpenshift {
		return openshiftPlatformEnvValue, nil
	}

	return kubernetesPlatformEnvValue, nil
}

func getDiscoveryClient() (discovery.DiscoveryInterface, error) {
	kubeconfigProvider := config.KubeConfigProvider{}
	kubeconfig, err := kubeconfigProvider.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}
