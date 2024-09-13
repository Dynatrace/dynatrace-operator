/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dynakube

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testAPIURL = "http://test-endpoint/api"

func TestTokens(t *testing.T) {
	testName := "test-name"
	testValue := "test-value"

	t.Run(`GetTokensName returns custom token name`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName},
			Spec:       DynaKubeSpec{Tokens: testValue},
		}
		assert.Equal(t, dk.Tokens(), testValue)
	})
	t.Run(`GetTokensName uses instance name as default value`, func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testName}}
		assert.Equal(t, dk.Tokens(), testName)
	})
}

func TestTenantUUID(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		apiUrl := "https://demo.dev.dynatracelabs.com/api"
		expectedTenantId := "demo"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative, no domain)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("missing API URL protocol", func(t *testing.T) {
		apiUrl := "demo.dev.dynatracelabs.com/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from API URL 'demo.dev.dynatracelabs.com/api'"

		actualTenantId, err := tenantUUID(apiUrl)

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

		actualTenantId, err := tenantUUID(apiUrl)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})
}

func TestIsTokenScopeVerificationAllowed(t *testing.T) {
	dk := DynaKube{
		Status: DynaKubeStatus{
			DynatraceApi: DynatraceApiStatus{
				LastTokenScopeRequest: metav1.Time{},
			},
		},
	}

	timeProvider := timeprovider.New().Freeze()
	tests := map[string]struct {
		lastRequestTimeDeltaMinutes int
		updateExpected              bool
		threshold                   int
	}{
		"Do not update after 10 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -10,
			updateExpected:              false,
			threshold:                   -1,
		},
		"Do update after 20 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   -1,
		},
		"Do not update after 3 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -3,
			updateExpected:              false,
			threshold:                   5,
		},
		"Do update after 7 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -7,
			updateExpected:              true,
			threshold:                   5,
		},
		"Do not update after 17 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -17,
			updateExpected:              false,
			threshold:                   20,
		},
		"Do update after 22 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -22,
			updateExpected:              true,
			threshold:                   20,
		},
		"Do update immediately using 0m interval": {
			lastRequestTimeDeltaMinutes: 0,
			updateExpected:              true,
			threshold:                   0,
		},
		"Do update after 1 minute using 0m interval": {
			lastRequestTimeDeltaMinutes: -1,
			updateExpected:              true,
			threshold:                   0,
		},
		"Do update after 20 minutes using 0m interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk.Spec.DynatraceApiRequestThreshold = test.threshold

			lastRequestTime := timeProvider.Now().Add(time.Duration(test.lastRequestTimeDeltaMinutes) * time.Minute)
			dk.Status.DynatraceApi.LastTokenScopeRequest.Time = lastRequestTime

			assert.Equal(t, test.updateExpected, dk.IsTokenScopeVerificationAllowed(timeProvider))
		})
	}
}
