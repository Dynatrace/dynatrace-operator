package pod_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	otlpspec "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	podmutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/otlp"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metadatamutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	oneagentmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const testNamespace = "dynatrace"

func TestWebhook(t *testing.T) {
	clt := integrationtests.SetupWebhookTestEnvironment(t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					Labels: map[string]string{
						podmutator.InjectionInstanceLabel: "dynakube",
					},
				},
			}
			require.NoError(t, mgr.GetClient().Create(t.Context(), ns))

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynatace-webhook",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  dtwebhook.WebhookContainerName,
							Image: "dummy-webhook-img:1.0.0",
						},
					},
				},
			}
			require.NoError(t, mgr.GetClient().Create(t.Context(), pod))
			t.Setenv(env.PodName, pod.Name)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	// shared between test cases
	bootstrapperSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: testNamespace,
		},
	}
	createObject(t, clt, bootstrapperSecret)

	otlpExporterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OTLPExporterSecretName,
			Namespace: testNamespace,
		},
	}
	createObject(t, clt, otlpExporterSecret)

	t.Run("success", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: uuid.NewString(),
						},
					},
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.True(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationDynatraceInjected, false))
		assert.True(t, maputils.GetFieldBool(pod.Annotations, metadatamutator.AnnotationInjected, false))
		assert.True(t, maputils.GetFieldBool(pod.Annotations, oneagentmutator.AnnotationInjected, false))
	})

	t.Run("oneagent mutator failure", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[oneagentmutator.AnnotationInject] = "true"
		})

		assert.Contains(t, pod.Annotations, oneagentmutator.AnnotationReason)
	})

	t.Run("metadata mutator failure", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations[metadatamutator.AnnotationInject] = "true"
			pod.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "missing",
					UID:        types.UID(uuid.NewString()),
					Controller: ptr.To(true),
				},
			}
		})

		assert.Contains(t, pod.Annotations, metadatamutator.AnnotationReason)
	})
}

