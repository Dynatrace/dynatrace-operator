package istio

import (
	"context"
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type probeResult int

const (
	probeObjectFound probeResult = iota
	probeObjectNotFound
	probeTypeFound
	probeTypeNotFound
	probeUnknown
)

// IstioReconciler - manager istioclientset and config
type IstioReconciler struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme
	config      *rest.Config
	namespace   string
}

// NewIstioReconciler - creates new instance of istio controller
func NewIstioReconciler(config *rest.Config, scheme *runtime.Scheme) *IstioReconciler {
	c := &IstioReconciler{
		config:    config,
		scheme:    scheme,
		namespace: os.Getenv("POD_NAMESPACE"),
	}
	istioClient, err := c.initializeIstioClient(config)
	if err != nil {
		return nil
	}
	c.istioClient = istioClient

	return c
}

func (r *IstioReconciler) initializeIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		log.Error(err, "istio: failed to initialize client")
	}

	return ic, err
}

// ReconcileIstio - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (r *IstioReconciler) ReconcileIstio(instance *dynatracev1beta1.DynaKube) (updated bool, err error) {

	enabled, err := CheckIstioEnabled(r.config)
	if err != nil {
		return false, fmt.Errorf("istio: failed to verify Istio availability: %w", err)
	}
	log.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, nil
	}

	apiHost := instance.CommunicationHostForClient()
	if upd, err := r.reconcileIstioConfigurations(instance, []dtclient.CommunicationHost{apiHost}, "api-url"); err != nil {
		return false, fmt.Errorf("istio: error reconciling config for Dynatrace API URL: %w", err)
	} else if upd {
		return true, nil
	}

	// Fetch endpoints via Dynatrace client
	ci := instance.ConnectionInfo()
	if upd, err := r.reconcileIstioConfigurations(instance, ci.CommunicationHosts, "communication-endpoint"); err != nil {
		return false, fmt.Errorf("istio: error reconciling config for Dynatrace communication endpoints: %w", err)
	} else if upd {
		return true, nil
	}

	return false, nil
}

func (r *IstioReconciler) reconcileIstioConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	add, err := r.reconcileIstioCreateConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}
	rem, err := r.reconcileIstioRemoveConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}

	return add || rem, nil
}

func (r *IstioReconciler) reconcileIstioRemoveConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	labelSelector := labels.SelectorFromSet(buildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[buildNameForEndpoint(instance.GetName(), ch.Protocol, ch.Host, ch.Port)] = true
	}

	vsUpd, err := r.removeIstioConfigurationForVirtualService(listOps, seen, instance.GetNamespace())
	if err != nil {
		return false, err
	}
	seUpd, err := r.removeIstioConfigurationForServiceEntry(listOps, seen, instance.GetNamespace())
	if err != nil {
		return false, err
	}

	return vsUpd || seUpd, nil
}

