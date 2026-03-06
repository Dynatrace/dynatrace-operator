package istio

import (
	"context"
	goerrors "errors"
	"fmt"
	"net"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sserviceentry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8svirtualservice"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("dynakube-istio")
)

const (
	OperatorComponent     = "operator"
	operatorConditionName = "Operator"

	CodeModuleComponent     = "oneagent"
	codeModuleConditionName = "OneAgent"

	ActiveGateComponent     = "activegate"
	activeGateConditionName = "ActiveGate"

	istioGVRName    = "networking.istio.io"
	istioGVRVersion = "v1beta1"
)

var (
	istioGVR = fmt.Sprintf("%s/%s", istioGVRName, istioGVRVersion)
)

// communicationHostReconciler holds the shared logic for managing istio ServiceEntry
// and VirtualService objects for a given set of communication hosts.
type communicationHostReconciler struct {
	serviceEntry   k8sserviceentry.QueryObject
	virtualService k8svirtualservice.QueryObject
}

func newCommunicationHostReconciler(kubeClient client.Client, apiReader client.Reader) *communicationHostReconciler {
	return &communicationHostReconciler{
		serviceEntry:   k8sserviceentry.Query(kubeClient, apiReader, log),
		virtualService: k8svirtualservice.Query(kubeClient, apiReader, log),
	}
}

// APIUrlReconciler reconciles istio objects for the Dynatrace API URL.
type APIUrlReconciler struct {
	*communicationHostReconciler
}

func NewAPIUrlReconciler(kubeClient client.Client, apiReader client.Reader) *APIUrlReconciler {
	return &APIUrlReconciler{communicationHostReconciler: newCommunicationHostReconciler(kubeClient, apiReader)}
}

func (r *APIUrlReconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for the Dynatrace API url")

	if dk == nil {
		return errors.New("can't reconcile api url of nil dynakube")
	}

	if !dk.Spec.EnableIstio {
		if isIstioConfigured(dk, OperatorComponent) {
			err := r.cleanupIstio(ctx, dk, OperatorComponent)
			if err != nil {
				// We don't error out here to avoid stuck reconciliations in case cleanup fails
				log.Error(err, "failed to cleanup the istio configuration", "component", OperatorComponent)
			}

			meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(operatorConditionName))
		}

		return nil
	}

	apiCommunicationHost, err := NewCommunicationHost(dk.Spec.APIURL)
	if err != nil {
		return err
	}

	err = r.reconcileCommunicationHosts(ctx, []CommunicationHost{apiCommunicationHost}, dk, OperatorComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	}

	log.Info("reconciled istio objects for API url")

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), operatorConditionName)

	return nil
}

// CodeModuleReconciler reconciles istio objects for OneAgent code module communication hosts.
type CodeModuleReconciler struct {
	*communicationHostReconciler
}

func NewCodeModuleReconciler(kubeClient client.Client, apiReader client.Reader) *CodeModuleReconciler {
	return &CodeModuleReconciler{communicationHostReconciler: newCommunicationHostReconciler(kubeClient, apiReader)}
}

func (r *CodeModuleReconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for oneagent-code-modules communication hosts")

	if dk == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}

	migrateDeprecatedCondition(dk.Conditions())

	if !dk.Spec.EnableIstio || !dk.OneAgent().IsAppInjectionNeeded() {
		if isIstioConfigured(dk, codeModuleConditionName) {
			log.Info("appinjection disabled, cleaning up")

			err := r.cleanupIstio(ctx, dk, CodeModuleComponent)
			if err != nil {
				// We don't error out here to avoid stuck reconciliations in case cleanup fails
				log.Error(err, "failed to cleanup the istio configuration", "component", codeModuleConditionName)
			}

			meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(codeModuleConditionName))
		}

		return nil
	}

	oaCommunicationHosts, err := NewCommunicationHosts(dk.Status.OneAgent.ConnectionInfo.Endpoints)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), codeModuleConditionName, err)

		return err
	}

	err = r.reconcileCommunicationHostsForComponent(ctx, oaCommunicationHosts, dk, CodeModuleComponent)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), codeModuleConditionName, err)

		return err
	}

	if len(oaCommunicationHosts) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(codeModuleConditionName))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), codeModuleConditionName)

	return nil
}

// ActiveGateReconciler reconciles istio objects for ActiveGate communication hosts.
type ActiveGateReconciler struct {
	*communicationHostReconciler
}

func NewActiveGateReconciler(kubeClient client.Client, apiReader client.Reader) *ActiveGateReconciler {
	return &ActiveGateReconciler{communicationHostReconciler: newCommunicationHostReconciler(kubeClient, apiReader)}
}

