package istio

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/go-logr/logr"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func handleIstioConfigurationForServiceEntry(istioConfig *istioConfiguration, log logr.Logger) (bool, error) {

	probe, err := kubeobjects.KubernetesObjectProbe(ServiceEntryGVK, istioConfig.instance.GetNamespace(), istioConfig.name, istioConfig.reconciler.config)
	if probe == kubeobjects.ProbeObjectFound {
		return false, nil
	} else if probe == kubeobjects.ProbeUnknown {
		log.Error(err, "istio: failed to query ServiceEntry")
		return false, err
	}

	serviceEntry := buildServiceEntry(istioConfig.name, istioConfig.instance.GetNamespace(), istioConfig.commHost.Host, istioConfig.commHost.Protocol, istioConfig.commHost.Port)
	err = createIstioConfigurationForServiceEntry(istioConfig.instance, serviceEntry, istioConfig.role, istioConfig.reconciler.istioClient, istioConfig.reconciler.scheme)
	if err != nil {
		log.Error(err, "istio: failed to create ServiceEntry")
		return false, err
	}
	log.Info("istio: ServiceEntry created", "objectName", istioConfig.name, "host", istioConfig.commHost.Host, "port", istioConfig.commHost.Port)

	return true, nil
}

func createIstioConfigurationForServiceEntry(dynaKube *dynatracev1beta1.DynaKube,
	serviceEntry *istiov1alpha3.ServiceEntry, role string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme) error {

	serviceEntry.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, serviceEntry, scheme); err != nil {
		return err
	}
	sve, err := istioClient.NetworkingV1alpha3().ServiceEntries(dynaKube.GetNamespace()).Create(context.TODO(), serviceEntry, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if sve == nil {
		return fmt.Errorf("could not create service entry with spec %v", serviceEntry.Spec)
	}
	return nil
}

func removeIstioConfigurationForServiceEntry(istioConfig *istioConfiguration, seen map[string]bool) (bool, error) {

	list, err := istioConfig.reconciler.istioClient.NetworkingV1alpha3().ServiceEntries(istioConfig.instance.GetNamespace()).List(context.TODO(), *istioConfig.listOps)
	if err != nil {
		log.Error(err, "istio: error listing service entries")
		return false, err
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			log.Info("istio: removing", "kind", se.Kind, "name", se.GetName())
			err = istioConfig.reconciler.istioClient.NetworkingV1alpha3().
				ServiceEntries(istioConfig.instance.GetNamespace()).
				Delete(context.TODO(), se.GetName(), metav1.DeleteOptions{})
			if err != nil {
				log.Error(err, "istio: error deleting service entry", "name", se.GetName())
				continue
			}
			del = true
		}
	}

	return del, nil
}
