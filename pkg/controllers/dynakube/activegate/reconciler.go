package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
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
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type authTokenReconciler interface {
	Reconcile(ctx context.Context, agClient agclient.APIClient, dk *dynakube.DynaKube) error
}

type istioReconciler interface {
	ReconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube) error
}

type connectionReconciler interface {
	Reconcile(ctx context.Context, agClient agclient.APIClient, dk *dynakube.DynaKube) error
}

type pullSecretReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube, tokens token.Tokens) error
}

type statefulsetReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type capabilityReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type customPropertiesReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type tlsReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	authTokenReconciler            authTokenReconciler
	istioReconciler                istioReconciler
	connectionReconciler           connectionReconciler
	versionReconcilerFunc          func(dtc dtclient.Client) version.Reconciler
	pullSecretReconciler           pullSecretReconciler
	statefulsetReconcilerFunc      func(capability capability.Capability) statefulsetReconciler
	capabilityReconcilerFunc       func(capability capability.Capability, statefulsetReconciler statefulsetReconciler, customPropertiesReconciler customPropertiesReconciler, tlsSecretReconciler tlsReconciler) capabilityReconciler
	customPropertiesReconcilerFunc func(customPropertiesOwnerName string, customPropertiesSource *value.Source) customPropertiesReconciler
	tlsReconcilerFunc              func() tlsReconciler
	configMaps                     k8sconfigmap.QueryObject
	services                       k8sservice.QueryObject
	statefulSets                   k8sstatefulset.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		authTokenReconciler:  authtoken.NewReconciler(clt, apiReader),
		istioReconciler:      istio.NewReconciler(clt, apiReader),
		connectionReconciler: agconnectioninfo.NewReconciler(clt, apiReader),
		versionReconcilerFunc: func(dtc dtclient.Client) version.Reconciler {
			return version.NewReconciler(apiReader, dtc, timeprovider.New().Freeze())
		},
		pullSecretReconciler: dtpullsecret.NewReconciler(clt, apiReader),
		customPropertiesReconcilerFunc: func(customPropertiesOwnerName string, customPropertiesSource *value.Source) customPropertiesReconciler {
			return customproperties.NewReconciler(clt, apiReader, customPropertiesOwnerName, customPropertiesSource)
		},
		statefulsetReconcilerFunc: func(capability capability.Capability) statefulsetReconciler {
			return statefulset.NewReconciler(clt, apiReader, capability)
		},
		tlsReconcilerFunc: func() tlsReconciler {
			return tls.NewReconciler(clt, apiReader)
		},
		capabilityReconcilerFunc: func(capability capability.Capability, statefulsetReconciler statefulsetReconciler, customPropertiesReconciler customPropertiesReconciler, tlsSecretReconciler tlsReconciler) capabilityReconciler {
			return capabilityInternal.NewReconciler(clt, capability, statefulsetReconciler, customPropertiesReconciler, tlsSecretReconciler)
		},
		configMaps:   k8sconfigmap.Query(clt, apiReader, log),
		services:     k8sservice.Query(clt, apiReader, log),
		statefulSets: k8sstatefulset.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, dtClient dtclient.Client, tokens token.Tokens) error {
	// If AG is not used or was not cleaned up due to being previously enabled
	// Split the `if` for better logging.
	if !dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), statefulset.ActiveGateStatefulSetConditionType) == nil {
			log.Info("activeGate not enabled, skipping")

			return nil
		}

		// didn't want to use "defer" for the condition removal, that would be change the behavior bit much for a bug fix
		// the sub reconcilers are either nice enough to not fail during cleanup or not
		log.Info("activeGate was disabled, starting cleanup")
	}

	err := r.connectionReconciler.Reconcile(ctx, dtClient.AsV2().ActiveGate, dk)
	if err != nil {
		return err
	}

	err = r.createActiveGateTenantConnectionInfoConfigMap(ctx, dk)
	if err != nil {
		return err
	}

	err = r.versionReconcilerFunc(dtClient).ReconcileActiveGate(ctx, dk)
	if err != nil {
		return err
	}

	err = r.pullSecretReconciler.Reconcile(ctx, dk, tokens)
	if err != nil {
		return err
	}

	err = r.istioReconciler.ReconcileActiveGate(ctx, dk)
	if err != nil {
		return err
	}

	err = r.authTokenReconciler.Reconcile(ctx, dtClient.AsV2().ActiveGate, dk)
	if err != nil {
		return errors.WithMessage(err, "could not reconcile Dynatrace ActiveGateAuthToken secrets")
	}

	agCapability := capability.NewMultiCapability(dk)
	if agCapability.Enabled() {
		return r.createCapability(ctx, dk, agCapability)
	} else {
		if err := r.deleteCapability(ctx, dk); err != nil {
			return err
		}
	}
	// TODO: move cleanup to ActiveGate reconciler
	meta.RemoveStatusCondition(dk.Conditions(), statefulset.ActiveGateStatefulSetConditionType)

	return nil
}

func (r *Reconciler) createActiveGateTenantConnectionInfoConfigMap(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.ActiveGate().IsEnabled() {
		// TODO: Add clean up of the config map
		return nil
	}

	configMapData := extractPublicData(dk)

	configMap, err := k8sconfigmap.Build(dk,
		dk.ActiveGate().GetConnectionInfoConfigMapName(),
		configMapData,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.configMaps.CreateOrUpdate(ctx, configMap)
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

func (r *Reconciler) createCapability(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) error {
	customPropertiesReconcilerInstance := r.customPropertiesReconcilerFunc(dk.ActiveGate().GetServiceAccountOwner(), agCapability.Properties().CustomProperties)
	statefulsetReconcilerInstance := r.statefulsetReconcilerFunc(agCapability)
	tlsSecretReconciler := r.tlsReconcilerFunc()

	capabilityReconcilerInstance := r.capabilityReconcilerFunc(agCapability, statefulsetReconcilerInstance, customPropertiesReconcilerInstance, tlsSecretReconciler)

	return capabilityReconcilerInstance.Reconcile(ctx, dk)
}

func (r *Reconciler) deleteCapability(ctx context.Context, dk *dynakube.DynaKube) error {
	if err := r.deleteStatefulset(ctx, dk); err != nil {
		return err
	}

	if err := r.deleteService(ctx, dk); err != nil {
		return err
	}

	// we must run tls reconciler to ensure that the TLS secret is deleted
	// TODO: consider to not mix two different patterns
	tlsSecretReconciler := r.tlsReconcilerFunc()
	if err := tlsSecretReconciler.Reconcile(ctx, dk); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) deleteService(ctx context.Context, dk *dynakube.DynaKube) error {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dk.Name),
			Namespace: dk.Namespace,
		},
	}

	return client.IgnoreNotFound(r.services.Delete(ctx, &svc))
}

func (r *Reconciler) deleteStatefulset(ctx context.Context, dk *dynakube.DynaKube) error {
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.CalculateStatefulSetName(dk.Name),
			Namespace: dk.Namespace,
		},
	}

	return client.IgnoreNotFound(r.statefulSets.Delete(ctx, &sts))
}
