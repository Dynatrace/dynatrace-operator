package istio

import (
	"net"

	istio "istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	protocolHTTP  = "http"
	protocolHTTPS = "https"
)

func buildVirtualService(meta metav1.ObjectMeta, commHosts []CommunicationHost) *istiov1beta1.VirtualService {
	var nonIPhosts []CommunicationHost

	for _, commHost := range commHosts {
		if !isIP(commHost.Host) {
			nonIPhosts = append(nonIPhosts, commHost)
		}
	}

	if len(nonIPhosts) == 0 {
		return nil
	}

	return &istiov1beta1.VirtualService{
		ObjectMeta: meta,
		Spec:       buildVirtualServiceSpec(nonIPhosts),
	}
}

func buildVirtualServiceSpec(commHosts []CommunicationHost) istio.VirtualService {
	hosts := make([]string, len(commHosts))

	var (
		tlses  []*istio.TLSRoute
		routes []*istio.HTTPRoute
	)

	for i, commHost := range commHosts {
		hosts[i] = commHost.Host

		switch commHost.Protocol {
		case protocolHTTPS:
			tlses = append(tlses, buildVirtualServiceTLSRoute(commHost.Host, commHost.Port))
		case protocolHTTP:
			routes = append(routes, buildVirtualServiceHTTPRoute(commHost.Host, commHost.Port))
		}
	}

	return istio.VirtualService{
		Hosts: hosts,
		Http:  routes,
		Tls:   tlses,
	}
}

func buildVirtualServiceHTTPRoute(host string, port uint32) *istio.HTTPRoute {
	return &istio.HTTPRoute{
		Match: []*istio.HTTPMatchRequest{{
			Port: port,
		}},
		Route: []*istio.HTTPRouteDestination{{
			Destination: &istio.Destination{
				Host: host,
				Port: &istio.PortSelector{
					Number: port,
				},
			},
		}},
	}
}

func buildVirtualServiceTLSRoute(host string, port uint32) *istio.TLSRoute {
	return &istio.TLSRoute{
		Match: []*istio.TLSMatchAttributes{{
			SniHosts: []string{host},
			Port:     port,
		}},
		Route: []*istio.RouteDestination{{
			Destination: &istio.Destination{
				Host: host,
				Port: &istio.PortSelector{
					Number: port,
				},
			},
		}},
	}
}

func isIP(host string) bool {
	return net.ParseIP(host) != nil
}
