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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run(`ActiveGateImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/activegate:latest"
		dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{Image: customImg}}}
		assert.Equal(t, customImg, dk.ActiveGateImage())
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
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL, OneAgent: OneAgentSpec{Version: "1.234.5"}}}
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5", dk.ImmutableOneAgentImage())
	})

	t.Run(`OneAgentImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/oneagent:latest"
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{Image: customImg}}}
		assert.Equal(t, customImg, dk.ImmutableOneAgentImage())
	})
}
