package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantUUIDFromApiUrl(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		apiURL := "https://demo.dev.dynatracelabs.com/api"
		expectedTenantID := "demo"

		actualTenantID, err := TenantUUIDFromAPIURL(apiURL)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiURL)
		assert.Equalf(t, expectedTenantID, actualTenantID, "Expected that tenant id of %s is %s, but found %s",
			apiURL, expectedTenantID, actualTenantID,
		)
	})

	t.Run("happy path (alternative)", func(t *testing.T) {
		apiURL := "https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest"
		expectedTenantID := "tenant"

		actualTenantID, err := TenantUUIDFromAPIURL(apiURL)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiURL)
		assert.Equalf(t, expectedTenantID, actualTenantID, "Expected that tenant id of %s is %s, but found %s",
			apiURL, expectedTenantID, actualTenantID,
		)
	})

	t.Run("happy path (alternative, no domain)", func(t *testing.T) {
		apiURL := "https://dynakube-activegate/e/tenant/api/v2/metrics/ingest"
		expectedTenantID := "tenant"

		actualTenantID, err := TenantUUIDFromAPIURL(apiURL)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiURL)
		assert.Equalf(t, expectedTenantID, actualTenantID, "Expected that tenant id of %s is %s, but found %s",
			apiURL, expectedTenantID, actualTenantID,
		)
	})

	t.Run("missing API URL protocol", func(t *testing.T) {
		apiURL := "demo.dev.dynatracelabs.com/api"
		expectedTenantID := ""
		expectedError := "problem getting tenant id from API URL 'demo.dev.dynatracelabs.com/api'"

		actualTenantID, err := TenantUUIDFromAPIURL(apiURL)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiURL, expectedError,
		)
		assert.Equalf(t, expectedTenantID, actualTenantID, "Expected that tenant id of %s is %s, but found %s",
			apiURL, expectedTenantID, actualTenantID,
		)
	})

	t.Run("suffix-only, relative API URL", func(t *testing.T) {
		apiURL := "/api"
		expectedTenantID := ""
		expectedError := "problem getting tenant id from API URL '/api'"

		actualTenantID, err := TenantUUIDFromAPIURL(apiURL)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiURL, expectedError,
		)
		assert.Equalf(t, expectedTenantID, actualTenantID, "Expected that tenant id of %s is %s, but found %s",
			apiURL, expectedTenantID, actualTenantID,
		)
	})
}
