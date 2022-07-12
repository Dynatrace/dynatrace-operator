package istio

import (
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// IstioReconciler - manager istioclientset and config
type IstioReconciler struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme
	config      *rest.Config
	namespace   string
}

type istioConfiguration struct {
	instance   *dynatracev1beta1.DynaKube
	reconciler *IstioReconciler
	name       string
	commHost   *dtclient.CommunicationHost
	role       string
	listOps    *metav1.ListOptions
}

// NewIstioReconciler - creates new instance of istio controller
func NewIstioReconciler(config *rest.Config, scheme *runtime.Scheme) *IstioReconciler {
	reconciler := &IstioReconciler{
		config:    config,
		scheme:    scheme,
		namespace: os.Getenv("POD_NAMESPACE"),
	}
	istioClient, err := reconciler.initializeIstioClient(config)
	if err != nil {
		return nil
	}
	reconciler.istioClient = istioClient

	return reconciler
}

func (reconciler *IstioReconciler) initializeIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		log.Error(err, "istio: failed to initialize client")
	}

	return ic, err
}

// ReconcileIstio - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (reconciler *IstioReconciler) ReconcileIstio(instance *dynatracev1beta1.DynaKube) (bool, error) {
	enabled, err := CheckIstioEnabled(reconciler.config)
	if err != nil {
		return false, fmt.Errorf("istio: failed to verify Istio availability: %w", err)
	}
	log.Info("istio: status", "enabled", enabled)

	if !enabled {
		return false, nil
	}

	apiHost, err := dtclient.ParseEndpoint(instance.Spec.APIURL)
	if err != nil {
		return false, err
	}

	upd, err := reconciler.reconcileIstioConfigurations(instance, []dtclient.CommunicationHost{apiHost}, "api-url")
	if err != nil {
		return false, errors.WithMessage(err, "istio: error reconciling config for Dynatrace API URL")
	} else if upd {
		return true, nil
	}

	// Fetch endpoints via Dynatrace client
	ci := instance.ConnectionInfo()
	upd, err = reconciler.reconcileIstioConfigurations(instance, ci.CommunicationHosts, "communication-endpoint")
	if err != nil {
		return false, errors.WithMessage(err, "istio: error reconciling config for Dynatrace communication endpoints:")
	} else if upd {
		return true, nil
	}

	return false, nil
}

func (reconciler *IstioReconciler) reconcileIstioConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	add, err := reconciler.reconcileIstioCreateConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}
	rem, err := reconciler.reconcileIstioRemoveConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}

	return add || rem, nil
}

func (reconciler *IstioReconciler) reconcileIstioRemoveConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {

	labelSelector := labels.SelectorFromSet(buildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seenComHosts := map[string]bool{}
	for _, comHost := range comHosts {
		seenComHosts[buildNameForEndpoint(instance.GetName(), comHost.Protocol, comHost.Host, comHost.Port)] = true
	}

	istioConfig := &istioConfiguration{
		reconciler: reconciler,
		listOps:    listOps,
		instance:   instance,
	}

	vsUpd, err := removeIstioConfigurationForVirtualService(istioConfig, seenComHosts)
	if err != nil {
		return false, err
	}
	seUpd, err := removeIstioConfigurationForServiceEntry(istioConfig, seenComHosts)
	if err != nil {
		return false, err
	}

	return vsUpd || seUpd, nil
}

func (reconciler *IstioReconciler) reconcileIstioCreateConfigurations(instance *dynatracev1beta1.DynaKube,
	communicationHosts []dtclient.CommunicationHost, role string) (bool, error) {

	crdProbe := verifyIstioCrdAvailability(instance, reconciler.config)
	if crdProbe != kubeobjects.ProbeTypeFound {
		log.Info("istio: failed to lookup CRD for ServiceEntry/VirtualService: Did you install Istio recently? Please restart the Operator.")
		return false, nil
	}

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := buildNameForEndpoint(instance.GetName(), commHost.Protocol, commHost.Host, commHost.Port)

		istioConfig := &istioConfiguration{
			instance:   instance,
			reconciler: reconciler,
			name:       name,
			commHost:   &commHost,
			role:       role,
		}

		createdServiceEntry, err := handleIstioConfigurationForServiceEntry(istioConfig, log)
		if err != nil {
			return false, err
		}
		createdVirtualService, err := handleIstioConfigurationForVirtualService(istioConfig, log)
		if err != nil {
			return false, err
		}

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated, nil
}
