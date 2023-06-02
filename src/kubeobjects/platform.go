package kubeobjects

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
)

const (
	openshiftPlatformEnvValue  = "openshift"
	kubernetesPlatformEnvValue = "kubernetes"
)

type Platform int

const (
	Kubernetes Platform = iota
	Openshift
)

const SccGVR = "security.openshift.io/v1"

type discoveryClientCreation interface {
	getDiscoveryClient() (discovery.DiscoveryInterface, error)
}

type DiscoveryClientResolver struct{}

func (p *DiscoveryClientResolver) getDiscoveryClient() (discovery.DiscoveryInterface, error) {
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

type PlatformResolver struct {
	discoveryClientCreation
}

func NewPlatformResolver() *PlatformResolver {
	return &PlatformResolver{
		discoveryClientCreation: &DiscoveryClientResolver{},
	}
}

func (p *PlatformResolver) IsOpenshift(t *testing.T) bool {
	client, err := p.getDiscoveryClient()
	if err != nil {
		t.Fatal("failed to detect platform from cluster", err)
		return false
	}

	_, err = client.ServerResourcesForGroupVersion(SccGVR)
	return !k8serrors.IsNotFound(err)
}

func (p *PlatformResolver) GetPlatform(t *testing.T) string {
	isOpenshift := p.IsOpenshift(t)
	if isOpenshift {
		return openshiftPlatformEnvValue
	}
	return kubernetesPlatformEnvValue
}
