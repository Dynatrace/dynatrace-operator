package certificates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func createTestMutatingWebhookConfig(_ *testing.T) *admissionregistrationv1.MutatingWebhookConfiguration {
	return &admissionregistrationv1.MutatingWebhookConfiguration{
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{},
			{ClientConfig: admissionregistrationv1.WebhookClientConfig{}},
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte{0, 1, 2, 3, 4},
				},
			},
		},
	}
}

func createTestValidatingWebhookConfig(_ *testing.T) *admissionregistrationv1.ValidatingWebhookConfiguration {
	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{},
			{ClientConfig: admissionregistrationv1.WebhookClientConfig{}},
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte{0, 1, 2, 3, 4},
				},
			},
		},
	}
}

func TestGetClientConfigsFromMutatingWebhook(t *testing.T) {
	t.Run(`returns nil when config is nil`, func(t *testing.T) {
		clientConfigs := getClientConfigsFromMutatingWebhook(nil)
		assert.Nil(t, clientConfigs)
	})
	t.Run(`returns client configs of all configured webhooks`, func(t *testing.T) {
		const expectedClientConfigs = 3

		clientConfigs := getClientConfigsFromMutatingWebhook(createTestMutatingWebhookConfig(t))

		assert.NotNil(t, clientConfigs)
		assert.Len(t, clientConfigs, expectedClientConfigs)
	})
}

func TestGetClientConfigsFromValidatingWebhook(t *testing.T) {
	t.Run(`returns nil when config is nil`, func(t *testing.T) {
		clientConfigs := getClientConfigsFromValidatingWebhook(nil)
		assert.Nil(t, clientConfigs)
	})
	t.Run(`returns client configs of all configured webhooks`, func(t *testing.T) {
		const expectedClientConfigs = 3

		clientConfigs := getClientConfigsFromValidatingWebhook(createTestValidatingWebhookConfig(t))

		assert.NotNil(t, clientConfigs)
		assert.Len(t, clientConfigs, expectedClientConfigs)
	})
}