func TestOTLPWebhook(t *testing.T) {
	metadataAnnotations := map[string]string{
		"metadata.dynatrace.com/service.name": "checkout service",
		"metadata.dynatrace.com/custom.key":   "value:with/special chars",
	}

	clt := integrationtests.SetupWebhookTestEnvironment(t,
		getWebhookInstallOptions(),

		func(mgr ctrl.Manager) error {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					Labels: map[string]string{
						podmutator.InjectionInstanceLabel: "dynakube",
					},
				},
			}
			require.NoError(t, mgr.GetClient().Create(t.Context(), ns))

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynatace-webhook",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  dtwebhook.WebhookContainerName,
							Image: "dummy-webhook-img:1.0.0",
						},
					},
				},
			}
			require.NoError(t, mgr.GetClient().Create(t.Context(), pod))
			t.Setenv(env.PodName, pod.Name)

			return podmutation.AddWebhookToManager(t.Context(), mgr, testNamespace, false)
		},
	)

	t.Run("otlp exporter", func(t *testing.T) {
		const dataIngestToken = "test-token"
		apiURL := "https://example.live.dynatrace.com"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{ // match test namespace label applied earlier
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
		}

		apiTokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynatrace.APIToken:        []byte(dataIngestToken),
				dynatrace.DataIngestToken: []byte(dataIngestToken),
			},
		}
		createObject(t, clt, apiTokenSecret)

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, func(pod *corev1.Pod) {
			pod.Annotations = metadataAnnotations
		})

		// verify mutation occurred by presence of OTLP env vars (annotation may not be set when no OneAgent injection)

		appContainer := pod.Spec.Containers[0]
		// Expect DT_API_TOKEN env var via secret ref
		var dtTokenEnv *corev1.EnvVar
		for i := range appContainer.Env {
			if appContainer.Env[i].Name == exporter.DynatraceAPITokenEnv {
				dtTokenEnv = &appContainer.Env[i]
				break
			}
		}

		require.NotNil(t, dtTokenEnv, "expected DT_API_TOKEN env var to be injected")
		require.NotNil(t, dtTokenEnv.ValueFrom)
		require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		// Headers env vars should reference DT_API_TOKEN via authorization header literal
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPMetricsHeadersEnv, Value: exporter.OTLPAuthorizationHeader})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPLogsHeadersEnv, Value: exporter.OTLPAuthorizationHeader})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPTraceHeadersEnv, Value: exporter.OTLPAuthorizationHeader})

		// Endpoint base constructed by BuildOTLPEndpoint(apiURL) => apiURL + /v2/otlp plus per-signal suffix
		baseEndpoint := apiURL + "/v2/otlp"
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPMetricsEndpointEnv, Value: baseEndpoint + "/v1/metrics"})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPLogsEndpointEnv, Value: baseEndpoint + "/v1/logs"})
		assert.Contains(t, appContainer.Env, corev1.EnvVar{Name: exporter.OTLPTraceEndpointEnv, Value: baseEndpoint + "/v1/traces"})

		raEnv := env.FindEnvVar(appContainer.Env, resourceattributes.OTELResourceAttributesEnv)

		require.NotNil(t, raEnv, "OTEL_RESOURCE_ATTRIBUTES missing")

		parsed := parseResourceAttributes(raEnv.Value)

		assert.Equal(t, url.QueryEscape(metadataAnnotations["metadata.dynatrace.com/service.name"]), parsed["service.name"])
		assert.Equal(t, url.QueryEscape(metadataAnnotations["metadata.dynatrace.com/custom.key"]), parsed["custom.key"])
		assert.Equal(t, testNamespace, parsed["k8s.namespace.name"])
		assert.Equal(t, "app", parsed["k8s.container.name"])
	})

	t.Run("data ingest token secret missing", func(t *testing.T) {
		apiURL := "https://example.live.dynatrace.com"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{ // match test namespace label applied earlier
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
		}

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
		assert.Equal(t, otlp.NoOTLPExporterConfigSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})

	t.Run("otlp exporter activegate", func(t *testing.T) {
		const dataIngestToken = "test-token"
		const agCertData = "ag-cert-data"

		apiURL := "https://example.live.dynatrace.com"
		tenantUUID := uuid.NewString()

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: tenantUUID,
						},
					},
				},
			},
		}

		apiTokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynatrace.APIToken:        []byte(dataIngestToken),
				dynatrace.DataIngestToken: []byte(dataIngestToken),
			},
		}
		createObject(t, clt, apiTokenSecret)

		agCertSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterCertsSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynakube.TLSCertKey: []byte(agCertData),
			},
		}
		createObject(t, clt, agCertSecret)

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)
		appContainer := pod.Spec.Containers[0]

		envMap := map[string]corev1.EnvVar{}
		for _, e := range appContainer.Env {
			envMap[e.Name] = e
		}

		dtTokenEnv, ok := envMap[exporter.DynatraceAPITokenEnv]
		require.True(t, ok, "DT_API_TOKEN missing")
		require.NotNil(t, dtTokenEnv.ValueFrom)
		require.NotNil(t, dtTokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenEnv.ValueFrom.SecretKeyRef.Key)

		expectedService := fmt.Sprintf("%s-%s.%s", dk.Name, agconsts.MultiActiveGateName, testNamespace)
		expectedBase := fmt.Sprintf("https://%s/e/%s/api/v2/otlp", expectedService, tenantUUID)

		assert.Equal(t, expectedBase+"/v1/metrics", envMap[exporter.OTLPMetricsEndpointEnv].Value)
		assert.Equal(t, expectedBase+"/v1/logs", envMap[exporter.OTLPLogsEndpointEnv].Value)
		assert.Equal(t, expectedBase+"/v1/traces", envMap[exporter.OTLPTraceEndpointEnv].Value)
	})

	t.Run("otlp exporter activegate - certificate secret missing", func(t *testing.T) {
		const dataIngestToken = "test-token"

		apiURL := "https://example.live.dynatrace.com"
		tenantUUID := uuid.NewString()

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.InjectionAutomaticKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.RoutingCapability.DisplayName,
					},
				},
				OTLPExporterConfiguration: &otlpspec.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: podmutator.InjectionInstanceLabel, Operator: metav1.LabelSelectorOpExists},
						},
					},
					Signals: otlpspec.SignalConfiguration{
						Metrics: &otlpspec.MetricsSignal{},
						Logs:    &otlpspec.LogsSignal{},
						Traces:  &otlpspec.TracesSignal{},
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
						ConnectionInfo: communication.ConnectionInfo{
							TenantUUID: tenantUUID,
						},
					},
				},
			},
		}

		apiTokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OTLPExporterSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				dynatrace.APIToken:        []byte(dataIngestToken),
				dynatrace.DataIngestToken: []byte(dataIngestToken),
			},
		}
		createObject(t, clt, apiTokenSecret)

		createDynaKube(t, clt, dk)

		pod := createPod(t, clt, nil)

		assert.False(t, maputils.GetFieldBool(pod.Annotations, podmutator.AnnotationOTLPInjected, false))
		assert.Equal(t, otlp.NoOTLPExporterActiveGateCertSecretReason, pod.Annotations[podmutator.AnnotationOTLPReason])
	})
}

