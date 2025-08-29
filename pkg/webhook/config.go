package webhook

const (
	// SecretCertsName is the name of the secret where the webhook certificates are stored.
	SecretCertsName = "dynatrace-webhook-certs"

	// DeploymentName is the name used for the Deployment of any webhooks and WebhookConfiguration objects.
	DeploymentName = "dynatrace-webhook"

	WebhookContainerName = "webhook"
)
