package deploymentmetadata

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube dynatracev1beta1.DynaKube, clusterID string) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
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
	configMapQuery := kubeobjects.NewConfigMapQuery(ctx, r.client, r.apiReader, log)
	configMap, err := kubeobjects.CreateConfigMap(r.scheme, &r.dynakube,
		kubeobjects.NewConfigMapNameModifier(GetDeploymentMetadataConfigMapName(r.dynakube.Name)),
		kubeobjects.NewConfigMapNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewConfigMapDataModifier(configMapData))
	if err != nil {
		return errors.WithStack(err)
	}

	if len(configMapData) > 0 {
		return configMapQuery.CreateOrUpdate(*configMap)
	}
	return configMapQuery.Delete(*configMap)
}

func GetDeploymentMetadataConfigMapName(dynakubeName string) string {
	return fmt.Sprintf("%s-deployment-metadata", dynakubeName)
}
