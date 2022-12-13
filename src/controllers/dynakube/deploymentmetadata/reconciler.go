package deploymentmetadata

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context   context.Context
	client    client.Client
	apiReader client.Reader
	dynakube  dynatracev1beta1.DynaKube
	clusterID string
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube dynatracev1beta1.DynaKube, clusterID string) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		context:   ctx,
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		clusterID: clusterID,
	}
}

func (r *Reconciler) Reconcile() error {
	configMapData := map[string]string{}

	r.addOneAgentDeploymentMetadata(configMapData)
	r.addActiveGateDeploymentMetadata(configMapData)

	return r.maintainMetadataConfigMap(configMapData)
}

func (r *Reconciler) addOneAgentDeploymentMetadata(data map[string]string) {
	if !r.dynakube.NeedsOneAgent() {
		return
	}
	data[OneAgentMetadataKey] = NewDeploymentMetadata(r.clusterID, GetOneAgentDeploymentType(r.dynakube)).AsString()
}

func (r *Reconciler) addActiveGateDeploymentMetadata(data map[string]string) {
	if !r.dynakube.NeedsActiveGate() {
		return
	}
	data[ActiveGateMetadataKey] = NewDeploymentMetadata(r.clusterID, ActiveGateMetadataKey).AsString()
}

func (r *Reconciler) maintainMetadataConfigMap(data map[string]string) error {
	configMapQuery := kubeobjects.NewConfigMapQuery(r.context, r.client, r.apiReader, log)
	configMap := kubeobjects.NewConfigMap(GetDeploymentMetadataConfigMapName(r.dynakube.Name), r.dynakube.Namespace, data)

	return configMapQuery.CreateOrUpdate(*configMap)
}

func GetDeploymentMetadataConfigMapName(dynakubeName string) string {
	return fmt.Sprintf("%s-deployment-metadata", dynakubeName)
}