func (r *ActiveGateReconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for activegate communication hosts")

	if dk == nil {
		return errors.New("can't reconcile activegate communication hosts of nil dynakube")
	}

	if !dk.Spec.EnableIstio || !dk.ActiveGate().IsEnabled() {
		if isIstioConfigured(dk, activeGateConditionName) {
			log.Info("activegate disabled, cleaning up")

			err := r.cleanupIstio(ctx, dk, ActiveGateComponent)
			if err != nil {
				// We don't error out here to avoid stuck reconciliations in case cleanup fails
				log.Error(err, "failed to cleanup the istio configuration", "component", activeGateConditionName)
			}

			meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(activeGateConditionName))
		}

		return nil
	}

	agCommunicationHosts, err := NewCommunicationHosts(dk.Status.ActiveGate.ConnectionInfo.Endpoints)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), activeGateConditionName, err)

		return err
	}

	err = r.reconcileCommunicationHostsForComponent(ctx, agCommunicationHosts, dk, ActiveGateComponent)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), activeGateConditionName, err)

		return err
	}

	if len(agCommunicationHosts) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(activeGateConditionName))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), activeGateConditionName)

	return nil
}

func (r *communicationHostReconciler) cleanupIstio(ctx context.Context, owner client.Object, component string) error {
	err1 := r.cleanupIPServiceEntry(ctx, owner, component)
	err2 := r.cleanupFQDNServiceEntry(ctx, owner, component)

	// try to clean up all entries even if one fails
	return goerrors.Join(err1, err2)
}

func isIstioConfigured(dk *dynakube.DynaKube, conditionComponent string) bool {
	istioCondition := meta.FindStatusCondition(*dk.Conditions(), getConditionTypeName(conditionComponent))

	return istioCondition != nil
}

func (r *communicationHostReconciler) reconcileCommunicationHostsForComponent(ctx context.Context, comHosts []CommunicationHost, owner client.Object, componentName string) error {
	err := r.reconcileCommunicationHosts(ctx, comHosts, owner, componentName)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	log.Info("reconciled istio objects for communication hosts", "component", componentName)

	return nil
}

func (r *communicationHostReconciler) reconcileCommunicationHosts(ctx context.Context, comHosts []CommunicationHost, owner client.Object, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	errIPServiceEntry := r.reconcileIPServiceEntry(ctx, ipHosts, owner, component)
	errFQDNServiceEntry := r.reconcileFQDNServiceEntry(ctx, fqdnHosts, owner, component)

	return goerrors.Join(errIPServiceEntry, errFQDNServiceEntry)
}

func splitCommunicationHost(comHosts []CommunicationHost) (ipHosts, fqdnHosts []CommunicationHost) {
	for _, commHost := range comHosts {
		if net.ParseIP(commHost.Host) != nil {
			ipHosts = append(ipHosts, commHost)
		} else {
			fqdnHosts = append(fqdnHosts, commHost)
		}
	}

	return
}

func (r *communicationHostReconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []CommunicationHost, owner client.Object, component string) error {
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)

	if len(ipHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			k8slabel.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryIPs(objectMeta, ipHosts)

		_, err := r.serviceEntry.WithOwner(owner).CreateOrUpdate(ctx, serviceEntry)
		if err != nil {
			return err
		}
	} else {
		err := r.cleanupIPServiceEntry(ctx, owner, component)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *communicationHostReconciler) cleanupIPServiceEntry(ctx context.Context, owner client.Object, component string) error {
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)

	return r.serviceEntry.DeleteForNamespace(ctx, entryName, owner.GetNamespace())
}

func (r *communicationHostReconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []CommunicationHost, owner client.Object, component string) error {
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)

	if len(fqdnHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			k8slabel.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryFQDNs(objectMeta, fqdnHosts)

		_, err := r.serviceEntry.WithOwner(owner).CreateOrUpdate(ctx, serviceEntry)
		if err != nil {
			return err
		}

		virtualService := buildVirtualService(objectMeta, fqdnHosts)

		_, err = r.virtualService.WithOwner(owner).CreateOrUpdate(ctx, virtualService)
		if err != nil {
			return err
		}
	} else {
		err := r.cleanupFQDNServiceEntry(ctx, owner, component)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *communicationHostReconciler) cleanupFQDNServiceEntry(ctx context.Context, owner client.Object, component string) error {
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)

	errServiceEntry := r.serviceEntry.DeleteForNamespace(ctx, entryName, owner.GetNamespace())
	errVirtualService := r.virtualService.DeleteForNamespace(ctx, entryName, owner.GetNamespace())

	return goerrors.Join(errServiceEntry, errVirtualService)
}

func buildObjectMeta(name, namespace string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
}

// IsInstalled checks whether Istio is installed on the cluster by querying
// the discovery API for the Istio networking group version.
func IsInstalled(discoveryClient discovery.DiscoveryInterface) (bool, error) {
	_, err := discoveryClient.ServerResourcesForGroupVersion(istioGVR)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}
