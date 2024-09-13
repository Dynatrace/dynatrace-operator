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

	"github.com/stretchr/testify/assert"
)

func TestDefaultActiveGateImage(t *testing.T) {
	t.Run(`ActiveGateImage with no API URL`, func(t *testing.T) {
		dk := DynaKube{}
		assert.Equal(t, "", dk.DefaultActiveGateImage(""))
	})

	t.Run(`ActiveGateImage adds raw postfix`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/activegate:1.234.5-raw", dk.DefaultActiveGateImage("1.234.5"))
	})

	t.Run("ActiveGateImage doesn't add 'raw' postfix if present", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/activegate:1.234.5-raw", dk.DefaultActiveGateImage("1.234.5-raw"))
	})

	t.Run(`ActiveGateImage truncates build date`, func(t *testing.T) {
		version := "1.239.14.20220325-164521"
		expectedImage := "test-endpoint/linux/activegate:1.239.14-raw"
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}

		assert.Equal(t, expectedImage, dk.DefaultActiveGateImage(version))
	})
}

func TestCustomActiveGateImage(t *testing.T) {
	t.Run(`ActiveGateImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/activegate:latest"
		dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{CapabilityProperties: CapabilityProperties{
			Image: customImg,
		}}}}
		assert.Equal(t, customImg, dk.CustomActiveGateImage())
	})

	t.Run(`ActiveGateImage with no custom image`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{CapabilityProperties: CapabilityProperties{}}}}
		assert.Equal(t, "", dk.CustomActiveGateImage())
	})
}
