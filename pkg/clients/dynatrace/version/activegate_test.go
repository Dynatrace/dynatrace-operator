package version

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestActiveGateVersion(t *testing.T) {
	setupMockedClient := func(t *testing.T, os string, err error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().
			WithPaasToken().
			Return(req).Once()
		req.EXPECT().
			Execute(new(struct {
				LatestGatewayVersion string `json:"latestGatewayVersion"`
			})).
			Run(func(model any) {
				resp := model.(*struct {
					LatestGatewayVersion string `json:"latestGatewayVersion"`
				})
				resp.LatestGatewayVersion = "1.2.3"
			}).
			Return(err).Once()
		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getLatestActiveGateVersionPath(os)).Return(req).Once()

		return NewClient(client)
	}

	t.Run("ok - returns version", func(t *testing.T) {
		client := setupMockedClient(t, installer.OSUnix, nil)

		version, err := client.GetLatestActiveGateVersion(t.Context(), installer.OSUnix)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("empty os", func(t *testing.T) {
		client := NewClient(nil)

		_, err := client.GetLatestActiveGateVersion(t.Context(), "")
		assert.Error(t, err, "os is empty")
	})

	t.Run("server error", func(t *testing.T) {
		expectErr := errors.New("boom")
		client := setupMockedClient(t, installer.OSUnix, expectErr)

		_, err := client.GetLatestActiveGateVersion(t.Context(), installer.OSUnix)
		assert.ErrorIs(t, err, expectErr)
	})
}
