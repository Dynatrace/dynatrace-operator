package istio

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"strconv"
	"strings"

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
	ignoredSubdomain = "ignored.subdomain"
	subnetMask       = "/32"
	protocolTcp      = "TCP"
)

func buildServiceEntryFQDNs(meta metav1.ObjectMeta, hostHosts []dtclient.CommunicationHost) *istiov1alpha3.ServiceEntry {
	var hosts []string
	var ports []*istio.ServicePort
	for _, commHost := range hostHosts {
		portStr := strconv.Itoa(int(commHost.Port))
		protocolStr := strings.ToUpper(commHost.Protocol)
		hosts = append(hosts, commHost.Host)
		ports = append(ports, &istio.ServicePort{
			Name:     commHost.Protocol + "-" + portStr,
			Number:   commHost.Port,
			Protocol: protocolStr,
		})
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
	var addresses []string
	for _, commHost := range commHosts {
		portStr := strconv.Itoa(int(commHost.Port))
		addresses = append(addresses, commHost.Host+subnetMask)
		ports = append(ports, &istio.ServicePort{
			Name:     protocolTcp + "-" + portStr,
			Number:   commHost.Port,
			Protocol: protocolTcp,
		})

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

func handleIstioConfigurationForServiceEntry(istioConfig *configuration, serviceEntry *istiov1alpha3.ServiceEntry) (bool, error) {
	err := createIstioConfigurationForServiceEntry(istioConfig.instance, serviceEntry, istioConfig.role, istioConfig.reconciler.istioClient, istioConfig.reconciler.scheme)
	if errors.IsAlreadyExists(err) {
		return false, nil
	}
	if err != nil {
		log.Error(err, "failed to create ServiceEntry")
		return false, err
	}

	log.Info("ServiceEntry created", "objectName", istioConfig.name, "hosts", getHosts(istioConfig.commHosts), "ports", getCommunicationPorts(istioConfig.commHosts))

	return true, nil
}

func createIstioConfigurationForServiceEntry(dynaKube *dynatracev1beta1.DynaKube, //nolint:revive // argument-limit doesn't apply to constructors
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

func removeIstioConfigurationForServiceEntry(istioConfig *configuration, seen map[string]bool) (bool, error) {
	list, err := istioConfig.reconciler.istioClient.NetworkingV1alpha3().ServiceEntries(istioConfig.instance.GetNamespace()).List(context.TODO(), *istioConfig.listOps)
	if err != nil {
		log.Error(err, "error listing service entries")
		return false, err
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			log.Info("removing service entry", "kind", se.Kind, "name", se.GetName())
			err = istioConfig.reconciler.istioClient.NetworkingV1alpha3().
				ServiceEntries(istioConfig.instance.GetNamespace()).
				Delete(context.TODO(), se.GetName(), metav1.DeleteOptions{})
			if err != nil {
				log.Error(err, "error deleting service entry", "name", se.GetName())
				continue
			}
			del = true
		}
	}

	return del, nil
}

func getHosts(commHosts []dtclient.CommunicationHost) []string {
	var hosts []string
	for _, commHost := range commHosts {
		hosts = append(hosts, commHost.Host)
	}
	return hosts
}

func getCommunicationPorts(commHosts []dtclient.CommunicationHost) []uint32 {
	var ports []uint32
	for _, commHost := range commHosts {
		ports = append(ports, commHost.Port)
	}
	return ports
}
