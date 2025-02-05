package api

const (
	LatestTag                            = "latest"
	RawTag                               = "raw"
	InternalFlagPrefix                   = "internal.operator.dynatrace.com/"
	AnnotationSecretHash                 = InternalFlagPrefix + "secret-hash"
	AnnotationTelemetryServiceSecretHash = InternalFlagPrefix + "ts-secret-hash"
)
