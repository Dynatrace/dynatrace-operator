package certificates

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admissionregistration/v1"
	"testing"
)

func createTestMutatingWebhookConfig(_ *testing.T) *v1.MutatingWebhookConfiguration {
	return &v1.MutatingWebhookConfiguration{
		Webhooks: []v1.MutatingWebhook{
			{},
			{ClientConfig: v1.WebhookClientConfig{}},
			{
				ClientConfig: v1.WebhookClientConfig{
					CABundle: []byte{0, 1, 2, 3, 4},
				},
			},
		},
	}
}

func createTestValidatingWebhookConfig(_ *testing.T) *v1.ValidatingWebhookConfiguration {
	return &v1.ValidatingWebhookConfiguration{
		Webhooks: []v1.ValidatingWebhook{
			{},
			{ClientConfig: v1.WebhookClientConfig{}},
			{
				ClientConfig: v1.WebhookClientConfig{
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
		assert.Equal(t, expectedClientConfigs, len(clientConfigs))
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
		assert.Equal(t, expectedClientConfigs, len(clientConfigs))
	})
}
