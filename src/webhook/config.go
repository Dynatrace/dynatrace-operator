package webhook

const (
	// LabelInstance can be set in a Namespace and indicates the corresponding DynaKube object assigned to it.
	LabelInstance = "dynakube.internal.dynatrace.com/instance"

	// AnnotationDynatraceInjected is set to "true" by the webhook to Pods to indicate that it has been injected.
	AnnotationDynatraceInjected = "dynakube.dynatrace.com/injected"

	OneAgentPrefix = "oneagent"
	// AnnotationOneAgentInject can be set at pod level to enable/disable OneAgent injection.
	AnnotationOneAgentInject   = OneAgentPrefix + ".dynatrace.com/inject"
	AnnotationOneAgentInjected = OneAgentPrefix + ".dynatrace.com/injected"

	DataIngestPrefix = "data-ingest"
	// AnnotationDataIngestInject can be set at pod level to enable/disable data-ingest injection.
	AnnotationDataIngestInject   = DataIngestPrefix + ".dynatrace.com/inject"
	AnnotationDataIngestInjected = DataIngestPrefix + ".dynatrace.com/injected"

	// AnnotationFlavor can be set on a Pod to configure which code modules flavor to download. It's set to "default"
	// if not set.
	AnnotationFlavor = "oneagent.dynatrace.com/flavor"

	// AnnotationTechnologies can be set on a Pod to configure which code module technologies to download. It's set to
	// "all" if not set.
	AnnotationTechnologies = "oneagent.dynatrace.com/technologies"

	// AnnotationInstallPath can be set on a Pod to configure on which directory the OneAgent will be available from,
	// defaults to DefaultInstallPath if not set.
	AnnotationInstallPath = "oneagent.dynatrace.com/install-path"

	// AnnotationInstallerUrl can be set on a Pod to configure the installer url for downloading the agent
	// defaults to the PaaS installer download url of your tenant
	AnnotationInstallerUrl = "oneagent.dynatrace.com/installer-url"

	// AnnotationFailurePolicy can be set on a Pod to control what the init container does on failures. When set to
	// "fail", the init container will exit with error code 1. Defaults to "silent".
	AnnotationFailurePolicy = "oneagent.dynatrace.com/failure-policy"

	// DefaultInstallPath is the default directory to install the app-only OneAgent package.
	DefaultInstallPath = "/opt/dynatrace/oneagent-paas"

	// SecretConfigName is the name of the secret where the Operator replicates the config data.
	SecretConfigName = "dynatrace-dynakube-config"

	// SecretCertsName is the name of the secret where the webhook certificates are stored.
	SecretCertsName = "dynatrace-webhook-certs"

	// DeploymentName is the name used for the Deployment of any webhooks and WebhookConfiguration objects.
	DeploymentName = "dynatrace-webhook"

	// InstallContainerName is the name used for the install container
	InstallContainerName = "install-oneagent"
)
