package validation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	validation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"
)

func TestWebhook(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		envtest.WebhookInstallOptions{
			// TODO(avorima): Load this from a file using Paths
			ValidatingWebhooks: []*admissionregistrationv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dynatrace-webhook",
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: "v1beta4.dynakube.webhook.dynatrace.com",
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{
									Path: new("/validate-dynatrace-com-v1beta4-dynakube"),
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
										APIVersions: []string{"v1beta4"},
										Resources:   []string{"dynakubes"},
									},
								},
							},
							MatchPolicy:             new(admissionregistrationv1.Exact),
							SideEffects:             new(admissionregistrationv1.SideEffectClassNone),
							TimeoutSeconds:          new(int32(10)),
							AdmissionReviewVersions: []string{"v1"},
						},
						{
							Name: "v1beta5.dynakube.webhook.dynatrace.com",
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{
									Path: new("/validate-dynatrace-com-v1beta5-dynakube"),
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
										APIVersions: []string{"v1beta5"},
										Resources:   []string{"dynakubes"},
									},
								},
							},
							MatchPolicy:             new(admissionregistrationv1.Exact),
							SideEffects:             new(admissionregistrationv1.SideEffectClassNone),
							TimeoutSeconds:          new(int32(10)),
							AdmissionReviewVersions: []string{"v1"},
						},
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
		},
		validation.SetupWebhookWithManager,
	)

	servedVersions := []string{
		"v1beta5",
	}
	seenGVKs := sets.New[string]()

	for _, version := range servedVersions {
		t.Run(version, func(t *testing.T) {
			compareWebhookResult(t, clt, version, "default", seenGVKs)
		})
	}

	unServedVersions := []string{
		"v1beta4",
	}
	for _, version := range unServedVersions {
		t.Run(version, func(t *testing.T) {
			oldObj := readTestData(t, version, "default")

			err := clt.Create(t.Context(), oldObj)
			require.True(t, meta.IsNoMatchError(err))
		})
	}

	invalidTests := []struct {
		name string
		spec dynakube.DynaKubeSpec
	}{
		{
			name: "duplicate telemetryingest protocols",
			spec: dynakube.DynaKubeSpec{
				APIURL: "https://test.localhost/api",
				TelemetryIngest: &telemetryingest.Spec{
					Protocols: []otelcgen.Protocol{
						"zipkin",
						"zipkin",
					},
				},
			},
		},
		{
			name: "duplicate activegate capabilities",
			spec: dynakube.DynaKubeSpec{
				APIURL: "https://test.localhost/api",
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						"routing",
						"routing",
					},
				},
			},
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: tt.spec,
			}

			err := clt.Create(t.Context(), dk)
			require.True(t, apierrors.IsInvalid(err), err)
		})
	}
}

func compareWebhookResult(t *testing.T, clt client.Client, version, name string, seen sets.Set[string]) {
	t.Helper()
	oldObj := readTestData(t, version, name)

	require.NoError(t, clt.Create(t.Context(), oldObj))
	t.Cleanup(func() {
		// t.Context is no longer valid during cleanup
		assert.NoError(t, clt.Delete(context.Background(), oldObj))
	})

	// Sanity check to reduce chances of human error
	require.NotContains(t, seen, oldObj.GroupVersionKind().String(), "duplicate entry")
	seen.Insert(oldObj.GroupVersionKind().String())

	// Path forward: a served version is stored as the Hub (latest) version.
	latestObj := getConvertedObject(t, clt, oldObj, "latest-"+name+".yaml")
	require.NotEqual(t, oldObj.GroupVersionKind(), latestObj.GroupVersionKind())

	// Path backward: reading the object back in its served version exercises
	// ConvertFrom (Hub -> spoke), where silently dropped fields - like
	// spec.telemetryIngest used to be - would otherwise go unnoticed.
	backObj := getConvertedObject(t, clt, oldObj, version+"-"+name+".converted.yaml")
	require.Equal(t, oldObj.GroupVersionKind(), backObj.GroupVersionKind())
}

// getConvertedObject reads the stored object in the version declared by the
// golden file, clears server-managed fields and asserts it matches the file.
func getConvertedObject(t *testing.T, clt client.Client, srcObj *unstructured.Unstructured, expectedFile string) *unstructured.Unstructured {
	t.Helper()

	expectData, err := os.ReadFile(filepath.Join("testdata", expectedFile))
	require.NoError(t, err)

	expectObj := &unstructured.Unstructured{}
	require.NoError(t, yaml.Unmarshal(expectData, &expectObj.Object))

	gotObj := &unstructured.Unstructured{}
	gotObj.SetGroupVersionKind(expectObj.GroupVersionKind())

	require.NoError(t, clt.Get(t.Context(), client.ObjectKeyFromObject(srcObj), gotObj))
	// Clear server-side fields for comparison
	gotObj.SetCreationTimestamp(metav1.Time{})
	gotObj.SetGeneration(0)
	gotObj.SetResourceVersion("")
	gotObj.SetUID("")
	gotObj.SetManagedFields(nil)

	gotData, err := yaml.Marshal(gotObj)
	require.NoError(t, err)

	assert.Equal(t, string(expectData), string(gotData))

	return gotObj
}

func readTestData(t *testing.T, version, name string) *unstructured.Unstructured {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", version+"-"+name+".yaml"))
	require.NoError(t, err)

	// Use unstructured to
	// a) not duplicate conversion code and
	// b) simulate external tools like kubectl
	obj := &unstructured.Unstructured{}
	require.NoError(t, yaml.Unmarshal(data, &obj.Object))

	return obj
}
