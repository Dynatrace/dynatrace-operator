package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	IstioGVRName    = "networking.istio.io"
	IstioGVRVersion = "v1alpha3"
	IstioGVR        = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
)

// TODO: Remove (e2e)
// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func BuildNameForEndpoint(commHosts []dtclient.CommunicationHost, name string) string {
	result := make([]string, len(commHosts))
	for index, commHost := range commHosts {
		result[index] = fmt.Sprintf("%s-%s-%s-%d", name, commHost.Protocol, commHost.Host, commHost.Port)
	}
	sum := sha256.Sum256([]byte(strings.Join(result, "\n")))
	return hex.EncodeToString(sum[:])
}

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
