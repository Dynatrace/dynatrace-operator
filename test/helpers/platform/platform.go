//go:build e2e

package platform

import (
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	if err == nil {
		return true, nil
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return false, err
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
	kubeconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}
