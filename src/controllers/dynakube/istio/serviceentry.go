package istio

import (
	"net"
	"strconv"
	"strings"

	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	ignoredSubdomain = "ignored.subdomain"
	subnetMask       = "/32"
	protocolTcp      = "TCP"
)

func buildServiceEntry(name, namespace, host, protocol string, port uint32) *istiov1alpha3.ServiceEntry {
	if net.ParseIP(host) != nil { // It's an IP.
		return buildServiceEntryIP(name, namespace, host, port)
	}

	return buildServiceEntryFQDN(name, namespace, host, protocol, port)
}

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

func buildServiceEntryIP(name, namespace, host string, port uint32) *istiov1alpha3.ServiceEntry {
	portStr := strconv.Itoa(int(port))

	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: buildObjectMeta(name, namespace),
		Spec: istio.ServiceEntry{
			Hosts:     []string{ignoredSubdomain},
			Addresses: []string{host + subnetMask},
			Ports: []*istio.Port{{
				Name:     protocolTcp + "-" + portStr,
				Number:   port,
				Protocol: protocolTcp,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
