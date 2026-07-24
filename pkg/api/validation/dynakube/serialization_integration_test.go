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

package validation_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	validation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"
)

// updateSerializationGolden regenerates the serialization golden files instead
// of asserting against them. Run:
//
//	go test ./pkg/api/validation/dynakube/ -run TestSerialization -update
var updateSerializationGolden = flag.Bool("update", false, "update the serialization golden files in testdata/")

var latestGVK = latest.GroupVersion.WithKind("DynaKube")

// TestSerialization exercises how the latest (v1beta6) DynaKube is persisted by
// the API server, which is where the `omitzero` struct tags take effect on the
// write path.
//
// Each case creates the object from the TYPED struct (client marshals it with
// its JSON tags, so omitzero is applied) and then reads it back as raw
// *unstructured.Unstructured (no re-decoding into typed structs), which is what
// the golden files capture. This is intentionally different from the conversion
// integration test, which both sets every field and round-trips through the
// unstructured client, so it never exercises the tags.
func TestSerialization(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		serializationWebhookOptions(),
		validation.SetupWebhookWithManager,
	)

	tests := []struct {
		name   string
		spec   dynakube.DynaKubeSpec
		status *dynakube.DynaKubeStatus
	}{
		{
			// Only apiUrl set: nothing else should appear, no empty {} blocks
			// anywhere in spec, and no status.
			name: "minimal",
			spec: dynakube.DynaKubeSpec{
				APIURL: "https://minimal.dev.dynatracelabs.com/api",
			},
		},
		{
			// A few nested fields set, their struct-typed siblings left empty:
			// metadataEnrichment.enabled without namespaceSelector, and
			// oneAgent.hostMonitoring.version without oneAgentResources.
			name: "partial",
			spec: dynakube.DynaKubeSpec{
				APIURL:             "https://partial.dev.dynatracelabs.com/api",
				MetadataEnrichment: metadataenrichment.Spec{Enabled: new(true)},
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Version: "1.0.0.20240101-000000"},
				},
			},
		},
		{
			// Regression guard: OneAgent connection info set while the version
			// is unset. A narrow promoted IsZero() used to drop this whole
			// status block; the golden file must show the connection info.
			name: "status-connection-info",
			spec: dynakube.DynaKubeSpec{
				APIURL: "https://status.dev.dynatracelabs.com/api",
			},
			status: &dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: "abc12345",
						Endpoints:  "https://abc12345.dev.dynatracelabs.com",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create from the typed struct: the client marshals it via the JSON
			// tags, so omitzero decides which struct fields are sent.
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: tt.name, Namespace: metav1.NamespaceDefault},
				Spec:       tt.spec,
			}
			require.NoError(t, clt.Create(t.Context(), dk))
			t.Cleanup(func() {
				// t.Context is no longer valid during cleanup
				assert.NoError(t, clt.Delete(context.Background(), dk))
			})

			// Status is not persisted on create; set it via the subresource.
			if tt.status != nil {
				dk.Status = *tt.status
				require.NoError(t, clt.Status().Update(t.Context(), dk))
			}

			// Read the raw stored object back as unstructured - no decoding into
			// typed structs, so this is exactly what the server persisted.
			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(latestGVK)
			require.NoError(t, clt.Get(t.Context(), client.ObjectKeyFromObject(dk), got))

			stripServerManagedFields(got)

			gotData, err := yaml.Marshal(got.Object)
			require.NoError(t, err)

			golden := filepath.Join("testdata", "serialized-"+tt.name+".yaml")
			if *updateSerializationGolden {
				require.NoError(t, os.WriteFile(golden, gotData, 0o600))
			}

			wantData, err := os.ReadFile(golden)
			require.NoError(t, err)

			assert.Equal(t, string(wantData), string(gotData))
		})
	}
}

// stripServerManagedFields removes the non-deterministic metadata the API server
// sets, so the golden files stay stable across runs.
func stripServerManagedFields(obj *unstructured.Unstructured) {
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetManagedFields(nil)
}

// serializationWebhookOptions registers the v1beta6 validating webhook so that
// creates are validated by the real webhook, matching production behavior.
func serializationWebhookOptions() envtest.WebhookInstallOptions {
	return envtest.WebhookInstallOptions{
		ValidatingWebhooks: []*admissionregistrationv1.ValidatingWebhookConfiguration{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "dynatrace-webhook"},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						Name: "v1beta6.dynakube.webhook.dynatrace.com",
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							Service: &admissionregistrationv1.ServiceReference{
								Path: new("/validate-dynatrace-com-v1beta6-dynakube"),
							},
						},
						Rules: []admissionregistrationv1.RuleWithOperations{
							{
								Operations: []admissionregistrationv1.OperationType{
									admissionregistrationv1.Create,
									admissionregistrationv1.Update,
								},
								Rule: admissionregistrationv1.Rule{
									APIGroups:   []string{"dynatrace.com"},
									APIVersions: []string{"v1beta6"},
									Resources:   []string{"dynakubes"},
								},
							},
						},
						MatchPolicy:             new(admissionregistrationv1.Exact),
						SideEffects:             new(admissionregistrationv1.SideEffectClassNone),
						TimeoutSeconds:          new(int32(10)),
						AdmissionReviewVersions: []string{"v1"},
					},
				},
			},
		},
	}
}
