package istio

import (
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	protocolHttp  = "http"
	protocolHttps = "https"
)

func buildVirtualService(name, namespace, host, protocol string, port uint32) *istiov1alpha3.VirtualService {
	if isIp(host) {
		return nil
	}

	return &istiov1alpha3.VirtualService{
		ObjectMeta: buildObjectMeta(name, namespace),
		Spec:       buildVirtualServiceSpec(host, protocol, port),
	}
}

func buildVirtualServiceSpec(host, protocol string, port uint32) istio.VirtualService {
	virtualServiceSpec := istio.VirtualService{}
	virtualServiceSpec.Hosts = []string{host}
	switch protocol {
	case protocolHttps:
		virtualServiceSpec.Tls = buildVirtualServiceTLSRoute(host, port)
	case protocolHttp:
		virtualServiceSpec.Http = buildVirtualServiceHttpRoute(port, host)
	}

	return virtualServiceSpec
}

func buildVirtualServiceHttpRoute(port uint32, host string) []*istio.HTTPRoute {
	return []*istio.HTTPRoute{{
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
	}}
}

func buildVirtualServiceTLSRoute(host string, port uint32) []*istio.TLSRoute {
	return []*istio.TLSRoute{{
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
	}}
}
