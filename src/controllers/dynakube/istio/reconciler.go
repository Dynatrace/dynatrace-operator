package istio

import (
	"context"
	"net"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type Reconciler struct {
	client    *Client
}

func NewReconciler(istio *Client) *Reconciler {
	return &Reconciler{
		client:    istio,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, instance *dynatracev1beta1.DynaKube, communicationHosts []dtclient.CommunicationHost) error {
	log.Info("reconciling")

	isInstalled, err := CheckIstioInstalled(r.client.Discovery())
	if err != nil {
		return err
	} else if !isInstalled {
		log.Info("istio not installed, skipping reconciliation")
		return nil
	}

	apiHost, err := dtclient.ParseEndpoint(instance.Spec.APIURL)
	if err != nil {
		return err
	}

	upd, err := r.reconcileIstioConfigurations(ctx, instance, []dtclient.CommunicationHost{apiHost}, "api-url")
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace API URL")
	} else if upd {
		return nil
	}

	upd, err = r.reconcileIstioConfigurations(ctx, instance, communicationHosts, "communication-endpoint")
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for Dynatrace communication endpoints:")
	} else if upd {
		return nil
	}

	return nil
}

func (r *Reconciler) reconcileIstioConfigurations(ctx context.Context, instance *dynatracev1beta1.DynaKube, comHosts []dtclient.CommunicationHost, role string) (bool, error) {
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

	err := r.reconcileCreateConfigurations(ctx, instance, ipHosts, hostHosts, role)
	if err != nil {
		return false, err
	}

	err = r.reconcileRemoveConfigurations(ctx, instance, ipHosts, hostHosts, role)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (r *Reconciler) reconcileCreateConfigurations(ctx context.Context, instance *dynatracev1beta1.DynaKube,
	ipHosts, hostHosts []dtclient.CommunicationHost, role string) error {

	if len(ipHosts) != 0 {
		name := BuildNameForIPServiceEntry(instance.GetName())
		serviceEntry := buildServiceEntryIPs(buildObjectMeta(name, instance.GetNamespace()), ipHosts)

		err := r.client.ApplyServiceEntry(ctx, instance, serviceEntry)
		if err != nil {
			return err
		}
	}

	if len(hostHosts) != 0 {
		name := BuildNameForFQDNServiceEntry(instance.GetName())
		serviceEntry := buildServiceEntryFQDNs(buildObjectMeta(name, instance.GetNamespace()), hostHosts)

		err := r.client.ApplyServiceEntry(ctx, instance, serviceEntry)
		if err != nil {
			return err
		}
	}

	virtualService := buildVirtualService(buildObjectMeta(instance.GetName(), instance.GetNamespace()), hostHosts)
	err := r.client.ApplyVirtualService(ctx, instance, virtualService)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileRemoveConfigurations(ctx context.Context, instance *dynatracev1beta1.DynaKube,
	ipHosts, hostHosts []dtclient.CommunicationHost, role string) error {
	if len(ipHosts) == 0 {
		name := BuildNameForIPServiceEntry(instance.GetName())
		err := r.client.DeleteServiceEntry(ctx, name)
		if err != nil {
			return err
		}
	}

	if len(hostHosts) == 0 {
		name := BuildNameForFQDNServiceEntry(instance.GetName())
		err := r.client.DeleteServiceEntry(ctx, name)
		if err != nil {
			return err
		}
	}
	return nil
}
