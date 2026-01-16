package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	capabilityInternal "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	agconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client                            client.Client
	dk                                *dynakube.DynaKube
	apiReader                         client.Reader
	authTokenReconciler               controllers.Reconciler
	istioReconciler                   istio.Reconciler
	connectionReconciler              controllers.Reconciler
	versionReconciler                 version.Reconciler
	pullSecretReconciler              controllers.Reconciler
	newStatefulsetReconcilerFunc      statefulset.NewReconcilerFunc
	newCapabilityReconcilerFunc       capabilityInternal.NewReconcilerFunc
	newCustomPropertiesReconcilerFunc func(customPropertiesOwnerName string, customPropertiesSource *value.Source) controllers.Reconciler
}

var _ controllers.Reconciler = (*Reconciler)(nil)

type ReconcilerBuilder func(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube,
	dtc dtclient.Client,
	istioClient *istio.Client,
	tokens token.Tokens,
) controllers.Reconciler

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube,
	dtc dtclient.Client,
	istioClient *istio.Client,
	tokens token.Tokens) controllers.Reconciler {
	var istioReconciler istio.Reconciler
	if istioClient != nil {
		istioReconciler = istio.NewReconciler(istioClient)
	}

	authTokenReconciler := authtoken.NewReconciler(clt, apiReader, dk, dtc)
	versionReconciler := version.NewReconciler(apiReader, dtc, timeprovider.New().Freeze())
	connectionInfoReconciler := agconnectioninfo.NewReconciler(clt, apiReader, dtc, dk)
	pullSecretReconciler := dtpullsecret.NewReconciler(clt, apiReader, dk, tokens)

	newCustomPropertiesReconcilerFunc := func(customPropertiesOwnerName string, customPropertiesSource *value.Source) controllers.Reconciler {
		return customproperties.NewReconciler(clt, apiReader, dk, customPropertiesOwnerName, customPropertiesSource)
	}

	return &Reconciler{
		client:                            clt,
		apiReader:                         apiReader,
		dk:                                dk,
		authTokenReconciler:               authTokenReconciler,
		istioReconciler:                   istioReconciler,
		connectionReconciler:              connectionInfoReconciler,
		versionReconciler:                 versionReconciler,
		pullSecretReconciler:              pullSecretReconciler,
		newCustomPropertiesReconcilerFunc: newCustomPropertiesReconcilerFunc,
		newStatefulsetReconcilerFunc:      statefulset.NewReconciler,
		newCapabilityReconcilerFunc:       capabilityInternal.NewReconciler,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	// If AG is not used or was not cleaned up due to being previously enabled
	// Split the `if` for better logging.
	if !r.dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), statefulset.ActiveGateStatefulSetConditionType) == nil {
			log.Info("activeGate not enabled, skipping")

			return nil
		}

		// didn't want to use "defer" for the condition removal, that would be change the behavior bit much for a bug fix
		// the sub reconcilers are either nice enough to not fail during cleanup or not
		log.Info("activeGate was disabled, starting cleanup")
	}

	err := r.connectionReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.createActiveGateTenantConnectionInfoConfigMap(ctx)
	if err != nil {
		return err
	}

	err = r.versionReconciler.ReconcileActiveGate(ctx, r.dk)
	if err != nil {
		return err
	}

	err = r.pullSecretReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	if r.istioReconciler != nil {
		err = r.istioReconciler.ReconcileActiveGateCommunicationHosts(ctx, r.dk)
		if err != nil {
			return err
		}
	}

	err = r.authTokenReconciler.Reconcile(ctx)
	if err != nil {
		return errors.WithMessage(err, "could not reconcile Dynatrace ActiveGateAuthToken secrets")
	}

	agCapability := capability.NewMultiCapability(r.dk)
	if agCapability.Enabled() {
		return r.createCapability(ctx, agCapability)
	} else {
		if err := r.deleteCapability(ctx); err != nil {
			return err
		}
	}
	// TODO: move cleanup to ActiveGate reconciler
	meta.RemoveStatusCondition(r.dk.Conditions(), statefulset.ActiveGateStatefulSetConditionType)

	return nil
}

func (r *Reconciler) createActiveGateTenantConnectionInfoConfigMap(ctx context.Context) error {
	if !r.dk.ActiveGate().IsEnabled() {
		// TODO: Add clean up of the config map
		return nil
	}

	configMapData := extractPublicData(r.dk)

	configMap, err := k8sconfigmap.Build(r.dk,
		r.dk.ActiveGate().GetConnectionInfoConfigMapName(),
		configMapData,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	query := k8sconfigmap.Query(r.client, r.apiReader, log)

	_, err = query.CreateOrUpdate(ctx, configMap)
	if err != nil {
		log.Info("could not create or update configMap for connection info", "name", configMap.Name)

		return err
	}

	return nil
}

func extractPublicData(dk *dynakube.DynaKube) map[string]string {
	data := map[string]string{}

	if dk.Status.ActiveGate.ConnectionInfo.TenantUUID != "" {
		data[connectioninfo.TenantUUIDKey] = dk.Status.ActiveGate.ConnectionInfo.TenantUUID
	}

	if dk.Status.ActiveGate.ConnectionInfo.Endpoints != "" {
		data[connectioninfo.CommunicationEndpointsKey] = dk.Status.ActiveGate.ConnectionInfo.Endpoints
	}

	return data
}

func (r *Reconciler) createCapability(ctx context.Context, agCapability capability.Capability) error {
	customPropertiesReconciler := r.newCustomPropertiesReconcilerFunc(r.dk.ActiveGate().GetServiceAccountOwner(), agCapability.Properties().CustomProperties) //nolint:typeCheck
	statefulsetReconciler := r.newStatefulsetReconcilerFunc(r.client, r.apiReader, r.dk, agCapability)                                                        //nolint:typeCheck
	tlsSecretReconciler := tls.NewReconciler(r.client, r.apiReader, r.dk)

	capabilityReconciler := r.newCapabilityReconcilerFunc(r.client, agCapability, r.dk, statefulsetReconciler, customPropertiesReconciler, tlsSecretReconciler)

	return capabilityReconciler.Reconcile(ctx)
}

func (r *Reconciler) deleteCapability(ctx context.Context) error {
	if err := r.deleteStatefulset(ctx); err != nil {
		return err
	}

	if err := r.deleteService(ctx); err != nil {
		return err
	}

	// we must run tls reconciler to ensure that the TLS secret is deleted
	// TODO: consider to not mix two different patterns
	tlsSecretReconciler := tls.NewReconciler(r.client, r.apiReader, r.dk)
	if err := tlsSecretReconciler.Reconcile(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) deleteService(ctx context.Context) error {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(r.dk.Name),
			Namespace: r.dk.Namespace,
		},
	}

	return client.IgnoreNotFound(r.client.Delete(ctx, &svc))
}

func (r *Reconciler) deleteStatefulset(ctx context.Context) error {
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.CalculateStatefulSetName(r.dk.Name),
			Namespace: r.dk.Namespace,
		},
	}

	return client.IgnoreNotFound(r.client.Delete(ctx, &sts))
}
