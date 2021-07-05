package istio

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
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
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type probeResult int

const (
	probeObjectFound probeResult = iota
	probeObjectNotFound
	probeTypeFound
	probeTypeNotFound
	probeUnknown
)

// Controller - manager istioclientset and config
type Controller struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme

	logger logr.Logger
	config *rest.Config
}

// NewController - creates new instance of istio controller
func NewController(config *rest.Config, scheme *runtime.Scheme) *Controller {
	c := &Controller{
		config: config,
		scheme: scheme,
		logger: log.Log.WithName("istio.controller"),
	}
	istioClient, err := c.initialiseIstioClient(config)
	if err != nil {
		return nil
	}
	c.istioClient = istioClient

	return c
}

func (c *Controller) initialiseIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		c.logger.Error(err, "istio: failed to initialize client")
	}

	return ic, err
}

// ReconcileIstio - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (c *Controller) ReconcileIstio(instance *dynatracev1alpha1.DynaKube) (updated bool, err error) {

	enabled, err := CheckIstioEnabled(c.config)
	if err != nil {
		return false, fmt.Errorf("istio: failed to verify Istio availability: %w", err)
	}
	c.logger.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, nil
	}

	apiHost := instance.CommunicationHostForClient()
	if upd, err := c.reconcileIstioConfigurations(instance, []*dtclient.CommunicationHost{&apiHost}, "api-url"); err != nil {
		return false, fmt.Errorf("istio: error reconciling config for Dynatrace API URL: %w", err)
	} else if upd {
		return true, nil
	}

	// Fetch endpoints via Dynatrace client
	ci := instance.ConnectionInfo()
	if upd, err := c.reconcileIstioConfigurations(instance, ci.CommunicationHosts, "communication-endpoint"); err != nil {
		return false, fmt.Errorf("istio: error reconciling config for Dynatrace communication endpoints: %w", err)
	} else if upd {
		return true, nil
	}

	return false, nil
}

func (c *Controller) reconcileIstioConfigurations(instance *dynatracev1alpha1.DynaKube,
	comHosts []*dtclient.CommunicationHost, role string) (bool, error) {

	add, err := c.reconcileIstioCreateConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}
	rem, err := c.reconcileIstioRemoveConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}

	return add || rem, nil
}

func (c *Controller) reconcileIstioRemoveConfigurations(instance *dynatracev1alpha1.DynaKube,
	comHosts []*dtclient.CommunicationHost, role string) (bool, error) {

	labelSelector := labels.SelectorFromSet(buildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[buildNameForEndpoint(instance.GetName(), ch.Protocol, ch.Host, ch.Port)] = true
	}

	vsUpd, err := c.removeIstioConfigurationForVirtualService(listOps, seen, instance.GetNamespace())
	if err != nil {
		return false, err
	}
	seUpd, err := c.removeIstioConfigurationForServiceEntry(listOps, seen, instance.GetNamespace())
	if err != nil {
		return false, err
	}

	return vsUpd || seUpd, nil
}

func (c *Controller) removeIstioConfigurationForServiceEntry(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) (bool, error) {

	list, err := c.istioClient.NetworkingV1alpha3().ServiceEntries(namespace).List(context.TODO(), *listOps)
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("istio: error listing service entries, %v", err))
		return false, err
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			c.logger.Info(fmt.Sprintf("istio: removing %s: %v", se.Kind, se.GetName()))
			err = c.istioClient.NetworkingV1alpha3().
				ServiceEntries(namespace).
				Delete(context.TODO(), se.GetName(), metav1.DeleteOptions{})
			if err != nil {
				c.logger.Error(err, fmt.Sprintf("istio: error deleting service entry, %s : %v", se.GetName(), err))
				continue
			}
			del = true
		}
	}

	return del, nil
}

func (c *Controller) removeIstioConfigurationForVirtualService(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) (bool, error) {

	list, err := c.istioClient.NetworkingV1alpha3().VirtualServices(namespace).List(context.TODO(), *listOps)
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("istio: error listing virtual service, %v", err))
		return false, err
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			c.logger.Info(fmt.Sprintf("istio: removing %s: %v", vs.Kind, vs.GetName()))
			err = c.istioClient.NetworkingV1alpha3().
				VirtualServices(namespace).
				Delete(context.TODO(), vs.GetName(), metav1.DeleteOptions{})
			if err != nil {
				c.logger.Error(err, fmt.Sprintf("istio: error deleting virtual service, %s : %v", vs.GetName(), err))
				continue
			}
			del = true
		}
	}

	return del, nil
}

