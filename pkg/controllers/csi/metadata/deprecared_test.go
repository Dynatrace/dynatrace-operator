package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantUUIDFromApiUrl(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		apiUrl := "https://demo.dev.dynatracelabs.com/api"
		expectedTenantId := "demo"

		actualTenantId, err := TenantUUIDFromApiUrl(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := TenantUUIDFromApiUrl(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative, no domain)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := TenantUUIDFromApiUrl(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("missing API URL protocol", func(t *testing.T) {
		apiUrl := "demo.dev.dynatracelabs.com/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from API URL 'demo.dev.dynatracelabs.com/api'"

		actualTenantId, err := TenantUUIDFromApiUrl(apiUrl)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("suffix-only, relative API URL", func(t *testing.T) {
		apiUrl := "/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from API URL '/api'"

		actualTenantId, err := TenantUUIDFromApiUrl(apiUrl)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})
}
