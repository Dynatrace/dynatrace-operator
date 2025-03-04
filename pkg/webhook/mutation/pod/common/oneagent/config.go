package oneagent

const (
	AnnotationPrefix = "oneagent"
	// AnnotationOneAgentInject can be set at pod level to enable/disable OneAgent injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	PreloadEnv           = "LD_PRELOAD"
	NetworkZoneEnv       = "DT_NETWORK_ZONE"
	DynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	ReleaseVersionEnv      = "DT_RELEASE_VERSION"
	ReleaseProductEnv      = "DT_RELEASE_PRODUCT"
	ReleaseStageEnv        = "DT_RELEASE_STAGE"
	ReleaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"
)
