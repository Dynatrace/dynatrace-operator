package mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"

	dataIngestInjectedEnvVarName = "DATA_INGEST_INJECTED"
	oneAgentInjectedEnvVarName   = "ONEAGENT_INJECTED"
	dynatraceMetadataEnvVarName  = "DT_DEPLOYMENT_METADATA"

	workloadKindEnvVarName = "DT_WORKLOAD_KIND"
	workloadNameEnvVarName = "DT_WORKLOAD_NAME"

	dataIngestVolumeName = "data-ingest-enrichment"

	dataIngestEndpointVolumeName = "data-ingest-endpoint"

	oneAgentBinVolumeName   = "oneagent-bin"
	oneAgentShareVolumeName = "oneagent-share"

	injectionConfigVolumeName = "injection-config"

	provisionedVolumeMode = "provisioned"
	installerVolumeMode   = "installer"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook")
)
