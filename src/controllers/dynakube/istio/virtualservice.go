package istio

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/go-logr/logr"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	protocolHttp  = "http"
	protocolHttps = "https"
)

func buildVirtualService(name, namespace, host, protocol string, port uint32) *istiov1alpha3.VirtualService {
	if IsIp(host) {
		return nil
	}

	return &istiov1alpha3.VirtualService{
		ObjectMeta: BuildObjectMeta(name, namespace),
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

func handleIstioConfigurationForVirtualService(instance *dynatracev1beta1.DynaKube,
	name string, communicationHost dtclient.CommunicationHost, role string, config *rest.Config, namespace string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme, log logr.Logger) (bool, error) {

	probe, err := kubeobjects.KubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), name, config)
	if probe == kubeobjects.ProbeObjectFound {
		return false, nil
	} else if probe == kubeobjects.ProbeUnknown {
		log.Error(err, "istio: failed to query VirtualService")
		return false, err
	}

	virtualService := buildVirtualService(name, namespace, communicationHost.Host, communicationHost.Protocol,
		communicationHost.Port)
	if virtualService == nil {
		return false, nil
	}

	err = createIstioConfigurationForVirtualService(instance, virtualService, role, istioClient, scheme)
	if err != nil {
		log.Error(err, "istio: failed to create VirtualService")
		return false, err
	}
	log.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host,
		"port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true, nil
}

func createIstioConfigurationForVirtualService(dynaKube *dynatracev1beta1.DynaKube,
	virtualService *istiov1alpha3.VirtualService, role string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme) error {

	virtualService.Labels = BuildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, virtualService, scheme); err != nil {
		return err
	}
	vs, err := istioClient.NetworkingV1alpha3().VirtualServices(dynaKube.GetNamespace()).Create(context.TODO(), virtualService, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if vs == nil {
		return fmt.Errorf("could not create virtual service with spec %v", virtualService.Spec)
	}

	return nil
}
