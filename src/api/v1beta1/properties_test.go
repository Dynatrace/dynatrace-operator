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

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testAPIURL = "http://test-endpoint/api"

func TestActiveGateImage(t *testing.T) {
	t.Run(`ActiveGateImage with no API URL`, func(t *testing.T) {
		dk := DynaKube{}
		assert.Equal(t, "", dk.ActiveGateImage())
	})

	t.Run(`ActiveGateImage with API URL`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/activegate:latest", dk.ActiveGateImage())
	})
	//
	//t.Run(`ActiveGateImage with custom image`, func(t *testing.T) {
	//	customImg := "registry/my/activegate:latest"
	//	dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{Image: customImg}}}
	//	assert.Equal(t, customImg, dk.ActiveGateImage())
	//})
}

func TestDynaKube_UseCSIDriver(t *testing.T) {
	t.Run(`DynaKube with application monitoring without csi driver`, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{},
				},
			},
		}
		assert.Equal(t, false, dk.NeedsCSIDriver())
	})

	t.Run(`DynaKube with application monitoring with csi driver enabled`, func(t *testing.T) {
		useCSIDriver := true
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{
						UseCSIDriver: &useCSIDriver,
					},
				},
			},
		}
		assert.Equal(t, true, dk.NeedsCSIDriver())
	})

	t.Run(`DynaKube with cloud native`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{CloudNativeFullStack: &CloudNativeFullStackSpec{}}}}
		assert.Equal(t, true, dk.NeedsCSIDriver())
	})
}

func TestOneAgentImage(t *testing.T) {
	t.Run(`OneAgentImage with no API URL`, func(t *testing.T) {
		dk := DynaKube{}
		assert.Equal(t, "", dk.ImmutableOneAgentImage())
	})

	t.Run(`OneAgentImage with API URL`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/oneagent:latest", dk.ImmutableOneAgentImage())
	})

	t.Run(`OneAgentImage with API URL and custom version`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL, OneAgent: OneAgentSpec{ClassicFullStack: &ClassicFullStackSpec{Version: "1.234.5"}}}}
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5", dk.ImmutableOneAgentImage())
	})

	t.Run(`OneAgentImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/oneagent:latest"
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{ClassicFullStack: &ClassicFullStackSpec{Image: customImg}}}}
		assert.Equal(t, customImg, dk.ImmutableOneAgentImage())
	})
}

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
	t.Run("", func(t *testing.T) {
		apiUrl := "https://demo.dev.dynatracelabs.com/api"
		expectedTenantId := "demo"

		actualTenantId, err := TenantUUID(apiUrl)

		assert.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("", func(t *testing.T) {
		apiUrl := "demo.dev.dynatracelabs.com/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from fqdn ''"

		actualTenantId, err := TenantUUID(apiUrl)

		assert.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("", func(t *testing.T) {
		apiUrl := "https://google.com"
		expectedTenantId := ""
		expectedError := "api url https://google.com does not end with /api"

		actualTenantId, err := TenantUUID(apiUrl)

		assert.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("", func(t *testing.T) {
		apiUrl := "/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from fqdn ''"

		actualTenantId, err := TenantUUID(apiUrl)

		assert.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})
}
