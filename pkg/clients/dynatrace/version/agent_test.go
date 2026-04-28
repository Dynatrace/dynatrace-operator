package version

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestAgentVersion(t *testing.T) {
	setupMockedClient := func(t *testing.T, os, installerType string, queryParams map[string]string, err error) *ClientImpl {
		req := coremock.NewRequest(t)
		req.EXPECT().WithPath([]string{os, installerType, "latest/metainfo"}).Return(req).Once()
		req.EXPECT().WithQueryParams(queryParams).Return(req).Once()
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().
			Execute(new(struct {
				LatestAgentVersion string `json:"latestAgentVersion"`
			})).
			Run(func(model any) {
				resp := model.(*struct {
					LatestAgentVersion string `json:"latestAgentVersion"`
				})
				resp.LatestAgentVersion = "1.2.3"
			}).
			Return(err).Once()
		apiClient := coremock.NewClient(t)
		apiClient.EXPECT().GET(t.Context(), "/v1/deployment/installer/agent").Return(req).Once()

		return NewClient(apiClient)
	}

	t.Run("ok, uses paas token, installer.TypeDefault", func(t *testing.T) {
		queryParams := map[string]string{
			"bitness": "64",
			"flavor":  arch.FlavorDefault,
		}
		client := setupMockedClient(t, installer.OSUnix, installer.TypeDefault, queryParams, nil)

		version, err := client.GetLatestAgentVersion(t.Context(), installer.OSUnix, installer.TypeDefault)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("ok, uses paas token, installer.TypePaaS", func(t *testing.T) {
		queryParams := map[string]string{
			"bitness": "64",
			"flavor":  arch.Flavor,
			"arch":    arch.Arch,
		}
		client := setupMockedClient(t, installer.OSUnix, installer.TypePaaS, queryParams, nil)

		version, err := client.GetLatestAgentVersion(t.Context(), installer.OSUnix, installer.TypePaaS)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", version)
	})

	t.Run("empty os", func(t *testing.T) {
		client := NewClient(nil)

		_, err := client.GetLatestAgentVersion(t.Context(), "", installer.TypeDefault)
		assert.ErrorIs(t, err, errEmptyOSOrInstallerType)
	})

	t.Run("empty installerType", func(t *testing.T) {
		client := NewClient(nil)

		_, err := client.GetLatestAgentVersion(t.Context(), installer.OSUnix, "")
		assert.ErrorIs(t, err, errEmptyOSOrInstallerType)
	})

	t.Run("server error", func(t *testing.T) {
		queryParams := map[string]string{
			"bitness": "64",
			"flavor":  arch.FlavorDefault,
		}
		expectErr := errors.New("boom")
		client := setupMockedClient(t, installer.OSUnix, installer.TypeDefault, queryParams, expectErr)

		_, err := client.GetLatestAgentVersion(t.Context(), installer.OSUnix, installer.TypeDefault)
		assert.ErrorIs(t, err, expectErr)
	})
}
