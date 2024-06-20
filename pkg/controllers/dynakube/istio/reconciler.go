package istio

import (
	"context"
	"net"
	"strings"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/activegate"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reconciler interface {
	ReconcileAPIUrl(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error
	ReconcileCodeModuleCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error
	ReconcileActiveGateCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error
	ReconcileCSIDriver(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error
}

type reconciler struct {
	client *Client
}

type ReconcilerBuilder func(istio *Client) Reconciler

func NewReconciler(istio *Client) Reconciler {
	return &reconciler{
		client: istio,
	}
}

func (r *reconciler) ReconcileCSIDriver(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error {
	log.Info("reconciling istio components for the CSI driver")

	if dynakube == nil {
		return errors.New("can't reconcile csi driver of nil dynakube")
	}

	codeModulesURL := dynakube.Status.CodeModules.ImageID

	if codeModulesURL == "" {
		return errors.New("imageID of codeModules status is empty")
	}

	if !hasCorrectFormat(codeModulesURL) {
		codeModulesURL = "https://" + codeModulesURL
	}

	codeModulesHost, err := dtclient.ParseEndpoint(codeModulesURL)
	if err != nil {
		return err
	}

	err = r.reconcileCommunicationHosts(ctx, []dtclient.CommunicationHost{codeModulesHost}, CSIDiverComponent)
	if err != nil {
		return errors.WithMessage(err, "error reconciling config for codeModulesImage")
	}

	log.Info("reconciled istio objects for CSI driver")

	return nil
}

// if the URL does not have the correct format parsing does not work‚
func hasCorrectFormat(url string) bool {
	return strings.HasPrefix(url, "https://")
}

func (r *reconciler) ReconcileAPIUrl(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error {
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

func (r *reconciler) ReconcileCodeModuleCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error {
	log.Info("reconciling istio components for oneagent-code-modules communication hosts")

	if dynakube == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}

	oneAgentCommunicationHosts := oaconnectioninfo.GetCommunicationHosts(dynakube)

	err := r.reconcileCommunicationHostsForComponent(ctx, oneAgentCommunicationHosts, OneAgentComponent)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ReconcileActiveGateCommunicationHosts(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error {
	log.Info("reconciling istio components for activegate communication hosts")

	if dynakube == nil {
		return errors.New("can't reconcile oneagent communication hosts of nil dynakube")
	}

	if !dynakube.NeedsActiveGate() {
		return nil
	}

	activeGateEndpoints := activegate.GetEndpointsAsCommunicationHosts(dynakube)

	err := r.reconcileCommunicationHostsForComponent(ctx, activeGateEndpoints, ActiveGateComponent)
	if err != nil {
		return err
	}

	return nil
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

func (r *reconciler) reconcileIPServiceEntry(ctx context.Context, ipHosts []dtclient.CommunicationHost, component string) error {
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

func (r *reconciler) reconcileFQDNServiceEntry(ctx context.Context, fqdnHosts []dtclient.CommunicationHost, component string) error {
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
