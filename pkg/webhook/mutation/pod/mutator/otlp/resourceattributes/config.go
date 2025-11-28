package resourceattributes

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

const (
	OTELResourceAttributesEnv = "OTEL_RESOURCE_ATTRIBUTES"
)
