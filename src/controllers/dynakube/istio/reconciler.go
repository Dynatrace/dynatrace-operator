package istio

import (
	"context"
	"net"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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

	err = r.reconcileCommunicationHosts(ctx, dynakube, []dtclient.CommunicationHost{apiHost}, OperatorComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	}

	return nil
}

func (r *Reconciler) ReconcileOneAgentCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("reconciling istio components for oneagent communication hosts")
	if dynakube == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}
	communicationHosts := connectioninfo.GetOneAgentCommunicationHosts(dynakube)

	err := r.reconcileCommunicationHosts(ctx, dynakube, communicationHosts, OneAgentComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication hosts")
	}

	return nil
}

func (r *Reconciler) reconcileCommunicationHosts(ctx context.Context, owner metav1.Object, comHosts []dtclient.CommunicationHost, component string) error {
	ipHosts, fqdnHosts := splitCommunicationHost(comHosts)

	err := r.reconcileIPServiceEntry(ctx, owner, ipHosts, component)
	if err != nil {
		return err
	}

	err = r.reconcileFQDNServiceEntry(ctx, owner, fqdnHosts, component)
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

func (r *Reconciler) reconcileIPServiceEntry(ctx context.Context, owner metav1.Object, ipHosts []dtclient.CommunicationHost, component string) error {
	if owner == nil {
		return errors.New("unable to create service entry for IPs if owner is nil")
	}
	entryName := BuildNameForIPServiceEntry(owner.GetName(), component)
	if len(ipHosts) != 0 {
		meta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			kubeobjects.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryIPs(meta, ipHosts)
		err := r.client.CreateOrUpdateServiceEntry(ctx, owner, serviceEntry)
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

func (r *Reconciler) reconcileFQDNServiceEntry(ctx context.Context, owner metav1.Object, fqdnHosts []dtclient.CommunicationHost, component string) error {
	if owner == nil {
		return errors.New("unable to create service entry and virtual service for Hosts if owner is nil")
	}
	entryName := BuildNameForFQDNServiceEntry(owner.GetName(), component)
	if len(fqdnHosts) != 0 {
		meta := buildObjectMeta(
			entryName,
			owner.GetNamespace(),
			kubeobjects.NewCoreLabels(owner.GetName(), component).BuildLabels(),
		)

		serviceEntry := buildServiceEntryFQDNs(meta, fqdnHosts)
		err := r.client.CreateOrUpdateServiceEntry(ctx, owner, serviceEntry)
		if err != nil {
			return err
		}

		virtualService := buildVirtualService(meta, fqdnHosts)
		err = r.client.CreateOrUpdateVirtualService(ctx, owner, virtualService)
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
