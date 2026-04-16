package istio

import (
	"context"
	goerrors "errors"
	"net"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sserviceentry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8svirtualservice"
	"github.com/pkg/errors"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorComponent     = "operator"
	operatorConditionName = "Operator"

	CodeModuleComponent     = "oneagent"
	codeModuleConditionName = "OneAgent"

	ActiveGateComponent     = "activegate"
	activeGateConditionName = "ActiveGate"
)

// Reconciler holds the shared logic for managing istio ServiceEntry
// and VirtualService objects for a given set of communication hosts.
type Reconciler struct {
	serviceEntry   k8sserviceentry.QueryObject
	virtualService k8svirtualservice.QueryObject
}

func NewReconciler(kubeClient client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		serviceEntry:   k8sserviceentry.Query(kubeClient, apiReader),
		virtualService: k8svirtualservice.Query(kubeClient, apiReader),
	}
}
func (r *Reconciler) ReconcileAPIURL(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "dynakube-istio")

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

func (r *Reconciler) ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)

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
		setServiceEntryFailedConditionForComponent(dk.Conditions(), codeModuleConditionName)

		return err
	}

	err = r.reconcileCommunicationHostsForComponent(ctx, oaCommunicationHosts, dk, CodeModuleComponent)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), codeModuleConditionName)

		return err
	}

	if len(oaCommunicationHosts) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(codeModuleConditionName))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), codeModuleConditionName)

	return nil
}

func (r *Reconciler) ReconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)

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
		setServiceEntryFailedConditionForComponent(dk.Conditions(), activeGateConditionName)

		return err
	}

	err = r.reconcileCommunicationHostsForComponent(ctx, agCommunicationHosts, dk, ActiveGateComponent)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), activeGateConditionName)

		return err
	}

	if len(agCommunicationHosts) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(activeGateConditionName))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), activeGateConditionName)

	return nil
}

func (r *Reconciler) cleanupIstio(ctx context.Context, owner client.Object, component string) error {
	err1 := r.cleanupIPServiceEntry(ctx, owner, component)
	err2 := r.cleanupFQDNServiceEntry(ctx, owner, component)

	// try to clean up all entries even if one fails
	return goerrors.Join(err1, err2)
}

func isIstioConfigured(dk *dynakube.DynaKube, conditionComponent string) bool {
	istioCondition := meta.FindStatusCondition(*dk.Conditions(), getConditionTypeName(conditionComponent))

	return istioCondition != nil
}

func (r *Reconciler) reconcileCommunicationHostsForComponent(ctx context.Context, comHosts []CommunicationHost, owner client.Object, componentName string) error {
	log := logd.FromContext(ctx)

	err := r.reconcileCommunicationHosts(ctx, comHosts, owner, componentName)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	log.Info("reconciled istio objects for communication hosts", "component", componentName)

	return nil
}

func (r *Reconciler) reconcileCommunicationHosts(ctx context.Context, comHosts []CommunicationHost, owner client.Object, component string) error {
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

func (r *Reconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []CommunicationHost, owner client.Object, component string) error {
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

func (r *Reconciler) cleanupIPServiceEntry(ctx context.Context, owner client.Object, component string) error {
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)

	return r.serviceEntry.DeleteForNamespace(ctx, entryName, owner.GetNamespace())
}

func (r *Reconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []CommunicationHost, owner client.Object, component string) error {
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

func (r *Reconciler) cleanupFQDNServiceEntry(ctx context.Context, owner client.Object, component string) error {
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

func IsInstalled(ctx context.Context, apiReader client.Reader) bool {
	vs := &istiov1beta1.VirtualService{}
	if err := apiReader.Get(ctx, client.ObjectKey{Namespace: "default", Name: "default"}, vs); err != nil {
		return !meta.IsNoMatchError(err)
	}

	return true
}
