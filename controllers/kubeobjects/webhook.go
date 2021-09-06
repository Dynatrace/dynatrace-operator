package kubeobjects

import admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

func GetWebhookClientConfigs(
	mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration,
	validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) []*admissionregistrationv1.WebhookClientConfig {
	var configs []*admissionregistrationv1.WebhookClientConfig
	configs = append(configs, getMutatingClientConfig(mutatingWebhookConfiguration)...)
	configs = append(configs, getValidatingClientConfig(validatingWebhookConfiguration)...)
	return configs
}

func getMutatingClientConfig(mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration) []*admissionregistrationv1.WebhookClientConfig {
	var configs []*admissionregistrationv1.WebhookClientConfig
	for _, config := range mutatingWebhookConfiguration.Webhooks {
		configs = append(configs, &config.ClientConfig)
	}
	return configs
}

func getValidatingClientConfig(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) []*admissionregistrationv1.WebhookClientConfig {
	var configs []*admissionregistrationv1.WebhookClientConfig
	for _, config := range validatingWebhookConfiguration.Webhooks {
		configs = append(configs, &config.ClientConfig)
	}
	return configs
}
