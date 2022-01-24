package kubeobjects

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/go-logr/logr"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type probeResult int

const (
	probeObjectFound probeResult = iota
	probeObjectNotFound
	ProbeTypeFound
	ProbeTypeNotFound
	probeUnknown
)

var (
	IstioGVRName = "networking.istio.io"

	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   IstioGVRName,
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   IstioGVRName,
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}
)

// CheckIstioEnabled checks if Istio is installed
func CheckIstioEnabled(cfg *rest.Config) (bool, error) {
	discoveryclient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	apiGroupList, err := discoveryclient.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == IstioGVRName {
			return true, nil
		}
	}
	return false, nil
}

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func BuildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}

func buildObjectMeta(name string, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func BuildIstioLabels(name, role string) map[string]string {
	return map[string]string{
		"dynatrace":            "oneagent",
		"oneagent":             name,
		"dynatrace-istio-role": role,
	}
}

func isIp(host string) bool {
	return net.ParseIP(host) != nil
}

func VerifyIstioCrdAvailability(instance *dynatracev1beta1.DynaKube, config *rest.Config) probeResult {
	var probe probeResult

	probe, _ = kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), "", config)
	if probe == ProbeTypeNotFound {
		return probe
	}

	probe, _ = kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), "", config)
	if probe == ProbeTypeNotFound {
		return probe
	}

	return ProbeTypeFound
}

func kubernetesObjectProbe(gvk schema.GroupVersionKind,
	namespace string, name string, config *rest.Config) (probeResult, error) {

	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return probeUnknown, err
	}
	if name == "" {
		err = runtimeClient.List(context.TODO(), &objQuery, client.InNamespace(namespace))
	} else {
		err = runtimeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &objQuery)
	}

	return mapErrorToObjectProbeResult(err)
}

func mapErrorToObjectProbeResult(err error) (probeResult, error) {
	if err != nil {
		if errors.IsNotFound(err) {
			return probeObjectNotFound, err
		} else if meta.IsNoMatchError(err) {
			return ProbeTypeNotFound, err
		}

		return probeUnknown, err
	}

	return probeObjectFound, nil
}

func HandleIstioConfigurationForServiceEntry(instance *dynatracev1beta1.DynaKube,
	name string, communicationHost dtclient.CommunicationHost, role string, config *rest.Config, namespace string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme, log logr.Logger) (bool, error) {

	probe, err := kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), name, config)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		//log.Error(err, "istio: failed to query ServiceEntry")
		return false, err
	}

	serviceEntry := buildServiceEntry(name, namespace, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	err = createIstioConfigurationForServiceEntry(instance, serviceEntry, role, istioClient, scheme)
	if err != nil {
		//log.Error(err, "istio: failed to create ServiceEntry")
		return false, err
	}
	log.Info("istio: ServiceEntry created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port)

	return true, nil
}

func HandleIstioConfigurationForVirtualService(instance *dynatracev1beta1.DynaKube,
	name string, communicationHost dtclient.CommunicationHost, role string, config *rest.Config, namespace string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme, log logr.Logger) (bool, error) {

	probe, err := kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), name, config)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		//log.Error(err, "istio: failed to query VirtualService")
		return false, err
	}

	virtualService := BuildVirtualService(name, namespace, communicationHost.Host, communicationHost.Protocol,
		communicationHost.Port)
	if virtualService == nil {
		return false, nil
	}

	err = createIstioConfigurationForVirtualService(instance, virtualService, role, istioClient, scheme)
	if err != nil {
		//log.Error(err, "istio: failed to create VirtualService")
		return false, err
	}
	log.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host,
		"port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true, nil
}

func createIstioConfigurationForServiceEntry(dynaKube *dynatracev1beta1.DynaKube,
	serviceEntry *istiov1alpha3.ServiceEntry, role string,
	istioClient istioclientset.Interface, scheme *runtime.Scheme) error {

	serviceEntry.Labels = BuildIstioLabels(dynaKube.GetName(), role)
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
