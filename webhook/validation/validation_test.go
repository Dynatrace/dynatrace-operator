package validation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testApiUrl    = "https://f.q.d.n/api"
)

func setupTestEnvironment(_ *testing.T) *envtest.Environment {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "default", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths:                    []string{filepath.Join("..", "config", "common", "webhook", "validation")},
			IgnoreErrorIfPathMissing: true,
		},
	}
	return testEnv
}

func TestAddDynakubeValidationWebhookToManager(t *testing.T) {
	testEnv := setupTestEnvironment(t)
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	require.NoError(t, err)
	require.NotNil(t, mgr)

	err = AddDynakubeValidationWebhookToManager(mgr)
	assert.NoError(t, err)
}

func TestDynakubeValidator_Handle(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		})

		assertAllowedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					CapabilityProperties: dynatracev1beta1.CapabilityProperties{
						Enabled: true,
					},
				},
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					CapabilityProperties: dynatracev1beta1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		})

		assertAllowedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.ActiveGateCapability{
						dynatracev1beta1.Routing,
						dynatracev1beta1.KubeMon,
						dynatracev1beta1.DataIngest,
					},
				},
			},
		})

	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
					HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, errorConflictingMode)

		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
					HostMonitoring:        &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, errorConflictingMode)

		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					CapabilityProperties: dynatracev1beta1.CapabilityProperties{
						Enabled: true,
					},
				},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.ActiveGateCapability{
						dynatracev1beta1.Routing,
					},
				},
			},
		}, errorConflictingMode)

		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.ActiveGateCapability{
						dynatracev1beta1.Routing,
						dynatracev1beta1.Routing,
					},
				},
			},
		}, errorConflictingMode)
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "",
			},
		}, errorNoApiUrl)
	})
	t.Run(`invalid API URL`, func(t *testing.T) {
		assertDeniedResponse(t, dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: exampleApiUrl,
			},
		}, errorNoApiUrl)
	})
}

func assertDeniedResponse(t *testing.T, dynakube dynatracev1beta1.DynaKube, reason string) {
	response := handleRequest(t, dynakube)
	assert.False(t, response.Allowed)
	assert.Equal(t, metav1.StatusReason(reason), response.Result.Reason)
}

func assertAllowedResponse(t *testing.T, dynakube dynatracev1beta1.DynaKube) {
	response := handleRequest(t, dynakube)
	assert.True(t, response.Allowed)
}

func handleRequest(t *testing.T, dynakube dynatracev1beta1.DynaKube) admission.Response {
	clt := fake.NewClient()
	validator := &dynakubeValidator{
		logger: logger.NewDTLogger(),
		clt:    clt,
	}

	data, err := json.Marshal(dynakube)
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
