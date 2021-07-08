package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var (
	istioGVRName = "networking.istio.io"

	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}
)

// CheckIstioEnabled checks if Istio is installed
func CheckIstioEnabled(cfg *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	apiGroupList, err := client.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == istioGVRName {
			return true, nil
		}
	}
	return false, nil
}

// BuildServiceEntry returns an Istio ServiceEntry object for the given communication endpoint.
func buildServiceEntry(name, namespace, host, protocol string, port uint32) *istiov1alpha3.ServiceEntry {
	if net.ParseIP(host) != nil { // It's an IP.
		return buildServiceEntryIP(name, namespace, host, port)
	}

	return buildServiceEntryFQDN(name, namespace, host, protocol, port)
}

// BuildVirtualService returns an Istio VirtualService object for the given communication endpoint.
func buildVirtualService(name, namespace, host, protocol string, port uint32) *istiov1alpha3.VirtualService {
	if net.ParseIP(host) != nil { // It's an IP.
		return nil
	}

	return &istiov1alpha3.VirtualService{
		ObjectMeta: buildObjectMeta(name, namespace),
		Spec:       buildVirtualServiceSpec(host, protocol, port),
	}
}

// buildServiceEntryFQDN returns an Istio ServiceEntry object for the given communication endpoint with a FQDN host.
func buildServiceEntryFQDN(name, namespace, host, protocol string, port uint32) *istiov1alpha3.ServiceEntry {
	portStr := strconv.Itoa(int(port))
	protocolStr := strings.ToUpper(protocol)

	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: buildObjectMeta(name, namespace),
		Spec: istio.ServiceEntry{
			Hosts: []string{host},
			Ports: []*istio.Port{{
				Name:     protocol + "-" + portStr,
				Number:   port,
				Protocol: protocolStr,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

// buildServiceEntryIP returns an Istio ServiceEntry object for the given communication endpoint with IP.
func buildServiceEntryIP(name, namespace, host string, port uint32) *istiov1alpha3.ServiceEntry {
	portStr := strconv.Itoa(int(port))

	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: buildObjectMeta(name, namespace),
		Spec: istio.ServiceEntry{
			Hosts:     []string{"ignored.subdomain"},
			Addresses: []string{host + "/32"},
			Ports: []*istio.Port{{
				Name:     "TCP-" + portStr,
				Number:   port,
				Protocol: "TCP",
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func buildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}

func buildObjectMeta(name string, namespace string) v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func mapErrorToObjectProbeResult(err error) (probeResult, error) {
	if err != nil {
		if errors.IsNotFound(err) {
			return probeObjectNotFound, err
		} else if meta.IsNoMatchError(err) {
			return probeTypeNotFound, err
		}

		return probeUnknown, err
	}

	return probeObjectFound, nil
}

func buildIstioLabels(name, role string) map[string]string {
	return map[string]string{
		"dynatrace":            "oneagent",
		"oneagent":             name,
		"dynatrace-istio-role": role,
	}
}
