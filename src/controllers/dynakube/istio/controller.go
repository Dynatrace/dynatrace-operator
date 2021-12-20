package istio

import (
	"context"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// Controller - manager istioclientset and config
type Controller struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme
	config      *rest.Config
	namespace   string
}

// NewController - creates new instance of istio controller
func NewController(config *rest.Config, scheme *runtime.Scheme) *Controller {
	c := &Controller{
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

func (c *Controller) initializeIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		log.Error(err, "istio: failed to initialize client")
	}

	return ic, err
}

// ReconcileIstio - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (c *Controller) ReconcileIstio(instance *dynatracev1beta1.DynaKube) (updated bool, err error) {

	enabled, err := kubeobjects.CheckIstioEnabled(c.config)
	if err != nil {
		return false, fmt.Errorf("istio: failed to verify Istio availability: %w", err)
	}
	log.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, nil
	}

	apiHost := instance.CommunicationHostForClient()
	if upd, err := c.reconcileIstioConfigurations(instance, []dtclient.CommunicationHost{apiHost}, "api-url"); err != nil {
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

func (c *Controller) reconcileIstioConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

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

func (c *Controller) reconcileIstioRemoveConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	labelSelector := labels.SelectorFromSet(kubeobjects.BuildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seen := map[string]bool{}
	for _, ch := range comHosts {
		seen[kubeobjects.BuildNameForEndpoint(instance.GetName(), ch.Protocol, ch.Host, ch.Port)] = true
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
		log.Error(err, "istio: error listing service entries")
		return false, err
	}

	del := false
	for _, se := range list.Items {
		if _, inUse := seen[se.GetName()]; !inUse {
			log.Info("istio: removing", "kind", se.Kind, "name", se.GetName())
			err = c.istioClient.NetworkingV1alpha3().
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

func (c *Controller) removeIstioConfigurationForVirtualService(listOps *metav1.ListOptions,
	seen map[string]bool, namespace string) (bool, error) {

	list, err := c.istioClient.NetworkingV1alpha3().VirtualServices(namespace).List(context.TODO(), *listOps)
	if err != nil {
		log.Error(err, "istio: error listing virtual service")
		return false, err
	}

	del := false
	for _, vs := range list.Items {
		if _, inUse := seen[vs.GetName()]; !inUse {
			log.Info("istio: removing", "kind", vs.Kind, "name", vs.GetName())
			err = c.istioClient.NetworkingV1alpha3().
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

func (c *Controller) reconcileIstioCreateConfigurations(instance *dynatracev1beta1.DynaKube,
	communicationHosts []dtclient.CommunicationHost, role string) (bool, error) {

	crdProbe := kubeobjects.VerifyIstioCrdAvailability(instance, c.config)
	if crdProbe != kubeobjects.ProbeTypeFound {
		log.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false, nil
	}

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := kubeobjects.BuildNameForEndpoint(instance.GetName(), commHost.Protocol, commHost.Host, commHost.Port)

		createdServiceEntry, err := kubeobjects.HandleIstioConfigurationForServiceEntry(instance, name, commHost,
			role, c.config, c.namespace, c.istioClient, c.scheme)
		if err != nil {
			return false, err
		}
		createdVirtualService, err := kubeobjects.HandleIstioConfigurationForVirtualService(instance, name, commHost,
			role, c.config, c.namespace, c.istioClient, c.scheme)
		if err != nil {
			return false, err
		}

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated, nil
}
