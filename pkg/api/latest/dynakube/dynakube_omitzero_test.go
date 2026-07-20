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
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOmitzero verifies that the struct-typed fields tagged with `omitzero`
// (see dynakube_types.go and dynakube_status.go) are dropped from the marshaled
// JSON when they hold a zero value, and are still rendered when they carry a
// value. `omitempty` has no effect on non-pointer struct fields, which is why
// these fields need `omitzero` (Go 1.24+).
//
// Status types that embed status.VersionStatus (oneagent.Status,
// activegate.Status, kubemon.Status) each define a complete IsZero() so that
// omitzero considers ALL fields, not just the promoted version fields. Without
// it, omitzero would drop the whole status object (losing connection info) when
// the version happens to be unset — see the round-trip subtest below.
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
		statusJSON := mustMarshal(t, DynaKubeStatus{})
		apiTokenJSON := mustMarshal(t, APITokenStatus{})
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
		assert.NotContains(t, templatesJSON, `"sqlExtensionExecutor"`)
		assert.NotContains(t, templatesJSON, `"extensionExecutionController"`)

		// Assert: DynaKubeStatus
		assert.NotContains(t, statusJSON, `"oneAgent"`)
		assert.NotContains(t, statusJSON, `"activeGate"`)
		assert.NotContains(t, statusJSON, `"kubernetesMonitoring"`)
		assert.NotContains(t, statusJSON, `"codeModules"`)
		assert.NotContains(t, statusJSON, `"metadataEnrichment"`)
		assert.NotContains(t, statusJSON, `"kspm"`)
		assert.NotContains(t, statusJSON, `"updatedTimestamp"`)
		assert.NotContains(t, statusJSON, `"apiToken"`)

		// Assert: APITokenStatus
		assert.NotContains(t, apiTokenJSON, `"availableOptionalScopes"`)

		// Assert: DynaKubeList
		assert.NotContains(t, listJSON, `"metadata"`)
	})

	t.Run("populated struct fields are still rendered", func(t *testing.T) {
		// Arrange: give every omitzero-tagged struct field a non-zero value.
		enabled := true
		dk := DynaKube{
			Status: DynaKubeStatus{
				OneAgent:             oneagent.Status{ConnectionInfo: communication.ConnectionInfo{TenantUUID: "test-uuid"}},
				ActiveGate:           activegate.Status{ConnectionInfo: communication.ConnectionInfo{TenantUUID: "test-uuid"}},
				KubernetesMonitoring: kubemon.Status{ConnectionInfo: communication.ConnectionInfo{TenantUUID: "test-uuid"}},
				CodeModules:          oneagent.CodeModulesStatus{VersionStatus: status.VersionStatus{Version: "1.0.0"}},
				MetadataEnrichment:   metadataenrichment.Status{Rules: []metadataenrichment.Rule{{}}},
				KSPM:                 kspm.Status{TokenSecretHash: "some-hash"},
				UpdatedTimestamp:     metav1.Now(),
				APIToken:             APITokenStatus{AvailableOptionalScopes: AvailableOptionalScopes{SettingsRead: &enabled}},
			},
			Spec: DynaKubeSpec{
				APIURL:             "https://test.dev.dynatracelabs.com/api",
				MetadataEnrichment: metadataenrichment.Spec{Enabled: &enabled},
				OneAgent:           oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{}},
				ActiveGate:         activegate.Spec{Annotations: map[string]string{"key": "value"}},
				Templates: TemplatesSpec{
					KSPMNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{Labels: map[string]string{"key": "value"}},
					OpenTelemetryCollector:         OpenTelemetryCollectorSpec{Labels: map[string]string{"key": "value"}},
					SQLExtensionExecutor:           extensions.DatabaseExecutorSpec{Tolerations: []corev1.Toleration{{}}},
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
		assert.Contains(t, dkJSON, `"sqlExtensionExecutor"`)
		assert.Contains(t, dkJSON, `"extensionExecutionController"`)

		// Assert: DynaKubeStatus
		assert.Contains(t, dkJSON, `"kubernetesMonitoring"`)
		assert.Contains(t, dkJSON, `"codeModules"`)
		assert.Contains(t, dkJSON, `"kspm"`)
		assert.Contains(t, dkJSON, `"updatedTimestamp"`)
		assert.Contains(t, dkJSON, `"apiToken"`)
		assert.Contains(t, dkJSON, `"availableOptionalScopes"`)

		// Assert: DynaKubeList
		assert.Contains(t, listJSON, `"metadata"`)
	})

	// Regression guard for ICP-973: status is JSON round-tripped on every
	// status update (client.Status().Update). A status carrying connection info
	// but no version must survive that round-trip; a fully empty status must be
	// dropped. This is exactly the data-loss bug that a partial promoted
	// IsZero() would reintroduce.
	t.Run("status survives marshal/unmarshal round-trip", func(t *testing.T) {
		// Arrange: connection info set, version deliberately left unset.
		dk := DynaKube{}
		dk.Status.OneAgent.ConnectionInfo = communication.ConnectionInfo{TenantUUID: "test-uuid", Endpoints: "test-endpoints"}
		dk.Status.ActiveGate.ConnectionInfo = communication.ConnectionInfo{TenantUUID: "test-uuid", Endpoints: "test-endpoints"}

		// Act
		data, err := json.Marshal(dk)
		require.NoError(t, err)

		var back DynaKube
		require.NoError(t, json.Unmarshal(data, &back))

		// Assert: connection info preserved through the round-trip.
		assert.Equal(t, "test-uuid", back.Status.OneAgent.ConnectionInfo.TenantUUID)
		assert.Equal(t, "test-endpoints", back.Status.OneAgent.ConnectionInfo.Endpoints)
		assert.Equal(t, "test-uuid", back.Status.ActiveGate.ConnectionInfo.TenantUUID)
		assert.Equal(t, "test-endpoints", back.Status.ActiveGate.ConnectionInfo.Endpoints)

		// Assert: a fully empty status is dropped entirely.
		emptyJSON := mustMarshal(t, DynaKube{})
		assert.NotContains(t, emptyJSON, `"status"`)
	})
}
