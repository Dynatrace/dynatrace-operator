package istio

import (
	"context"
	goerrors "errors"
	"fmt"
	"net"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	log = logd.Get().WithName("dynakube-istio")
)

const (
	OperatorComponent = "operator"

	CodeModuleComponent     = "oneagent"
	codeModuleConditionName = "OneAgent"

	ActiveGateComponent     = "activegate"
	activeGateConditionName = "ActiveGate"

	IstioGVRName    = "networking.istio.io"
	IstioGVRVersion = "v1beta1"
)

var (
	IstioGVR = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
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

	apiCommunicationHost, err := NewCommunicationHost(dk.Spec.APIURL)
	if err != nil {
		return err
	}

	err = r.reconcileCommunicationHosts(ctx, []CommunicationHost{apiCommunicationHost}, OperatorComponent)
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

	migrateDeprecatedCondition(dk.Conditions())

	if !dk.OneAgent().IsAppInjectionNeeded() {
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

	err = r.reconcileCommunicationHostsForComponent(ctx, oaCommunicationHosts, CodeModuleComponent)
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

func (r *reconciler) ReconcileActiveGateCommunicationHosts(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling istio components for activegate communication hosts")

	if dk == nil {
		return errors.New("can't reconcile activegate communication hosts of nil dynakube")
	}

	if !dk.ActiveGate().IsEnabled() {
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

	err = r.reconcileCommunicationHostsForComponent(ctx, agCommunicationHosts, ActiveGateComponent)
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

func (r *reconciler) cleanupIstio(ctx context.Context, dk *dynakube.DynaKube, component string) error {
	err1 := r.cleanupIPServiceEntry(ctx, component)
	err2 := r.cleanupFQDNServiceEntry(ctx, component)

	// try to clean up all entries even if one fails
	return goerrors.Join(err1, err2)
}

func isIstioConfigured(dk *dynakube.DynaKube, conditionComponent string) bool {
	istioCondition := meta.FindStatusCondition(*dk.Conditions(), getConditionTypeName(conditionComponent))

	return istioCondition != nil
}

func (r *reconciler) reconcileCommunicationHostsForComponent(ctx context.Context, comHosts []CommunicationHost, componentName string) error {
	err := r.reconcileCommunicationHosts(ctx, comHosts, componentName)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	log.Info("reconciled istio objects for communication hosts", "component", componentName)

	return nil
}

func (r *reconciler) reconcileCommunicationHosts(ctx context.Context, comHosts []CommunicationHost, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	errIPServiceEntry := r.reconcileIPServiceEntry(ctx, ipHosts, component)
	errFQDNServiceEntry := r.reconcileFQDNServiceEntry(ctx, fqdnHosts, component)

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

func (r *reconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)

	if len(ipHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			k8slabel.NewCoreLabels(owner.GetName(), component).BuildLabels(),
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

func (r *reconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)

	if len(fqdnHosts) != 0 {
		objectMeta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			k8slabel.NewCoreLabels(owner.GetName(), component).BuildLabels(),
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
