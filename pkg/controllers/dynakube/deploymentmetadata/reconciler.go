package deploymentmetadata

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dynakube  dynatracev1beta1.DynaKube
	clusterID string
	scheme    *runtime.Scheme
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube dynatracev1beta1.DynaKube, clusterID string) controllers.Reconciler

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube dynatracev1beta1.DynaKube, clusterID string) controllers.Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		clusterID: clusterID,
		scheme:    scheme,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	configMapData := map[string]string{}

	r.addOneAgentDeploymentMetadata(configMapData)
	r.addActiveGateDeploymentMetadata(configMapData)
	r.addOperatorVersionInfo(configMapData)

	return r.maintainMetadataConfigMap(ctx, configMapData)
}

func (r *Reconciler) addOneAgentDeploymentMetadata(configMapData map[string]string) {
	if !r.dynakube.NeedsOneAgent() {
		return
	}

	configMapData[OneAgentMetadataKey] = NewDeploymentMetadata(r.clusterID, GetOneAgentDeploymentType(r.dynakube)).AsString()
}

func (r *Reconciler) addActiveGateDeploymentMetadata(configMapData map[string]string) {
	if !r.dynakube.NeedsActiveGate() {
		return
	}

	configMapData[ActiveGateMetadataKey] = NewDeploymentMetadata(r.clusterID, ActiveGateMetadataKey).AsString()
}

func (r *Reconciler) addOperatorVersionInfo(configMapData map[string]string) {
	if !r.dynakube.NeedsOneAgent() { // Currently only used for oneAgent args
		return
	}

	configMapData[OperatorVersionKey] = version.Version
}

func (r *Reconciler) maintainMetadataConfigMap(ctx context.Context, configMapData map[string]string) error {
	configMapQuery := configmap.NewQuery(ctx, r.client, r.apiReader, log)

	configMap, err := configmap.CreateConfigMap(r.scheme, &r.dynakube,
		configmap.NewModifier(GetDeploymentMetadataConfigMapName(r.dynakube.Name)),
		configmap.NewNamespaceModifier(r.dynakube.Namespace),
		configmap.NewConfigMapDataModifier(configMapData))
	if err != nil {
		return errors.WithStack(err)
	}

	if len(configMapData) > 0 {
		return configMapQuery.CreateOrUpdate(*configMap)
	}

	return configMapQuery.Delete(*configMap)
}

func GetDeploymentMetadataConfigMapName(dynakubeName string) string {
	return dynakubeName + "-deployment-metadata"
}
