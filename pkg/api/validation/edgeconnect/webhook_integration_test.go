package validation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	validation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"
)

func TestWebhook(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		envtest.WebhookInstallOptions{
			// TODO(avorima): Load this from a file using Paths
			ValidatingWebhooks: []*admissionv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dynatrace-webhook",
					},
					Webhooks: []admissionv1.ValidatingWebhook{
						{
							Name: "v1alpha1.edgeconnect.webhook.dynatrace.com",
							ClientConfig: admissionv1.WebhookClientConfig{
								Service: &admissionv1.ServiceReference{
									Path: ptr.To("/validate-dynatrace-com-v1alpha1-edgeconnect"),
								},
							},
							Rules: []admissionv1.RuleWithOperations{
								{
									Operations: []admissionv1.OperationType{
										admissionv1.Create,
										admissionv1.Update,
									},
									Rule: admissionv1.Rule{
										APIGroups:   []string{"dynatrace.com"},
										APIVersions: []string{"v1alpha1"},
										Resources:   []string{"edgeconnects"},
									},
								},
							},
							MatchPolicy:             ptr.To(admissionv1.Exact),
							SideEffects:             ptr.To(admissionv1.SideEffectClassNone),
							TimeoutSeconds:          ptr.To[int32](10),
							AdmissionReviewVersions: []string{"v1"},
						},
						{
							Name: "v1alpha2.edgeconnect.webhook.dynatrace.com",
							ClientConfig: admissionv1.WebhookClientConfig{
								Service: &admissionv1.ServiceReference{
									Path: ptr.To("/validate-dynatrace-com-v1alpha2-edgeconnect"),
								},
							},
							Rules: []admissionv1.RuleWithOperations{
								{
									Operations: []admissionv1.OperationType{
										admissionv1.Create,
										admissionv1.Update,
									},
									Rule: admissionv1.Rule{
										APIGroups:   []string{"dynatrace.com"},
										APIVersions: []string{"v1alpha2"},
										Resources:   []string{"edgeconnects"},
									},
								},
							},
							MatchPolicy:             ptr.To(admissionv1.Exact),
							SideEffects:             ptr.To(admissionv1.SideEffectClassNone),
							TimeoutSeconds:          ptr.To[int32](10),
							AdmissionReviewVersions: []string{"v1"},
						},
					},
				},
			},
		},
		validation.SetupWebhookWithManager,
	)

	versions := []string{
		"v1alpha1",
	}
	seenGVKs := sets.New[string]()

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			compareWebhookResult(t, clt, version, "default", seenGVKs)
		})
	}
}

func compareWebhookResult(t *testing.T, clt client.Client, version, name string, seen sets.Set[string]) {
	t.Helper()
	oldData, err := os.ReadFile(filepath.Join("testdata", version+"-"+name+".yaml"))
	require.NoError(t, err)

	// Use unstructured to
	// a) not duplicate conversion code and
	// b) simulate external tools like kubectl
	oldObj := &unstructured.Unstructured{}
	require.NoError(t, yaml.Unmarshal(oldData, &oldObj.Object))

	require.NoError(t, clt.Create(t.Context(), oldObj))
	t.Cleanup(func() {
		// t.Context is no longer valid during cleanup
		assert.NoError(t, clt.Delete(context.Background(), oldObj))
	})

	expectData, err := os.ReadFile(filepath.Join("testdata", "latest-"+name+".yaml"))
	require.NoError(t, err)

	expectObj := &unstructured.Unstructured{}
	require.NoError(t, yaml.Unmarshal(expectData, &expectObj.Object))

	// Sanity checks to reduce chances of human error
	require.NotEqual(t, expectObj.GroupVersionKind(), oldObj.GroupVersionKind())
	require.NotContains(t, seen, oldObj.GroupVersionKind().String(), "duplicate entry")
	seen.Insert(oldObj.GroupVersionKind().String())

	gotObj := &unstructured.Unstructured{}
	gotObj.SetGroupVersionKind(expectObj.GroupVersionKind())

	require.NoError(t, clt.Get(t.Context(), client.ObjectKeyFromObject(oldObj), gotObj))
	// Clear server-side fields for comparison
	gotObj.SetCreationTimestamp(metav1.Time{})
	gotObj.SetGeneration(0)
	gotObj.SetResourceVersion("")
	gotObj.SetUID("")
	gotObj.SetManagedFields(nil)

	gotData, err := yaml.Marshal(gotObj)
	require.NoError(t, err)

	assert.Equal(t, string(expectData), string(gotData))
}
