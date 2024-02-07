package dynakube

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testApiUrl    = "https://f.q.d.n/api"
)

var defaultDynakubeObjectMeta = metav1.ObjectMeta{
	Name:      testName,
	Namespace: testNamespace,
}

var defaultCSIDaemonSet = appsv1.DaemonSet{
	ObjectMeta: metav1.ObjectMeta{Name: dtcsi.DaemonSetName, Namespace: testNamespace},
}

var dummyLabels = map[string]string{
	"dummy": "label",
}

var dummyNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "dummy",
		Labels: dummyLabels,
	},
}

var dummyLabels2 = map[string]string{
	"dummy": "label",
}

var dummyNamespace2 = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "dummy2",
		Labels: dummyLabels2,
	},
}

func TestDynakubeValidator_Handle(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 1, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				NamespaceSelector: metav1.LabelSelector{
					MatchLabels: dummyLabels,
				},
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
					},
				},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
						dynatracev1beta1.KubeMonCapability.DisplayName,
						dynatracev1beta1.MetricsIngestCapability.DisplayName,
					},
				},
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: false,
				},
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: false,
				},
			},
		},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels2,
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
						},
					},
				},
			}, &dummyNamespace, &dummyNamespace2, &defaultCSIDaemonSet)
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{
				errorCSIRequired,
				errorNoApiUrl,
				errorConflictingActiveGateSections,
				errorConflictingNamespaceSelector,
				fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1beta1.KubeMonCapability.DisplayName),
				fmt.Sprintf(errorInvalidActiveGateCapability, "me dumb"),
				fmt.Sprintf(errorNodeSelectorConflict, "conflict2")},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: "",
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
					},
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.KubeMonCapability.DisplayName,
							dynatracev1beta1.KubeMonCapability.DisplayName,
							"me dumb",
						},
					},
					KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
						Enabled: true,
					},
					Routing: dynatracev1beta1.RoutingSpec{
						Enabled: true,
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict1",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: dummyLabels,
					},
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict2",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
					},
				},
			}, &dummyNamespace, &dummyNamespace2)
	})
}

func assertDeniedResponse(t *testing.T, errMessages []string, dynakube *dynatracev1beta1.DynaKube, other ...client.Object) {
	response := handleRequest(t, dynakube, other...)
	assert.False(t, response.Allowed)

	message := response.Result.Message
	for _, errMsg := range errMessages {
		assert.Contains(t, message, errMsg)
	}
}

func assertAllowedResponseWithoutWarnings(t *testing.T, dynakube *dynatracev1beta1.DynaKube, other ...client.Object) {
	response := assertAllowedResponse(t, dynakube, other...)
	assert.Empty(t, response.Warnings)
}

func assertAllowedResponseWithWarnings(t *testing.T, warningAmount int, dynakube *dynatracev1beta1.DynaKube, other ...client.Object) {
	response := assertAllowedResponse(t, dynakube, other...)
	assert.Len(t, response.Warnings, warningAmount)
}

func assertAllowedResponse(t *testing.T, dynakube *dynatracev1beta1.DynaKube, other ...client.Object) admission.Response {
	response := handleRequest(t, dynakube, other...)
	assert.True(t, response.Allowed)

	return response
}

func handleRequest(t *testing.T, dynakube *dynatracev1beta1.DynaKube, other ...client.Object) admission.Response {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &dynakubeValidator{
		clt:       clt,
		apiReader: clt,
		cfg:       &rest.Config{},
	}

	data, err := json.Marshal(*dynakube)
	require.NoError(t, err)

	return validator.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      testName,
			Namespace: testNamespace,
			Object:    runtime.RawExtension{Raw: data},
		},
	})
}
