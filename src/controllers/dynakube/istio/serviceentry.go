package istio

import (
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ignoredSubdomain = "ignored.subdomain"
	subnetMask       = "/32"
	protocolTcp      = "TCP"
)

func BuildNameForIPServiceEntry(ownerName, role string) string {
	return ownerName + "-ip-" + role
}

func BuildNameForFQDNServiceEntry(ownerName, role string) string {
	return ownerName + "-fqdn-" + role
}


func buildServiceEntryFQDNs(meta metav1.ObjectMeta, hostHosts []dtclient.CommunicationHost) *istiov1alpha3.ServiceEntry {
	hosts := make([]string, len(hostHosts))
	portSet := make(map[uint32]bool)
	var ports []*istio.ServicePort

	for i, commHost := range hostHosts {
		portStr := strconv.Itoa(int(commHost.Port))
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
	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: meta,
		Spec: istio.ServiceEntry{
			Hosts:      hosts,
			Ports:      ports,
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

func buildServiceEntryIPs(meta metav1.ObjectMeta, commHosts []dtclient.CommunicationHost) *istiov1alpha3.ServiceEntry {
	var ports []*istio.ServicePort
	portSet := make(map[uint32]bool)
	addresses := make([]string, len(commHosts))
	for i, commHost := range commHosts {
		portStr := strconv.Itoa(int(commHost.Port))
		addresses[i] = commHost.Host + subnetMask
		if !portSet[commHost.Port] {
			ports = append(ports, &istio.ServicePort{
				Name:     protocolTcp + "-" + portStr,
				Number:   commHost.Port,
				Protocol: protocolTcp,
			})
			portSet[commHost.Port] = true
		}
	}

	return &istiov1alpha3.ServiceEntry{
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
