package otlp

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-otlp")
)

const (
	OTLPAnnotationPrefix = "otlp-exporter-configuration"
	// AnnotationOTLPInjectionEnabled controls whether the automatic injection of OTLP env vars and resource attributes should happen for a pod
	AnnotationOTLPInjectionEnabled = OTLPAnnotationPrefix + ".dynatrace.com/inject"
	// AnnotationOTLPInjected indicates whether the OTLP env vars and resource attributes have already been injected into a pod
	AnnotationOTLPInjected = OTLPAnnotationPrefix + ".dynatrace.com/injected"
	// AnnotationOTLPReason is add to provide extra info why an injection didn't happen.
	AnnotationOTLPReason = OTLPAnnotationPrefix + ".dynatrace.com/reason"

	NoOTLPExporterConfigSecretReason = "NoOTLPExporterConfigSecret"
)