func (c *Controller) reconcileIstioCreateConfigurations(instance *dynatracev1alpha1.DynaKube,
	communicationHosts []*dtclient.CommunicationHost, role string) (bool, error) {

	crdProbe := c.verifyIstioCrdAvailability(instance)
	if crdProbe != probeTypeFound {
		c.logger.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false, nil
	}

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := buildNameForEndpoint(instance.GetName(), commHost.Protocol, commHost.Host, commHost.Port)

		createdServiceEntry, err := c.handleIstioConfigurationForServiceEntry(instance, name, commHost, role)
		if err != nil {
			return false, err
		}
		createdVirtualService, err := c.handleIstioConfigurationForVirtualService(instance, name, commHost, role)
		if err != nil {
			return false, err
		}

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated, nil
}

func (c *Controller) verifyIstioCrdAvailability(instance *dynatracev1alpha1.DynaKube) probeResult {
	var probe probeResult

	probe, _ = c.kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), "")
	if probe == probeTypeNotFound {
		return probe
	}

	probe, _ = c.kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), "")
	if probe == probeTypeNotFound {
		return probe
	}

	return probeTypeFound
}

func (c *Controller) handleIstioConfigurationForVirtualService(instance *dynatracev1alpha1.DynaKube,
	name string, communicationHost *dtclient.CommunicationHost, role string) (bool, error) {

	probe, err := c.kubernetesObjectProbe(VirtualServiceGVK, instance.GetNamespace(), name)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		c.logger.Error(err, "istio: failed to query VirtualService")
		return false, err
	}

	virtualService := buildVirtualService(name, communicationHost.Host, communicationHost.Protocol,
		communicationHost.Port)
	if virtualService == nil {
		return false, nil
	}

	err = c.createIstioConfigurationForVirtualService(instance, virtualService, role)
	if err != nil {
		c.logger.Error(err, "istio: failed to create VirtualService")
		return false, err
	}
	c.logger.Info("istio: VirtualService created", "objectName", name, "host", communicationHost.Host,
		"port", communicationHost.Port, "protocol", communicationHost.Protocol)

	return true, nil
}

func (c *Controller) handleIstioConfigurationForServiceEntry(instance *dynatracev1alpha1.DynaKube,
	name string, communicationHost *dtclient.CommunicationHost, role string) (bool, error) {

	probe, err := c.kubernetesObjectProbe(ServiceEntryGVK, instance.GetNamespace(), name)
	if probe == probeObjectFound {
		return false, nil
	} else if probe == probeUnknown {
		c.logger.Error(err, "istio: failed to query ServiceEntry")
		return false, err
	}

	serviceEntry := buildServiceEntry(name, communicationHost.Host, communicationHost.Protocol, communicationHost.Port)
	err = c.createIstioConfigurationForServiceEntry(instance, serviceEntry, role)
	if err != nil {
		c.logger.Error(err, "istio: failed to create ServiceEntry")
		return false, err
	}
	c.logger.Info("istio: ServiceEntry created", "objectName", name, "host", communicationHost.Host, "port", communicationHost.Port)

	return true, nil
}

func (c *Controller) createIstioConfigurationForServiceEntry(dynaKube *dynatracev1alpha1.DynaKube,
	serviceEntry *istiov1alpha3.ServiceEntry, role string) error {

	serviceEntry.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, serviceEntry, c.scheme); err != nil {
		return err
	}
	sve, err := c.istioClient.NetworkingV1alpha3().ServiceEntries(dynaKube.GetNamespace()).Create(context.TODO(), serviceEntry, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if sve == nil {
		return fmt.Errorf("could not create service entry with spec %v", serviceEntry.Spec)
	}

	return nil
}

func (c *Controller) createIstioConfigurationForVirtualService(dynaKube *dynatracev1alpha1.DynaKube,
	virtualService *istiov1alpha3.VirtualService, role string) error {

	virtualService.Labels = buildIstioLabels(dynaKube.GetName(), role)
	if err := controllerutil.SetControllerReference(dynaKube, virtualService, c.scheme); err != nil {
		return err
	}
	vs, err := c.istioClient.NetworkingV1alpha3().VirtualServices(dynaKube.GetNamespace()).Create(context.TODO(), virtualService, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if vs == nil {
		return fmt.Errorf("could not create virtual service with spec %v", virtualService.Spec)
	}

	return nil
}

func (c *Controller) kubernetesObjectProbe(gvk schema.GroupVersionKind,
	namespace string, name string) (probeResult, error) {

	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]interface{})

	objQuery.SetGroupVersionKind(gvk)

	runtimeClient, err := client.New(c.config, client.Options{})
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
