package istio

import (
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

// Reconciler - manager istioclientset and config
type Reconciler struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme
	config      *rest.Config
	namespace   string
}

type configuration struct {
	instance   *dynatracev1beta1.DynaKube
	reconciler *Reconciler
	name       string
	commHost   *dtclient.CommunicationHost
	role       string
	listOps    *metav1.ListOptions
}

// NewReconciler - creates new instance of istio controller
func NewReconciler(config *rest.Config, scheme *runtime.Scheme) *Reconciler {
	reconciler := &Reconciler{
		config:    config,
		scheme:    scheme,
		namespace: os.Getenv(kubeobjects.EnvPodNamespace),
	}
	istioClient, err := reconciler.initializeIstioClient(config)
	if err != nil {
		return nil
	}
	reconciler.istioClient = istioClient

	return reconciler
}

func (reconciler *Reconciler) initializeIstioClient(config *rest.Config) (istioclientset.Interface, error) {
	ic, err := istioclientset.NewForConfig(config)
	if err != nil {
		log.Error(err, "failed to initialize client")
	}

	return ic, err
}

// Reconcile - runs the istio's reconcile workflow,
// creating/deleting VS & SE for external communications
func (reconciler *Reconciler) Reconcile(instance *dynatracev1beta1.DynaKube, communicationHosts []dtclient.CommunicationHost) (bool, error) {
	log.Info("reconciling")

	apiHost, err := dtclient.ParseEndpoint(instance.Spec.APIURL)
	if err != nil {
		return false, err
	}

	upd, err := reconciler.reconcileIstioConfigurations(instance, []dtclient.CommunicationHost{apiHost}, "api-url")
	if err != nil {
		return false, errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	} else if upd {
		return true, nil
	}

	// Fetch endpoints via Dynatrace client
	upd, err = reconciler.reconcileIstioConfigurations(instance, communicationHosts, "communication-endpoint")
	if err != nil {
		return false, errors.WithMessage(err, "error reconciling config for Dynatrace communication endpoints:")
	} else if upd {
		return true, nil
	}

	return false, nil
}

func (reconciler *Reconciler) reconcileIstioConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {
	add, err := reconciler.reconcileCreateConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}
	rem, err := reconciler.reconcileRemoveConfigurations(instance, comHosts, role)
	if err != nil {
		return false, err
	}

	return add || rem, nil
}

func (reconciler *Reconciler) reconcileRemoveConfigurations(instance *dynatracev1beta1.DynaKube,
	comHosts []dtclient.CommunicationHost, role string) (bool, error) {
	labelSelector := labels.SelectorFromSet(buildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seenComHosts := map[string]bool{}
	for _, comHost := range comHosts {
		seenComHosts[BuildNameForEndpoint(instance.GetName(), comHost.Protocol, comHost.Host, comHost.Port)] = true
	}

	istioConfig := &configuration{
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

func (reconciler *Reconciler) reconcileCreateConfigurations(instance *dynatracev1beta1.DynaKube,
	communicationHosts []dtclient.CommunicationHost, role string) (bool, error) {

	configurationUpdated := false
	for _, commHost := range communicationHosts {
		name := BuildNameForEndpoint(instance.GetName(), commHost.Protocol, commHost.Host, commHost.Port)
		commHost := commHost
		istioConfig := &configuration{
			instance:   instance,
			reconciler: reconciler,
			name:       name,
			commHost:   &commHost,
			role:       role,
		}

		createdServiceEntry, err := handleIstioConfigurationForServiceEntry(istioConfig)
		if err != nil {
			return false, err
		}
		createdVirtualService, err := handleIstioConfigurationForVirtualService(istioConfig)
		if err != nil {
			return false, err
		}

		configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService
	}

	return configurationUpdated, nil
}
