package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var (
	IstioGVRName    = "networking.istio.io"
	IstioGVRVersion = "v1alpha3"
	IstioGVR        = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)

	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   IstioGVRName,
		Version: IstioGVRVersion,
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   IstioGVRName,
		Version: IstioGVRVersion,
		Kind:    "ServiceEntry",
	}
)

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func BuildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}

func CheckIstioInstalled(cfg *rest.Config) (bool, error) {
	discoveryclient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}

	_, err = discoveryclient.ServerResourcesForGroupVersion(IstioGVR)
	if errors.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}

func buildObjectMeta(name string, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func buildIstioLabels(name, role string) map[string]string {
	return map[string]string{
		"dynatrace":            "oneagent",
		"oneagent":             name,
		"dynatrace-istio-role": role,
	}
}

func isIp(host string) bool {
	return net.ParseIP(host) != nil
}
