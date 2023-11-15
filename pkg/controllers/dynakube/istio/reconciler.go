package istio

import (
	"context"
	"net"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reconciler struct {
	client *Client
}

func NewReconciler(istio *Client) *Reconciler {
	return &Reconciler{
		client: istio,
	}
}

func (r *Reconciler) ReconcileAPIUrl(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("reconciling istio components for the Dynatrace API url")
	if dynakube == nil {
		return errors.New("can't reconcile api url of nil dynakube")
	}
	apiHost, err := dtclient.ParseEndpoint(dynakube.Spec.APIURL)
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

func (r *Reconciler) ReconcileCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("reconciling istio components for oneagent communication hosts")
	if dynakube == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}

	oneAgentCommunicationHosts := connectioninfo.GetOneAgentCommunicationHosts(dynakube)
	err := r.reconcileCommunicationHostsForComponent(ctx, oneAgentCommunicationHosts, OneAgentComponent)
	if err != nil {
		return err
	}

	activeGateEndpoints := connectioninfo.GetActiveGateEndpointsAsCommunicationHosts(dynakube)
	err = r.reconcileCommunicationHostsForComponent(ctx, activeGateEndpoints, ActiveGateComponent)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reconcileCommunicationHostsForComponent(ctx context.Context, comHosts []dtclient.CommunicationHost, componentName string) error {
	err := r.reconcileCommunicationHosts(ctx, comHosts, componentName)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}
	log.Info("reconciled istio objects for communication hosts", "component", componentName)
	return nil
}

func (r *Reconciler) reconcileCommunicationHosts(ctx context.Context, comHosts []dtclient.CommunicationHost, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	err := r.reconcileIPServiceEntry(ctx, ipHosts, component)
	if err != nil {
		return err
	}

	err = r.reconcileFQDNServiceEntry(ctx, fqdnHosts, component)
	if err != nil {
		return err
	}
	return nil
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

func (r *Reconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []dtclient.CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)
	if len(ipHosts) != 0 {
		meta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			labels.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryIPs(meta, ipHosts)
		err := r.client.CreateOrUpdateServiceEntry(ctx, serviceEntry)
		if err != nil {
			return err
		}
	} else {
		err := r.client.DeleteServiceEntry(ctx, entryName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []dtclient.CommunicationHost, component string) error {
	owner := r.client.Owner
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)
	if len(fqdnHosts) != 0 {
		meta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			labels.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryFQDNs(meta, fqdnHosts)
		err := r.client.CreateOrUpdateServiceEntry(ctx, serviceEntry)
		if err != nil {
			return err
		}

		virtualService := buildVirtualService(meta, fqdnHosts)
		err = r.client.CreateOrUpdateVirtualService(ctx, virtualService)
		if err != nil {
			return err
		}
	} else {
		err := r.client.DeleteServiceEntry(ctx, entryName)
		if err != nil {
			return err
		}

		err = r.client.DeleteVirtualService(ctx, entryName)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildObjectMeta(name, namespace string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
}
