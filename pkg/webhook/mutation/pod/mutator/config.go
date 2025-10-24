package mutator

const (
	// InjectionInstanceLabel can be set in a Namespace and indicates the corresponding DynaKube object assigned to it.
	InjectionInstanceLabel = "dynakube.internal.dynatrace.com/instance"

	// AnnotationFailurePolicy can be set on a Pod to control what the init container does on failures. When set to
	// "fail", the init container will exit with error code 1. Defaults to "silent".
	AnnotationFailurePolicy = "oneagent.dynatrace.com/failure-policy"

	// AnnotationDynatraceInjected is set to "true" by the webhook to Pods to indicate that it has been injected.
	AnnotationDynatraceInjected = "dynakube.dynatrace.com/injected"

	// AnnotationDynatraceReason is add to provide extra info why an injection didn't happen.
	AnnotationDynatraceReason = "dynakube.dynatrace.com/reason"

	// AnnotationDynatraceInject is set to "false" on the Pod to indicate that does not want any injection.
	AnnotationDynatraceInject = "dynatrace.com/inject"

	AnnotationContainerInjection = "container.inject.dynatrace.com"

	OTLPAnnotationPrefix = "otlp-exporter-configuration"
	// AnnotationOTLPInjectionEnabled controls whether the automatic injection of OTLP env vars and resource attributes should happen for a pod
	AnnotationOTLPInjectionEnabled = OTLPAnnotationPrefix + ".dynatrace.com/inject"
	// AnnotationOTLPInjected indicates whether the OTLP env vars and resource attributes have already been injected into a pod
	AnnotationOTLPInjected = OTLPAnnotationPrefix + ".dynatrace.com/injected"
	// AnnotationOTLPReason is add to provide extra info why an injection didn't happen.
	AnnotationOTLPReason = OTLPAnnotationPrefix + ".dynatrace.com/reason"

	// InstallContainerName is the name used for the install container
	InstallContainerName = "dynatrace-operator"
)
