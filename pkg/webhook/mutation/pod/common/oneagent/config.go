package oneagent

const (
	AnnotationPrefix = "oneagent"
	// AnnotationOneAgentInject can be set at pod level to enable/disable OneAgent injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	// AnnotationTechnologies can be set on a Pod to configure which code module technologies to download. It's set to
	// "all" if not set.
	AnnotationTechnologies = "oneagent.dynatrace.com/technologies"

	// AnnotationInstallPath can be set on a Pod to configure on which directory the OneAgent will be available from,
	// defaults to DefaultInstallPath if not set.
	AnnotationInstallPath = "oneagent.dynatrace.com/install-path"

	// DefaultInstallPath is the default directory to install the app-only OneAgent package.
	DefaultInstallPath = "/opt/dynatrace/oneagent-paas"

	PreloadEnv           = "LD_PRELOAD"
	NetworkZoneEnv       = "DT_NETWORK_ZONE"
	DynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	ReleaseVersionEnv      = "DT_RELEASE_VERSION"
	ReleaseProductEnv      = "DT_RELEASE_PRODUCT"
	ReleaseStageEnv        = "DT_RELEASE_STAGE"
	ReleaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	EmptyConnectionInfoReason = "EmptyConnectionInfo"
	UnknownCodeModuleReason   = "UnknownCodeModule"
	EmptyTenantUUIDReason     = "EmptyTenantUUID"

	DefaultUser   int64 = 1001
	DefaultGroup  int64 = 1001
	RootUserGroup int64 = 0
)
