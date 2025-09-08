package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
)

var (
	log = logd.Get().WithName("oa-mutation")
)

const (
	AnnotationPrefix = "oneagent"
	// AnnotationOneAgentInject can be set at pod level to enable/disable OneAgent injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	MissingTenantUUIDReason = "MissingTenantUUID"

	// AnnotationTechnologies can be set on a Pod to configure which code module technologies to download. It's set to
	// "all" if not set.
	AnnotationTechnologies = exp.OANodeImagePullTechnologiesKey

	// AnnotationFlavor can be set on a Pod to configure which code modules flavor to download.
	AnnotationFlavor = "oneagent.dynatrace.com/flavor"

	// AnnotationInstallPath can be set on a Pod to configure on which directory the OneAgent will be available from,
	// defaults to DefaultInstallPath if not set.
	AnnotationInstallPath = AnnotationPrefix + ".dynatrace.com/install-path"

	// AnnotationVolumeType can be set on a Pod to turn off the CSI volume usage.
	// This annotation ONLY takes affect if `node-image-pull` feature-flag is set on the DynaKube.
	AnnotationVolumeType = AnnotationPrefix + ".dynatrace.com/volume-type"

	// AnnotationOneAgentBinResource is used to specify the volume size for EmptyDir for oneagent-bin.
	AnnotationOneAgentBinResource = volumes.AnnotationResourcePrefix + "oneagent-bin"

	// DefaultInstallPath is the default directory to install the app-only OneAgent package.
	DefaultInstallPath = "/opt/dynatrace/oneagent-paas"

	AgentCodeModuleSource = "/opt/dynatrace/oneagent"

	PreloadEnv           = "LD_PRELOAD"
	NetworkZoneEnv       = "DT_NETWORK_ZONE"
	DynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	ReleaseVersionEnv      = "DT_RELEASE_VERSION"
	ReleaseProductEnv      = "DT_RELEASE_PRODUCT"
	ReleaseStageEnv        = "DT_RELEASE_STAGE"
	ReleaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	DefaultUser  int64 = 1001
	DefaultGroup int64 = 1001

	// DtStorageEnv is a temporary env we set on the containers injected with the OneAgent to control where it stores logs and such.
	// This should be replaced by the `storage` property in the ruxitagentproc.conf
	DtStorageEnv  = "DT_STORAGE"
	DtStoragePath = volumes.ConfigMountPath + "/oneagent"
)
