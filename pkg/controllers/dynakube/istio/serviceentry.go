package istio

import (
	"strconv"
	"strings"

	istio "istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ignoredSubdomain = "ignored.subdomain"
	subnetMask       = "/32"
	protocolTCP      = "TCP"
)

func BuildNameForIPServiceEntry(ownerName, component string) string {
	return ownerName + "-ip-" + component
}

func BuildNameForFQDNServiceEntry(ownerName, component string) string {
	return ownerName + "-fqdn-" + component
}

func buildServiceEntryFQDNs(meta metav1.ObjectMeta, hostHosts []CommunicationHost) *istiov1beta1.ServiceEntry {
	hosts := make([]string, len(hostHosts))
	portSet := make(map[uint32]bool)

	var ports []*istio.ServicePort

	for i, commHost := range hostHosts {
		portStr := strconv.FormatUint(uint64(commHost.Port), 10)
		protocolStr := strings.ToUpper(commHost.Protocol)
		hosts[i] = commHost.Host

		if !portSet[commHost.Port] {
			ports = append(ports, &istio.ServicePort{
				Name:     commHost.Protocol + "-" + portStr,
				Number:   commHost.Port,
				Protocol: protocolStr,
			})
			portSet[commHost.Port] = true
		}
	}

	return &istiov1beta1.ServiceEntry{
		ObjectMeta: meta,
		Spec: istio.ServiceEntry{
			Hosts:      hosts,
			Ports:      ports,
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

func buildServiceEntryIPs(meta metav1.ObjectMeta, commHosts []CommunicationHost) *istiov1beta1.ServiceEntry {
	var ports []*istio.ServicePort

	portSet := make(map[uint32]bool)
	addresses := make([]string, len(commHosts))

	for i, commHost := range commHosts {
		portStr := strconv.FormatUint(uint64(commHost.Port), 10)
		addresses[i] = commHost.Host + subnetMask

		if !portSet[commHost.Port] {
			ports = append(ports, &istio.ServicePort{
				Name:     protocolTCP + "-" + portStr,
				Number:   commHost.Port,
				Protocol: protocolTCP,
			})
			portSet[commHost.Port] = true
		}
	}

	return &istiov1beta1.ServiceEntry{
		ObjectMeta: meta,
		Spec: istio.ServiceEntry{
			Hosts:      []string{ignoredSubdomain},
			Addresses:  addresses,
			Ports:      ports,
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
