package deploymentmetadata

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	clusterID string
	dk        dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dk dynakube.DynaKube, clusterID string) controllers.Reconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dk dynakube.DynaKube, clusterID string) controllers.Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
		clusterID: clusterID,
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
	if !r.dk.NeedsOneAgent() {
		return
	}

	configMapData[OneAgentMetadataKey] = NewDeploymentMetadata(r.clusterID, GetOneAgentDeploymentType(r.dk)).AsString()
}

func (r *Reconciler) addActiveGateDeploymentMetadata(configMapData map[string]string) {
	if !r.dk.ActiveGate().IsEnabled() {
		return
	}

	configMapData[ActiveGateMetadataKey] = NewDeploymentMetadata(r.clusterID, ActiveGateMetadataKey).AsString()
}

func (r *Reconciler) addOperatorVersionInfo(configMapData map[string]string) {
	if !r.dk.NeedsOneAgent() { // Currently only used for oneAgent args
		return
	}

	configMapData[OperatorVersionKey] = version.Version
}

func (r *Reconciler) maintainMetadataConfigMap(ctx context.Context, configMapData map[string]string) error {
	configMap, err := configmap.Build(&r.dk,
		GetDeploymentMetadataConfigMapName(r.dk.Name),
		configMapData,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	configMapQuery := configmap.Query(r.client, r.apiReader, log)

	if len(configMapData) > 0 {
		_, err := configMapQuery.CreateOrUpdate(ctx, configMap)

		return err
	}

	return configMapQuery.Delete(ctx, configMap)
}

func GetDeploymentMetadataConfigMapName(dynakubeName string) string {
	return dynakubeName + "-deployment-metadata"
}
