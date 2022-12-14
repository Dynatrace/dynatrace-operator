package deploymentmetadata

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	orchestrationTech = "Operator"

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"

	DeploymentTypeApplicationMonitoring = "application_monitoring"
	DeploymentTypeFullStack             = "classic_fullstack"
	DeploymentTypeCloudNative           = "cloud_native_fullstack"
	DeploymentTypeHostMonitoring        = "host_monitoring"
	DeploymentTypeActiveGate            = "active_gate"

	EnvDtDeploymentMetadata = "DT_DEPLOYMENT_METADATA"

	OneAgentMetadataKey   = "oneagent"
	ActiveGateMetadataKey = "activegate"
)

var (
	log = logger.Factory.GetLogger("dynakube-deployment-metadata")
)
