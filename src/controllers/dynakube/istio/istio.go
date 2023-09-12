package istio

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	IstioGVRName    = "networking.istio.io"
	IstioGVRVersion = "v1alpha3"
	IstioGVR        = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
)

// TODO: Maybe move to Client
// CheckIstioInstalled run discovery query for server resource for group version
func CheckIstioInstalled(discoveryClient discovery.DiscoveryInterface) (bool, error) {
	_, err := discoveryClient.ServerResourcesForGroupVersion(IstioGVR)
	if errors.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}

func buildObjectMeta(name, namespace string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
}
