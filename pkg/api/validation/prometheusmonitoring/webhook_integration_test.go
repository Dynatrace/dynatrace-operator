// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	validation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
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
					Name: "dynatrace-webhook",
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: "v1alpha1.prometheusmonitoring.webhook.dynatrace.com",
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{
									Path: new("/validate-dynatrace-com-v1alpha1-prometheusmonitoring"),
								},
							},
							Rules: []admissionregistrationv1.RuleWithOperations{
								{
									Operations: []admissionregistrationv1.OperationType{
										admissionregistrationv1.Create,
										admissionregistrationv1.Update,
									},
									APIGroups:   []string{"dynatrace.com"},
									APIVersions: []string{"v1alpha1"},
									Resources:   []string{"prometheusmonitorings"},
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
		pm := newTestPrometheusMonitoring("missing-crds")

		recorder.reset()
		integrationtests.CreateKubernetesObject(t, clt, pm)
		assertMissingCRDWarning(t, recorder.get())

		recorder.reset()
		pm.Spec.PublicRegistryOverride = "example.io/updated"
		require.NoError(t, clt.Update(t.Context(), pm))
		assertMissingCRDWarning(t, recorder.get())
	})

	t.Run("prometheus CRDs present", func(t *testing.T) {
		installMonitoringCRDs(t, clt)

		pm := newTestPrometheusMonitoring("crds-present")

		recorder.reset()
		integrationtests.CreateKubernetesObject(t, clt, pm)
		assert.Empty(t, recorder.get())

		recorder.reset()
		pm.Spec.PublicRegistryOverride = "example.io/updated"
		require.NoError(t, clt.Update(t.Context(), pm))
		assert.Empty(t, recorder.get())
	})

	t.Run("multiple PrometheusMonitorings in one namespace", func(t *testing.T) {
		// Prometheus CRDs were installed by the previous subtest.
		t.Run("distinct selectors do not warn", func(t *testing.T) {
			ns := newTestNamespace(t, clt, "distinct-selectors")

			for _, name := range []string{"pm-one", "pm-two"} {
				selector := &metav1.LabelSelector{MatchLabels: map[string]string{"pm": name}}

				recorder.reset()
				integrationtests.CreateKubernetesObject(t, clt, newTestPrometheusMonitoringInNamespace(name, ns, selector))
				assert.Empty(t, recorder.get())
			}
		})

		t.Run("same selector warns about overlap", func(t *testing.T) {
			ns := newTestNamespace(t, clt, "same-selector")
			selector := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "shared"}}

			recorder.reset()
			integrationtests.CreateKubernetesObject(t, clt, newTestPrometheusMonitoringInNamespace("pm-one", ns, selector))
			assert.Empty(t, recorder.get())

			recorder.reset()
			integrationtests.CreateKubernetesObject(t, clt, newTestPrometheusMonitoringInNamespace("pm-two", ns, selector.DeepCopy()))

			warnings := recorder.get()
			require.Len(t, warnings, 1)
			assert.Contains(t, warnings[0], "pm-one")
			assert.Contains(t, warnings[0], "customResourceSelector")
		})

		t.Run("same selector in different namespaces does not warn", func(t *testing.T) {
			selector := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "shared"}}

			for _, nsName := range []string{"other-ns-a", "other-ns-b"} {
				ns := newTestNamespace(t, clt, nsName)

				recorder.reset()
				integrationtests.CreateKubernetesObject(t, clt, newTestPrometheusMonitoringInNamespace("pm", ns, selector.DeepCopy()))
				assert.Empty(t, recorder.get())
			}
		})
	})

	t.Run("invalid names are rejected", func(t *testing.T) {
		ns := newTestNamespace(t, clt, "invalid-names")

		tests := []struct {
			name    string
			pmName  string
			message string
		}{
			{name: "too long", pmName: "a" + strings.Repeat("b", 44), message: "must not be longer than 44 characters"},
			{name: "starts with digit", pmName: "1-starts-with-digit", message: "DNS-1035"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pm := newTestPrometheusMonitoringInNamespace(tt.pmName, ns, nil)

				err := clt.Create(t.Context(), pm)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.message)
			})
		}

		t.Run("max length is accepted", func(t *testing.T) {
			recorder.reset()
			integrationtests.CreateKubernetesObject(t, clt, newTestPrometheusMonitoringInNamespace("a"+strings.Repeat("b", 43), ns, nil))
		})
	})
}

func newTestNamespace(t *testing.T, clt client.Client, name string) string {
	t.Helper()

	integrationtests.CreateKubernetesObject(t, clt, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})

	return name
}

func newTestPrometheusMonitoringInNamespace(name, namespace string, selector *metav1.LabelSelector) *prometheusmonitoring.PrometheusMonitoring {
	pm := newTestPrometheusMonitoring(name)
	pm.Namespace = namespace
	pm.Spec.TargetAllocator.CustomResourceSelector = selector

	return pm
}

func newTestPrometheusMonitoring(name string) *prometheusmonitoring.PrometheusMonitoring {
	return &prometheusmonitoring.PrometheusMonitoring{
		Name:      name,
		Namespace: metav1.NamespaceDefault,
		Spec: prometheusmonitoring.PrometheusMonitoringSpec{
			DynaKubeName: "test-dynakube",
		},
	}
}

func assertMissingCRDWarning(t *testing.T, warnings []string) {
	t.Helper()

	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "ServiceMonitor.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "PodMonitor.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "Probe.monitoring.coreos.com")
	assert.Contains(t, warnings[0], "ScrapeConfig.monitoring.coreos.com")
}

// installMonitoringCRDs registers minimal stand-ins for the Prometheus Operator CRDs
// required by the PrometheusMonitoring validator, so tests can exercise the "CRDs present" path
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
		Name: plural + ".monitoring.coreos.com",
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
