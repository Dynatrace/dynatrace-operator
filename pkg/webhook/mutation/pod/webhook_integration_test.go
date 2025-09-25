package pod_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	podmutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metadatamutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	oneagentmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
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
		envtest.WebhookInstallOptions{
			// TODO(avorima): Load this from a file using Paths
			MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
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
		},

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