func (r *IstioReconciler) removeIstioConfigurationForServiceEntry(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) (bool, error) {

	list, err := r.istioClient.NetworkingV1alpha3().ServiceEntries(namespace).List(context.TODO(), *listOps)
	if err != nil {
		log.Error(err, "istio: error listing service entries")
		return false, err
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			log.Info("istio: removing", "kind", se.Kind, "name", se.GetName())
			err = r.istioClient.NetworkingV1alpha3().
				ServiceEntries(namespace).
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

func (r *IstioReconciler) removeIstioConfigurationForVirtualService(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) (bool, error) {

	list, err := r.istioClient.NetworkingV1alpha3().VirtualServices(namespace).List(context.TODO(), *listOps)
	if err != nil {
		log.Error(err, "istio: error listing virtual service")
		return false, err
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			log.Info("istio: removing", "kind", vs.Kind, "name", vs.GetName())
			err = r.istioClient.NetworkingV1alpha3().
				VirtualServices(namespace).
				Delete(context.TODO(), vs.GetName(), metav1.DeleteOptions{})
			if err != nil {
				log.Error(err, "istio: error deleting virtual service", "name", vs.GetName())
				continue
			}
			del = true
		}
	}

	return del, nil
}

func (r *IstioReconciler) reconcileIstioCreateConfigurations(instance *dynatracev1beta1.DynaKube,
	communicationHosts []dtclient.CommunicationHost, role string) (bool, error) {

	crdProbe := r.verifyIstioCrdAvailability(instance)
	if crdProbe != probeTypeFound {
		log.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false, nil
	}

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := buildNameForEndpoint(instance.GetName(), commHost.Protocol, commHost.Host, commHost.Port)

		createdServiceEntry, err := r.handleIstioConfigurationForServiceEntry(instance, name, commHost, role)
		if err != nil {
			return false, err
		}
		createdVirtualService, err := r.handleIstioConfigurationForVirtualService(instance, name, commHost, role)
		if err != nil {
			return false, err
		}

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated, nil
}

func (r *IstioReconciler) verifyIstioCrdAvailability(instance *dynatracev1beta1.DynaKube) probeResult {
	var probe probeResult

	probe, _ = r.kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), "")
	if probe == probeTypeNotFound {
		return probe
	}

	probe, _ = r.kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), "")
	if probe == probeTypeNotFound {
		return probe
	}

	return probeTypeFound
}

func (r *IstioReconciler) handleIstioConfigurationForVirtualService(instance *dynatracev1beta1.DynaKube,
	name string, communicationHost dtclient.CommunicationHost, role string) (bool, error) {

	probe, err := r.kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), name)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		log.Error(err, "istio: failed to query VirtualService")
		return false, err
	}

	virtualService := buildVirtualService(name, r.namespace, communicationHost.Host, communicationHost.Protocol,
		communicationHost.Port)
	if virtualService == nil {
		return false, nil
	}

	err = r.createIstioConfigurationForVirtualService(instance, virtualService, role)
	if err != nil {
		log.Error(err, "istio: failed to create VirtualService")
		return false, err
	}
	log.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host,
		"port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true, nil
}

func (r *IstioReconciler) handleIstioConfigurationForServiceEntry(instance *dynatracev1beta1.DynaKube,
	name string, communicationHost dtclient.CommunicationHost, role string) (bool, error) {

	probe, err := r.kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), name)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		log.Error(err, "istio: failed to query ServiceEntry")
		return false, err
	}

	serviceEntry := buildServiceEntry(name, r.namespace, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	err = r.createIstioConfigurationForServiceEntry(instance, serviceEntry, role)
	if err != nil {
		log.Error(err, "istio: failed to create ServiceEntry")
		return false, err
	}
	log.Info("istio: ServiceEntry created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port)

	return true, nil
}

func (r *IstioReconciler) createIstioConfigurationForServiceEntry(dynaKube *dynatracev1beta1.DynaKube,
	serviceEntry *istiov1alpha3.ServiceEntry, role string) error {

	serviceEntry.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, serviceEntry, r.scheme); err != nil {
		return err
	}
	sve, err := r.istioClient.NetworkingV1alpha3().ServiceEntries(dynaKube.GetNamespace()).Create(context.TODO(), serviceEntry, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if sve == nil {
		return fmt.Errorf("could not create service entry with spec %v", serviceEntry.Spec)
	}
	return nil
}

func (r *IstioReconciler) createIstioConfigurationForVirtualService(dynaKube *dynatracev1beta1.DynaKube,
	virtualService *istiov1alpha3.VirtualService, role string) error {

	virtualService.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, virtualService, r.scheme); err != nil {
		return err
	}
	vs, err := r.istioClient.NetworkingV1alpha3().VirtualServices(dynaKube.GetNamespace()).Create(context.TODO(), virtualService, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if vs == nil {
		return fmt.Errorf("could not create virtual service with spec %v", virtualService.Spec)
	}

	return nil
}

func (r *IstioReconciler) kubernetesObjectProbe(gvk schema.GroupVersionKind,
	namespace string, name string) (probeResult, error) {

	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	runtimeClient, err := client.New(r.config, client.Options{})
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
