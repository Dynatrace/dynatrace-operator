package istio

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	protocolHttp  = "http"
	protocolHttps = "https"
)

func buildVirtualService(meta metav1.ObjectMeta, commHosts []dtclient.CommunicationHost) *istiov1alpha3.VirtualService {
	var nonIPhosts []dtclient.CommunicationHost

	for _, commHost := range commHosts {
		if !isIp(commHost.Host) {
			nonIPhosts = append(nonIPhosts, commHost)
		}
	}
	if len(nonIPhosts) == 0 {
		return nil
	}

	return &istiov1alpha3.VirtualService{
		ObjectMeta: meta,
		Spec:       buildVirtualServiceSpec(nonIPhosts),
	}
}

func buildVirtualServiceSpec(commHosts []dtclient.CommunicationHost) istio.VirtualService {
	virtualServiceSpec := istio.VirtualService{}
	var (
		hosts  []string
		tlses  []*istio.TLSRoute
		routes []*istio.HTTPRoute
	)

	for _, commHost := range commHosts {
		hosts = append(hosts, commHost.Host)
		switch commHost.Protocol {
		case protocolHttps:
			tlses = append(tlses, buildVirtualServiceTLSRoute(commHost.Host, commHost.Port))
		case protocolHttp:
			routes = append(routes, buildVirtualServiceHttpRoute(commHost.Host, commHost.Port))
		}
	}

	virtualServiceSpec.Hosts = hosts

	if len(routes) != 0 {
		virtualServiceSpec.Http = routes
	}
	if len(tlses) != 0 {
		virtualServiceSpec.Tls = tlses
	}
	return virtualServiceSpec
}

func buildVirtualServiceHttpRoute(host string, port uint32) *istio.HTTPRoute {
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

func handleIstioConfigurationForVirtualService(istioConfig *configuration) (bool, error) {
	virtualService := buildVirtualService(metav1.ObjectMeta{Name: istioConfig.name, Namespace: istioConfig.instance.GetNamespace()}, istioConfig.commHost.Host, istioConfig.commHost.Protocol,
		istioConfig.commHost.Port)
	if virtualService == nil {
		return false, nil
	}

	err := createIstioConfigurationForVirtualService(istioConfig.instance, virtualService, istioConfig.role, istioConfig.reconciler.istioClient, istioConfig.reconciler.scheme)
	if errors.IsAlreadyExists(err) {
		return false, nil
	}
	if err != nil {
		log.Error(err, "failed to create VirtualService")
		return false, err
	}
	log.Info("VirtualService created", "objectName", istioConfig.name, "host", istioConfig.commHost.Host,
		"port", istioConfig.commHost.Port, "protocol", istioConfig.commHost.Protocol)

	return true, nil
}

func createIstioConfigurationForVirtualService(dynaKube *dynatracev1beta1.DynaKube, //nolint:revive // argument-limit doesn't apply to constructors
	virtualService *istiov1alpha3.VirtualService, role string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme) error {
	virtualService.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, virtualService, scheme); err != nil {
		return err
	}
	createdVirtualService, err := istioClient.NetworkingV1alpha3().VirtualServices(dynaKube.GetNamespace()).Create(context.TODO(), virtualService, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if createdVirtualService == nil {
		return fmt.Errorf("could not create virtual service with spec %v", virtualService.Spec)
	}

	return nil
}

func removeIstioConfigurationForVirtualService(istioConfig *configuration, seen map[string]bool) (bool, error) {
	list, err := istioConfig.reconciler.istioClient.NetworkingV1alpha3().VirtualServices(istioConfig.instance.GetNamespace()).List(context.TODO(), *istioConfig.listOps)
	if err != nil {
		log.Error(err, "error listing virtual service")
		return false, err
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			log.Info("removing virtual service", "kind", vs.Kind, "name", vs.GetName())
			err = istioConfig.reconciler.istioClient.NetworkingV1alpha3().
				VirtualServices(istioConfig.instance.GetNamespace()).
				Delete(context.TODO(), vs.GetName(), metav1.DeleteOptions{})
			if err != nil {
				log.Error(err, "error deleting virtual service", "name", vs.GetName())
				continue
			}
			del = true
		}
	}

	return del, nil
}
