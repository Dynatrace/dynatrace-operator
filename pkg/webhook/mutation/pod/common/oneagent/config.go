package oneagent

const (
	AnnotationPrefix = "oneagent"
	// AnnotationOneAgentInject can be set at pod level to enable/disable OneAgent injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"
)
