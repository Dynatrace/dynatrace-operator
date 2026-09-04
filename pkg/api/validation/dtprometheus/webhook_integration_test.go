// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	validation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestWebhook(t *testing.T) {
	recorder := &warningRecorder{}

	clt := integrationtests.SetupWebhookTestEnvironment(t,
		envtest.WebhookInstallOptions{
			ValidatingWebhooks: []*admissionregistrationv1.ValidatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dynatrace-webhook",
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: "v1alpha1.dtprometheus.webhook.dynatrace.com",
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{
									Path: new("/validate-dynatrace-com-v1alpha1-dtprometheus"),
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
										APIVersions: []string{"v1alpha1"},
										Resources:   []string{"dtprometheuses"},
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
		func(cfg *rest.Config) {
			cfg.WarningHandlerWithContext = recorder
		},
	)

	t.Run("prometheus CRDs missing", func(t *testing.T) {
		dtp := newTestDTPrometheus("missing-crds")

		recorder.reset()
		integrationtests.CreateKubernetesObject(t, clt, dtp)
		assertMissingCRDWarning(t, recorder.get())

		recorder.reset()
		dtp.Spec.PublicRegistryOverride = "example.io/updated"
		require.NoError(t, clt.Update(t.Context(), dtp))
		assertMissingCRDWarning(t, recorder.get())
	})

	t.Run("prometheus CRDs present", func(t *testing.T) {
		installMonitoringCRDs(t, clt)

		dtp := newTestDTPrometheus("crds-present")

		recorder.reset()
		integrationtests.CreateKubernetesObject(t, clt, dtp)
		assert.Empty(t, recorder.get())

		recorder.reset()
		dtp.Spec.PublicRegistryOverride = "example.io/updated"
		require.NoError(t, clt.Update(t.Context(), dtp))
		assert.Empty(t, recorder.get())
	})
}

func newTestDTPrometheus(name string) *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: dtprometheus.DTPrometheusSpec{
			DynaKubeName: "test-dynakube",
		},
	}
}

func assertMissingCRDWarning(t *testing.T, warnings []string) {
	t.Helper()

	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "servicemonitors.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "podmonitors.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "probes.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "scrapeconfigs.monitoring.coreos.com")
}

// installMonitoringCRDs registers minimal stand-ins for the Prometheus Operator CRDs
// required by the DTPrometheus validator, so tests can exercise the "CRDs present" path
// without depending on the real prometheus-operator CRDs being vendored into the repo.
func installMonitoringCRDs(t *testing.T, clt client.Client) {
	t.Helper()

	crds := []*apiextensionsv1.CustomResourceDefinition{
		newMonitoringCRD("ServiceMonitor", "servicemonitors", "v1"),
		newMonitoringCRD("PodMonitor", "podmonitors", "v1"),
		newMonitoringCRD("Probe", "probes", "v1"),
		newMonitoringCRD("ScrapeConfig", "scrapeconfigs", "v1alpha1"),
	}

	for _, crd := range crds {
		integrationtests.CreateKubernetesObject(t, clt, crd)

		waitForCRDEstablished(t, clt, crd.Name)
	}
}

func newMonitoringCRD(kind, plural, version string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: plural + ".monitoring.coreos.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "monitoring.coreos.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: plural,
				Kind:   kind,
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    version,
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type:                   "object",
							XPreserveUnknownFields: new(true),
						},
					},
				},
			},
		},
	}
}

func waitForCRDEstablished(t *testing.T, clt client.Client, name string) {
	t.Helper()

	err := wait.PollUntilContextTimeout(t.Context(), 100*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := clt.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
	require.NoError(t, err)
}

// warningRecorder implements rest.WarningHandlerWithContext and collects admission
// warnings returned by the API server so tests can assert on them directly. It is
// installed on the REST config before the client is built, since controller-runtime's
// client.New installs its own logging handler whenever none is set.
type warningRecorder struct {
	mu       sync.Mutex
	warnings []string
}

func (w *warningRecorder) HandleWarningHeaderWithContext(_ context.Context, _ int, _ string, message string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.warnings = append(w.warnings, message)
}

func (w *warningRecorder) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.warnings = nil
}

func (w *warningRecorder) get() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return append([]string(nil), w.warnings...)
}
