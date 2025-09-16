package otlp

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-otlp")
)

const (
	AnnotationPrefix = "otlp-exporter-configuration"
	// AnnotationOTLPInjectionEnabled controls whether the automatic injection of OTLP env vars and resource attributes should happen for a pod
	AnnotationOTLPInjectionEnabled = AnnotationPrefix + ".dynatrace.com/inject"
	// AnnotationOTLPInjected indicates whether the OTLP env vars and resource attributes have already been injected into a pod
	AnnotationOTLPInjected = AnnotationPrefix + ".dynatrace.com/injected"
)