func getWebhookInstallOptions() envtest.WebhookInstallOptions {
	return envtest.WebhookInstallOptions{
		MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
			// TODO(avorima): Load this from a file using Paths
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace-webhook",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name:               "webhook.pod.dynatrace.com",
						ReinvocationPolicy: ptr.To(admissionv1.IfNeededReinvocationPolicy),
						FailurePolicy:      ptr.To(admissionv1.Ignore),
						TimeoutSeconds:     ptr.To[int32](10),
						Rules: []admissionv1.RuleWithOperations{
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{""},
									APIVersions: []string{"v1"},
									Resources:   []string{"pods"},
									Scope:       ptr.To(admissionv1.NamespacedScope),
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
								},
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      podmutator.InjectionInstanceLabel,
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
						ClientConfig: admissionv1.WebhookClientConfig{
							Service: &admissionv1.ServiceReference{
								Name: "dynatrace-webhook",
								Path: ptr.To("/inject"),
							},
						},
						AdmissionReviewVersions: []string{"v1beta1", "v1"},
						SideEffects:             ptr.To(admissionv1.SideEffectClassNone),
					},
				},
			},
		},
	}
}

func createPod(t *testing.T, clt client.Client, mutateFn func(*corev1.Pod)) *corev1.Pod {
	t.Helper()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "pod-inject-test",
			Namespace:   testNamespace,
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:            "app",
					Image:           "docker.io/myapp:1.2.3",
					ImagePullPolicy: corev1.PullAlways,
				},
			},
		},
	}

	if mutateFn != nil {
		mutateFn(pod)
	}

	createObject(t, clt, pod)

	return pod
}

func createObject(t *testing.T, clt client.Client, obj client.Object) {
	t.Helper()
	require.NoError(t, clt.Create(t.Context(), obj))
	t.Cleanup(func() {
		// t.Context is no longer valid during cleanup
		assert.NoError(t, clt.Delete(context.Background(), obj))
	})
}

func createDynaKube(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	status := dk.Status
	createObject(t, clt, dk)
	dk.Status = status
	dk.UpdateStatus(t.Context(), clt)
}

func parseResourceAttributes(value string) map[string]string {
	res := map[string]string{}
	for _, p := range strings.Split(value, ",") {
		p = strings.TrimSpace(p)
		if p == "" || !strings.Contains(p, "=") {
			continue
		}
		key, val, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		if key != "" && val != "" {
			res[key] = val
		}
	}
	return res
}
