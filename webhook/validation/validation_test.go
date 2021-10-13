package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testApiUrl    = "https://f.q.d.n/api"
)

func TestDynakubeValidator_Handle(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   nil,
				},
			},
		}, nil)

		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
					HostMonitoring:   nil,
				},
			},
		}, nil)

		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, nil)

		assertAllowedResponse(t,
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
						},
					},
				},
			})

		assertAllowedResponse(t,
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
						},
					},
				},
			})

		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: true,
				},
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				},
			},
		}, nil)

		assertAllowedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
						dynatracev1beta1.KubeMonCapability.DisplayName,
						dynatracev1beta1.DataIngestCapability.DisplayName,
					},
				},
			},
		}, nil)

	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
					HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, nil, errorConflictingOneagentMode)

		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					HostMonitoring:        &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, nil, errorConflictingOneagentMode)

		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: true,
				},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
					},
				},
			},
		}, nil, errorConflictingActiveGateSections)

		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
						dynatracev1beta1.RoutingCapability.DisplayName,
					},
				},
			},
		}, nil, fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1beta1.RoutingCapability.DisplayName))

		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						"invalid-capability",
					},
				},
			},
		}, nil, fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability"))

		assertDeniedResponse(t,
			&dynatracev1beta1.DynaKube{
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: "conflicting-dk",
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			}, fmt.Sprintf(errorNodeSelectorConflict, "conflicting-dk"))
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "",
			},
		}, nil, errorNoApiUrl)
	})
	t.Run(`invalid API URL`, func(t *testing.T) {
		assertDeniedResponse(t, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: exampleApiUrl,
			},
		}, nil, errorNoApiUrl)
	})
}

func assertDeniedResponse(t *testing.T, dynakube, other *dynatracev1beta1.DynaKube, reason string) {
	response := handleRequest(t, dynakube, other)
	assert.False(t, response.Allowed)
	assert.Equal(t, metav1.StatusReason(reason), response.Result.Reason)
}

func assertAllowedResponse(t *testing.T, dynakube, other *dynatracev1beta1.DynaKube) {
	response := handleRequest(t, dynakube, other)
	assert.True(t, response.Allowed)
}

func handleRequest(t *testing.T, dynakube, other *dynatracev1beta1.DynaKube) admission.Response {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other)
	}
	validator := &dynakubeValidator{
		logger: logger.NewDTLogger(),
		clt:    clt,
	}

	data, err := json.Marshal(*dynakube)
	require.NoError(t, err)

	return validator.Handle(context.TODO(), admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Name:      testName,
			Namespace: testNamespace,
			Object:    runtime.RawExtension{Raw: data},
		},
	})
}

func TestDynakubeValidator_InjectClient(t *testing.T) {
	validator := &dynakubeValidator{}
	clt := fake.NewClient()
	err := validator.InjectClient(clt)

	assert.NoError(t, err)
	assert.NotNil(t, validator.clt)
	assert.Equal(t, clt, validator.clt)
}

func TestHasApiUrl(t *testing.T) {
	instance := &dynatracev1beta1.DynaKube{}
	assert.False(t, hasApiUrl(instance))

	instance.Spec.APIURL = testApiUrl
	assert.True(t, hasApiUrl(instance))
}
