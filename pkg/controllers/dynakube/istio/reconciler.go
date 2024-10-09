package istio

import (
	"context"
	goerrors "errors"
	"net"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/activegate"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reconciler interface {
	ReconcileAPIUrl(ctx context.Context, dk *dynakube.DynaKube) error
	ReconcileCodeModuleCommunicationHosts(ctx context.Context, dk *dynakube.DynaKube) error
	ReconcileActiveGateCommunicationHosts(ctx context.Context, dk *dynakube.DynaKube) error
}

type reconciler struct {
	client       *Client
	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(istio *Client) Reconciler

func NewReconciler(istio *Client) Reconciler {
	return &reconciler{
		client:       istio,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) ReconcileAPIUrl(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for the Dynatrace API url")

	if dk == nil {
		return errors.New("can't reconcile api url of nil dynakube")
	}

	apiHost, err := dtclient.ParseEndpoint(dk.Spec.APIURL)
	if err != nil {
		return err
	}

	err = r.reconcileCommunicationHosts(ctx, []dtclient.CommunicationHost{apiHost}, OperatorComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	}

	log.Info("reconciled istio objects for API url")

	return nil
}

func (r *reconciler) ReconcileCodeModuleCommunicationHosts(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for oneagent-code-modules communication hosts")

	if dk == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}

	if !dk.NeedAppInjection() {
		if isIstioConfigured(dk, CodeModuleComponent) {
			log.Info("appinjection disabled, cleaning up")

			return r.CleanupIstio(ctx, dk, CodeModuleComponent, OneAgentComponent)
		}

		return nil
	}

	oneAgentCommunicationHosts := oaconnectioninfo.GetCommunicationHosts(dk)

	err := r.reconcileCommunicationHostsForComponent(ctx, oneAgentCommunicationHosts, OneAgentComponent)
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), CodeModuleComponent, err)

		return err
	}

	if len(oneAgentCommunicationHosts) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(CodeModuleComponent))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), CodeModuleComponent)

	return nil
}

func (r *reconciler) ReconcileActiveGateCommunicationHosts(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for activegate communication hosts")

	if dk == nil {
		return errors.New("can't reconcile activegate communication hosts of nil dynakube")
	}

	if !dk.ActiveGate().IsEnabled() {
		if isIstioConfigured(dk, ActiveGateComponent) {
			log.Info("activegate disabled, cleaning up")

			return r.CleanupIstio(ctx, dk, ActiveGateComponent, strings.ToLower(ActiveGateComponent))
		}

		return nil
	}

	if !conditions.IsOutdated(r.timeProvider, dk, getConditionTypeName(ActiveGateComponent)) {
		log.Info("condition still within time threshold...skipping further reconciliation")

		return nil
	}

	activeGateEndpoints := activegate.GetEndpointsAsCommunicationHosts(dk)

	err := r.reconcileCommunicationHostsForComponent(ctx, activeGateEndpoints, strings.ToLower(ActiveGateComponent))
	if err != nil {
		setServiceEntryFailedConditionForComponent(dk.Conditions(), ActiveGateComponent, err)

		return err
	}

	if len(activeGateEndpoints) == 0 {
		meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(ActiveGateComponent))

		return nil
	}

	setServiceEntryUpdatedConditionForComponent(dk.Conditions(), ActiveGateComponent)

	return nil
}

func (r *reconciler) CleanupIstio(ctx context.Context, dk *dynakube.DynaKube, conditionComponent string, component string) error {
	meta.RemoveStatusCondition(dk.Conditions(), getConditionTypeName(conditionComponent))

	err1 := r.cleanupIPServiceEntry(ctx, component)
	err2 := r.cleanupFQDNServiceEntry(ctx, component)

	// try to clean up all entries even if one fails
	return goerrors.Join(err1, err2)
}

func isIstioConfigured(dk *dynakube.DynaKube, conditionComponent string) bool {
	istioCondition := meta.FindStatusCondition(*dk.Conditions(), getConditionTypeName(conditionComponent))

	return istioCondition != nil
}

func (r *reconciler) reconcileCommunicationHostsForComponent(ctx context.Context, comHosts []dtclient.CommunicationHost, componentName string) error {
	err := r.reconcileCommunicationHosts(ctx, comHosts, componentName)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	log.Info("reconciled istio objects for communication hosts", "component", componentName)

	return nil
}

func (r *reconciler) reconcileCommunicationHosts(ctx context.Context, comHosts []dtclient.CommunicationHost, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	errIPServiceEntry := r.reconcileIPServiceEntry(ctx, ipHosts, component)
	errFQDNServiceEntry := r.reconcileFQDNServiceEntry(ctx, fqdnHosts, component)

	return goerrors.Join(errIPServiceEntry, errFQDNServiceEntry)
}

func splitCommunicationHost(comHosts []dtclient.CommunicationHost) (ipHosts, fqdnHosts []dtclient.CommunicationHost) {
	for _, commHost := range comHosts {
		if net.ParseIP(commHost.Host) != nil {
			ipHosts = append(ipHosts, commHost)
		} else {
			fqdnHosts = append(fqdnHosts, commHost)
		}
	}

	return
}

func (r *reconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []dtclient.CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)

	if len(ipHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			labels.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryIPs(objectMeta, ipHosts)

		err := r.client.CreateOrUpdateServiceEntry(ctx, serviceEntry)
		if err != nil {
			return err
		}
	} else {
		err := r.cleanupIPServiceEntry(ctx, component)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) cleanupIPServiceEntry(ctx context.Context, component string) error {
	entryName := BuildNameForIPServiceEntry(r.client.Owner.GetName(), component)

	return r.client.DeleteServiceEntry(ctx, entryName)
}

func (r *reconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []dtclient.CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)

	if len(fqdnHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			labels.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryFQDNs(objectMeta, fqdnHosts)

		err := r.client.CreateOrUpdateServiceEntry(ctx, serviceEntry)
		if err != nil {
			return err
		}

		virtualService := buildVirtualService(objectMeta, fqdnHosts)

		err = r.client.CreateOrUpdateVirtualService(ctx, virtualService)
		if err != nil {
			return err
		}
	} else {
		err := r.cleanupFQDNServiceEntry(ctx, component)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) cleanupFQDNServiceEntry(ctx context.Context, component string) error {
	entryName := BuildNameForFQDNServiceEntry(r.client.Owner.GetName(), component)

	errServiceEntry := r.client.DeleteServiceEntry(ctx, entryName)
	errVirtualService := r.client.DeleteVirtualService(ctx, entryName)

	return goerrors.Join(errServiceEntry, errVirtualService)
}

func buildObjectMeta(name, namespace string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
}
