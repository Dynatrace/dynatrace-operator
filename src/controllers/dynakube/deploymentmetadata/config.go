package deploymentmetadata

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
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
	ActiveGateDeploymentType            = "active_gate"

	EnvDtDeploymentMetadata = "DT_DEPLOYMENT_METADATA"

	OneAgentMetadataKey   = "oneagent"
	ActiveGateMetadataKey = "activegate"
)

var (
	log = logger.Factory.GetLogger("dynakube-deployment-metadata")
)
