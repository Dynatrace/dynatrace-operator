package istio

import (
	"context"
	"net"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
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
	apiHost, err := dtclient.ParseEndpoint(dynakube.Spec.APIURL)
	if err != nil {
		return err
	}

	err = r.reconcileCommunicationHosts(ctx, dynakube, []dtclient.CommunicationHost{apiHost}, operatorComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	}

	return nil
}

func (r *Reconciler) ReconcileOneAgentCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("reconciling istio components for oneagent communication hosts")
	communicationHosts := connectioninfo.GetCommunicationHosts(dynakube)

	err := r.reconcileCommunicationHosts(ctx, dynakube, communicationHosts, oneAgentComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	return nil
}

func (r *Reconciler) reconcileCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, comHosts []dtclient.CommunicationHost, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	err := r.reconcileIPServiceEntry(ctx, dynakube, ipHosts, component)
	if err != nil {
		return err
	}

	err = r.reconcileFQDNServiceEntry(ctx, dynakube, fqdnHosts, component)
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

func (r *Reconciler) reconcileIPServiceEntry(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, ipHosts []dtclient.CommunicationHost, component string) error {
	if len(ipHosts) != 0 {
		meta := buildObjectMeta(
			BuildNameForIPServiceEntry(dynakube.GetName(), component),
			dynakube.GetNamespace(),
			kubeobjects.NewCoreLabels(dynakube.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryIPs(meta, ipHosts)
		err := r.client.ApplyServiceEntry(ctx, dynakube, serviceEntry)
		if err != nil {
			return err
		}
	} else {
		err := r.client.DeleteServiceEntry(ctx, component)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileFQDNServiceEntry(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, fqdnHosts []dtclient.CommunicationHost, component string) error {
	if len(fqdnHosts) != 0 {
		meta := buildObjectMeta(
			BuildNameForFQDNServiceEntry(dynakube.GetName(), component),
			dynakube.GetNamespace(),
			kubeobjects.NewCoreLabels(dynakube.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryFQDNs(meta, fqdnHosts)
		err := r.client.ApplyServiceEntry(ctx, dynakube, serviceEntry)
		if err != nil {
			return err
		}

		virtualService := buildVirtualService(meta, fqdnHosts)
		err = r.client.ApplyVirtualService(ctx, dynakube, virtualService)
		if err != nil {
			return err
		}
	} else {
		err := r.client.DeleteServiceEntry(ctx, component)
		if err != nil {
			return err
		}

		err = r.client.DeleteVirtualService(ctx, dynakube.GetName())
		if err != nil {
			return err
		}
	}
	return nil
}
