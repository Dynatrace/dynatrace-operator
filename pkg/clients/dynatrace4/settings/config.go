package settings

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("dtclient.settings")
)

const (
	validateOnlyQueryParam = "validateOnly"
	trueQueryParamValue    = "true"

	EffectiveValuesPath = "/v2/settings/effectiveValues"
	ObjectsPath         = "/v2/settings/objects"
)

type Client interface {
	GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error)
	GetK8sClusterMEDeleteThisMethod(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error)
	GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error)
	GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetLogMonSettingsResponse, error)
	CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel, kubeSystemUUID, scope string) (string, error)
	CreateOrUpdateKubernetesAppSetting(ctx context.Context, scope string) (string, error)
	GetRulesSettings(ctx context.Context, kubeSystemUUID string, entityID string) (GetRulesSettingsResponse, error)
	CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error)
}
