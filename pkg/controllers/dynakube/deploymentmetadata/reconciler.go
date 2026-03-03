package deploymentmetadata

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	configMaps k8sconfigmap.QueryObject
	clusterID  string
}

func NewReconciler(clt client.Client, apiReader client.Reader, clusterID string) *Reconciler {
	return &Reconciler{
		configMaps: k8sconfigmap.Query(clt, apiReader, log),
		clusterID:  clusterID,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	configMapData := map[string]string{}

	r.addOneAgentDeploymentMetadata(dk, configMapData)
	r.addActiveGateDeploymentMetadata(dk, configMapData)
	r.addOperatorVersionInfo(dk, configMapData)

	return r.maintainMetadataConfigMap(ctx, dk, configMapData)
}

func (r *Reconciler) addOneAgentDeploymentMetadata(dk *dynakube.DynaKube, configMapData map[string]string) {
	if !dk.OneAgent().IsDaemonsetRequired() {
		return
	}

	configMapData[OneAgentMetadataKey] = NewDeploymentMetadata(r.clusterID, GetOneAgentDeploymentType(*dk)).AsString()
}

func (r *Reconciler) addActiveGateDeploymentMetadata(dk *dynakube.DynaKube, configMapData map[string]string) {
	if !dk.ActiveGate().IsEnabled() {
		return
	}

	configMapData[ActiveGateMetadataKey] = NewDeploymentMetadata(r.clusterID, ActiveGateMetadataKey).AsString()
}

func (r *Reconciler) addOperatorVersionInfo(dk *dynakube.DynaKube, configMapData map[string]string) {
	if !dk.OneAgent().IsDaemonsetRequired() { // Currently only used for oneAgent args
		return
	}

	configMapData[OperatorVersionKey] = version.Version
}

func (r *Reconciler) maintainMetadataConfigMap(ctx context.Context, dk *dynakube.DynaKube, configMapData map[string]string) error {
	configMap, err := k8sconfigmap.Build(dk,
		GetDeploymentMetadataConfigMapName(dk.Name),
		configMapData,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	if len(configMapData) > 0 {
		_, err := r.configMaps.CreateOrUpdate(ctx, configMap)

		return err
	}

	return r.configMaps.Delete(ctx, configMap)
}

func GetDeploymentMetadataConfigMapName(dynakubeName string) string {
	return dynakubeName + "-deployment-metadata"
}
