package istio

import (
	"net"
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
	commHosts  []dtclient.CommunicationHost
	role       string
	listOps    *metav1.ListOptions
}

// NewReconciler - creates new instance of istio controller
func NewReconciler(config *rest.Config, scheme *runtime.Scheme, istio istioclientset.Interface) *Reconciler {
	return &Reconciler{
		istioClient: istio,
		config:      config,
		scheme:      scheme,
		namespace:   os.Getenv(kubeobjects.EnvPodNamespace),
	}
}

// Reconcile runs the istio reconcile workflow:
// creating/deleting VS & SE for external communications
func (r *Reconciler) Reconcile(instance *dynatracev1beta1.DynaKube, communicationHosts []dtclient.CommunicationHost) (bool, error) {
	log.Info("reconciling")

	isInstalled, err := CheckIstioInstalled(r.istioClient.Discovery())
	if err != nil {
		return false, err
	} else if !isInstalled {
		log.Info("istio not installed, skipping reconciliation")
		return false, nil
	}

	apiHost, err := dtclient.ParseEndpoint(instance.Spec.APIURL)
	if err != nil {
		return false, err
	}

	upd, err := r.reconcileIstioConfigurations(instance, []dtclient.CommunicationHost{apiHost}, "api-url")
	if err != nil {
		return false, errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	} else if upd {
		return true, nil
	}

	// Fetch endpoints via Dynatrace client
	upd, err = r.reconcileIstioConfigurations(instance, communicationHosts, "communication-endpoint")
	if err != nil {
		return false, errors.WithMessage(err, "error reconciling config for Dynatrace communication endpoints:")
	} else if upd {
		return true, nil
	}

	return false, nil
}

func (r *Reconciler) reconcileIstioConfigurations(instance *dynatracev1beta1.DynaKube, comHosts []dtclient.CommunicationHost, role string) (bool, error) {
	var ipHosts []dtclient.CommunicationHost
	var hostHosts []dtclient.CommunicationHost

	// Split ip hosts and host hosts
	for _, commHost := range comHosts {
		if net.ParseIP(commHost.Host) != nil {
			ipHosts = append(ipHosts, commHost)
		} else {
			hostHosts = append(hostHosts, commHost)
		}
	}

	add, err := r.reconcileCreateConfigurations(instance, ipHosts, hostHosts, role)
	if err != nil {
		return false, err
	}
	rem, err := r.reconcileRemoveConfigurations(instance, ipHosts, hostHosts, role)
	if err != nil {
		return false, err
	}

	return add || rem, nil
}

func (r *Reconciler) reconcileRemoveConfigurations(instance *dynatracev1beta1.DynaKube,
	ipHosts, hostHosts []dtclient.CommunicationHost, role string) (bool, error) {
	labelSelector := labels.SelectorFromSet(buildIstioLabels(instance.GetName(), role)).String()
	listOps := &metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	seenComHosts := map[string]bool{}
	seenComHosts[BuildNameForEndpoint(instance.GetName(), ipHosts)] = true
	seenComHosts[BuildNameForEndpoint(instance.GetName(), hostHosts)] = true

	istioConfig := &configuration{
		reconciler: r,
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

func (r *Reconciler) reconcileCreateConfigurations(instance *dynatracev1beta1.DynaKube,
	ipHosts, hostHosts []dtclient.CommunicationHost, role string) (bool, error) {
	configurationUpdated := false
	var createdServiceEntryIP, createdServiceEntryFQNS bool

	if len(ipHosts) != 0 {
		name := BuildNameForEndpoint(instance.GetName(), ipHosts)
		istioConfig := &configuration{
			instance:   instance,
			reconciler: r,
			name:       name,
			commHosts:  ipHosts,
			role:       role,
		}
		serviceEntry := buildServiceEntryIPs(buildObjectMeta(istioConfig.name, istioConfig.instance.GetNamespace()), ipHosts)

		var err error
		createdServiceEntryIP, err = handleIstioConfigurationForServiceEntry(istioConfig, serviceEntry)
		if err != nil {
			return false, err
		}
	}

	if len(hostHosts) != 0 {
		name := BuildNameForEndpoint(instance.GetName(), hostHosts)
		istioConfig := &configuration{
			instance:   instance,
			reconciler: r,
			name:       name,
			commHosts:  hostHosts,
			role:       role,
		}
		serviceEntry := buildServiceEntryFQDNs(buildObjectMeta(istioConfig.name, istioConfig.instance.GetNamespace()), hostHosts)

		var err error
		createdServiceEntryFQNS, err = handleIstioConfigurationForServiceEntry(istioConfig, serviceEntry)
		if err != nil {
			return false, err
		}
	}

	istioConfig := &configuration{
		instance:   instance,
		reconciler: r,
		name:       BuildNameForEndpoint(instance.GetName(), hostHosts),
		commHosts:  hostHosts,
		role:       role,
	}

	createdVirtualService, err := handleIstioConfigurationForVirtualService(istioConfig)
	if err != nil {
		return false, err
	}

	createdServiceEntry := createdServiceEntryIP || createdServiceEntryFQNS
	configurationUpdated = configurationUpdated || createdServiceEntry || createdVirtualService

	return configurationUpdated, nil
}
