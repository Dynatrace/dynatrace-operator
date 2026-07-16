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
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOmitzero verifies that the struct-typed fields tagged with `omitzero`
// (see dynakube_types.go) are dropped from the marshaled JSON when they hold a
// zero value, and are still rendered when they carry a value. `omitempty` has
// no effect on non-pointer struct fields, which is why these fields need
// `omitzero` (Go 1.24+).
func TestOmitzero(t *testing.T) {
	// mustMarshal marshals v and returns the JSON as a string.
	mustMarshal := func(t *testing.T, v any) string {
		t.Helper()

		out, err := json.Marshal(v)
		require.NoError(t, err)

		return string(out)
	}

	t.Run("empty struct fields are dropped", func(t *testing.T) {
		// Arrange: every omitzero-tagged struct field is left at its zero
		// value. Each field is asserted at the level where it lives, so the
		// drop is caused by the field's own omitzero tag and not by a dropped
		// parent.
		dkJSON := mustMarshal(t, DynaKube{})
		specJSON := mustMarshal(t, DynaKubeSpec{})
		templatesJSON := mustMarshal(t, TemplatesSpec{})
		listJSON := mustMarshal(t, DynaKubeList{})

		// Assert: DynaKube
		assert.NotContains(t, dkJSON, `"status"`)
		assert.NotContains(t, dkJSON, `"spec"`)

		// Assert: DynaKubeSpec
		assert.NotContains(t, specJSON, `"metadataEnrichment"`)
		assert.NotContains(t, specJSON, `"oneAgent"`)
		assert.NotContains(t, specJSON, `"templates"`)
		assert.NotContains(t, specJSON, `"activeGate"`)

		// Assert: TemplatesSpec
		assert.NotContains(t, templatesJSON, `"kspmNodeConfigurationCollector"`)
		assert.NotContains(t, templatesJSON, `"otelCollector"`)
		assert.NotContains(t, templatesJSON, `"extensionExecutionController"`)

		// Assert: DynaKubeList
		assert.NotContains(t, listJSON, `"metadata"`)
	})

	t.Run("populated struct fields are still rendered", func(t *testing.T) {
		// Arrange: give every omitzero-tagged struct field a non-zero value.
		enabled := true
		dk := DynaKube{
			Status: DynaKubeStatus{KubeSystemUUID: "some-uuid"},
			Spec: DynaKubeSpec{
				APIURL:             "https://test.dev.dynatracelabs.com/api",
				MetadataEnrichment: metadataenrichment.Spec{Enabled: &enabled},
				OneAgent:           oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{}},
				ActiveGate:         activegate.Spec{Annotations: map[string]string{"key": "value"}},
				Templates: TemplatesSpec{
					KSPMNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{Labels: map[string]string{"key": "value"}},
					OpenTelemetryCollector:         OpenTelemetryCollectorSpec{Labels: map[string]string{"key": "value"}},
					ExtensionExecutionController:   extensions.ExecutionControllerSpec{Labels: map[string]string{"key": "value"}},
				},
			},
		}
		list := DynaKubeList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}}

		// Act
		dkJSON := mustMarshal(t, dk)
		listJSON := mustMarshal(t, list)

		// Assert: DynaKube
		assert.Contains(t, dkJSON, `"status"`)
		assert.Contains(t, dkJSON, `"spec"`)

		// Assert: DynaKubeSpec
		assert.Contains(t, dkJSON, `"metadataEnrichment"`)
		assert.Contains(t, dkJSON, `"oneAgent"`)
		assert.Contains(t, dkJSON, `"templates"`)
		assert.Contains(t, dkJSON, `"activeGate"`)

		// Assert: TemplatesSpec
		assert.Contains(t, dkJSON, `"kspmNodeConfigurationCollector"`)
		assert.Contains(t, dkJSON, `"otelCollector"`)
		assert.Contains(t, dkJSON, `"extensionExecutionController"`)

		// Assert: DynaKubeList
		assert.Contains(t, listJSON, `"metadata"`)
	})
}
