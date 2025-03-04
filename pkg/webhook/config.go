package webhook

const (
	// InjectionInstanceLabel can be set in a Namespace and indicates the corresponding DynaKube object assigned to it.
	InjectionInstanceLabel = "dynakube.internal.dynatrace.com/instance"

	// AnnotationDynatraceInjected is set to "true" by the webhook to Pods to indicate that it has been injected.
	AnnotationDynatraceInjected = "dynakube.dynatrace.com/injected"

	// AnnotationDynatraceInject is set to "false" on the Pod to indicate that does not want any injection.
	AnnotationDynatraceInject = "dynatrace.com/inject"

	AnnotationContainerInjection = "container.inject.dynatrace.com"

	// SecretCertsName is the name of the secret where the webhook certificates are stored.
	SecretCertsName = "dynatrace-webhook-certs"

	// DeploymentName is the name used for the Deployment of any webhooks and WebhookConfiguration objects.
	DeploymentName = "dynatrace-webhook"

	WebhookContainerName = "webhook"

	// InstallContainerName is the name used for the install container
	InstallContainerName = "dynatrace-operator"
)
