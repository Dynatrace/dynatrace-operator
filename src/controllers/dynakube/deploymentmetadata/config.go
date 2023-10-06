package deploymentmetadata

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
)

const (
	orchestrationTech = "Operator"

	operatorScriptVersionKey = "script_version"
	orchestratorIDKey        = "orchestrator_id"
	orchestrationTechKey     = "orchestration_tech"

	ApplicationMonitoringDeploymentType = "application_monitoring"
	ClassicFullStackDeploymentType      = "classic_fullstack"
	CloudNativeDeploymentType           = "cloud_native_fullstack"
	HostMonitoringDeploymentType        = "host_monitoring"

	EnvDtDeploymentMetadata = "DT_DEPLOYMENT_METADATA"
	EnvDtOperatorVersion    = "DT_OPERATOR_VERSION"

	OneAgentMetadataKey   = "oneagent"
	ActiveGateMetadataKey = "activegate"
	OperatorVersionKey    = "operator"
)

var (
	log = logger.Factory.GetLogger("dynakube-deployment-metadata")
)
